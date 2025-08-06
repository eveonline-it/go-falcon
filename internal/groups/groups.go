package groups

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"go-falcon/internal/auth"
	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
	"go-falcon/pkg/handlers"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Module struct {
	*module.BaseModule
	permissionService *PermissionService
	groupService      *GroupService
	authModule        *auth.Module
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService, authModule *auth.Module) *Module {
	groupService := NewGroupService(mongodb)
	permissionService := NewPermissionService(mongodb, redis, groupService)
	
	return &Module{
		BaseModule:        module.NewBaseModule("groups", mongodb, redis, sdeService),
		permissionService: permissionService,
		groupService:      groupService,
		authModule:        authModule,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r)
	
	// Group management endpoints
	r.Route("/groups", func(r chi.Router) {
		r.With(m.authModule.OptionalJWTMiddleware).Get("/", m.listGroupsHandler)
		r.With(m.authModule.JWTMiddleware, m.RequirePermission("groups", "admin")).Post("/", m.createGroupHandler)
		r.With(m.authModule.JWTMiddleware, m.RequirePermission("groups", "admin")).Put("/{groupID}", m.updateGroupHandler)
		r.With(m.authModule.JWTMiddleware, m.RequirePermission("groups", "admin")).Delete("/{groupID}", m.deleteGroupHandler)
		
		// Membership management
		r.Route("/{groupID}/members", func(r chi.Router) {
			r.Use(m.authModule.JWTMiddleware, m.RequirePermission("groups", "admin"))
			r.Get("/", m.listMembersHandler)
			r.Post("/", m.addMemberHandler)
			r.Delete("/{characterID}", m.removeMemberHandler)
		})
	})
	
	// Permission checking endpoints
	r.Route("/permissions", func(r chi.Router) {
		r.Use(m.authModule.OptionalJWTMiddleware)
		r.Get("/check", m.checkPermissionHandler)
		r.Get("/user", m.getUserPermissionsHandler)
	})
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting groups-specific background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Initialize default groups
	go m.initializeDefaultGroups(ctx)
	
	// Start membership validation routine
	go m.runMembershipValidation(ctx)
	
	// Add groups-specific background processing here
	for {
		select {
		case <-ctx.Done():
			slog.Info("Groups background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Groups background tasks stopped")
			return
		default:
			// Groups-specific background work would go here
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}

func (m *Module) initializeDefaultGroups(ctx context.Context) {
	slog.Info("Initializing default groups")
	
	if err := m.groupService.InitializeDefaultGroups(ctx); err != nil {
		slog.Error("Failed to initialize default groups", slog.String("error", err.Error()))
	} else {
		slog.Info("Default groups initialized successfully")
	}
}

func (m *Module) runMembershipValidation(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Validate every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Group membership validation stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Group membership validation stopped")
			return
		case <-ticker.C:
			if err := m.validateCorporateMemberships(ctx); err != nil {
				slog.Error("Failed to validate corporate memberships", slog.String("error", err.Error()))
			}
		}
	}
}

func (m *Module) validateCorporateMemberships(ctx context.Context) error {
	slog.Info("Starting corporate membership validation")
	
	// This will be implemented as part of the scheduler integration
	// For now, just log that validation would happen here
	slog.Info("Corporate membership validation completed")
	return nil
}

// Handler methods

func (m *Module) listGroupsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "groups.list",
		attribute.String("service", "groups"),
		attribute.String("operation", "list_groups"),
	)
	defer span.End()

	user, authenticated := auth.GetAuthenticatedUser(r)
	
	groups, err := m.groupService.ListGroups(r.Context(), authenticated)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list groups")
		slog.Error("Failed to list groups", slog.String("error", err.Error()))
		http.Error(w, "Failed to list groups", http.StatusInternalServerError)
		return
	}

	// Add user's membership status if authenticated
	if authenticated {
		for i := range groups {
			isMember, err := m.groupService.IsUserMember(r.Context(), user.CharacterID, groups[i].ID.Hex())
			if err != nil {
				slog.Warn("Failed to check membership status", 
					slog.String("error", err.Error()),
					slog.String("group", groups[i].Name))
				continue
			}
			groups[i].IsMember = isMember
		}
	}

	span.SetAttributes(
		attribute.Int("groups.count", len(groups)),
		attribute.Bool("user.authenticated", authenticated),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"groups": groups,
		"count":  len(groups),
	})
}

func (m *Module) createGroupHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "groups.create",
		attribute.String("service", "groups"),
		attribute.String("operation", "create_group"),
	)
	defer span.End()

	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		span.SetStatus(codes.Error, "Invalid group data")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	group, err := m.groupService.CreateGroup(r.Context(), &req, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create group")
		slog.Error("Failed to create group", 
			slog.String("error", err.Error()),
			slog.String("name", req.Name),
			slog.Int("character_id", user.CharacterID))
		http.Error(w, "Failed to create group", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("group.id", group.ID.Hex()),
		attribute.String("group.name", group.Name),
		attribute.Bool("group.success", true),
	)

	slog.Info("Group created successfully",
		slog.String("id", group.ID.Hex()),
		slog.String("name", group.Name),
		slog.Int("created_by", user.CharacterID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

func (m *Module) updateGroupHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "groups.update",
		attribute.String("service", "groups"),
		attribute.String("operation", "update_group"),
	)
	defer span.End()

	groupID := chi.URLParam(r, "groupID")
	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	group, err := m.groupService.UpdateGroup(r.Context(), groupID, &req, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update group")
		slog.Error("Failed to update group", 
			slog.String("error", err.Error()),
			slog.String("group_id", groupID))
		http.Error(w, "Failed to update group", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("group.id", groupID),
		attribute.Bool("group.success", true),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
}

func (m *Module) deleteGroupHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "groups.delete",
		attribute.String("service", "groups"),
		attribute.String("operation", "delete_group"),
	)
	defer span.End()

	groupID := chi.URLParam(r, "groupID")
	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	err := m.groupService.DeleteGroup(r.Context(), groupID, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete group")
		slog.Error("Failed to delete group", 
			slog.String("error", err.Error()),
			slog.String("group_id", groupID))
		
		if err.Error() == "cannot delete default group" {
			http.Error(w, "Cannot delete default group", http.StatusForbidden)
			return
		}
		
		http.Error(w, "Failed to delete group", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("group.id", groupID),
		attribute.Bool("group.success", true),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Group deleted successfully",
	})
}

func (m *Module) listMembersHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "groups.members.list",
		attribute.String("service", "groups"),
		attribute.String("operation", "list_members"),
	)
	defer span.End()

	groupID := chi.URLParam(r, "groupID")

	members, err := m.groupService.ListGroupMembers(r.Context(), groupID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list members")
		slog.Error("Failed to list group members", 
			slog.String("error", err.Error()),
			slog.String("group_id", groupID))
		http.Error(w, "Failed to list members", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("group.id", groupID),
		attribute.Int("members.count", len(members)),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"members": members,
		"count":   len(members),
	})
}

func (m *Module) addMemberHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "groups.members.add",
		attribute.String("service", "groups"),
		attribute.String("operation", "add_member"),
	)
	defer span.End()

	groupID := chi.URLParam(r, "groupID")
	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CharacterID == 0 {
		span.SetStatus(codes.Error, "Missing character ID")
		http.Error(w, "Character ID is required", http.StatusBadRequest)
		return
	}

	membership, err := m.groupService.AddGroupMember(r.Context(), groupID, req.CharacterID, user.CharacterID, req.ExpiresAt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to add member")
		slog.Error("Failed to add group member", 
			slog.String("error", err.Error()),
			slog.String("group_id", groupID),
			slog.Int("character_id", req.CharacterID))
		http.Error(w, "Failed to add member", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("group.id", groupID),
		attribute.Int("character.id", req.CharacterID),
		attribute.Bool("membership.success", true),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(membership)
}

func (m *Module) removeMemberHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "groups.members.remove",
		attribute.String("service", "groups"),
		attribute.String("operation", "remove_member"),
	)
	defer span.End()

	groupID := chi.URLParam(r, "groupID")
	characterIDStr := chi.URLParam(r, "characterID")
	
	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		span.SetStatus(codes.Error, "Invalid character ID")
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	err = m.groupService.RemoveGroupMember(r.Context(), groupID, characterID, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to remove member")
		slog.Error("Failed to remove group member", 
			slog.String("error", err.Error()),
			slog.String("group_id", groupID),
			slog.Int("character_id", characterID))
		http.Error(w, "Failed to remove member", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("group.id", groupID),
		attribute.Int("character.id", characterID),
		attribute.Bool("removal.success", true),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Member removed successfully",
	})
}

func (m *Module) checkPermissionHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "groups.permissions.check",
		attribute.String("service", "groups"),
		attribute.String("operation", "check_permission"),
	)
	defer span.End()

	resource := r.URL.Query().Get("resource")
	action := r.URL.Query().Get("action")

	if resource == "" || action == "" {
		span.SetStatus(codes.Error, "Missing parameters")
		http.Error(w, "Resource and action parameters are required", http.StatusBadRequest)
		return
	}

	var characterID int
	user, authenticated := auth.GetAuthenticatedUser(r)
	if authenticated {
		characterID = user.CharacterID
	}

	allowed, groups, err := m.permissionService.CheckPermission(r.Context(), characterID, resource, action)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to check permission")
		slog.Error("Failed to check permission", 
			slog.String("error", err.Error()),
			slog.String("resource", resource),
			slog.String("action", action))
		http.Error(w, "Failed to check permission", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("permission.resource", resource),
		attribute.String("permission.action", action),
		attribute.Bool("permission.allowed", allowed),
		attribute.Bool("user.authenticated", authenticated),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed": allowed,
		"groups":  groups,
	})
}

func (m *Module) getUserPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "groups.permissions.user",
		attribute.String("service", "groups"),
		attribute.String("operation", "get_user_permissions"),
	)
	defer span.End()

	user, authenticated := auth.GetAuthenticatedUser(r)
	if !authenticated {
		// Return guest permissions for unauthenticated users
		permissions, err := m.permissionService.GetUserPermissions(r.Context(), 0)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "Failed to get guest permissions")
			http.Error(w, "Failed to get permissions", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(permissions)
		return
	}

	permissions, err := m.permissionService.GetUserPermissions(r.Context(), user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get user permissions")
		slog.Error("Failed to get user permissions", 
			slog.String("error", err.Error()),
			slog.Int("character_id", user.CharacterID))
		http.Error(w, "Failed to get permissions", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.Int("character.id", user.CharacterID),
		attribute.Bool("permissions.success", true),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(permissions)
}

// GetPermissionService returns the permission service for use by other modules
func (m *Module) GetPermissionService() *PermissionService {
	return m.permissionService
}

// GetGroupService returns the group service for use by other modules
func (m *Module) GetGroupService() *GroupService {
	return m.groupService
}

// AssignUserToDefaultGroups assigns a user to appropriate default groups
func (m *Module) AssignUserToDefaultGroups(ctx context.Context, characterID int, corporationID, allianceID *int) error {
	slog.Info("Assigning user to default groups",
		slog.Int("character_id", characterID),
		slog.Any("corporation_id", corporationID),
		slog.Any("alliance_id", allianceID))

	// Always assign to "full" group for authenticated users
	if err := m.groupService.AssignToDefaultGroup(ctx, characterID, "full"); err != nil {
		return err
	}

	// Assign to corporate group if applicable
	if (corporationID != nil && *corporationID > 0) || (allianceID != nil && *allianceID > 0) {
		if err := m.groupService.AssignToDefaultGroup(ctx, characterID, "corporate"); err != nil {
			slog.Warn("Failed to assign to corporate group", slog.String("error", err.Error()))
		}
	}

	// Check if user should be super admin
	superAdminCharID := config.GetEnvInt("SUPER_ADMIN_CHARACTER_ID", 0)
	if superAdminCharID > 0 && characterID == superAdminCharID {
		if err := m.groupService.AssignToDefaultGroup(ctx, characterID, "super_admin"); err != nil {
			slog.Error("Failed to assign super admin", 
				slog.String("error", err.Error()),
				slog.Int("character_id", characterID))
		} else {
			slog.Info("Assigned user as super admin", slog.Int("character_id", characterID))
		}
	}

	return nil
}