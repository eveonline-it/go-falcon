package casbin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/pkg/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CasbinService provides high-level operations for Casbin authorization
type CasbinService struct {
	authMiddleware *CasbinAuthMiddleware
	database       *mongo.Database
	hierarchyCollection    *mongo.Collection
	roleCollection         *mongo.Collection
	policyCollection       *mongo.Collection
	auditCollection        *mongo.Collection
}

// NewCasbinService creates a new Casbin service
func NewCasbinService(authMiddleware *CasbinAuthMiddleware, db *mongo.Database) *CasbinService {
	service := &CasbinService{
		authMiddleware:      authMiddleware,
		database:           db,
		hierarchyCollection: db.Collection("permission_hierarchies"),
		roleCollection:     db.Collection("role_assignments"),
		policyCollection:   db.Collection("permission_policies"),
		auditCollection:    db.Collection("permission_audit_logs"),
	}

	// Create indexes for optimal performance
	service.createIndexes()

	return service
}

// createIndexes creates MongoDB indexes for optimal performance
func (s *CasbinService) createIndexes() {
	ctx := context.Background()

	// Hierarchy indexes
	hierarchyIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"character_id", 1}}},
		{Keys: bson.D{{"user_id", 1}}},
		{Keys: bson.D{{"corporation_id", 1}}},
		{Keys: bson.D{{"alliance_id", 1}}},
	}

	// Role assignment indexes
	roleIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"subject_type", 1}, {"subject_id", 1}}},
		{Keys: bson.D{{"role_name", 1}}},
		{Keys: bson.D{{"is_active", 1}, {"expires_at", 1}}},
	}

	// Policy indexes
	policyIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"subject_type", 1}, {"subject_id", 1}}},
		{Keys: bson.D{{"resource", 1}, {"action", 1}}},
		{Keys: bson.D{{"is_active", 1}, {"expires_at", 1}}},
	}

	// Audit log indexes
	auditIndexes := []mongo.IndexModel{
		{Keys: bson.D{{"timestamp", -1}}},
		{Keys: bson.D{{"performed_by", 1}}},
		{Keys: bson.D{{"operation", 1}, {"operation_type", 1}}},
		{Keys: bson.D{{"subject_type", 1}, {"subject_id", 1}}},
	}

	// Create indexes
	collections := map[string][]mongo.IndexModel{
		"permission_hierarchies": hierarchyIndexes,
		"role_assignments":       roleIndexes,
		"permission_policies":    policyIndexes,
		"permission_audit_logs":  auditIndexes,
	}

	for collectionName, indexes := range collections {
		collection := s.database.Collection(collectionName)
		if _, err := collection.Indexes().CreateMany(ctx, indexes); err != nil {
			slog.Warn("Failed to create indexes", "collection", collectionName, "error", err)
		}
	}

	slog.Info("Created Casbin service indexes")
}

// SyncUserHierarchy syncs a user's hierarchy information
func (s *CasbinService) SyncUserHierarchy(ctx context.Context, userID string, characters []middleware.UserCharacter) error {
	// Remove existing hierarchies for this user
	_, err := s.hierarchyCollection.DeleteMany(ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("failed to remove existing hierarchies: %w", err)
	}

	// Insert new hierarchies
	now := time.Now()
	var hierarchies []interface{}

	for _, char := range characters {
		hierarchy := PermissionHierarchy{
			CharacterID:     char.CharacterID,
			CharacterName:   char.Name,
			CorporationID:   char.CorporationID,
			AllianceID:      char.AllianceID,
			UserID:          userID,
			IsPrimary:       char.IsPrimary,
			CreatedAt:       now,
			UpdatedAt:       now,
			LastSyncAt:      now,
		}
		hierarchies = append(hierarchies, hierarchy)
	}

	if len(hierarchies) > 0 {
		_, err = s.hierarchyCollection.InsertMany(ctx, hierarchies)
		if err != nil {
			return fmt.Errorf("failed to insert hierarchies: %w", err)
		}
	}

	return nil
}

// CheckPermission checks if a user has permission for a specific resource and action
func (s *CasbinService) CheckPermission(ctx context.Context, userID string, resource, action string) (*PermissionCheckResponse, error) {
	// Get user's expanded context
	expandedCtx, err := s.getUserExpandedContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user context: %w", err)
	}

	checkTime := time.Now()
	response := &PermissionCheckResponse{
		CheckedAt: checkTime,
	}

	// Check permissions using Casbin middleware
	allowed, err := s.authMiddleware.checkHierarchicalPermission(ctx, expandedCtx, resource, action)
	if err != nil {
		return nil, fmt.Errorf("permission check failed: %w", err)
	}

	response.Allowed = allowed
	if allowed {
		response.Reason = "Permission granted"
		// TODO: Add which subjects granted the permission
	} else {
		response.Reason = "Permission denied"
		// TODO: Add which subjects denied the permission
	}

	// Audit log the permission check
	s.auditPermissionCheck(ctx, userID, resource, action, allowed, nil)

	return response, nil
}

// GrantPermission grants a permission to a subject
func (s *CasbinService) GrantPermission(ctx context.Context, request *PolicyCreateRequest, performedBy int64) error {
	// Create subject string
	subject := fmt.Sprintf("%s:%s", request.SubjectType, request.SubjectID)
	
	// Add policy to Casbin
	err := s.authMiddleware.AddPolicy(subject, request.Resource, request.Action, request.Effect)
	if err != nil {
		return fmt.Errorf("failed to add Casbin policy: %w", err)
	}

	// Store policy metadata
	now := time.Now()
	policy := PermissionPolicy{
		SubjectType:   request.SubjectType,
		SubjectID:     request.SubjectID,
		Resource:      request.Resource,
		Action:        request.Action,
		Domain:        request.Domain,
		Effect:        request.Effect,
		CreatedBy:     performedBy,
		CreatedAt:     now,
		ExpiresAt:     request.ExpiresAt,
		Reason:        request.Reason,
		IsActive:      true,
	}

	_, err = s.policyCollection.InsertOne(ctx, policy)
	if err != nil {
		// Try to rollback Casbin policy
		s.authMiddleware.RemovePolicy(subject, request.Resource, request.Action, request.Effect)
		return fmt.Errorf("failed to store policy metadata: %w", err)
	}

	// Audit log
	s.auditPolicyOperation(ctx, "grant", request.SubjectType, request.SubjectID, request.Resource, request.Action, request.Effect, performedBy)

	return nil
}

// RevokePermission revokes a permission from a subject
func (s *CasbinService) RevokePermission(ctx context.Context, subjectType, subjectID, resource, action, effect string, performedBy int64) error {
	// Create subject string
	subject := fmt.Sprintf("%s:%s", subjectType, subjectID)
	
	// Remove policy from Casbin
	err := s.authMiddleware.RemovePolicy(subject, resource, action, effect)
	if err != nil {
		return fmt.Errorf("failed to remove Casbin policy: %w", err)
	}

	// Mark policy as inactive in metadata
	filter := bson.M{
		"subject_type": subjectType,
		"subject_id":   subjectID,
		"resource":     resource,
		"action":       action,
		"effect":       effect,
		"is_active":    true,
	}

	update := bson.M{
		"$set": bson.M{
			"is_active": false,
			"updated_at": time.Now(),
		},
	}

	_, err = s.policyCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		slog.Warn("Failed to update policy metadata", "error", err)
	}

	// Audit log
	s.auditPolicyOperation(ctx, "revoke", subjectType, subjectID, resource, action, effect, performedBy)

	return nil
}

// AssignRole assigns a role to a subject
func (s *CasbinService) AssignRole(ctx context.Context, request *RoleCreateRequest, performedBy int64) error {
	// Create subject string
	subject := fmt.Sprintf("%s:%s", request.SubjectType, request.SubjectID)
	
	// Add role to Casbin
	err := s.authMiddleware.AddRoleForUser(subject, request.RoleName)
	if err != nil {
		return fmt.Errorf("failed to add Casbin role: %w", err)
	}

	// Store role assignment metadata
	now := time.Now()
	roleAssignment := RoleAssignment{
		RoleName:    request.RoleName,
		SubjectType: request.SubjectType,
		SubjectID:   request.SubjectID,
		Domain:      request.Domain,
		GrantedBy:   performedBy,
		GrantedAt:   now,
		ExpiresAt:   request.ExpiresAt,
		Reason:      request.Reason,
		IsActive:    true,
	}

	_, err = s.roleCollection.InsertOne(ctx, roleAssignment)
	if err != nil {
		// Try to rollback Casbin role
		s.authMiddleware.RemoveRoleForUser(subject, request.RoleName)
		return fmt.Errorf("failed to store role assignment metadata: %w", err)
	}

	// Audit log
	s.auditRoleOperation(ctx, "assign", request.SubjectType, request.SubjectID, request.RoleName, performedBy)

	return nil
}

// RevokeRole revokes a role from a subject
func (s *CasbinService) RevokeRole(ctx context.Context, subjectType, subjectID, roleName string, performedBy int64) error {
	// Create subject string
	subject := fmt.Sprintf("%s:%s", subjectType, subjectID)
	
	// Remove role from Casbin
	err := s.authMiddleware.RemoveRoleForUser(subject, roleName)
	if err != nil {
		return fmt.Errorf("failed to remove Casbin role: %w", err)
	}

	// Mark role assignment as inactive
	filter := bson.M{
		"subject_type": subjectType,
		"subject_id":   subjectID,
		"role_name":    roleName,
		"is_active":    true,
	}

	update := bson.M{
		"$set": bson.M{
			"is_active": false,
			"updated_at": time.Now(),
		},
	}

	_, err = s.roleCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		slog.Warn("Failed to update role assignment metadata", "error", err)
	}

	// Audit log
	s.auditRoleOperation(ctx, "revoke", subjectType, subjectID, roleName, performedBy)

	return nil
}

// GetEffectivePermissions gets all effective permissions for a user
func (s *CasbinService) GetEffectivePermissions(ctx context.Context, userID string) (*GetEffectivePermissionsResponse, error) {
	expandedCtx, err := s.getUserExpandedContext(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user context: %w", err)
	}

	response := &GetEffectivePermissionsResponse{
		UserID:    userID,
		CheckedAt: time.Now(),
	}

	// Get subjects for all levels
	subjects := s.authMiddleware.buildSubjects(expandedCtx)
	
	// Get roles for each subject
	for _, subject := range subjects {
		roles, err := s.authMiddleware.GetRolesForUser(subject)
		if err == nil {
			response.Roles = append(response.Roles, roles...)
		}
	}

	// Get direct policies for each subject
	for _, subject := range subjects {
		permissions, err := s.authMiddleware.GetPermissionsForUser(subject)
		if err == nil {
			for _, perm := range permissions {
			if len(perm) >= 4 {
				policyInfo := PermissionPolicyInfo{
					Resource: perm[1],
					Action:   perm[2],
					Effect:   perm[3],
					Domain:   perm[4],
					Source:   subject,
				}
				response.Policies = append(response.Policies, policyInfo)
			}
		}
		}
	}

	// Build character, corporation, and alliance info
	// TODO: Implement detailed subject info building

	return response, nil
}

// getUserExpandedContext builds expanded auth context for a user
func (s *CasbinService) getUserExpandedContext(ctx context.Context, userID string) (*middleware.ExpandedAuthContext, error) {
	// Get hierarchies from database
	cursor, err := s.hierarchyCollection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to find user hierarchies: %w", err)
	}
	defer cursor.Close(ctx)

	var hierarchies []PermissionHierarchy
	if err := cursor.All(ctx, &hierarchies); err != nil {
		return nil, fmt.Errorf("failed to decode hierarchies: %w", err)
	}

	if len(hierarchies) == 0 {
		return nil, fmt.Errorf("no hierarchies found for user %s", userID)
	}

	// Build expanded context
	var characterIDs []int64
	var corporationIDs []int64
	var allianceIDs []int64
	
	corpMap := make(map[int64]bool)
	allianceMap := make(map[int64]bool)
	
	var primaryChar PermissionHierarchy
	for _, hierarchy := range hierarchies {
		characterIDs = append(characterIDs, hierarchy.CharacterID)
		
		if !corpMap[hierarchy.CorporationID] {
			corporationIDs = append(corporationIDs, hierarchy.CorporationID)
			corpMap[hierarchy.CorporationID] = true
		}
		
		if hierarchy.AllianceID > 0 && !allianceMap[hierarchy.AllianceID] {
			allianceIDs = append(allianceIDs, hierarchy.AllianceID)
			allianceMap[hierarchy.AllianceID] = true
		}
		
		if hierarchy.IsPrimary {
			primaryChar = hierarchy
		}
	}

	expandedCtx := &middleware.ExpandedAuthContext{
		AuthContext: &middleware.AuthContext{
			UserID:          userID,
			PrimaryCharID:   primaryChar.CharacterID,
			RequestType:     "internal",
			IsAuthenticated: true,
		},
		CharacterIDs:   characterIDs,
		CorporationIDs: corporationIDs,
		AllianceIDs:    allianceIDs,
		PrimaryCharacter: struct {
			ID            int64  `json:"id"`
			Name          string `json:"name"`
			CorporationID int64  `json:"corporation_id"`
			AllianceID    int64  `json:"alliance_id,omitempty"`
		}{
			ID:            primaryChar.CharacterID,
			Name:          primaryChar.CharacterName,
			CorporationID: primaryChar.CorporationID,
			AllianceID:    primaryChar.AllianceID,
		},
		Roles:       []string{}, // Will be populated by CASBIN
		Permissions: []string{}, // Will be populated by CASBIN
	}

	return expandedCtx, nil
}

// auditPermissionCheck logs a permission check
func (s *CasbinService) auditPermissionCheck(ctx context.Context, userID, resource, action string, result bool, performedBy *int64) {
	auditLog := PermissionAuditLog{
		Operation:       "check",
		OperationType:   "permission",
		SubjectType:     "user",
		SubjectID:       userID,
		Resource:        resource,
		Action:          action,
		Domain:          "global",
		Result:          &result,
		Timestamp:       time.Now(),
	}

	if performedBy != nil {
		auditLog.PerformedBy = *performedBy
	}

	_, err := s.auditCollection.InsertOne(ctx, auditLog)
	if err != nil {
		slog.Warn("Failed to audit permission check", "error", err)
	}
}

// auditPolicyOperation logs a policy operation
func (s *CasbinService) auditPolicyOperation(ctx context.Context, operation, subjectType, subjectID, resource, action, effect string, performedBy int64) {
	auditLog := PermissionAuditLog{
		Operation:       operation,
		OperationType:   "policy",
		SubjectType:     subjectType,
		SubjectID:       subjectID,
		Resource:        resource,
		Action:          action,
		Domain:          "global",
		Effect:          effect,
		PerformedBy:     performedBy,
		Timestamp:       time.Now(),
	}

	_, err := s.auditCollection.InsertOne(ctx, auditLog)
	if err != nil {
		slog.Warn("Failed to audit policy operation", "error", err)
	}
}

// auditRoleOperation logs a role operation
func (s *CasbinService) auditRoleOperation(ctx context.Context, operation, subjectType, subjectID, roleName string, performedBy int64) {
	auditLog := PermissionAuditLog{
		Operation:       operation,
		OperationType:   "role",
		SubjectType:     subjectType,
		SubjectID:       subjectID,
		TargetType:      "role",
		TargetID:        roleName,
		Domain:          "global",
		PerformedBy:     performedBy,
		Timestamp:       time.Now(),
	}

	_, err := s.auditCollection.InsertOne(ctx, auditLog)
	if err != nil {
		slog.Warn("Failed to audit role operation", "error", err)
	}
}