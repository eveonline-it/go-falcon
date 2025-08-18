package services

import (
	"context"
	"fmt"
	"time"

	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Extensions to existing services to support the routes

// GroupService extensions

// ListGroupsForSubjects returns groups that can be used as permission subjects
func (gs *GroupService) ListGroupsForSubjects(ctx context.Context) ([]models.Group, int64, error) {
	// Get all groups that are not system-internal groups
	filter := bson.M{"enabled": true}
	groups, total, err := gs.repository.ListGroups(ctx, filter, 1, 1000)
	return groups, total, err
}

// ListMembers returns paginated list of group members
func (gs *GroupService) ListMembers(ctx context.Context, groupID primitive.ObjectID, query *dto.MemberListQuery) ([]dto.GroupMemberResponse, int64, error) {
	// Convert ObjectID to string for the existing method
	groupIDStr := groupID.Hex()
	
	// Call existing method
	response, err := gs.GetGroupMembers(ctx, groupIDStr, query.Page, query.PageSize)
	if err != nil {
		return nil, 0, err
	}

	// Convert MembershipResponse to GroupMemberResponse
	var members []dto.GroupMemberResponse
	for _, member := range response.Members {
		memberResp := dto.GroupMemberResponse{
			CharacterID:      member.CharacterID,
			CharacterName:    fmt.Sprintf("Character-%d", member.CharacterID), // TODO: Get actual name
			AssignedAt:       member.AssignedAt,
			AssignedBy:       0, // TODO: Get actual assigned by
			ExpiresAt:        member.ExpiresAt,
			LastValidated:    time.Now(), // TODO: Get actual last validated
			ValidationStatus: member.ValidationStatus,
			IsActive:         member.ValidationStatus == "valid",
		}
		members = append(members, memberResp)
	}

	return members, int64(response.Pagination.Total), nil
}

// GranularPermissionService extensions

// ListPermissions lists permission assignments with filtering and pagination
func (gps *GranularPermissionService) ListPermissions(ctx context.Context, service, resource, action, subjectType, subjectID string, page, pageSize int) ([]dto.PermissionAssignmentResponse, int64, error) {
	// Build filter
	filter := bson.M{"enabled": true}
	
	if service != "" {
		filter["service"] = service
	}
	if resource != "" {
		filter["resource"] = resource
	}
	if action != "" {
		filter["action"] = action
	}
	if subjectType != "" {
		filter["subject_type"] = subjectType
	}
	if subjectID != "" {
		filter["subject_id"] = subjectID
	}

	// TODO: Implement actual database query - for now return empty
	return []dto.PermissionAssignmentResponse{}, 0, nil
}

// GetUserPermissionSummary returns a summary of user's permissions
func (gps *GranularPermissionService) GetUserPermissionSummary(ctx context.Context, characterID int) (*dto.UserPermissionSummaryResponse, error) {
	// TODO: Implement actual permission summary - for now return empty
	return &dto.UserPermissionSummaryResponse{
		CharacterID:         characterID,
		CharacterName:       fmt.Sprintf("Character-%d", characterID),
		Groups:              []string{},
		GranularPermissions: make(map[string][]string),
		LastUpdated:         time.Now(),
		TotalPermissions:    0,
	}, nil
}

// GetServicePermissions returns permissions for a specific service
func (gps *GranularPermissionService) GetServicePermissions(ctx context.Context, serviceName string) (*dto.ServicePermissionSummaryResponse, error) {
	// TODO: Implement actual service permissions - for now return empty
	return &dto.ServicePermissionSummaryResponse{
		Service:     serviceName,
		Resources:   make(map[string][]string),
		Assignments: []dto.PermissionAssignmentResponse{},
		Total:       0,
	}, nil
}

// ValidateSubject validates if a subject exists and can receive permissions
func (gps *GranularPermissionService) ValidateSubject(ctx context.Context, subjectType, subjectID string) (bool, error) {
	return gps.validateSubject(ctx, subjectType, subjectID) == nil, nil
}

// GetAuditLogs returns paginated audit logs
func (gps *GranularPermissionService) GetAuditLogs(ctx context.Context, query *dto.AuditLogQuery) ([]dto.AuditLogEntry, int64, error) {
	// TODO: Implement actual audit log retrieval - for now return empty
	return []dto.AuditLogEntry{}, 0, nil
}

// IsSuperAdminByCharacterID checks if a character ID is a super admin
func (gps *GranularPermissionService) IsSuperAdminByCharacterID(ctx context.Context, characterID int) (bool, error) {
	// Check against SUPER_ADMIN_CHARACTER_ID environment variable
	// This should be implemented to read from config
	// For now, return false
	return false, nil
}