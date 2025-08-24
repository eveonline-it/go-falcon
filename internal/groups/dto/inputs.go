package dto

// CreateGroupInput represents the input for creating a new group
type CreateGroupInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Body          struct {
		Name        string `json:"name" minLength:"3" maxLength:"100" required:"true" description:"Group name"`
		Description string `json:"description" maxLength:"500" description:"Group description"`
		Type        string `json:"type" enum:"custom" required:"true" description:"Group type (only 'custom' allowed for manual creation)"`
	} `json:"body"`
}

// UpdateGroupInput represents the input for updating a group
type UpdateGroupInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	ID            string `path:"id" required:"true" description:"Group ID"`
	Body          struct {
		Name        *string `json:"name" minLength:"3" maxLength:"100" description:"Group name"`
		Description *string `json:"description" maxLength:"500" description:"Group description"`
		IsActive    *bool   `json:"is_active" description:"Whether the group is active"`
	} `json:"body"`
}

// GetGroupInput represents the input for getting a specific group
type GetGroupInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	ID            string `path:"id" required:"true" description:"Group ID"`
}

// ListGroupsInput represents the input for listing groups
type ListGroupsInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Type          string `query:"type" enum:"system,corporation,alliance,custom" description:"Filter by group type"`
	Page          int    `query:"page" minimum:"1" default:"1" description:"Page number"`
	Limit         int    `query:"limit" minimum:"1" maximum:"100" default:"20" description:"Items per page"`
}

// AddMemberInput represents the input for adding a member to a group
type AddMemberInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	GroupID       string `path:"group_id" required:"true" description:"Group ID"`
	Body          struct {
		CharacterID int64 `json:"character_id" required:"true" description:"Character ID to add to the group"`
	} `json:"body"`
}

// RemoveMemberInput represents the input for removing a member from a group
type RemoveMemberInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	GroupID       string `path:"group_id" required:"true" description:"Group ID"`
	CharacterID   string `path:"character_id" required:"true" description:"Character ID to remove from the group"`
}

// ListMembersInput represents the input for listing group members
type ListMembersInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	GroupID       string `path:"group_id" required:"true" description:"Group ID"`
	Page          int    `query:"page" minimum:"1" default:"1" description:"Page number"`
	Limit         int    `query:"limit" minimum:"1" maximum:"100" default:"20" description:"Items per page"`
}

// CheckMembershipInput represents the input for checking if a character is a member of a group
type CheckMembershipInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	GroupID       string `path:"group_id" required:"true" description:"Group ID"`
	CharacterID   string `path:"character_id" required:"true" description:"Character ID to check"`
}

// GetCharacterGroupsInput represents the input for getting groups a character belongs to
type GetCharacterGroupsInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	CharacterID   string `path:"character_id" required:"true" description:"Character ID"`
	Type          string `query:"type" enum:"system,corporation,alliance,custom" description:"Filter by group type"`
}

// DeleteGroupInput represents the input for deleting a group
type DeleteGroupInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	ID            string `path:"id" required:"true" description:"Group ID"`
}

// GetMyGroupsInput represents the input for getting current user's groups
type GetMyGroupsInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Type          string `query:"type" enum:"system,corporation,alliance,custom" description:"Filter by group type"`
}

// GetUserGroupsInput represents the input for getting groups by user_id
type GetUserGroupsInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	UserID        string `path:"user_id" required:"true" description:"User ID"`
	Type          string `query:"type" enum:"system,corporation,alliance,custom" description:"Filter by group type"`
}

// ListPermissionsInput represents the input for listing all permissions
type ListPermissionsInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	Service       string `query:"service" description:"Filter by service name"`
	Category      string `query:"category" description:"Filter by permission category"`
	IsStatic      string `query:"is_static" enum:"true,false" description:"Filter by static/dynamic permissions"`
}

// GetPermissionInput represents the input for getting a specific permission
type GetPermissionInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	PermissionID  string `path:"permission_id" required:"true" description:"Permission ID"`
}

// GrantPermissionToGroupInput represents the input for granting a permission to a group
type GrantPermissionToGroupInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	GroupID       string `path:"group_id" required:"true" description:"Group ID"`
	Body          struct {
		PermissionID string `json:"permission_id" required:"true" minLength:"3" description:"Permission ID to grant"`
	} `json:"body"`
}

// RevokePermissionFromGroupInput represents the input for revoking a permission from a group
type RevokePermissionFromGroupInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	GroupID       string `path:"group_id" required:"true" description:"Group ID"`
	PermissionID  string `path:"permission_id" required:"true" description:"Permission ID to revoke"`
}

// ListGroupPermissionsInput represents the input for listing permissions assigned to a group
type ListGroupPermissionsInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	GroupID       string `path:"group_id" required:"true" description:"Group ID"`
	IsActive      string `query:"is_active" enum:"true,false" description:"Filter by active/inactive permissions"`
}

// CheckPermissionInput represents the input for checking a specific permission
type CheckPermissionInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	PermissionID  string `path:"permission_id" required:"true" description:"Permission ID to check"`
	CharacterID   string `query:"character_id" description:"Character ID to check (optional, defaults to authenticated user)"`
}

// UpdateGroupPermissionStatusInput represents the input for updating group permission status
type UpdateGroupPermissionStatusInput struct {
	Authorization string `header:"Authorization" description:"Bearer token for authentication"`
	Cookie        string `header:"Cookie" description:"Cookie header containing falcon_auth_token"`
	GroupID       string `path:"group_id" required:"true" description:"Group ID"`
	PermissionID  string `path:"permission_id" required:"true" description:"Permission ID to update"`
	Body          struct {
		IsActive bool `json:"is_active" required:"true" description:"Set permission active/inactive status"`
	} `json:"body"`
}
