package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/models"
	"go-falcon/pkg/database"
	"go-falcon/pkg/handlers"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GranularPermissionService handles the new granular permission system
type GranularPermissionService struct {
	mongodb      *database.MongoDB
	redis        *database.Redis
	groupService *GroupService
	repository   *Repository
}

// NewGranularPermissionService creates a new granular permission service
func NewGranularPermissionService(mongodb *database.MongoDB, redis *database.Redis, groupService *GroupService) *GranularPermissionService {
	return &GranularPermissionService{
		mongodb:      mongodb,
		redis:        redis,
		groupService: groupService,
		repository:   NewRepository(mongodb),
	}
}

// InitializeDefaultServices creates default service definitions if they don't exist
func (gps *GranularPermissionService) InitializeDefaultServices(ctx context.Context) error {
	slog.Info("Initializing default granular permission services")

	defaultServices := []models.Service{
		{
			Name:        "scheduler",
			DisplayName: "Task Scheduler",
			Description: "Task scheduling and management system with cron scheduling and distributed locking",
			Resources: []models.ResourceConfig{
				{
					Name:        "tasks",
					DisplayName: "Scheduled Tasks",
					Description: "Task definitions, management, and lifecycle operations",
					Actions:     []string{"read", "write", "delete", "execute", "admin"},
					Enabled:     true,
				},
				{
					Name:        "executions",
					DisplayName: "Task Executions",
					Description: "Task execution history and runtime details",
					Actions:     []string{"read"},
					Enabled:     true,
				},
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "sde",
			DisplayName: "Static Data Export",
			Description: "EVE Online static data management with automated processing and scheduler integration",
			Resources: []models.ResourceConfig{
				{
					Name:        "entities",
					DisplayName: "SDE Entities",
					Description: "EVE Online static data entities including agents, blueprints, types, and universe data",
					Actions:     []string{"read"},
					Enabled:     true,
				},
				{
					Name:        "management",
					DisplayName: "SDE Management",
					Description: "SDE update processes, index rebuilding, and administrative operations",
					Actions:     []string{"read", "write", "admin"},
					Enabled:     true,
				},
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "dev",
			DisplayName: "Development Tools",
			Description: "ESI testing and SDE data access tools for development and debugging",
			Resources: []models.ResourceConfig{
				{
					Name:        "tools",
					DisplayName: "Development Tools",
					Description: "ESI testing endpoints, SDE data access, and development utilities",
					Actions:     []string{"read", "write"},
					Enabled:     true,
				},
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "users",
			DisplayName: "User Management",
			Description: "User profile management and character administration",
			Resources: []models.ResourceConfig{
				{
					Name:        "profiles",
					DisplayName: "User Profiles",
					Description: "User profiles, character management, and account administration",
					Actions:     []string{"read", "write", "delete"},
					Enabled:     true,
				},
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "notifications",
			DisplayName: "Notification System",
			Description: "User notification management and messaging system",
			Resources: []models.ResourceConfig{
				{
					Name:        "messages",
					DisplayName: "Notification Messages",
					Description: "User notifications, alerts, and messaging functionality",
					Actions:     []string{"read", "write", "delete"},
					Enabled:     true,
				},
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "groups",
			DisplayName: "Groups Management",
			Description: "Group and permission management system",
			Resources: []models.ResourceConfig{
				{
					Name:        "management",
					DisplayName: "Group Management",
					Description: "Group creation, membership, and permission administration",
					Actions:     []string{"read", "write", "delete", "admin"},
					Enabled:     true,
				},
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "auth",
			DisplayName: "Authentication",
			Description: "EVE Online SSO authentication and session management",
			Resources: []models.ResourceConfig{
				{
					Name:        "users",
					DisplayName: "User Authentication",
					Description: "User authentication, session management, and profile access",
					Actions:     []string{"read", "write", "delete", "admin"},
					Enabled:     true,
				},
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, service := range defaultServices {
		// Check if service already exists
		if err := gps.repository.CreateService(ctx, &service); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				slog.Info("Service already exists, skipping", slog.String("service", service.Name))
				continue
			}
			slog.Error("Failed to create default service",
				slog.String("service", service.Name),
				slog.String("error", err.Error()))
			continue
		}

		slog.Info("Created default service", slog.String("service", service.Name))
	}

	slog.Info("Default granular permission services initialization complete")
	return nil
}

// InitializeIndexes creates necessary database indexes for the new permission system
func (gps *GranularPermissionService) InitializeIndexes(ctx context.Context) error {
	servicesCollection := gps.mongodb.Database.Collection("permission_services")
	permissionsCollection := gps.mongodb.Database.Collection("permission_assignments")
	auditCollection := gps.mongodb.Database.Collection("permission_audit_logs")

	// Services collection indexes
	serviceIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "enabled", Value: 1}},
		},
	}

	// Permission assignments collection indexes
	permissionIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "service", Value: 1}, {Key: "resource", Value: 1}, {Key: "action", Value: 1}, {Key: "subject_type", Value: 1}, {Key: "subject_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "subject_type", Value: 1}, {Key: "subject_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "service", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "granted_by", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "expires_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "enabled", Value: 1}},
		},
	}

	// Audit log indexes
	auditIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "performed_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "performed_by", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "service", Value: 1}, {Key: "resource", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "subject_type", Value: 1}, {Key: "subject_id", Value: 1}},
		},
	}

	// Create indexes
	if _, err := servicesCollection.Indexes().CreateMany(ctx, serviceIndexes); err != nil {
		return fmt.Errorf("failed to create services indexes: %w", err)
	}

	if _, err := permissionsCollection.Indexes().CreateMany(ctx, permissionIndexes); err != nil {
		return fmt.Errorf("failed to create permissions indexes: %w", err)
	}

	if _, err := auditCollection.Indexes().CreateMany(ctx, auditIndexes); err != nil {
		return fmt.Errorf("failed to create audit indexes: %w", err)
	}

	slog.Info("Granular permission system indexes created successfully")
	return nil
}

// Service Management

func (gps *GranularPermissionService) CreateService(ctx context.Context, req *dto.ServiceCreateRequest, createdBy int) (*models.Service, error) {
	// Check if service already exists
	_, err := gps.repository.GetService(ctx, req.Name)
	if err == nil {
		return nil, fmt.Errorf("service with name '%s' already exists", req.Name)
	}
	if err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to check existing service: %w", err)
	}

	// Set default enabled state for resources
	for i := range req.Resources {
		req.Resources[i].Enabled = true
	}

	service := &models.Service{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Resources:   req.Resources,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := gps.repository.CreateService(ctx, service); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Log the creation
	gps.logAudit(ctx, "create_service", req.Name, "", "", "", "", createdBy, "Service created", nil, map[string]any{
		"service_name": req.Name,
		"display_name": req.DisplayName,
	})

	slog.Info("Service created", slog.String("name", req.Name), slog.Int("created_by", createdBy))
	return service, nil
}

func (gps *GranularPermissionService) GetService(ctx context.Context, serviceName string) (*models.Service, error) {
	service, err := gps.repository.GetService(ctx, serviceName)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("service not found: %s", serviceName)
		}
		return nil, fmt.Errorf("failed to query service: %w", err)
	}

	return service, nil
}

func (gps *GranularPermissionService) ListServices(ctx context.Context) ([]models.Service, error) {
	return gps.repository.ListServices(ctx, bson.M{})
}

func (gps *GranularPermissionService) UpdateService(ctx context.Context, serviceName string, req *dto.ServiceUpdateRequest, updatedBy int) (*models.Service, error) {
	// Get existing service
	service, err := gps.repository.GetService(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	oldValues := map[string]any{
		"display_name": service.DisplayName,
		"description":  service.Description,
		"resources":    service.Resources,
		"enabled":      service.Enabled,
	}

	newValues := make(map[string]any)

	if req.DisplayName != nil {
		service.DisplayName = *req.DisplayName
		newValues["display_name"] = *req.DisplayName
	}

	if req.Description != nil {
		service.Description = *req.Description
		newValues["description"] = *req.Description
	}

	if req.Resources != nil {
		// Set enabled state for new resources
		for i := range req.Resources {
			if !req.Resources[i].Enabled {
				req.Resources[i].Enabled = true // Default to enabled
			}
		}
		service.Resources = req.Resources
		newValues["resources"] = req.Resources
	}

	if req.Enabled != nil {
		service.Enabled = *req.Enabled
		newValues["enabled"] = *req.Enabled
	}

	service.UpdatedAt = time.Now()

	if err := gps.repository.UpdateService(ctx, service); err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	// Log the update
	gps.logAudit(ctx, "update_service", serviceName, "", "", "", "", updatedBy, "Service updated", oldValues, newValues)

	slog.Info("Service updated", slog.String("name", serviceName), slog.Int("updated_by", updatedBy))
	return service, nil
}

func (gps *GranularPermissionService) DeleteService(ctx context.Context, serviceName string, deletedBy int) error {
	// Remove all permissions for this service first
	if err := gps.repository.DeleteService(ctx, serviceName); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Log the deletion
	gps.logAudit(ctx, "delete_service", serviceName, "", "", "", "", deletedBy, "Service deleted", nil, nil)

	slog.Info("Service deleted", slog.String("name", serviceName), slog.Int("deleted_by", deletedBy))
	return nil
}

// Permission Assignment Management

func (gps *GranularPermissionService) GrantPermission(ctx context.Context, req *dto.PermissionAssignmentRequest, grantedBy int) (*models.PermissionAssignment, error) {
	// Validate that the service and resource exist
	service, err := gps.GetService(ctx, req.Service)
	if err != nil {
		return nil, fmt.Errorf("invalid service: %w", err)
	}

	// Check if resource exists in service
	var resourceFound bool
	var validActions []string
	for _, resource := range service.Resources {
		if resource.Name == req.Resource && resource.Enabled {
			resourceFound = true
			validActions = resource.Actions
			break
		}
	}

	if !resourceFound {
		return nil, fmt.Errorf("resource '%s' not found or disabled in service '%s'", req.Resource, req.Service)
	}

	// Check if action is valid for this resource
	var actionValid bool
	for _, action := range validActions {
		if action == req.Action {
			actionValid = true
			break
		}
	}

	if !actionValid {
		return nil, fmt.Errorf("action '%s' not valid for resource '%s'", req.Action, req.Resource)
	}

	// Validate subject exists (basic validation)
	if err := gps.validateSubject(ctx, req.SubjectType, req.SubjectID); err != nil {
		return nil, fmt.Errorf("invalid subject: %w", err)
	}

	assignment := &models.PermissionAssignment{
		Service:     req.Service,
		Resource:    req.Resource,
		Action:      req.Action,
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
		GrantedBy:   grantedBy,
		GrantedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
		Reason:      req.Reason,
		Enabled:     true,
	}

	if err := gps.repository.CreatePermissionAssignment(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to grant permission: %w", err)
	}

	// Log the grant
	gps.logAudit(ctx, "grant", req.Service, req.Resource, req.Action, req.SubjectType, req.SubjectID, grantedBy, req.Reason, nil, map[string]any{
		"expires_at": req.ExpiresAt,
	})

	slog.Info("Permission granted",
		slog.String("service", req.Service),
		slog.String("resource", req.Resource),
		slog.String("action", req.Action),
		slog.String("subject_type", req.SubjectType),
		slog.String("subject_id", req.SubjectID),
		slog.Int("granted_by", grantedBy))

	return assignment, nil
}

func (gps *GranularPermissionService) RevokePermission(ctx context.Context, service, resource, action, subjectType, subjectID string, revokedBy int, reason string) error {
	filter := bson.M{
		"service":      service,
		"resource":     resource,
		"action":       action,
		"subject_type": subjectType,
		"subject_id":   subjectID,
	}

	if err := gps.repository.DeletePermissionAssignment(ctx, filter); err != nil {
		return fmt.Errorf("failed to revoke permission: %w", err)
	}

	// Log the revocation
	gps.logAudit(ctx, "revoke", service, resource, action, subjectType, subjectID, revokedBy, reason, nil, nil)

	slog.Info("Permission revoked",
		slog.String("service", service),
		slog.String("resource", resource),
		slog.String("action", action),
		slog.String("subject_type", subjectType),
		slog.String("subject_id", subjectID),
		slog.Int("revoked_by", revokedBy))

	return nil
}

// Permission Checking

func (gps *GranularPermissionService) CheckPermission(ctx context.Context, req *models.GranularPermissionCheck) (*models.PermissionResult, error) {
	result := &models.PermissionResult{
		CharacterID: req.CharacterID,
		Allowed:     false,
		Groups:      []string{},
		Reason:      "Access denied",
	}

	// Check using repository
	allowed, groups, err := gps.repository.CheckPermission(ctx, req.CharacterID, req.Service, req.Resource, req.Action)
	if err != nil {
		return result, fmt.Errorf("failed to check permission: %w", err)
	}

	result.Allowed = allowed
	result.Groups = groups
	if allowed {
		result.Reason = fmt.Sprintf("Access granted via: %v", groups)
	}

	return result, nil
}

func (gps *GranularPermissionService) CheckPermissionFromRequest(r *http.Request, service, resource, action string) (*models.PermissionResult, error) {
	// Get character ID from request context (set by auth middleware)
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		return &models.PermissionResult{
			Allowed: false,
			Reason:  "Authentication required",
		}, fmt.Errorf("failed to get character ID: %w", err)
	}

	req := &models.GranularPermissionCheck{
		CharacterID: characterID,
		Service:     service,
		Resource:    resource,
		Action:      action,
	}

	return gps.CheckPermission(r.Context(), req)
}

// IsSuperAdmin checks if a user is a super admin via request
func (gps *GranularPermissionService) IsSuperAdmin(r *http.Request) (bool, error) {
	characterID, err := handlers.GetCharacterIDFromRequest(r)
	if err != nil {
		return false, err
	}

	groups, err := gps.groupService.GetUserGroups(r.Context(), characterID)
	if err != nil {
		return false, err
	}

	for _, group := range groups {
		if group.Name == "super_admin" {
			return true, nil
		}
	}

	return false, nil
}

// Helper methods

func (gps *GranularPermissionService) validateSubject(ctx context.Context, subjectType, subjectID string) error {
	switch subjectType {
	case "group":
		// Validate group exists
		_, err := gps.groupService.GetGroupByID(ctx, subjectID)
		return err
	case "member":
		// For now, just validate it's a valid integer
		if subjectID == "" || subjectID == "0" {
			return fmt.Errorf("invalid member ID")
		}
		return nil
	case "corporation", "alliance":
		// For now, just validate it's not empty
		// TODO: Validate against ESI when we have corporation/alliance validation
		if subjectID == "" || subjectID == "0" {
			return fmt.Errorf("invalid %s ID", subjectType)
		}
		return nil
	default:
		return fmt.Errorf("invalid subject type: %s", subjectType)
	}
}

func (gps *GranularPermissionService) logAudit(ctx context.Context, action, service, resource, permission, subjectType, subjectID string, performedBy int, reason string, oldValues, newValues map[string]any) {
	collection := gps.mongodb.Database.Collection("permission_audit_logs")

	auditLog := &models.PermissionAuditLog{
		Action:      action,
		Service:     service,
		Resource:    resource,
		Permission:  permission,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		PerformedBy: performedBy,
		PerformedAt: time.Now(),
		Reason:      reason,
		OldValues:   oldValues,
		NewValues:   newValues,
	}

	auditLog.ID = primitive.NewObjectID()
	_, err := collection.InsertOne(ctx, auditLog)
	if err != nil {
		slog.Error("Failed to log permission audit", slog.String("error", err.Error()))
	}
}

// CleanupExpiredPermissions removes expired permission assignments
func (gps *GranularPermissionService) CleanupExpiredPermissions(ctx context.Context) (int64, error) {
	return gps.repository.CleanupExpiredPermissions(ctx)
}