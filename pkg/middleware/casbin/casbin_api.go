package casbin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-falcon/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CasbinAPIHandler provides HTTP endpoints for managing Casbin policies
type CasbinAPIHandler struct {
	service *CasbinService
}

// NewCasbinAPIHandler creates a new Casbin API handler
func NewCasbinAPIHandler(service *CasbinService) *CasbinAPIHandler {
	return &CasbinAPIHandler{
		service: service,
	}
}

// RegisterRoutes registers Casbin management routes
func (h *CasbinAPIHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin/permissions", func(r chi.Router) {
		// Policy management
		r.Post("/policies", h.CreatePolicy)
		r.Get("/policies", h.ListPolicies)
		r.Delete("/policies/{policyID}", h.DeletePolicy)
		
		// Role management
		r.Post("/roles", h.AssignRole)
		r.Get("/roles", h.ListRoles)
		r.Delete("/roles/{roleID}", h.RevokeRole)
		
		// Permission checking
		r.Post("/check", h.CheckPermission)
		r.Post("/check/batch", h.BatchCheckPermissions)
		
		// User permissions
		r.Get("/users/{userID}/effective", h.GetEffectivePermissions)
		r.Post("/users/{userID}/sync", h.SyncUserHierarchy)
		
		// Hierarchy management
		r.Get("/hierarchies/{userID}", h.GetUserHierarchy)
		r.Post("/hierarchies/sync", h.SyncAllHierarchies)
		
		// Audit logs
		r.Get("/audit", h.GetAuditLogs)
	})
}

// CreatePolicy creates a new permission policy
func (h *CasbinAPIHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var request PolicyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get performer from context
	expandedCtx := middleware.GetExpandedAuthContext(r.Context())
	if expandedCtx == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Grant the permission
	err := h.service.GrantPermission(r.Context(), &request, expandedCtx.PrimaryCharacter.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to grant permission: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Permission granted successfully",
		"policy": map[string]interface{}{
			"subject_type": request.SubjectType,
			"subject_id":   request.SubjectID,
			"resource":     request.Resource,
			"action":       request.Action,
			"effect":       request.Effect,
			"granted_by":   expandedCtx.PrimaryCharacter.ID,
			"granted_at":   time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListPolicies lists all permission policies
func (h *CasbinAPIHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for filtering
	subjectType := r.URL.Query().Get("subject_type")
	subjectID := r.URL.Query().Get("subject_id")
	resource := r.URL.Query().Get("resource")
	action := r.URL.Query().Get("action")
	
	// Build filter
	filter := bson.M{"is_active": true}
	if subjectType != "" {
		filter["subject_type"] = subjectType
	}
	if subjectID != "" {
		filter["subject_id"] = subjectID
	}
	if resource != "" {
		filter["resource"] = resource
	}
	if action != "" {
		filter["action"] = action
	}

	cursor, err := h.service.policyCollection.Find(r.Context(), filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch policies: %v", err), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	var policies []PermissionPolicy
	if err := cursor.All(r.Context(), &policies); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode policies: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"count":    len(policies),
		"policies": policies,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeletePolicy deletes a permission policy
func (h *CasbinAPIHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	policyID := chi.URLParam(r, "policyID")
	if policyID == "" {
		http.Error(w, "Policy ID required", http.StatusBadRequest)
		return
	}

	// Get performer from context
	expandedCtx := middleware.GetExpandedAuthContext(r.Context())
	if expandedCtx == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Find policy details
	var policy PermissionPolicy
	err := h.service.policyCollection.FindOne(r.Context(), bson.M{
		"_id": policyID,
		"is_active": true,
	}).Decode(&policy)
	if err != nil {
		http.Error(w, "Policy not found", http.StatusNotFound)
		return
	}

	// Revoke the permission
	err = h.service.RevokePermission(
		r.Context(),
		policy.SubjectType,
		policy.SubjectID,
		policy.Resource,
		policy.Action,
		policy.Effect,
		expandedCtx.PrimaryCharacter.ID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to revoke permission: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Permission revoked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AssignRole assigns a role to a subject
func (h *CasbinAPIHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var request RoleCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get performer from context
	expandedCtx := middleware.GetExpandedAuthContext(r.Context())
	if expandedCtx == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Assign the role
	err := h.service.AssignRole(r.Context(), &request, expandedCtx.PrimaryCharacter.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to assign role: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Role assigned successfully",
		"assignment": map[string]interface{}{
			"role_name":    request.RoleName,
			"subject_type": request.SubjectType,
			"subject_id":   request.SubjectID,
			"granted_by":   expandedCtx.PrimaryCharacter.ID,
			"granted_at":   time.Now(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListRoles lists all role assignments
func (h *CasbinAPIHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for filtering
	subjectType := r.URL.Query().Get("subject_type")
	subjectID := r.URL.Query().Get("subject_id")
	roleName := r.URL.Query().Get("role_name")
	
	// Build filter
	filter := bson.M{"is_active": true}
	if subjectType != "" {
		filter["subject_type"] = subjectType
	}
	if subjectID != "" {
		filter["subject_id"] = subjectID
	}
	if roleName != "" {
		filter["role_name"] = roleName
	}

	cursor, err := h.service.roleCollection.Find(r.Context(), filter)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch roles: %v", err), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	var roles []RoleAssignment
	if err := cursor.All(r.Context(), &roles); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode roles: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"count":   len(roles),
		"roles":   roles,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RevokeRole revokes a role assignment
func (h *CasbinAPIHandler) RevokeRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleID")
	if roleID == "" {
		http.Error(w, "Role ID required", http.StatusBadRequest)
		return
	}

	// Get performer from context
	expandedCtx := middleware.GetExpandedAuthContext(r.Context())
	if expandedCtx == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Find role assignment details
	var roleAssignment RoleAssignment
	err := h.service.roleCollection.FindOne(r.Context(), bson.M{
		"_id": roleID,
		"is_active": true,
	}).Decode(&roleAssignment)
	if err != nil {
		http.Error(w, "Role assignment not found", http.StatusNotFound)
		return
	}

	// Revoke the role
	err = h.service.RevokeRole(
		r.Context(),
		roleAssignment.SubjectType,
		roleAssignment.SubjectID,
		roleAssignment.RoleName,
		expandedCtx.PrimaryCharacter.ID,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to revoke role: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Role revoked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckPermission checks if a user has a specific permission
func (h *CasbinAPIHandler) CheckPermission(w http.ResponseWriter, r *http.Request) {
	var request PermissionCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.service.CheckPermission(r.Context(), request.UserID, request.Resource, request.Action)
	if err != nil {
		http.Error(w, fmt.Sprintf("Permission check failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  response,
	})
}

// BatchCheckPermissions performs batch permission checking
func (h *CasbinAPIHandler) BatchCheckPermissions(w http.ResponseWriter, r *http.Request) {
	var request BatchPermissionCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	results := make(map[string]PermissionCheckResponse)
	checkTime := time.Now()

	for _, perm := range request.Permissions {
		key := fmt.Sprintf("%s.%s", perm.Resource, perm.Action)
		
		result, err := h.service.CheckPermission(r.Context(), request.UserID, perm.Resource, perm.Action)
		if err != nil {
			results[key] = PermissionCheckResponse{
				Allowed:   false,
				Reason:    fmt.Sprintf("Check failed: %v", err),
				CheckedAt: checkTime,
			}
		} else {
			results[key] = *result
		}
	}

	response := BatchPermissionCheckResponse{
		Results:   results,
		CheckedAt: checkTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  response,
	})
}

// GetEffectivePermissions gets all effective permissions for a user
func (h *CasbinAPIHandler) GetEffectivePermissions(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	response, err := h.service.GetEffectivePermissions(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get effective permissions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  response,
	})
}

// SyncUserHierarchy syncs a user's character hierarchy
func (h *CasbinAPIHandler) SyncUserHierarchy(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	var characters []middleware.UserCharacter
	if err := json.NewDecoder(r.Body).Decode(&characters); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.SyncUserHierarchy(r.Context(), userID, characters)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sync user hierarchy: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "User hierarchy synced successfully",
		"user_id": userID,
		"character_count": len(characters),
		"synced_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUserHierarchy gets a user's character hierarchy
func (h *CasbinAPIHandler) GetUserHierarchy(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	cursor, err := h.service.hierarchyCollection.Find(r.Context(), bson.M{"user_id": userID})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch hierarchy: %v", err), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	var hierarchies []PermissionHierarchy
	if err := cursor.All(r.Context(), &hierarchies); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode hierarchies: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":     true,
		"user_id":     userID,
		"count":       len(hierarchies),
		"hierarchies": hierarchies,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SyncAllHierarchies syncs all user hierarchies (admin operation)
func (h *CasbinAPIHandler) SyncAllHierarchies(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement bulk hierarchy synchronization
	// This would typically iterate through all users and sync their hierarchies from EVE ESI
	
	response := map[string]interface{}{
		"success": true,
		"message": "Bulk hierarchy sync not yet implemented",
		"todo":    "Implement integration with EVE ESI to sync all user hierarchies",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAuditLogs gets permission audit logs
func (h *CasbinAPIHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	operation := r.URL.Query().Get("operation")
	subjectID := r.URL.Query().Get("subject_id")
	performedBy := r.URL.Query().Get("performed_by")

	// Set defaults
	limit := int64(50)
	offset := int64(0)

	if limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			offset = o
		}
	}

	// Build filter
	filter := bson.M{}
	if operation != "" {
		filter["operation"] = operation
	}
	if subjectID != "" {
		filter["subject_id"] = subjectID
	}
	if performedBy != "" {
		if pbID, err := strconv.ParseInt(performedBy, 10, 64); err == nil {
			filter["performed_by"] = pbID
		}
	}

	// Find logs with pagination
	findOptions := options.Find()
	findOptions.SetSort(bson.M{"timestamp": -1}) // Latest first
	findOptions.SetSkip(offset)
	findOptions.SetLimit(limit)
	
	cursor, err := h.service.auditCollection.Find(r.Context(), filter, findOptions)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch audit logs: %v", err), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	var logs []PermissionAuditLog
	if err := cursor.All(r.Context(), &logs); err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode audit logs: %v", err), http.StatusInternalServerError)
		return
	}

	// Get total count
	total, _ := h.service.auditCollection.CountDocuments(r.Context(), filter)

	response := map[string]interface{}{
		"success": true,
		"count":   len(logs),
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"logs":    logs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}