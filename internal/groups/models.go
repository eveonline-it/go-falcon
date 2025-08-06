package groups

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Group represents a permission group in the system
type Group struct {
	ID                  primitive.ObjectID    `bson:"_id,omitempty" json:"id"`
	Name                string                `bson:"name" json:"name"`
	Description         string                `bson:"description" json:"description"`
	IsDefault           bool                  `bson:"is_default" json:"is_default"`
	Permissions         map[string][]string   `bson:"permissions" json:"permissions"`
	DiscordRoles        []DiscordRole         `bson:"discord_roles" json:"discord_roles"`
	AutoAssignmentRules *AutoAssignmentRules  `bson:"auto_assignment_rules,omitempty" json:"auto_assignment_rules,omitempty"`
	CreatedAt           time.Time             `bson:"created_at" json:"created_at"`
	UpdatedAt           time.Time             `bson:"updated_at" json:"updated_at"`
	CreatedBy           int                   `bson:"created_by" json:"created_by"`
	IsMember            bool                  `bson:"-" json:"is_member"` // Runtime field, not stored
}

// DiscordRole represents a Discord role assignment for a group
type DiscordRole struct {
	ServerID   string `bson:"server_id" json:"server_id"`
	ServerName string `bson:"server_name,omitempty" json:"server_name,omitempty"`
	RoleName   string `bson:"role_name" json:"role_name"`
}

// AutoAssignmentRules defines rules for automatic group assignment
type AutoAssignmentRules struct {
	CorporationIDs      []int     `bson:"corporation_ids,omitempty" json:"corporation_ids,omitempty"`
	AllianceIDs         []int     `bson:"alliance_ids,omitempty" json:"alliance_ids,omitempty"`
	MinSecurityStatus   *float64  `bson:"min_security_status,omitempty" json:"min_security_status,omitempty"`
}

// GroupMembership represents a user's membership in a group
type GroupMembership struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CharacterID      int                `bson:"character_id" json:"character_id"`
	GroupID          primitive.ObjectID `bson:"group_id" json:"group_id"`
	AssignedAt       time.Time          `bson:"assigned_at" json:"assigned_at"`
	AssignedBy       int                `bson:"assigned_by" json:"assigned_by"`
	ExpiresAt        *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	LastValidated    *time.Time         `bson:"last_validated,omitempty" json:"last_validated,omitempty"`
	ValidationStatus string             `bson:"validation_status" json:"validation_status"` // valid, invalid, pending
}

// GroupMemberInfo represents detailed information about a group member
type GroupMemberInfo struct {
	GroupMembership
	CharacterName string `json:"character_name,omitempty"`
	GroupName     string `json:"group_name,omitempty"`
}

// Request/Response types

type CreateGroupRequest struct {
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	Permissions         map[string][]string  `json:"permissions"`
	DiscordRoles        []DiscordRole        `json:"discord_roles"`
	AutoAssignmentRules *AutoAssignmentRules `json:"auto_assignment_rules,omitempty"`
}

func (r *CreateGroupRequest) Validate() error {
	if r.Name == "" {
		return errors.New("group name is required")
	}
	if r.Description == "" {
		return errors.New("group description is required")
	}
	if r.Permissions == nil {
		r.Permissions = make(map[string][]string)
	}
	return nil
}

type UpdateGroupRequest struct {
	Description         *string              `json:"description,omitempty"`
	Permissions         map[string][]string  `json:"permissions,omitempty"`
	DiscordRoles        []DiscordRole        `json:"discord_roles,omitempty"`
	AutoAssignmentRules *AutoAssignmentRules `json:"auto_assignment_rules,omitempty"`
}

type AddMemberRequest struct {
	CharacterID int        `json:"character_id"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// UserPermissionMatrix represents all permissions for a user
type UserPermissionMatrix struct {
	CharacterID int                            `json:"character_id"`
	Groups      []string                       `json:"groups"`
	Permissions map[string]map[string]bool     `json:"permissions"` // resource -> action -> allowed
	IsGuest     bool                           `json:"is_guest"`
}

// GroupService handles all group-related database operations
type GroupService struct {
	mongodb *database.MongoDB
}

func NewGroupService(mongodb *database.MongoDB) *GroupService {
	return &GroupService{
		mongodb: mongodb,
	}
}

// InitializeIndexes creates necessary database indexes for optimal performance
func (gs *GroupService) InitializeIndexes(ctx context.Context) error {
	groupsCollection := gs.mongodb.Database.Collection("groups")
	membershipsCollection := gs.mongodb.Database.Collection("group_memberships")

	// Groups collection indexes
	groupIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "is_default", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: 1}},
		},
	}

	// Memberships collection indexes
	membershipIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "character_id", Value: 1}, {Key: "group_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "character_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "group_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "expires_at", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "validation_status", Value: 1}},
		},
	}

	// Create indexes
	if _, err := groupsCollection.Indexes().CreateMany(ctx, groupIndexes); err != nil {
		return fmt.Errorf("failed to create groups indexes: %w", err)
	}

	if _, err := membershipsCollection.Indexes().CreateMany(ctx, membershipIndexes); err != nil {
		return fmt.Errorf("failed to create memberships indexes: %w", err)
	}

	slog.Info("Groups database indexes created successfully")
	return nil
}

// InitializeDefaultGroups creates the default system groups
func (gs *GroupService) InitializeDefaultGroups(ctx context.Context) error {
	defaultGroups := []Group{
		{
			Name:        "guest",
			Description: "Unauthenticated users with read-only access to public resources",
			IsDefault:   true,
			Permissions: map[string][]string{
				"public": {"read"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: 0, // System created
		},
		{
			Name:        "full",
			Description: "Authenticated EVE Online characters with standard access",
			IsDefault:   true,
			Permissions: map[string][]string{
				"public":  {"read"},
				"user":    {"read", "write"},
				"profile": {"read", "write"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: 0,
		},
		{
			Name:        "corporate",
			Description: "Members of enabled EVE Online corporations or alliances",
			IsDefault:   true,
			Permissions: map[string][]string{
				"public":      {"read"},
				"user":        {"read", "write"},
				"profile":     {"read", "write"},
				"corporation": {"read"},
				"alliance":    {"read"},
			},
			AutoAssignmentRules: &AutoAssignmentRules{
				CorporationIDs: []int{}, // Will be populated via configuration
				AllianceIDs:    []int{}, // Will be populated via configuration
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: 0,
		},
		{
			Name:        "administrators",
			Description: "Application administrators with elevated privileges",
			IsDefault:   true,
			Permissions: map[string][]string{
				"public":      {"read"},
				"user":        {"read", "write", "delete"},
				"profile":     {"read", "write", "delete"},
				"corporation": {"read", "write"},
				"alliance":    {"read", "write"},
				"groups":      {"read", "write", "admin"},
				"system":      {"read"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: 0,
		},
		{
			Name:        "super_admin",
			Description: "Ultimate system authority with unrestricted access",
			IsDefault:   true,
			Permissions: map[string][]string{
				"*": {"*"}, // All permissions on all resources
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			CreatedBy: 0,
		},
	}

	collection := gs.mongodb.Database.Collection("groups")

	for _, group := range defaultGroups {
		// Use upsert to avoid duplicates
		filter := bson.M{"name": group.Name, "is_default": true}
		update := bson.M{
			"$setOnInsert": group,
			"$set": bson.M{
				"updated_at": time.Now(),
				"permissions": group.Permissions, // Always update permissions
			},
		}
		
		opts := options.Update().SetUpsert(true)
		result, err := collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			return fmt.Errorf("failed to create default group %s: %w", group.Name, err)
		}

		if result.UpsertedCount > 0 {
			slog.Info("Created default group", slog.String("name", group.Name))
		}
	}

	return nil
}

// ListGroups retrieves all groups, with optional visibility filtering
func (gs *GroupService) ListGroups(ctx context.Context, includeSystemGroups bool) ([]Group, error) {
	collection := gs.mongodb.Database.Collection("groups")
	
	filter := bson.M{}
	if !includeSystemGroups {
		// Only show non-default groups to regular users
		filter["is_default"] = false
	}

	cursor, err := collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}

	return groups, nil
}

// GetGroupByName retrieves a group by its name
func (gs *GroupService) GetGroupByName(ctx context.Context, name string) (*Group, error) {
	collection := gs.mongodb.Database.Collection("groups")
	
	var group Group
	err := collection.FindOne(ctx, bson.M{"name": name}).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("group not found: %s", name)
		}
		return nil, fmt.Errorf("failed to query group: %w", err)
	}

	return &group, nil
}

// GetGroupByID retrieves a group by its ID
func (gs *GroupService) GetGroupByID(ctx context.Context, groupID string) (*Group, error) {
	collection := gs.mongodb.Database.Collection("groups")
	
	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	var group Group
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("group not found: %s", groupID)
		}
		return nil, fmt.Errorf("failed to query group: %w", err)
	}

	return &group, nil
}

// CreateGroup creates a new custom group
func (gs *GroupService) CreateGroup(ctx context.Context, req *CreateGroupRequest, createdBy int) (*Group, error) {
	collection := gs.mongodb.Database.Collection("groups")

	// Check if group name already exists
	existing := collection.FindOne(ctx, bson.M{"name": req.Name})
	if existing.Err() == nil {
		return nil, fmt.Errorf("group with name '%s' already exists", req.Name)
	}

	group := Group{
		ID:                  primitive.NewObjectID(),
		Name:                req.Name,
		Description:         req.Description,
		IsDefault:           false, // Custom groups are never default
		Permissions:         req.Permissions,
		DiscordRoles:        req.DiscordRoles,
		AutoAssignmentRules: req.AutoAssignmentRules,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
		CreatedBy:           createdBy,
	}

	if group.DiscordRoles == nil {
		group.DiscordRoles = []DiscordRole{}
	}

	_, err := collection.InsertOne(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return &group, nil
}

// UpdateGroup updates an existing group
func (gs *GroupService) UpdateGroup(ctx context.Context, groupID string, req *UpdateGroupRequest, updatedBy int) (*Group, error) {
	collection := gs.mongodb.Database.Collection("groups")

	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Check if group exists and is not a default group (can't modify default groups)
	var existing Group
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&existing)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("failed to query group: %w", err)
	}

	if existing.IsDefault {
		return nil, fmt.Errorf("cannot modify default group")
	}

	// Build update document
	update := bson.M{
		"updated_at": time.Now(),
	}

	if req.Description != nil {
		update["description"] = *req.Description
	}
	if req.Permissions != nil {
		update["permissions"] = req.Permissions
	}
	if req.DiscordRoles != nil {
		update["discord_roles"] = req.DiscordRoles
	}
	if req.AutoAssignmentRules != nil {
		update["auto_assignment_rules"] = req.AutoAssignmentRules
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": update})
	if err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	// Return updated group
	return gs.GetGroupByID(ctx, groupID)
}

// DeleteGroup deletes a custom group (cannot delete default groups)
func (gs *GroupService) DeleteGroup(ctx context.Context, groupID string, deletedBy int) error {
	collection := gs.mongodb.Database.Collection("groups")

	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %w", err)
	}

	// Check if group exists and is not a default group
	var group Group
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&group)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("group not found")
		}
		return fmt.Errorf("failed to query group: %w", err)
	}

	if group.IsDefault {
		return fmt.Errorf("cannot delete default group")
	}

	// Remove all memberships for this group
	membershipsCollection := gs.mongodb.Database.Collection("group_memberships")
	_, err = membershipsCollection.DeleteMany(ctx, bson.M{"group_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to remove group memberships: %w", err)
	}

	// Delete the group
	_, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	slog.Info("Group deleted", 
		slog.String("group_id", groupID), 
		slog.String("group_name", group.Name),
		slog.Int("deleted_by", deletedBy))

	return nil
}

// IsUserMember checks if a user is a member of a specific group
func (gs *GroupService) IsUserMember(ctx context.Context, characterID int, groupID string) (bool, error) {
	collection := gs.mongodb.Database.Collection("group_memberships")

	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return false, fmt.Errorf("invalid group ID: %w", err)
	}

	filter := bson.M{
		"character_id": characterID,
		"group_id":     objectID,
		"$or": []bson.M{
			{"expires_at": bson.M{"$exists": false}},
			{"expires_at": nil},
			{"expires_at": bson.M{"$gt": time.Now()}},
		},
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return count > 0, nil
}

// GetUserGroups retrieves all groups a user belongs to
func (gs *GroupService) GetUserGroups(ctx context.Context, characterID int) ([]Group, error) {
	// Get all active memberships for the user
	memberships, err := gs.getUserMemberships(ctx, characterID)
	if err != nil {
		return nil, err
	}

	if len(memberships) == 0 {
		// Return guest group for users with no memberships
		guestGroup, err := gs.GetGroupByName(ctx, "guest")
		if err != nil {
			return []Group{}, nil // Return empty if guest group doesn't exist
		}
		return []Group{*guestGroup}, nil
	}

	// Get group details for each membership
	groupIDs := make([]primitive.ObjectID, len(memberships))
	for i, membership := range memberships {
		groupIDs[i] = membership.GroupID
	}

	collection := gs.mongodb.Database.Collection("groups")
	cursor, err := collection.Find(ctx, bson.M{"_id": bson.M{"$in": groupIDs}})
	if err != nil {
		return nil, fmt.Errorf("failed to query user groups: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}

	return groups, nil
}

func (gs *GroupService) getUserMemberships(ctx context.Context, characterID int) ([]GroupMembership, error) {
	collection := gs.mongodb.Database.Collection("group_memberships")

	filter := bson.M{
		"character_id": characterID,
		"$or": []bson.M{
			{"expires_at": bson.M{"$exists": false}},
			{"expires_at": nil},
			{"expires_at": bson.M{"$gt": time.Now()}},
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query memberships: %w", err)
	}
	defer cursor.Close(ctx)

	var memberships []GroupMembership
	if err := cursor.All(ctx, &memberships); err != nil {
		return nil, fmt.Errorf("failed to decode memberships: %w", err)
	}

	return memberships, nil
}

// ListGroupMembers retrieves all members of a specific group
func (gs *GroupService) ListGroupMembers(ctx context.Context, groupID string) ([]GroupMemberInfo, error) {
	collection := gs.mongodb.Database.Collection("group_memberships")

	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Use aggregation to join with user profiles for character names
	pipeline := []bson.M{
		{"$match": bson.M{
			"group_id": objectID,
			"$or": []bson.M{
				{"expires_at": bson.M{"$exists": false}},
				{"expires_at": nil},
				{"expires_at": bson.M{"$gt": time.Now()}},
			},
		}},
		{"$lookup": bson.M{
			"from":         "user_profiles",
			"localField":   "character_id",
			"foreignField": "character_id",
			"as":           "profile",
		}},
		{"$lookup": bson.M{
			"from":         "groups",
			"localField":   "group_id",
			"foreignField": "_id",
			"as":           "group",
		}},
		{"$addFields": bson.M{
			"character_name": bson.M{"$arrayElemAt": []interface{}{"$profile.character_name", 0}},
			"group_name":     bson.M{"$arrayElemAt": []interface{}{"$group.name", 0}},
		}},
		{"$project": bson.M{
			"profile": 0,
			"group":   0,
		}},
		{"$sort": bson.M{"assigned_at": -1}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to query group members: %w", err)
	}
	defer cursor.Close(ctx)

	var members []GroupMemberInfo
	if err := cursor.All(ctx, &members); err != nil {
		return nil, fmt.Errorf("failed to decode members: %w", err)
	}

	return members, nil
}

// AddGroupMember adds a user to a group
func (gs *GroupService) AddGroupMember(ctx context.Context, groupID string, characterID, assignedBy int, expiresAt *time.Time) (*GroupMembership, error) {
	collection := gs.mongodb.Database.Collection("group_memberships")

	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID: %w", err)
	}

	// Check if group exists
	if _, err := gs.GetGroupByID(ctx, groupID); err != nil {
		return nil, err
	}

	// Check if membership already exists
	existing := collection.FindOne(ctx, bson.M{"character_id": characterID, "group_id": objectID})
	if existing.Err() == nil {
		return nil, fmt.Errorf("user is already a member of this group")
	}

	membership := GroupMembership{
		ID:               primitive.NewObjectID(),
		CharacterID:      characterID,
		GroupID:          objectID,
		AssignedAt:       time.Now(),
		AssignedBy:       assignedBy,
		ExpiresAt:        expiresAt,
		ValidationStatus: "valid",
	}

	_, err = collection.InsertOne(ctx, membership)
	if err != nil {
		return nil, fmt.Errorf("failed to add group member: %w", err)
	}

	return &membership, nil
}

// RemoveGroupMember removes a user from a group
func (gs *GroupService) RemoveGroupMember(ctx context.Context, groupID string, characterID, removedBy int) error {
	collection := gs.mongodb.Database.Collection("group_memberships")

	objectID, err := primitive.ObjectIDFromHex(groupID)
	if err != nil {
		return fmt.Errorf("invalid group ID: %w", err)
	}

	result, err := collection.DeleteOne(ctx, bson.M{"character_id": characterID, "group_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to remove group member: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user is not a member of this group")
	}

	slog.Info("Group member removed", 
		slog.String("group_id", groupID),
		slog.Int("character_id", characterID),
		slog.Int("removed_by", removedBy))

	return nil
}

// AssignToDefaultGroup assigns a user to a default group
func (gs *GroupService) AssignToDefaultGroup(ctx context.Context, characterID int, groupName string) error {
	group, err := gs.GetGroupByName(ctx, groupName)
	if err != nil {
		return fmt.Errorf("default group not found: %s", groupName)
	}

	if !group.IsDefault {
		return fmt.Errorf("group is not a default group: %s", groupName)
	}

	// Check if already a member
	isMember, err := gs.IsUserMember(ctx, characterID, group.ID.Hex())
	if err != nil {
		return err
	}

	if isMember {
		return nil // Already a member, nothing to do
	}

	_, err = gs.AddGroupMember(ctx, group.ID.Hex(), characterID, 0, nil) // System assigned
	return err
}

// CleanupExpiredMemberships removes expired group memberships
func (gs *GroupService) CleanupExpiredMemberships(ctx context.Context) (int, error) {
	collection := gs.mongodb.Database.Collection("group_memberships")

	filter := bson.M{
		"expires_at": bson.M{
			"$exists": true,
			"$ne":     nil,
			"$lt":     time.Now(),
		},
	}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired memberships: %w", err)
	}

	slog.Info("Cleaned up expired memberships", slog.Int64("count", result.DeletedCount))
	return int(result.DeletedCount), nil
}