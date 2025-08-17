package groups

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"go-falcon/internal/auth"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Admin routes for the new granular permission system
func (m *Module) AdminRoutes(r chi.Router) {
	r.Route("/admin/permissions", func(r chi.Router) {
		// Require super admin for all permission management
		r.Use(m.authModule.JWTMiddleware, m.RequireSuperAdmin())
		
		// Service management
		r.Route("/services", func(r chi.Router) {
			r.Get("/", m.listServicesHandler)
			r.Post("/", m.createServiceHandler)
			r.Get("/{serviceName}", m.getServiceHandler)
			r.Put("/{serviceName}", m.updateServiceHandler)
			r.Delete("/{serviceName}", m.deleteServiceHandler)
		})
		
		// Permission assignment management
		r.Route("/assignments", func(r chi.Router) {
			r.Get("/", m.listPermissionAssignmentsHandler)
			r.Post("/", m.grantPermissionHandler)
			r.Post("/bulk", m.bulkGrantPermissionsHandler)
			r.Delete("/{assignmentID}", m.revokePermissionHandler)
		})
		
		// Permission checking and queries
		r.Route("/check", func(r chi.Router) {
			r.Post("/", m.adminCheckPermissionHandler)
			r.Get("/user/{characterID}", m.getUserPermissionSummaryHandler)
			r.Get("/service/{serviceName}", m.getServicePermissionsHandler)
		})
		
		// Audit logs
		r.Get("/audit", m.getPermissionAuditLogsHandler)
		
		// Utility endpoints
		r.Get("/subjects/groups", m.listGroupSubjectsHandler)
		r.Get("/subjects/validate", m.validateSubjectHandler)
	})
}

// Service Management Handlers

func (m *Module) listServicesHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.services.list",
		attribute.String("service", "groups"),
		attribute.String("operation", "list_services"),
	)
	defer span.End()

	services, err := m.granularPermissionService.ListServices(r.Context())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list services")
		slog.Error("Failed to list services", slog.String("error", err.Error()))
		http.Error(w, "Failed to list services", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.Int("services.count", len(services)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"services": services,
		"count":    len(services),
	})
}

func (m *Module) createServiceHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.services.create",
		attribute.String("service", "groups"),
		attribute.String("operation", "create_service"),
	)
	defer span.End()

	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		span.SetStatus(codes.Error, "Invalid service data")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	service, err := m.granularPermissionService.CreateService(r.Context(), &req, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create service")
		slog.Error("Failed to create service", 
			slog.String("error", err.Error()),
			slog.String("name", req.Name),
			slog.Int("character_id", user.CharacterID))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("service.name", service.Name),
		attribute.Bool("service.success", true),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(service)
}

func (m *Module) getServiceHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.services.get",
		attribute.String("service", "groups"),
		attribute.String("operation", "get_service"),
	)
	defer span.End()

	serviceName := chi.URLParam(r, "serviceName")
	
	service, err := m.granularPermissionService.GetService(r.Context(), serviceName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get service")
		if err.Error() == "service not found: "+serviceName {
			http.Error(w, "Service not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get service", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("service.name", serviceName))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service)
}

func (m *Module) updateServiceHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.services.update",
		attribute.String("service", "groups"),
		attribute.String("operation", "update_service"),
	)
	defer span.End()

	serviceName := chi.URLParam(r, "serviceName")
	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	service, err := m.granularPermissionService.UpdateService(r.Context(), serviceName, &req, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update service")
		if err.Error() == "service not found: "+serviceName {
			http.Error(w, "Service not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("service.name", serviceName))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service)
}

func (m *Module) deleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.services.delete",
		attribute.String("service", "groups"),
		attribute.String("operation", "delete_service"),
	)
	defer span.End()

	serviceName := chi.URLParam(r, "serviceName")
	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	err := m.granularPermissionService.DeleteService(r.Context(), serviceName, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete service")
		if err.Error() == "service not found: "+serviceName {
			http.Error(w, "Service not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.String("service.name", serviceName))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Service deleted successfully",
	})
}

// Permission Assignment Handlers

func (m *Module) grantPermissionHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.assignments.grant",
		attribute.String("service", "groups"),
		attribute.String("operation", "grant_permission"),
	)
	defer span.End()

	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		span.SetStatus(codes.Error, "Invalid permission data")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	assignment, err := m.granularPermissionService.GrantPermission(r.Context(), &req, user.CharacterID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to grant permission")
		slog.Error("Failed to grant permission", 
			slog.String("error", err.Error()),
			slog.String("service", req.Service),
			slog.String("resource", req.Resource),
			slog.String("action", req.Action))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("permission.service", req.Service),
		attribute.String("permission.resource", req.Resource),
		attribute.String("permission.action", req.Action),
		attribute.Bool("permission.success", true),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(assignment)
}

func (m *Module) adminCheckPermissionHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.check",
		attribute.String("service", "groups"),
		attribute.String("operation", "check_permission"),
	)
	defer span.End()

	var req GranularPermissionCheck
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Service == "" || req.Resource == "" || req.Action == "" || req.CharacterID == 0 {
		span.SetStatus(codes.Error, "Missing required fields")
		http.Error(w, "Service, resource, action, and character_id are required", http.StatusBadRequest)
		return
	}

	result, err := m.granularPermissionService.CheckPermission(r.Context(), &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to check permission")
		slog.Error("Failed to check permission", slog.String("error", err.Error()))
		http.Error(w, "Failed to check permission", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("permission.service", req.Service),
		attribute.String("permission.resource", req.Resource),
		attribute.String("permission.action", req.Action),
		attribute.Bool("permission.allowed", result.Allowed),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (m *Module) listGroupSubjectsHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.subjects.groups",
		attribute.String("service", "groups"),
		attribute.String("operation", "list_group_subjects"),
	)
	defer span.End()

	groups, err := m.groupService.ListGroups(r.Context(), true)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to list groups")
		http.Error(w, "Failed to list groups", http.StatusInternalServerError)
		return
	}

	// Convert to subject info format
	subjects := make([]SubjectInfo, len(groups))
	for i, group := range groups {
		subjects[i] = SubjectInfo{
			Type:        "group",
			ID:          group.ID.Hex(),
			Name:        group.Name,
			Description: group.Description,
		}
	}

	span.SetAttributes(attribute.Int("subjects.count", len(subjects)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subjects": subjects,
		"count":    len(subjects),
	})
}

func (m *Module) validateSubjectHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.subjects.validate",
		attribute.String("service", "groups"),
		attribute.String("operation", "validate_subject"),
	)
	defer span.End()

	subjectType := r.URL.Query().Get("type")
	subjectID := r.URL.Query().Get("id")

	if subjectType == "" || subjectID == "" {
		span.SetStatus(codes.Error, "Missing parameters")
		http.Error(w, "Subject type and ID are required", http.StatusBadRequest)
		return
	}

	err := m.granularPermissionService.validateSubject(r.Context(), subjectType, subjectID)
	valid := err == nil

	var errorMessage string
	if !valid {
		errorMessage = err.Error()
	}

	span.SetAttributes(
		attribute.String("subject.type", subjectType),
		attribute.String("subject.id", subjectID),
		attribute.Bool("subject.valid", valid),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   valid,
		"error":   errorMessage,
		"type":    subjectType,
		"id":      subjectID,
	})
}

// Placeholder handlers for remaining endpoints

func (m *Module) listPermissionAssignmentsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement list permission assignments with filtering
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "List permissions endpoint - TODO: implement filtering and pagination",
	})
}

func (m *Module) bulkGrantPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement bulk permission granting
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Bulk grant permissions endpoint - TODO: implement",
	})
}

func (m *Module) revokePermissionHandler(w http.ResponseWriter, r *http.Request) {
	span, r := handlers.StartHTTPSpan(r, "admin.permissions.assignments.revoke",
		attribute.String("service", "groups"),
		attribute.String("operation", "revoke_permission"),
	)
	defer span.End()

	// For now, we'll revoke by service/resource/action/subject combination
	service := r.URL.Query().Get("service")
	resource := r.URL.Query().Get("resource")
	action := r.URL.Query().Get("action")
	subjectType := r.URL.Query().Get("subject_type")
	subjectID := r.URL.Query().Get("subject_id")
	reason := r.URL.Query().Get("reason")

	if service == "" || resource == "" || action == "" || subjectType == "" || subjectID == "" {
		span.SetStatus(codes.Error, "Missing parameters")
		http.Error(w, "Service, resource, action, subject_type, and subject_id are required", http.StatusBadRequest)
		return
	}

	user, ok := auth.GetAuthenticatedUser(r)
	if !ok {
		span.SetStatus(codes.Error, "No authenticated user")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	err := m.granularPermissionService.RevokePermission(r.Context(), service, resource, action, subjectType, subjectID, user.CharacterID, reason)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to revoke permission")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	span.SetAttributes(
		attribute.String("permission.service", service),
		attribute.String("permission.resource", resource),
		attribute.String("permission.action", action),
		attribute.Bool("permission.success", true),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Permission revoked successfully",
	})
}

func (m *Module) getUserPermissionSummaryHandler(w http.ResponseWriter, r *http.Request) {
	characterIDStr := chi.URLParam(r, "characterID")
	characterID, err := strconv.Atoi(characterIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	// TODO: Implement comprehensive user permission summary
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"character_id": characterID,
		"message":      "User permission summary endpoint - TODO: implement",
	})
}

func (m *Module) getServicePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "serviceName")
	
	// TODO: Implement service-specific permission listing
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": serviceName,
		"message": "Service permissions endpoint - TODO: implement",
	})
}

func (m *Module) getPermissionAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement audit log retrieval with filtering and pagination
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Permission audit logs endpoint - TODO: implement",
	})
}