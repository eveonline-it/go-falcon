package groups

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-falcon/pkg/config"
)

// DiscordService handles Discord role management across multiple servers
type DiscordService struct {
	serviceURL string
	client     *http.Client
}

func NewDiscordService() *DiscordService {
	serviceURL := config.GetEnv("DISCORD_SERVICE_URL", "")
	
	return &DiscordService{
		serviceURL: serviceURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DiscordServer represents a Discord server
type DiscordServer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Available   bool   `json:"available"`
	MemberCount int    `json:"member_count,omitempty"`
}

// DiscordServerRole represents a role on a Discord server
type DiscordServerRole struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ServerID string `json:"server_id"`
	Color    int    `json:"color,omitempty"`
	Position int    `json:"position,omitempty"`
}

// DiscordMemberRole represents a user's role assignment on a server
type DiscordMemberRole struct {
	UserID   string `json:"user_id"`
	ServerID string `json:"server_id"`
	RoleName string `json:"role_name"`
	RoleID   string `json:"role_id,omitempty"`
}

// BulkRoleAssignment represents multiple role assignments for a user
type BulkRoleAssignment struct {
	UserID      string                   `json:"user_id"`
	Assignments []DiscordRoleAssignment  `json:"assignments"`
}

// DiscordRoleAssignment represents a single role assignment
type DiscordRoleAssignment struct {
	ServerID string `json:"server_id"`
	RoleName string `json:"role_name"`
	Action   string `json:"action"` // "assign" or "remove"
}

// GetServers retrieves all managed Discord servers
func (ds *DiscordService) GetServers(ctx context.Context) ([]DiscordServer, error) {
	if ds.serviceURL == "" {
		return nil, fmt.Errorf("Discord service URL not configured")
	}

	url := fmt.Sprintf("%s/discord/servers", ds.serviceURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ds.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Discord service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discord service returned status %d", resp.StatusCode)
	}

	var servers []DiscordServer
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, fmt.Errorf("failed to decode servers response: %w", err)
	}

	return servers, nil
}

// GetServerRoles retrieves all roles for a specific server
func (ds *DiscordService) GetServerRoles(ctx context.Context, serverID string) ([]DiscordServerRole, error) {
	if ds.serviceURL == "" {
		return nil, fmt.Errorf("Discord service URL not configured")
	}

	url := fmt.Sprintf("%s/discord/servers/%s/roles", ds.serviceURL, serverID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ds.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Discord service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discord service returned status %d", resp.StatusCode)
	}

	var roles []DiscordServerRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode roles response: %w", err)
	}

	return roles, nil
}

// GetRoleByName retrieves a specific role by server and name
func (ds *DiscordService) GetRoleByName(ctx context.Context, serverID, roleName string) (*DiscordServerRole, error) {
	if ds.serviceURL == "" {
		return nil, fmt.Errorf("Discord service URL not configured")
	}

	url := fmt.Sprintf("%s/discord/roles/%s/%s", ds.serviceURL, serverID, roleName)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ds.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Discord service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Role not found
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discord service returned status %d", resp.StatusCode)
	}

	var role DiscordServerRole
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		return nil, fmt.Errorf("failed to decode role response: %w", err)
	}

	return &role, nil
}

// AssignRole assigns a role to a user on a specific server
func (ds *DiscordService) AssignRole(ctx context.Context, serverID, userID, roleName string) error {
	if ds.serviceURL == "" {
		return fmt.Errorf("Discord service URL not configured")
	}

	url := fmt.Sprintf("%s/discord/servers/%s/members/%s/roles", ds.serviceURL, serverID, userID)
	
	payload := map[string]string{
		"role_name": roleName,
		"action":    "assign",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ds.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Discord service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Discord service returned status %d", resp.StatusCode)
	}

	return nil
}

// RemoveRole removes a role from a user on a specific server
func (ds *DiscordService) RemoveRole(ctx context.Context, serverID, userID, roleName string) error {
	if ds.serviceURL == "" {
		return fmt.Errorf("Discord service URL not configured")
	}

	url := fmt.Sprintf("%s/discord/servers/%s/members/%s/roles/%s", ds.serviceURL, serverID, userID, roleName)
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ds.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Discord service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Discord service returned status %d", resp.StatusCode)
	}

	return nil
}

// GetUserRoles retrieves all roles for a user across all servers
func (ds *DiscordService) GetUserRoles(ctx context.Context, userID string) ([]DiscordMemberRole, error) {
	if ds.serviceURL == "" {
		return nil, fmt.Errorf("Discord service URL not configured")
	}

	url := fmt.Sprintf("%s/discord/members/%s/roles", ds.serviceURL, userID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ds.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Discord service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discord service returned status %d", resp.StatusCode)
	}

	var roles []DiscordMemberRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode roles response: %w", err)
	}

	return roles, nil
}

// BulkAssignRoles assigns/removes multiple roles for a user across servers
func (ds *DiscordService) BulkAssignRoles(ctx context.Context, userID string, assignments []DiscordRoleAssignment) error {
	if ds.serviceURL == "" {
		return fmt.Errorf("Discord service URL not configured")
	}

	if len(assignments) == 0 {
		return nil // Nothing to do
	}

	url := fmt.Sprintf("%s/discord/members/%s/roles/bulk", ds.serviceURL, userID)
	
	payload := BulkRoleAssignment{
		UserID:      userID,
		Assignments: assignments,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal bulk assignment request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ds.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Discord service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Discord service returned status %d", resp.StatusCode)
	}

	return nil
}

// SyncGroupRoles synchronizes Discord roles for all members of a group
func (ds *DiscordService) SyncGroupRoles(ctx context.Context, groupName string, members []GroupMemberInfo, discordRoles []DiscordRole) error {
	if ds.serviceURL == "" {
		slog.Info("Discord service URL not configured, skipping Discord sync for group", 
			slog.String("group", groupName))
		return nil
	}

	slog.Info("Syncing Discord roles for group", 
		slog.String("group", groupName),
		slog.Int("members", len(members)),
		slog.Int("discord_roles", len(discordRoles)))

	successCount := 0
	errorCount := 0

	for _, member := range members {
		err := ds.syncMemberRoles(ctx, member.CharacterID, discordRoles)
		if err != nil {
			slog.Warn("Failed to sync Discord roles for member",
				slog.Int("character_id", member.CharacterID),
				slog.String("group", groupName),
				slog.String("error", err.Error()))
			errorCount++
		} else {
			successCount++
		}
	}

	slog.Info("Group Discord role sync completed",
		slog.String("group", groupName),
		slog.Int("success", successCount),
		slog.Int("errors", errorCount))

	return nil
}

// syncMemberRoles syncs all Discord roles for a single member
func (ds *DiscordService) syncMemberRoles(ctx context.Context, characterID int, roles []DiscordRole) error {
	userID := fmt.Sprintf("%d", characterID) // Convert character ID to Discord user ID format
	
	var assignments []DiscordRoleAssignment
	for _, role := range roles {
		assignments = append(assignments, DiscordRoleAssignment{
			ServerID: role.ServerID,
			RoleName: role.RoleName,
			Action:   "assign",
		})
	}

	return ds.BulkAssignRoles(ctx, userID, assignments)
}

// ValidateServerAvailability checks if Discord servers are available
func (ds *DiscordService) ValidateServerAvailability(ctx context.Context) (map[string]bool, error) {
	servers, err := ds.GetServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get servers: %w", err)
	}

	availability := make(map[string]bool)
	for _, server := range servers {
		availability[server.ID] = server.Available
	}

	return availability, nil
}

// ValidateRoleExists checks if a role exists on a specific server
func (ds *DiscordService) ValidateRoleExists(ctx context.Context, serverID, roleName string) (bool, error) {
	role, err := ds.GetRoleByName(ctx, serverID, roleName)
	if err != nil {
		return false, err
	}
	
	return role != nil, nil
}

// GetDiscordHealthStatus returns the health status of the Discord service
func (ds *DiscordService) GetDiscordHealthStatus(ctx context.Context) (*DiscordHealthStatus, error) {
	if ds.serviceURL == "" {
		return &DiscordHealthStatus{
			Available:    false,
			ServiceURL:   "",
			ServerCount:  0,
			ErrorMessage: "Discord service URL not configured",
		}, nil
	}

	servers, err := ds.GetServers(ctx)
	if err != nil {
		return &DiscordHealthStatus{
			Available:    false,
			ServiceURL:   ds.serviceURL,
			ServerCount:  0,
			ErrorMessage: err.Error(),
		}, nil
	}

	availableServers := 0
	for _, server := range servers {
		if server.Available {
			availableServers++
		}
	}

	return &DiscordHealthStatus{
		Available:        true,
		ServiceURL:       ds.serviceURL,
		ServerCount:      len(servers),
		AvailableServers: availableServers,
		Servers:          servers,
	}, nil
}

// DiscordHealthStatus represents the health status of Discord integration
type DiscordHealthStatus struct {
	Available        bool            `json:"available"`
	ServiceURL       string          `json:"service_url"`
	ServerCount      int             `json:"server_count"`
	AvailableServers int             `json:"available_servers"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	Servers          []DiscordServer `json:"servers,omitempty"`
}

// IsDiscordEnabled returns true if Discord integration is enabled
func (ds *DiscordService) IsDiscordEnabled() bool {
	return ds.serviceURL != ""
}

// BatchProcessGroupRoles processes Discord role assignments for multiple groups
func (ds *DiscordService) BatchProcessGroupRoles(ctx context.Context, groups []Group, groupService *GroupService) error {
	if !ds.IsDiscordEnabled() {
		slog.Info("Discord integration disabled, skipping batch role processing")
		return nil
	}

	slog.Info("Starting batch Discord role processing", slog.Int("groups", len(groups)))

	totalMembers := 0
	successfulGroups := 0
	errorCount := 0

	for _, group := range groups {
		if len(group.DiscordRoles) == 0 {
			continue // No Discord roles configured
		}

		members, err := groupService.ListGroupMembers(ctx, group.ID.Hex())
		if err != nil {
			slog.Error("Failed to get members for batch Discord processing",
				slog.String("group", group.Name),
				slog.String("error", err.Error()))
			errorCount++
			continue
		}

		err = ds.SyncGroupRoles(ctx, group.Name, members, group.DiscordRoles)
		if err != nil {
			slog.Error("Failed to sync Discord roles for group",
				slog.String("group", group.Name),
				slog.String("error", err.Error()))
			errorCount++
		} else {
			successfulGroups++
			totalMembers += len(members)
		}
	}

	slog.Info("Batch Discord role processing completed",
		slog.Int("successful_groups", successfulGroups),
		slog.Int("total_members_processed", totalMembers),
		slog.Int("errors", errorCount))

	return nil
}