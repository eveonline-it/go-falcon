package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/groups/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository handles database operations for groups
type Repository struct {
	mongodb                *database.MongoDB
	groups                 *mongo.Collection
	memberships           *mongo.Collection
	services              *mongo.Collection
	permissions           *mongo.Collection
	auditLogs             *mongo.Collection
}

// NewRepository creates a new repository instance
func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb:     mongodb,
		groups:      mongodb.Database.Collection("groups"),
		memberships: mongodb.Database.Collection("group_memberships"),
		services:    mongodb.Database.Collection("permission_services"),
		permissions: mongodb.Database.Collection("permission_assignments"),
		auditLogs:   mongodb.Database.Collection("permission_audit_logs"),
	}
}

// Group Operations

// CreateGroup creates a new group
func (r *Repository) CreateGroup(ctx context.Context, group *models.Group) error {
	group.ID = primitive.NewObjectID()
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	
	_, err := r.groups.InsertOne(ctx, group)
	return err
}

// GetGroup retrieves a group by ID
func (r *Repository) GetGroup(ctx context.Context, groupID primitive.ObjectID) (*models.Group, error) {
	var group models.Group
	err := r.groups.FindOne(ctx, bson.M{"_id": groupID}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetGroupByName retrieves a group by name
func (r *Repository) GetGroupByName(ctx context.Context, name string) (*models.Group, error) {
	var group models.Group
	err := r.groups.FindOne(ctx, bson.M{"name": name}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateGroup updates an existing group
func (r *Repository) UpdateGroup(ctx context.Context, group *models.Group) error {
	group.UpdatedAt = time.Now()
	_, err := r.groups.ReplaceOne(ctx, bson.M{"_id": group.ID}, group)
	return err
}

// DeleteGroup deletes a group
func (r *Repository) DeleteGroup(ctx context.Context, groupID primitive.ObjectID) error {
	_, err := r.groups.DeleteOne(ctx, bson.M{"_id": groupID})
	return err
}

// ListGroups lists groups with filtering and pagination
func (r *Repository) ListGroups(ctx context.Context, filter bson.M, page, pageSize int) ([]models.Group, int64, error) {
	// Count total documents
	total, err := r.groups.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := r.groups.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var groups []models.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

// GetDefaultGroups retrieves all default groups
func (r *Repository) GetDefaultGroups(ctx context.Context) ([]models.Group, error) {
	cursor, err := r.groups.Find(ctx, bson.M{"is_default": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []models.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}

	return groups, nil
}

// Membership Operations

// CreateMembership creates a new group membership
func (r *Repository) CreateMembership(ctx context.Context, membership *models.GroupMembership) error {
	membership.ID = primitive.NewObjectID()
	membership.AssignedAt = time.Now()
	membership.ValidationStatus = models.ValidationStatusPending
	
	_, err := r.memberships.InsertOne(ctx, membership)
	return err
}

// GetMembership retrieves a specific membership
func (r *Repository) GetMembership(ctx context.Context, characterID int, groupID primitive.ObjectID) (*models.GroupMembership, error) {
	var membership models.GroupMembership
	filter := bson.M{
		"character_id": characterID,
		"group_id":     groupID,
	}
	
	err := r.memberships.FindOne(ctx, filter).Decode(&membership)
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

// DeleteMembership removes a group membership
func (r *Repository) DeleteMembership(ctx context.Context, characterID int, groupID primitive.ObjectID) error {
	filter := bson.M{
		"character_id": characterID,
		"group_id":     groupID,
	}
	
	_, err := r.memberships.DeleteOne(ctx, filter)
	return err
}

// GetUserMemberships retrieves all memberships for a user
func (r *Repository) GetUserMemberships(ctx context.Context, characterID int) ([]models.GroupMembership, error) {
	cursor, err := r.memberships.Find(ctx, bson.M{"character_id": characterID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var memberships []models.GroupMembership
	if err := cursor.All(ctx, &memberships); err != nil {
		return nil, err
	}

	return memberships, nil
}

// GetGroupMemberships retrieves all memberships for a group
func (r *Repository) GetGroupMemberships(ctx context.Context, groupID primitive.ObjectID, page, pageSize int) ([]models.GroupMembership, int64, error) {
	filter := bson.M{"group_id": groupID}
	
	// Count total documents
	total, err := r.memberships.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "assigned_at", Value: -1}})

	cursor, err := r.memberships.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var memberships []models.GroupMembership
	if err := cursor.All(ctx, &memberships); err != nil {
		return nil, 0, err
	}

	return memberships, total, nil
}

// UpdateMembershipValidation updates the validation status of a membership
func (r *Repository) UpdateMembershipValidation(ctx context.Context, characterID int, groupID primitive.ObjectID, status string) error {
	filter := bson.M{
		"character_id": characterID,
		"group_id":     groupID,
	}
	
	update := bson.M{
		"$set": bson.M{
			"validation_status": status,
			"last_validated":    time.Now(),
		},
	}
	
	_, err := r.memberships.UpdateOne(ctx, filter, update)
	return err
}

// Service Operations (Granular Permissions)

// CreateService creates a new service
func (r *Repository) CreateService(ctx context.Context, service *models.Service) error {
	service.ID = primitive.NewObjectID()
	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()
	service.Enabled = true
	
	_, err := r.services.InsertOne(ctx, service)
	return err
}

// GetService retrieves a service by name
func (r *Repository) GetService(ctx context.Context, name string) (*models.Service, error) {
	var service models.Service
	err := r.services.FindOne(ctx, bson.M{"name": name}).Decode(&service)
	if err != nil {
		return nil, err
	}
	return &service, nil
}

// UpdateService updates an existing service
func (r *Repository) UpdateService(ctx context.Context, service *models.Service) error {
	service.UpdatedAt = time.Now()
	_, err := r.services.ReplaceOne(ctx, bson.M{"name": service.Name}, service)
	return err
}

// DeleteService deletes a service and all its permissions
func (r *Repository) DeleteService(ctx context.Context, name string) error {
	// First delete all permissions for this service
	_, err := r.permissions.DeleteMany(ctx, bson.M{"service": name})
	if err != nil {
		return err
	}
	
	// Then delete the service
	_, err = r.services.DeleteOne(ctx, bson.M{"name": name})
	return err
}

// ListServices lists all services
func (r *Repository) ListServices(ctx context.Context, filter bson.M) ([]models.Service, error) {
	cursor, err := r.services.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var services []models.Service
	if err := cursor.All(ctx, &services); err != nil {
		return nil, err
	}

	return services, nil
}

// Permission Assignment Operations

// CreatePermissionAssignment creates a new permission assignment
func (r *Repository) CreatePermissionAssignment(ctx context.Context, assignment *models.PermissionAssignment) error {
	assignment.ID = primitive.NewObjectID()
	assignment.GrantedAt = time.Now()
	assignment.Enabled = true
	
	_, err := r.permissions.InsertOne(ctx, assignment)
	return err
}

// GetPermissionAssignment retrieves a specific permission assignment
func (r *Repository) GetPermissionAssignment(ctx context.Context, id primitive.ObjectID) (*models.PermissionAssignment, error) {
	var assignment models.PermissionAssignment
	err := r.permissions.FindOne(ctx, bson.M{"_id": id}).Decode(&assignment)
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

// DeletePermissionAssignment removes a permission assignment
func (r *Repository) DeletePermissionAssignment(ctx context.Context, filter bson.M) error {
	_, err := r.permissions.DeleteOne(ctx, filter)
	return err
}

// GetUserPermissions retrieves all permission assignments for a user
func (r *Repository) GetUserPermissions(ctx context.Context, characterID int) ([]models.PermissionAssignment, error) {
	// Get user's group memberships first
	memberships, err := r.GetUserMemberships(ctx, characterID)
	if err != nil {
		return nil, err
	}
	
	var groupIDs []string
	for _, membership := range memberships {
		groupIDs = append(groupIDs, membership.GroupID.Hex())
	}
	
	// Build filter to include direct member permissions and group permissions
	filter := bson.M{
		"enabled": true,
		"$or": []bson.M{
			{"subject_type": "member", "subject_id": fmt.Sprintf("%d", characterID)},
			{"subject_type": "group", "subject_id": bson.M{"$in": groupIDs}},
		},
	}
	
	// Check for expiration
	now := time.Now()
	filter["$or"] = append(filter["$or"].([]bson.M), bson.M{
		"expires_at": bson.M{"$exists": false},
	}, bson.M{
		"expires_at": bson.M{"$gt": now},
	})
	
	cursor, err := r.permissions.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var permissions []models.PermissionAssignment
	if err := cursor.All(ctx, &permissions); err != nil {
		return nil, err
	}

	return permissions, nil
}

// CheckPermission checks if a user has a specific permission
func (r *Repository) CheckPermission(ctx context.Context, characterID int, service, resource, action string) (bool, []string, error) {
	permissions, err := r.GetUserPermissions(ctx, characterID)
	if err != nil {
		return false, nil, err
	}
	
	var matchingGroups []string
	
	for _, perm := range permissions {
		if perm.Service == service && perm.Resource == resource && perm.Action == action {
			if perm.SubjectType == "group" {
				// Get group name for response
				groupID, err := primitive.ObjectIDFromHex(perm.SubjectID)
				if err == nil {
					group, err := r.GetGroup(ctx, groupID)
					if err == nil {
						matchingGroups = append(matchingGroups, group.Name)
					}
				}
			} else {
				matchingGroups = append(matchingGroups, perm.SubjectType+":"+perm.SubjectID)
			}
			return true, matchingGroups, nil
		}
	}
	
	return false, nil, nil
}

// ListPermissionAssignments lists permission assignments with filtering
func (r *Repository) ListPermissionAssignments(ctx context.Context, filter bson.M, page, pageSize int) ([]models.PermissionAssignment, int64, error) {
	// Count total documents
	total, err := r.permissions.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "granted_at", Value: -1}})

	cursor, err := r.permissions.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var assignments []models.PermissionAssignment
	if err := cursor.All(ctx, &assignments); err != nil {
		return nil, 0, err
	}

	return assignments, total, nil
}

// Audit Log Operations

// CreateAuditLog creates a new audit log entry
func (r *Repository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	log.ID = primitive.NewObjectID()
	log.Timestamp = time.Now()
	
	_, err := r.auditLogs.InsertOne(ctx, log)
	return err
}

// GetAuditLogs retrieves audit logs with filtering and pagination
func (r *Repository) GetAuditLogs(ctx context.Context, filter bson.M, page, pageSize int) ([]models.AuditLog, int64, error) {
	// Count total documents
	total, err := r.auditLogs.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Find with pagination
	skip := (page - 1) * pageSize
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := r.auditLogs.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var logs []models.AuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// Cleanup and Maintenance Operations

// CleanupExpiredMemberships removes expired memberships
func (r *Repository) CleanupExpiredMemberships(ctx context.Context) (int64, error) {
	filter := bson.M{
		"expires_at": bson.M{
			"$exists": true,
			"$lt":     time.Now(),
		},
	}
	
	result, err := r.memberships.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	
	return result.DeletedCount, nil
}

// CleanupExpiredPermissions removes expired permission assignments
func (r *Repository) CleanupExpiredPermissions(ctx context.Context) (int64, error) {
	filter := bson.M{
		"expires_at": bson.M{
			"$exists": true,
			"$lt":     time.Now(),
		},
	}
	
	result, err := r.permissions.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	
	return result.DeletedCount, nil
}

// GetMembershipStats retrieves membership statistics
func (r *Repository) GetMembershipStats(ctx context.Context) (*models.MembershipStats, error) {
	now := time.Now()
	
	// Count total memberships
	total, err := r.memberships.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	
	// Count active memberships (not expired)
	activeFilter := bson.M{
		"$or": []bson.M{
			{"expires_at": bson.M{"$exists": false}},
			{"expires_at": bson.M{"$gt": now}},
		},
	}
	active, err := r.memberships.CountDocuments(ctx, activeFilter)
	if err != nil {
		return nil, err
	}
	
	// Count expired memberships
	expiredFilter := bson.M{
		"expires_at": bson.M{
			"$exists": true,
			"$lt":     now,
		},
	}
	expired, err := r.memberships.CountDocuments(ctx, expiredFilter)
	if err != nil {
		return nil, err
	}
	
	return &models.MembershipStats{
		TotalMembers:   int(total),
		ActiveMembers:  int(active),
		ExpiredMembers: int(expired),
		LastUpdated:    now,
	}, nil
}