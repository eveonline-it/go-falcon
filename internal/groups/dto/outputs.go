package dto

import (
	"time"
)

// GroupOutput represents a group in API responses
type GroupOutput struct {
	ID           string     `json:"id" description:"Group ID"`
	Name         string     `json:"name" description:"Group name"`
	Description  string     `json:"description" description:"Group description"`
	Type         string     `json:"type" description:"Group type"`
	SystemName   *string    `json:"system_name,omitempty" description:"System group identifier"`
	EVEEntityID  *int64     `json:"eve_entity_id,omitempty" description:"EVE Corporation/Alliance ID"`
	IsActive     bool       `json:"is_active" description:"Whether the group is active"`
	MemberCount  *int64     `json:"member_count,omitempty" description:"Number of active members"`
	CreatedBy    *int64     `json:"created_by,omitempty" description:"Character ID who created this group"`
	CreatedAt    time.Time  `json:"created_at" description:"Creation timestamp"`
	UpdatedAt    time.Time  `json:"updated_at" description:"Last update timestamp"`
}

// GroupMembershipOutput represents a group membership in API responses
type GroupMembershipOutput struct {
	ID          string    `json:"id" description:"Membership ID"`
	GroupID     string    `json:"group_id" description:"Group ID"`
	CharacterID int64     `json:"character_id" description:"Character ID"`
	IsActive    bool      `json:"is_active" description:"Whether the membership is active"`
	AddedBy     *int64    `json:"added_by,omitempty" description:"Character ID who added this membership"`
	AddedAt     time.Time `json:"added_at" description:"When the membership was added"`
	UpdatedAt   time.Time `json:"updated_at" description:"Last update timestamp"`
}

// ListGroupsOutput represents the response for listing groups
type ListGroupsOutput struct {
	Body ListGroupsResponse `json:"body"`
}

// ListGroupsResponse represents the actual response data for listing groups
type ListGroupsResponse struct {
	Groups []GroupOutput `json:"groups" description:"List of groups"`
	Total  int64         `json:"total" description:"Total number of groups matching the criteria"`
	Page   int           `json:"page" description:"Current page number"`
	Limit  int           `json:"limit" description:"Items per page"`
}

// ListMembersOutput represents the response for listing group members
type ListMembersOutput struct {
	Members []GroupMembershipOutput `json:"members" description:"List of group members"`
	Total   int64                   `json:"total" description:"Total number of members matching the criteria"`
	Page    int                     `json:"page" description:"Current page number"`
	Limit   int                     `json:"limit" description:"Items per page"`
}

// MembershipCheckOutput represents the response for checking membership
type MembershipCheckOutput struct {
	IsMember bool      `json:"is_member" description:"Whether the character is a member of the group"`
	IsActive bool      `json:"is_active" description:"Whether the membership is active (only relevant if is_member is true)"`
	AddedAt  *time.Time `json:"added_at,omitempty" description:"When the membership was added (only relevant if is_member is true)"`
}

// CharacterGroupsOutput represents the response for getting character groups
type CharacterGroupsOutput struct {
	Groups []GroupOutput `json:"groups" description:"List of groups the character belongs to"`
	Total  int64         `json:"total" description:"Total number of groups"`
}

// SuccessOutput represents a simple success response
type SuccessOutput struct {
	Message string `json:"message" description:"Success message"`
}

// HealthOutput represents the health check response
type HealthOutput struct {
	Body HealthResponse `json:"body"`
}

// HealthResponse represents the actual health response data
type HealthResponse struct {
	Health string `json:"health" description:"Health status"`
}