package routes

import (
	"go-falcon/internal/groups/dto"
	"go-falcon/internal/groups/models"
)

// convertGroupToResponse converts a models.Group to dto.GroupResponse
func convertGroupToResponse(group *models.Group, isMember bool, memberCount int) *dto.GroupResponse {
	response := &dto.GroupResponse{
		ID:                  group.ID.Hex(),
		Name:                group.Name,
		Description:         group.Description,
		IsDefault:           group.IsDefault,
		DiscordRoles:        group.DiscordRoles,
		AutoAssignmentRules: group.AutoAssignmentRules,
		CreatedAt:           group.CreatedAt,
		UpdatedAt:           group.UpdatedAt,
		CreatedBy:           group.CreatedBy,
		IsMember:            isMember,
		MemberCount:         memberCount,
	}
	return response
}

// convertServiceToResponse converts a models.Service to dto.ServiceResponse
func convertServiceToResponse(service *models.Service) *dto.ServiceResponse {
	return &dto.ServiceResponse{
		ID:          service.ID.Hex(),
		Name:        service.Name,
		DisplayName: service.DisplayName,
		Description: service.Description,
		Resources:   service.Resources,
		Enabled:     service.Enabled,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   service.UpdatedAt,
	}
}

// convertPermissionAssignmentToResponse converts a models.PermissionAssignment to dto.PermissionAssignmentResponse
func convertPermissionAssignmentToResponse(assignment *models.PermissionAssignment, subjectName string) *dto.PermissionAssignmentResponse {
	return &dto.PermissionAssignmentResponse{
		ID:          assignment.ID.Hex(),
		Service:     assignment.Service,
		Resource:    assignment.Resource,
		Action:      assignment.Action,
		SubjectType: assignment.SubjectType,
		SubjectID:   assignment.SubjectID,
		SubjectName: subjectName,
		GrantedBy:   assignment.GrantedBy,
		GrantedAt:   assignment.GrantedAt,
		ExpiresAt:   assignment.ExpiresAt,
		Reason:      assignment.Reason,
		Enabled:     assignment.Enabled,
	}
}

// convertMembershipToResponse converts a models.GroupMembership to dto.MembershipResponse
func convertMembershipToResponse(membership *models.GroupMembership) *dto.MembershipResponse {
	return &dto.MembershipResponse{
		CharacterID:        membership.CharacterID,
		AssignedAt:         membership.AssignedAt,
		ExpiresAt:          membership.ExpiresAt,
		ValidationStatus:   membership.ValidationStatus,
		LastValidated:      membership.LastValidated,
		AssignmentSource:   membership.AssignmentSource,
		AssignmentMetadata: membership.AssignmentMetadata,
	}
}

// convertPermissionResultToDTO converts a models.PermissionResult to dto.PermissionResult
func convertPermissionResultToDTO(result *models.PermissionResult) *dto.PermissionResult {
	return &dto.PermissionResult{
		Allowed: result.Allowed,
		Reason:  result.Reason,
		Groups:  []string{}, // TODO: Add groups if needed
	}
}

// convertDTOPermissionCheckToModel converts dto.PermissionCheckGranularRequest to models.GranularPermissionCheck
func convertDTOPermissionCheckToModel(req *dto.PermissionCheckGranularRequest) *models.GranularPermissionCheck {
	return &models.GranularPermissionCheck{
		CharacterID: req.CharacterID,
		Service:     req.Service,
		Resource:    req.Resource,
		Action:      req.Action,
	}
}