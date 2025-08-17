package groups

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GranularPermissionService handles the new granular permission system
type GranularPermissionService struct {
	mongodb *database.MongoDB
	redis   *database.Redis
	groupService *GroupService
}

func NewGranularPermissionService(mongodb *database.MongoDB, redis *database.Redis, groupService *GroupService) *GranularPermissionService {
	return &GranularPermissionService{
		mongodb: mongodb,
		redis:   redis,
		groupService: groupService,
	}
}

// InitializeDefaultServices creates default service definitions if they don't exist
func (gps *GranularPermissionService) InitializeDefaultServices(ctx context.Context) error {
	slog.Info("Initializing default granular permission services")
	
	defaultServices := []Service{
		{
			Name:        "scheduler",
			DisplayName: "Task Scheduler",
			Description: "Task scheduling and management system with cron scheduling and distributed locking",
			Resources: []Resource{
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
			Resources: []Resource{
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
			Resources: []Resource{
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
			Resources: []Resource{
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
			Resources: []Resource{
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
			Resources: []Resource{
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
			Resources: []Resource{
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
	
	collection := gps.mongodb.Database.Collection("services")
	
	for _, service := range defaultServices {
		// Check if service already exists
		var existing Service
		err := collection.FindOne(ctx, bson.M{"name": service.Name}).Decode(&existing)
		if err == nil {
			// Service already exists, skip
			slog.Info("Service already exists, skipping", slog.String("service", service.Name))
			continue
		}
		
		if err != mongo.ErrNoDocuments {
			slog.Error("Failed to check if service exists", 
				slog.String("service", service.Name),
				slog.String("error", err.Error()))
			continue
		}
		
		// Insert the service
		service.ID = primitive.NewObjectID()
		_, err = collection.InsertOne(ctx, service)
		if err != nil {
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
	servicesCollection := gps.mongodb.Database.Collection("services")
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

func (gps *GranularPermissionService) CreateService(ctx context.Context, req *CreateServiceRequest, createdBy int) (*Service, error) {
	collection := gps.mongodb.Database.Collection("services")

	// Check if service already exists
	existing := collection.FindOne(ctx, bson.M{"name": req.Name})
	if existing.Err() == nil {
		return nil, fmt.Errorf("service with name '%s' already exists", req.Name)
	}

	// Set default enabled state for resources
	for i := range req.Resources {
		req.Resources[i].Enabled = true
	}

	service := Service{
		ID:          primitive.NewObjectID(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Resources:   req.Resources,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := collection.InsertOne(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Log the creation
	gps.logAudit(ctx, "create_service", req.Name, "", "", "", "", createdBy, "Service created", nil, map[string]any{
		"service_name": req.Name,
		"display_name": req.DisplayName,
	})

	slog.Info("Service created", slog.String("name", req.Name), slog.Int("created_by", createdBy))
	return &service, nil
}

func (gps *GranularPermissionService) GetService(ctx context.Context, serviceName string) (*Service, error) {
	collection := gps.mongodb.Database.Collection("services")
	
	var service Service
	err := collection.FindOne(ctx, bson.M{"name": serviceName}).Decode(&service)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("service not found: %s", serviceName)
		}
		return nil, fmt.Errorf("failed to query service: %w", err)
	}

	return &service, nil
}

func (gps *GranularPermissionService) ListServices(ctx context.Context) ([]Service, error) {
	collection := gps.mongodb.Database.Collection("services")
	
	cursor, err := collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer cursor.Close(ctx)

	var services []Service
	if err := cursor.All(ctx, &services); err != nil {
		return nil, fmt.Errorf("failed to decode services: %w", err)
	}

	return services, nil
}

func (gps *GranularPermissionService) UpdateService(ctx context.Context, serviceName string, req *UpdateServiceRequest, updatedBy int) (*Service, error) {
	collection := gps.mongodb.Database.Collection("services")

	// Build update document
	update := bson.M{
		"updated_at": time.Now(),
	}

	oldValues := make(map[string]any)
	newValues := make(map[string]any)

	if req.DisplayName != nil {
		update["display_name"] = *req.DisplayName
		newValues["display_name"] = *req.DisplayName
	}
	if req.Description != nil {
		update["description"] = *req.Description
		newValues["description"] = *req.Description
	}
	if req.Resources != nil {
		// Set enabled state for new resources
		for i := range req.Resources {
			if req.Resources[i].Enabled == false {
				req.Resources[i].Enabled = true // Default to enabled
			}
		}
		update["resources"] = req.Resources
		newValues["resources"] = req.Resources
	}
	if req.Enabled != nil {
		update["enabled"] = *req.Enabled
		newValues["enabled"] = *req.Enabled
	}

	result, err := collection.UpdateOne(ctx, bson.M{"name": serviceName}, bson.M{"$set": update})
	if err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	// Log the update
	gps.logAudit(ctx, "update_service", serviceName, "", "", "", "", updatedBy, "Service updated", oldValues, newValues)

	// Return updated service
	return gps.GetService(ctx, serviceName)
}

func (gps *GranularPermissionService) DeleteService(ctx context.Context, serviceName string, deletedBy int) error {
	collection := gps.mongodb.Database.Collection("services")
	permissionsCollection := gps.mongodb.Database.Collection("permission_assignments")

	// Remove all permissions for this service first
	_, err := permissionsCollection.DeleteMany(ctx, bson.M{"service": serviceName})
	if err != nil {
		return fmt.Errorf("failed to remove service permissions: %w", err)
	}

	// Delete the service
	result, err := collection.DeleteOne(ctx, bson.M{"name": serviceName})
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("service not found: %s", serviceName)
	}

	// Log the deletion
	gps.logAudit(ctx, "delete_service", serviceName, "", "", "", "", deletedBy, "Service deleted", nil, nil)

	slog.Info("Service deleted", slog.String("name", serviceName), slog.Int("deleted_by", deletedBy))
	return nil
}

// Permission Assignment Management

func (gps *GranularPermissionService) GrantPermission(ctx context.Context, req *CreatePermissionRequest, grantedBy int) (*PermissionAssignment, error) {
	collection := gps.mongodb.Database.Collection("permission_assignments")

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

	// Check if permission already exists
	existing := collection.FindOne(ctx, bson.M{
		"service":      req.Service,
		"resource":     req.Resource,
		"action":       req.Action,
		"subject_type": req.SubjectType,
		"subject_id":   req.SubjectID,
	})

	if existing.Err() == nil {
		return nil, fmt.Errorf("permission already exists for this subject")
	}

	assignment := PermissionAssignment{
		ID:          primitive.NewObjectID(),
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

	_, err = collection.InsertOne(ctx, assignment)
	if err != nil {
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

	return &assignment, nil
}

func (gps *GranularPermissionService) RevokePermission(ctx context.Context, service, resource, action, subjectType, subjectID string, revokedBy int, reason string) error {
	collection := gps.mongodb.Database.Collection("permission_assignments")

	result, err := collection.DeleteOne(ctx, bson.M{
		"service":      service,
		"resource":     resource,
		"action":       action,
		"subject_type": subjectType,
		"subject_id":   subjectID,
	})

	if err != nil {
		return fmt.Errorf("failed to revoke permission: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("permission not found")
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

func (gps *GranularPermissionService) CheckPermission(ctx context.Context, req *GranularPermissionCheck) (*PermissionResult, error) {
	result := &PermissionResult{
		Service:     req.Service,
		Resource:    req.Resource,
		Action:      req.Action,
		CharacterID: req.CharacterID,
		CheckedAt:   time.Now(),
		Allowed:     false,
		GrantedThrough: []string{},
	}

	// Get user's groups and other membership info
	groups, err := gps.groupService.GetUserGroups(ctx, req.CharacterID)
	if err != nil {
		return result, fmt.Errorf("failed to get user groups: %w", err)
	}

	// Check if user is super_admin or administrator - they have all permissions
	for _, group := range groups {
		if group.Name == "super_admin" {
			result.Allowed = true
			result.GrantedThrough = append(result.GrantedThrough, "group:super_admin")
			return result, nil
		}
		if group.Name == "administrators" {
			result.Allowed = true
			result.GrantedThrough = append(result.GrantedThrough, "group:administrators")
			return result, nil
		}
	}

	// Check permissions through multiple subject types
	collection := gps.mongodb.Database.Collection("permission_assignments")

	// Build query for all possible subjects
	subjectQueries := []bson.M{
		// Direct member permission
		{
			"service":      req.Service,
			"resource":     req.Resource,
			"action":       req.Action,
			"subject_type": "member",
			"subject_id":   fmt.Sprintf("%d", req.CharacterID),
			"enabled":      true,
			"$or": []bson.M{
				{"expires_at": bson.M{"$exists": false}},
				{"expires_at": nil},
				{"expires_at": bson.M{"$gt": time.Now()}},
			},
		},
	}

	// Add group-based permissions
	for _, group := range groups {
		subjectQueries = append(subjectQueries, bson.M{
			"service":      req.Service,
			"resource":     req.Resource,
			"action":       req.Action,
			"subject_type": "group",
			"subject_id":   group.ID.Hex(),
			"enabled":      true,
			"$or": []bson.M{
				{"expires_at": bson.M{"$exists": false}},
				{"expires_at": nil},
				{"expires_at": bson.M{"$gt": time.Now()}},
			},
		})
	}

	// TODO: Add corporation and alliance permissions when we have that data
	// This would require getting user's corporation/alliance from their profile

	// Execute query
	cursor, err := collection.Find(ctx, bson.M{"$or": subjectQueries})
	if err != nil {
		return result, fmt.Errorf("failed to query permissions: %w", err)
	}
	defer cursor.Close(ctx)

	var assignments []PermissionAssignment
	if err := cursor.All(ctx, &assignments); err != nil {
		return result, fmt.Errorf("failed to decode permissions: %w", err)
	}

	// If we found any matching permissions, access is allowed
	if len(assignments) > 0 {
		result.Allowed = true
		
		// Record how permission was granted
		for _, assignment := range assignments {
			switch assignment.SubjectType {
			case "member":
				result.GrantedThrough = append(result.GrantedThrough, "member:direct")
			case "group":
				// Find group name
				for _, group := range groups {
					if group.ID.Hex() == assignment.SubjectID {
						result.GrantedThrough = append(result.GrantedThrough, fmt.Sprintf("group:%s", group.Name))
						break
					}
				}
			case "corporation":
				result.GrantedThrough = append(result.GrantedThrough, fmt.Sprintf("corporation:%s", assignment.SubjectID))
			case "alliance":
				result.GrantedThrough = append(result.GrantedThrough, fmt.Sprintf("alliance:%s", assignment.SubjectID))
			}
		}
	}

	return result, nil
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

	auditLog := PermissionAuditLog{
		ID:          primitive.NewObjectID(),
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

	_, err := collection.InsertOne(ctx, auditLog)
	if err != nil {
		slog.Error("Failed to log permission audit", slog.String("error", err.Error()))
	}
}