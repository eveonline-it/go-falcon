package dto

// No imports needed for HUMA input structures

// HUMA Input/Output structures for Groups API endpoints

// Traditional Group Management DTOs

type GroupListInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	Page          int    `query:"page" minimum:"1" default:"1" doc:"Page number"`
	PageSize      int    `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Items per page"`
	IsDefault     bool   `query:"is_default" doc:"Filter by default groups"`
	Search        string `query:"search" doc:"Search groups by name"`
	ShowMembers   bool   `query:"show_members" doc:"Include member counts"`
}

type GroupListOutput struct {
	Body GroupListResponse `json:"body"`
}

type CreateGroupInput struct {
	Authorization string             `header:"Authorization"`
	Cookie        string             `header:"Cookie"`
	Body          GroupCreateRequest `json:"body"`
}

type CreateGroupOutput struct {
	Body GroupResponse `json:"body"`
}

type GetGroupInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	GroupID       string `path:"groupID" doc:"Group ID"`
}

type GetGroupOutput struct {
	Body GroupResponse `json:"body"`
}

type UpdateGroupInput struct {
	Authorization string             `header:"Authorization"`
	Cookie        string             `header:"Cookie"`
	GroupID       string             `path:"groupID" doc:"Group ID"`
	Body          GroupUpdateRequest `json:"body"`
}

type UpdateGroupOutput struct {
	Body GroupResponse `json:"body"`
}

type DeleteGroupInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	GroupID       string `path:"groupID" doc:"Group ID"`
}

type DeleteGroupOutput struct {
	Body DeleteResponse `json:"body"`
}

// Group Membership DTOs

type AddMemberInput struct {
	Authorization string            `header:"Authorization"`
	Cookie        string            `header:"Cookie"`
	GroupID       string            `path:"groupID" doc:"Group ID"`
	Body          MembershipRequest `json:"body"`
}

type AddMemberOutput struct {
	Body MembershipResponse `json:"body"`
}

type RemoveMemberInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	GroupID       string `path:"groupID" doc:"Group ID"`
	CharacterID   int    `path:"characterID" doc:"Character ID"`
}

type RemoveMemberOutput struct {
	Body DeleteResponse `json:"body"`
}

type ListMembersInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	GroupID       string `path:"groupID" doc:"Group ID"`
	Page          int    `query:"page" minimum:"1" default:"1" doc:"Page number"`
	PageSize      int    `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Items per page"`
}

type ListMembersOutput struct {
	Body GroupMemberListResponse `json:"body"`
}

// Permission Query DTOs

type CheckPermissionInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	Resource      string `query:"resource" doc:"Resource to check"`
	Action        string `query:"action" doc:"Action to check"`
}

type CheckPermissionOutput struct {
	Body PermissionResult `json:"body"`
}

type GetUserPermissionsInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
}

type GetUserPermissionsOutput struct {
	Body UserPermissionsResponse `json:"body"`
}

// HUMA Admin Permission DTOs

type ListServicesInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	Page          int    `query:"page" minimum:"1" default:"1" doc:"Page number"`
	PageSize      int    `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Items per page"`
}

type ListServicesOutput struct {
	Body ServiceListResponse `json:"body"`
}

type CreateServiceInput struct {
	Authorization string               `header:"Authorization"`
	Cookie        string               `header:"Cookie"`
	Body          ServiceCreateRequest `json:"body"`
}

type CreateServiceOutput struct {
	Body ServiceResponse `json:"body"`
}

type GetServiceInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	ServiceName   string `path:"serviceName" doc:"Service name"`
}

type GetServiceOutput struct {
	Body ServiceResponse `json:"body"`
}

type UpdateServiceInput struct {
	Authorization string               `header:"Authorization"`
	Cookie        string               `header:"Cookie"`
	ServiceName   string               `path:"serviceName" doc:"Service name"`
	Body          ServiceUpdateRequest `json:"body"`
}

type UpdateServiceOutput struct {
	Body ServiceResponse `json:"body"`
}

type DeleteServiceInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	ServiceName   string `path:"serviceName" doc:"Service name"`
}

type DeleteServiceOutput struct {
	Body DeleteResponse `json:"body"`
}

// Permission Assignment DTOs

type CreatePermissionAssignmentInput struct {
	Authorization string                       `header:"Authorization"`
	Cookie        string                       `header:"Cookie"`
	Body          PermissionAssignmentRequest `json:"body"`
}

type CreatePermissionAssignmentOutput struct {
	Body PermissionAssignmentResponse `json:"body"`
}

type ListPermissionAssignmentsInput struct {
	Authorization string  `header:"Authorization"`
	Cookie        string  `header:"Cookie"`
	Service       string  `query:"service" doc:"Filter by service"`
	Resource      string  `query:"resource" doc:"Filter by resource"`
	Action        string  `query:"action" doc:"Filter by action"`
	SubjectType   string  `query:"subject_type" doc:"Filter by subject type"`
	SubjectID     string  `query:"subject_id" doc:"Filter by subject ID"`
	Page          int     `query:"page" minimum:"1" default:"1" doc:"Page number"`
	PageSize      int     `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Items per page"`
}

type ListPermissionAssignmentsOutput struct {
	Body PermissionAssignmentListResponse `json:"body"`
}

type RevokePermissionAssignmentInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	AssignmentID  string `path:"assignmentID" doc:"Permission assignment ID"`
}

type RevokePermissionAssignmentOutput struct {
	Body DeleteResponse `json:"body"`
}

type CheckGranularPermissionInput struct {
	Authorization string                          `header:"Authorization"`
	Cookie        string                          `header:"Cookie"`
	Body          PermissionCheckGranularRequest `json:"body"`
}

type CheckGranularPermissionOutput struct {
	Body PermissionResult `json:"body"`
}

type GetUserPermissionSummaryInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	CharacterID   int    `path:"characterID" doc:"Character ID"`
}

type GetUserPermissionSummaryOutput struct {
	Body UserPermissionSummaryResponse `json:"body"`
}

type GetServicePermissionsInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	ServiceName   string `path:"serviceName" doc:"Service name"`
}

type GetServicePermissionsOutput struct {
	Body ServicePermissionSummaryResponse `json:"body"`
}

// Utility DTOs

type ListAvailableGroupsInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
}

type ListAvailableGroupsOutput struct {
	Body SubjectListResponse `json:"body"`
}

type ValidateSubjectInput struct {
	Authorization string `header:"Authorization"`
	Cookie        string `header:"Cookie"`
	Type          string `query:"type" doc:"Subject type (group, member, corporation, alliance)"`
	ID            string `query:"id" doc:"Subject ID"`
}

type ValidateSubjectOutput struct {
	Body SubjectValidationResponse `json:"body"`
}

type GetAuditLogsInput struct {
	Authorization string     `header:"Authorization"`
	Cookie        string     `header:"Cookie"`
	Page          int        `query:"page" minimum:"1" default:"1" doc:"Page number"`
	PageSize      int        `query:"page_size" minimum:"1" maximum:"100" default:"20" doc:"Items per page"`
	Service       string     `query:"service" doc:"Filter by service"`
	Action        string     `query:"action" doc:"Filter by action"`
	SubjectID     string     `query:"subject_id" doc:"Filter by subject ID"`
	StartDate     string `query:"start_date" format:"date-time" doc:"Filter by start date (RFC3339)"`
	EndDate       string `query:"end_date" format:"date-time" doc:"Filter by end date (RFC3339)"`
}

type GetAuditLogsOutput struct {
	Body AuditLogResponse `json:"body"`
}

// Additional Response DTOs

type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type SubjectValidationResponse struct {
	Valid   bool   `json:"valid"`
	Type    string `json:"type"`
	ID      string `json:"id"`
	Message string `json:"message"`
}