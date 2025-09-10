package services

import (
	"context"
	"fmt"
	"time"

	authModels "go-falcon/internal/auth/models"
	groupsModels "go-falcon/internal/groups/models"
	wsModels "go-falcon/internal/websocket/models"
	"log/slog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// GroupInfo represents group information for room assignment
type GroupInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// IntegrationService handles integration with user and group modules
type IntegrationService struct {
	db          *mongo.Database
	roomManager *RoomManager
	redisHub    *RedisHub
}

// NewIntegrationService creates a new integration service
func NewIntegrationService(db *mongo.Database, roomManager *RoomManager, redisHub *RedisHub) *IntegrationService {
	return &IntegrationService{
		db:          db,
		roomManager: roomManager,
		redisHub:    redisHub,
	}
}

// AssignUserToRooms assigns a user connection to appropriate rooms
func (is *IntegrationService) AssignUserToRooms(ctx context.Context, connection *wsModels.Connection) error {
	// 1. Join personal room
	if err := is.roomManager.JoinPersonalRoom(connection.UserID, connection.ID); err != nil {
		slog.Error("Failed to join personal room", "error", err, "user_id", connection.UserID, "connection_id", connection.ID)
		return fmt.Errorf("failed to join personal room: %w", err)
	}

	connection.AddRoom(fmt.Sprintf("user:%s", connection.UserID))

	// 2. Get user's group memberships
	groups, err := is.getUserGroups(ctx, connection.CharacterID)
	if err != nil {
		slog.Error("Failed to get user groups", "error", err, "character_id", connection.CharacterID)
		// Don't return error - personal room is enough for basic functionality
		return nil
	}

	// 3. Join group rooms
	for _, group := range groups {
		groupRoomID := fmt.Sprintf("group:%s", group.ID)
		if err := is.roomManager.JoinGroupRoom(group.ID, group.Name, connection.ID); err != nil {
			slog.Error("Failed to join group room", "error", err, "group_id", group.ID, "connection_id", connection.ID)
		} else {
			connection.AddRoom(groupRoomID)
			slog.Info("Connection joined group room", "connection_id", connection.ID, "group_id", group.ID, "group_name", group.Name)
		}
	}

	return nil
}

// getUserGroups retrieves all groups a character belongs to
func (is *IntegrationService) getUserGroups(ctx context.Context, characterID int64) ([]GroupInfo, error) {
	// Add explicit timeout for MongoDB operations to prevent hanging
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get group memberships for the character
	membershipCollection := is.db.Collection(groupsModels.MembershipsCollection)

	filter := bson.M{
		"character_id": characterID,
		"is_active":    true,
	}

	cursor, err := membershipCollection.Find(queryCtx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find group memberships: %w", err)
	}
	defer cursor.Close(queryCtx)

	var memberships []groupsModels.GroupMembership
	if err := cursor.All(queryCtx, &memberships); err != nil {
		return nil, fmt.Errorf("failed to decode group memberships: %w", err)
	}

	if len(memberships) == 0 {
		return []GroupInfo{}, nil
	}

	// Get group information for each membership
	groupCollection := is.db.Collection(groupsModels.GroupsCollection)

	groupIDs := make([]interface{}, len(memberships))
	for i, membership := range memberships {
		groupIDs[i] = membership.GroupID
	}

	groupFilter := bson.M{
		"_id":       bson.M{"$in": groupIDs},
		"is_active": true,
	}

	groupCursor, err := groupCollection.Find(queryCtx, groupFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to find groups: %w", err)
	}
	defer groupCursor.Close(queryCtx)

	var groups []groupsModels.Group
	if err := groupCursor.All(queryCtx, &groups); err != nil {
		return nil, fmt.Errorf("failed to decode groups: %w", err)
	}

	// Convert to GroupInfo
	groupInfos := make([]GroupInfo, len(groups))
	for i, group := range groups {
		groupInfos[i] = GroupInfo{
			ID:   group.ID.Hex(),
			Name: group.Name,
		}
	}

	return groupInfos, nil
}

// HandleUserProfileUpdate handles user profile updates
func (is *IntegrationService) HandleUserProfileUpdate(ctx context.Context, userID string, characterID int64, profileData map[string]interface{}) {
	// Broadcast profile update to user's personal room
	if is.redisHub != nil {
		is.redisHub.BroadcastUserProfileUpdate(ctx, userID, characterID, profileData)
	}

	slog.Info("User profile update handled", "user_id", userID, "character_id", characterID)
}

// HandleGroupMembershipChange handles group membership changes
func (is *IntegrationService) HandleGroupMembershipChange(ctx context.Context, characterID int64, groupID string, groupName string, joined bool) error {
	// Get user ID from character ID
	userID, err := is.getUserIDFromCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("failed to get user ID from character: %w", err)
	}

	// Get all active connections for this user
	if is.roomManager.connectionMgr != nil {
		connections := is.roomManager.connectionMgr.GetConnectionsByCharacter(characterID)

		groupRoomID := fmt.Sprintf("group:%s", groupID)

		for _, conn := range connections {
			if joined {
				// Join group room
				if err := is.roomManager.JoinGroupRoom(groupID, groupName, conn.ID); err != nil {
					slog.Error("Failed to join group room", "error", err, "connection_id", conn.ID, "group_id", groupID)
				} else {
					conn.AddRoom(groupRoomID)
					slog.Info("Connection joined group room due to membership change", "connection_id", conn.ID, "group_id", groupID)
				}
			} else {
				// Leave group room
				if err := is.roomManager.RemoveConnectionFromRoom(groupRoomID, conn.ID); err != nil {
					slog.Error("Failed to leave group room", "error", err, "connection_id", conn.ID, "group_id", groupID)
				} else {
					conn.RemoveRoom(groupRoomID)
					slog.Info("Connection left group room due to membership change", "connection_id", conn.ID, "group_id", groupID)
				}
			}
		}
	}

	// Broadcast membership change
	if is.redisHub != nil {
		is.redisHub.BroadcastGroupMembershipChange(ctx, userID, groupID, groupName, joined)
	}

	return nil
}

// getUserIDFromCharacter gets user ID from character ID
func (is *IntegrationService) getUserIDFromCharacter(ctx context.Context, characterID int64) (string, error) {
	collection := is.db.Collection("user_profiles")

	var user authModels.UserProfile
	filter := bson.M{"character_id": characterID}

	if err := collection.FindOne(ctx, filter).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			return "", fmt.Errorf("user not found for character ID: %d", characterID)
		}
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	return user.UserID, nil
}

// GetUserRoomAssignments returns room assignments for a user
func (is *IntegrationService) GetUserRoomAssignments(ctx context.Context, userID string, characterID int64) ([]string, error) {
	rooms := []string{}

	// Personal room
	personalRoom := fmt.Sprintf("user:%s", userID)
	rooms = append(rooms, personalRoom)

	// Group rooms based on memberships
	groups, err := is.getUserGroups(ctx, characterID)
	if err != nil {
		slog.Error("Failed to get user groups for room assignment", "error", err, "character_id", characterID)
	} else {
		for _, group := range groups {
			groupRoom := fmt.Sprintf("group:%s", group.ID)
			rooms = append(rooms, groupRoom)
		}
	}

	return rooms, nil
}

// RefreshUserRoomAssignments refreshes room assignments for a user
func (is *IntegrationService) RefreshUserRoomAssignments(ctx context.Context, userID string, characterID int64) error {
	if is.roomManager.connectionMgr == nil {
		return fmt.Errorf("connection manager not available")
	}

	// Get all connections for this character
	connections := is.roomManager.connectionMgr.GetConnectionsByCharacter(characterID)
	if len(connections) == 0 {
		return nil // No active connections
	}

	// Get expected room assignments
	expectedRooms, err := is.GetUserRoomAssignments(ctx, userID, characterID)
	if err != nil {
		return fmt.Errorf("failed to get room assignments: %w", err)
	}

	// Update room assignments for each connection
	for _, conn := range connections {
		currentRooms := is.roomManager.GetConnectionRooms(conn.ID)

		// Find rooms to leave
		for _, currentRoom := range currentRooms {
			found := false
			for _, expectedRoom := range expectedRooms {
				if currentRoom == expectedRoom {
					found = true
					break
				}
			}
			if !found {
				is.roomManager.RemoveConnectionFromRoom(currentRoom, conn.ID)
				conn.RemoveRoom(currentRoom)
				slog.Info("Connection left room during refresh", "connection_id", conn.ID, "room_id", currentRoom)
			}
		}

		// Find rooms to join
		for _, expectedRoom := range expectedRooms {
			if !is.roomManager.IsConnectionInRoom(expectedRoom, conn.ID) {
				// Determine room type and join appropriately
				if expectedRoom == fmt.Sprintf("user:%s", userID) {
					is.roomManager.JoinPersonalRoom(userID, conn.ID)
				} else {
					// Extract group ID from room ID
					if len(expectedRoom) > 6 && expectedRoom[:6] == "group:" {
						groupID := expectedRoom[6:]
						// Find group name
						groupName := ""
						groups, _ := is.getUserGroups(ctx, characterID)
						for _, group := range groups {
							if group.ID == groupID {
								groupName = group.Name
								break
							}
						}
						is.roomManager.JoinGroupRoom(groupID, groupName, conn.ID)
					}
				}
				conn.AddRoom(expectedRoom)
				slog.Info("Connection joined room during refresh", "connection_id", conn.ID, "room_id", expectedRoom)
			}
		}
	}

	return nil
}
