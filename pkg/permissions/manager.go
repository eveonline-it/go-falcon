package permissions

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PermissionManager handles permission registration, storage, and checking
type PermissionManager struct {
	db                 *mongo.Database
	staticPermissions  map[string]Permission
	dynamicPermissions map[string]Permission
	mu                 sync.RWMutex
	
	// Collections
	permissionsCollection     *mongo.Collection
	groupPermissionsCollection *mongo.Collection
}

// NewPermissionManager creates a new permission manager instance
func NewPermissionManager(db *mongo.Database) *PermissionManager {
	pm := &PermissionManager{
		db:                        db,
		staticPermissions:         make(map[string]Permission),
		dynamicPermissions:        make(map[string]Permission),
		permissionsCollection:     db.Collection("permissions"),
		groupPermissionsCollection: db.Collection("group_permissions"),
	}
	
	// Load static permissions
	pm.loadStaticPermissions()
	
	// Initialize database indexes
	if err := pm.ensureIndexes(context.Background()); err != nil {
		slog.Error("[Permissions] Failed to create database indexes", "error", err)
	}
	
	return pm
}

// InitializeSystemGroupPermissions assigns static permissions to system groups
func (pm *PermissionManager) InitializeSystemGroupPermissions(ctx context.Context) error {
	slog.Info("[Permissions] Initializing system group permissions")
	
	// Define static permission assignments for system groups
	systemGroupPermissions := map[string][]string{
		"Super Administrator": {
			// System administration permissions
			"system:admin:full",
			"system:config:manage",
			"users:management:full",
			"users:profiles:view",
			"auth:tokens:manage",
			"groups:management:full",
			"groups:memberships:manage",
			"groups:permissions:manage",
			"groups:view:all",
			"scheduler:tasks:full",
		},
		"Authenticated Users": {
			// Basic permissions for authenticated users
			"groups:view:all",
			"users:profiles:view",
		},
		"Guest Users": {
			// Very limited permissions for guest users
			// Most endpoints should be protected
		},
	}
	
	// Get system groups from database
	for groupName, permissionIDs := range systemGroupPermissions {
		// Find the system group
		groupFilter := bson.M{
			"name":      groupName,
			"type":      "system",
			"is_active": true,
		}
		
		var group struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		
		err := pm.db.Collection("groups").FindOne(ctx, groupFilter).Decode(&group)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				slog.Warn("[Permissions] System group not found, skipping permission assignment", 
					"group_name", groupName)
				continue
			}
			return fmt.Errorf("failed to find system group %s: %w", groupName, err)
		}
		
		// Assign each permission to the group
		assigned := 0
		for _, permissionID := range permissionIDs {
			// Check if permission exists
			if !pm.permissionExists(permissionID) {
				slog.Warn("[Permissions] Permission not found, skipping assignment", 
					"permission_id", permissionID, "group_name", groupName)
				continue
			}
			
			// Check if already assigned
			filter := bson.M{
				"group_id":      group.ID,
				"permission_id": permissionID,
			}
			
			var existingAssignment struct{}
			err := pm.groupPermissionsCollection.FindOne(ctx, filter).Decode(&existingAssignment)
			if err == nil {
				// Already assigned, skip
				continue
			} else if err != mongo.ErrNoDocuments {
				slog.Warn("[Permissions] Error checking existing assignment", 
					"error", err, "permission_id", permissionID, "group_name", groupName)
				continue
			}
			
			// Assign permission (system assignment, no grantedBy)
			assignment := GroupPermission{
				GroupID:      group.ID,
				PermissionID: permissionID,
				GrantedBy:    nil, // System assignment
				GrantedAt:    time.Now(),
				IsActive:     true,
				UpdatedAt:    time.Now(),
			}
			
			_, err = pm.groupPermissionsCollection.InsertOne(ctx, assignment)
			if err != nil {
				slog.Error("[Permissions] Failed to assign permission", 
					"error", err, "permission_id", permissionID, "group_name", groupName)
				continue
			}
			
			assigned++
		}
		
		slog.Info("[Permissions] Assigned permissions to system group", 
			"group_name", groupName, 
			"assigned", assigned, 
			"total", len(permissionIDs))
	}
	
	slog.Info("[Permissions] System group permission initialization completed")
	return nil
}

// loadStaticPermissions loads all static permissions into memory
func (pm *PermissionManager) loadStaticPermissions() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	for id, permission := range StaticPermissions {
		pm.staticPermissions[id] = permission
	}
	
	slog.Info("[Permissions] Loaded static permissions", "count", len(pm.staticPermissions))
}

// RegisterServicePermissions registers permissions for a specific service
func (pm *PermissionManager) RegisterServicePermissions(ctx context.Context, servicePermissions []Permission) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	var operations []mongo.WriteModel
	
	for _, perm := range servicePermissions {
		// Validate permission structure
		if err := pm.validatePermission(perm); err != nil {
			return fmt.Errorf("invalid permission %s: %w", perm.ID, err)
		}
		
		// Prevent overriding static permissions
		if pm.isStaticPermission(perm.ID) {
			return fmt.Errorf("cannot register dynamic permission with static ID: %s", perm.ID)
		}
		
		// Set creation time if not set
		if perm.CreatedAt.IsZero() {
			perm.CreatedAt = time.Now()
		}
		
		// Add to in-memory registry
		pm.dynamicPermissions[perm.ID] = perm
		
		// Prepare database upsert
		filter := bson.M{"_id": perm.ID}
		update := bson.M{"$set": perm}
		operation := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true)
		operations = append(operations, operation)
	}
	
	// Bulk upsert to database
	if len(operations) > 0 {
		opts := options.BulkWrite().SetOrdered(false)
		result, err := pm.permissionsCollection.BulkWrite(ctx, operations, opts)
		if err != nil {
			return fmt.Errorf("failed to store permissions in database: %w", err)
		}
		
		slog.Info("[Permissions] Registered service permissions",
			"service", servicePermissions[0].Service,
			"count", len(servicePermissions),
			"upserted", result.UpsertedCount,
			"modified", result.ModifiedCount)
	}
	
	return nil
}

// HasPermission checks if a character has a specific permission through group membership
func (pm *PermissionManager) HasPermission(ctx context.Context, characterID int64, permissionID string) (bool, error) {
	// Check if permission exists
	if !pm.permissionExists(permissionID) {
		return false, fmt.Errorf("permission not found: %s", permissionID)
	}
	
	// Super admin has all permissions (except for specific restrictions)
	if pm.isSuperAdmin(ctx, characterID) {
		return true, nil
	}
	
	// Check group permissions via aggregation pipeline
	pipeline := []bson.M{
		// Match group memberships for this character
		{
			"$match": bson.M{
				"character_id": characterID,
				"is_active":    true,
			},
		},
		// Lookup group permissions
		{
			"$lookup": bson.M{
				"from":         "group_permissions",
				"localField":   "group_id",
				"foreignField": "group_id",
				"as":           "permissions",
			},
		},
		// Unwind permissions array
		{
			"$unwind": bson.M{
				"path":                       "$permissions",
				"preserveNullAndEmptyArrays": false,
			},
		},
		// Match the specific permission
		{
			"$match": bson.M{
				"permissions.permission_id": permissionID,
				"permissions.is_active":     true,
			},
		},
		// Limit to one result (we just need to know if it exists)
		{
			"$limit": 1,
		},
	}
	
	cursor, err := pm.db.Collection("group_memberships").Aggregate(ctx, pipeline)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}
	defer cursor.Close(ctx)
	
	return cursor.Next(ctx), nil
}

// CheckPermission returns detailed permission check result
func (pm *PermissionManager) CheckPermission(ctx context.Context, characterID int64, permissionID string) (*PermissionCheck, error) {
	result := &PermissionCheck{
		CharacterID:  characterID,
		PermissionID: permissionID,
		Granted:      false,
	}
	
	// Check if permission exists
	if !pm.permissionExists(permissionID) {
		return result, fmt.Errorf("permission not found: %s", permissionID)
	}
	
	// Super admin check
	if pm.isSuperAdmin(ctx, characterID) {
		result.Granted = true
		result.GrantedVia = "Super Administrator"
		return result, nil
	}
	
	// Check group permissions with group name resolution
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"character_id": characterID,
				"is_active":    true,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "groups",
				"localField":   "group_id",
				"foreignField": "_id",
				"as":           "group",
			},
		},
		{
			"$unwind": "$group",
		},
		{
			"$lookup": bson.M{
				"from":         "group_permissions",
				"localField":   "group_id",
				"foreignField": "group_id",
				"as":           "permissions",
			},
		},
		{
			"$unwind": bson.M{
				"path":                       "$permissions",
				"preserveNullAndEmptyArrays": false,
			},
		},
		{
			"$match": bson.M{
				"permissions.permission_id": permissionID,
				"permissions.is_active":     true,
			},
		},
		{
			"$limit": 1,
		},
		{
			"$project": bson.M{
				"group_name": "$group.name",
			},
		},
	}
	
	cursor, err := pm.db.Collection("group_memberships").Aggregate(ctx, pipeline)
	if err != nil {
		return result, fmt.Errorf("failed to check permission: %w", err)
	}
	defer cursor.Close(ctx)
	
	if cursor.Next(ctx) {
		var doc struct {
			GroupName string `bson:"group_name"`
		}
		if err := cursor.Decode(&doc); err == nil {
			result.Granted = true
			result.GrantedVia = doc.GroupName
		} else {
			result.Granted = true
			result.GrantedVia = "Unknown Group"
		}
	}
	
	return result, nil
}

// GetPermission retrieves a permission by ID (static or dynamic)
func (pm *PermissionManager) GetPermission(permissionID string) (Permission, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	// Check static permissions first
	if perm, exists := pm.staticPermissions[permissionID]; exists {
		return perm, true
	}
	
	// Check dynamic permissions
	if perm, exists := pm.dynamicPermissions[permissionID]; exists {
		return perm, true
	}
	
	return Permission{}, false
}

// GetAllPermissions returns all permissions (static + dynamic)
func (pm *PermissionManager) GetAllPermissions() map[string]Permission {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	all := make(map[string]Permission)
	
	// Add static permissions
	for id, perm := range pm.staticPermissions {
		all[id] = perm
	}
	
	// Add dynamic permissions
	for id, perm := range pm.dynamicPermissions {
		all[id] = perm
	}
	
	return all
}

// GrantPermissionToGroup grants a permission to a group
func (pm *PermissionManager) GrantPermissionToGroup(ctx context.Context, groupID primitive.ObjectID, permissionID string, grantedBy int64) error {
	// Verify permission exists
	if !pm.permissionExists(permissionID) {
		return fmt.Errorf("permission not found: %s", permissionID)
	}
	
	// Check if permission is static and restricted
	if pm.isStaticPermission(permissionID) && pm.isRestrictedStaticPermission(permissionID) {
		return fmt.Errorf("permission %s is restricted and cannot be manually granted", permissionID)
	}
	
	// Upsert group permission
	filter := bson.M{
		"group_id":      groupID,
		"permission_id": permissionID,
	}
	
	update := bson.M{
		"$set": bson.M{
			"group_id":      groupID,
			"permission_id": permissionID,
			"granted_by":    grantedBy,
			"granted_at":    time.Now(),
			"is_active":     true,
			"updated_at":    time.Now(),
		},
	}
	
	opts := options.Update().SetUpsert(true)
	_, err := pm.groupPermissionsCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to grant permission: %w", err)
	}
	
	slog.Info("[Permissions] Granted permission to group",
		"group_id", groupID.Hex(),
		"permission_id", permissionID,
		"granted_by", grantedBy)
	
	return nil
}

// RevokePermissionFromGroup revokes a permission from a group
func (pm *PermissionManager) RevokePermissionFromGroup(ctx context.Context, groupID primitive.ObjectID, permissionID string) error {
	// Check if permission is static and restricted
	if pm.isStaticPermission(permissionID) && pm.isRestrictedStaticPermission(permissionID) {
		return fmt.Errorf("permission %s is restricted and cannot be manually revoked", permissionID)
	}
	
	filter := bson.M{
		"group_id":      groupID,
		"permission_id": permissionID,
	}
	
	update := bson.M{
		"$set": bson.M{
			"is_active":  false,
			"updated_at": time.Now(),
		},
	}
	
	result, err := pm.groupPermissionsCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to revoke permission: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("permission assignment not found")
	}
	
	slog.Info("[Permissions] Revoked permission from group",
		"group_id", groupID.Hex(),
		"permission_id", permissionID)
	
	return nil
}

// Helper methods

func (pm *PermissionManager) validatePermission(perm Permission) error {
	if perm.ID == "" {
		return fmt.Errorf("permission ID cannot be empty")
	}
	if perm.Service == "" {
		return fmt.Errorf("service cannot be empty")
	}
	if perm.Resource == "" {
		return fmt.Errorf("resource cannot be empty")
	}
	if perm.Action == "" {
		return fmt.Errorf("action cannot be empty")
	}
	return nil
}

func (pm *PermissionManager) permissionExists(permissionID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	_, exists := pm.staticPermissions[permissionID]
	if exists {
		return true
	}
	
	_, exists = pm.dynamicPermissions[permissionID]
	return exists
}

func (pm *PermissionManager) isStaticPermission(permissionID string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	_, exists := pm.staticPermissions[permissionID]
	return exists
}

func (pm *PermissionManager) isRestrictedStaticPermission(permissionID string) bool {
	// Define permissions that cannot be manually granted/revoked
	restrictedPermissions := map[string]bool{
		"system:admin:full": true, // Only for super admin group
	}
	
	return restrictedPermissions[permissionID]
}

func (pm *PermissionManager) isSuperAdmin(ctx context.Context, characterID int64) bool {
	// Check if character is in Super Administrator group
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"character_id": characterID,
				"is_active":    true,
			},
		},
		{
			"$lookup": bson.M{
				"from":         "groups",
				"localField":   "group_id",
				"foreignField": "_id",
				"as":           "group",
			},
		},
		{
			"$unwind": "$group",
		},
		{
			"$match": bson.M{
				"group.name":      "Super Administrator",
				"group.is_active": true,
			},
		},
		{
			"$limit": 1,
		},
	}
	
	cursor, err := pm.db.Collection("group_memberships").Aggregate(ctx, pipeline)
	if err != nil {
		slog.Error("[Permissions] Failed to check super admin status", "error", err, "character_id", characterID)
		return false
	}
	defer cursor.Close(ctx)
	
	return cursor.Next(ctx)
}

func (pm *PermissionManager) ensureIndexes(ctx context.Context) error {
	// Permissions collection indexes
	permIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "service", Value: 1},
				{Key: "resource", Value: 1},
				{Key: "action", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_static", Value: 1}},
		},
	}
	
	_, err := pm.permissionsCollection.Indexes().CreateMany(ctx, permIndexes)
	if err != nil {
		return fmt.Errorf("failed to create permissions indexes: %w", err)
	}
	
	// Group permissions collection indexes
	groupPermIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "group_id", Value: 1},
				{Key: "permission_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "group_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "permission_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "is_active", Value: 1}},
		},
	}
	
	_, err = pm.groupPermissionsCollection.Indexes().CreateMany(ctx, groupPermIndexes)
	if err != nil {
		return fmt.Errorf("failed to create group permissions indexes: %w", err)
	}
	
	return nil
}