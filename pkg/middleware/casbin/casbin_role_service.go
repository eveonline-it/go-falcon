package casbin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/casbin/casbin/v2"
)

// RoleAssignmentService provides role-based assignment operations using CASBIN
type RoleAssignmentService struct {
	enforcer *casbin.Enforcer
	logger   *slog.Logger
}

// NewRoleAssignmentService creates a new role assignment service
func NewRoleAssignmentService(enforcer *casbin.Enforcer) *RoleAssignmentService {
	return &RoleAssignmentService{
		enforcer: enforcer,
		logger:   slog.Default(),
	}
}

// AssignRole assigns a role to a user or character
func (s *RoleAssignmentService) AssignRole(ctx context.Context, req *RoleAssignmentRequest) (*RoleAssignmentResponse, error) {
	s.logger.Info("Assigning role", 
		slog.String("user_id", req.UserID),
		slog.String("role", req.Role),
		slog.String("domain", req.Domain))

	// Build subject identifier
	subject := fmt.Sprintf("user:%s", req.UserID)
	if req.CharacterID != nil {
		subject = fmt.Sprintf("character:%d", *req.CharacterID)
	}

	// Default domain to global if not specified
	domain := req.Domain
	if domain == "" {
		domain = "global"
	}

	// Add role using CASBIN
	added, err := s.enforcer.AddRoleForUser(subject, req.Role, domain)
	if err != nil {
		s.logger.Error("Failed to assign role",
			slog.String("subject", subject),
			slog.String("role", req.Role),
			slog.String("domain", domain),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to assign role: %w", err)
	}

	var message string
	if added {
		message = fmt.Sprintf("Role '%s' successfully assigned to %s", req.Role, subject)
	} else {
		message = fmt.Sprintf("Role '%s' was already assigned to %s", req.Role, subject)
	}

	s.logger.Info("Role assignment completed",
		slog.String("subject", subject),
		slog.String("role", req.Role),
		slog.Bool("newly_added", added))

	return &RoleAssignmentResponse{
		Success:   true,
		Message:   message,
		UserID:    req.UserID,
		Role:      req.Role,
		Domain:    domain,
		Timestamp: time.Now(),
	}, nil
}

// RemoveRole removes a role from a user or character
func (s *RoleAssignmentService) RemoveRole(ctx context.Context, req *RoleRemovalRequest) (*RoleAssignmentResponse, error) {
	s.logger.Info("Removing role",
		slog.String("user_id", req.UserID),
		slog.String("role", req.Role),
		slog.String("domain", req.Domain))

	// Build subject identifier
	subject := fmt.Sprintf("user:%s", req.UserID)
	if req.CharacterID != nil {
		subject = fmt.Sprintf("character:%d", *req.CharacterID)
	}

	// Default domain to global if not specified
	domain := req.Domain
	if domain == "" {
		domain = "global"
	}

	// Remove role using CASBIN
	removed, err := s.enforcer.DeleteRoleForUser(subject, req.Role, domain)
	if err != nil {
		s.logger.Error("Failed to remove role",
			slog.String("subject", subject),
			slog.String("role", req.Role),
			slog.String("domain", domain),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to remove role: %w", err)
	}

	var message string
	if removed {
		message = fmt.Sprintf("Role '%s' successfully removed from %s", req.Role, subject)
	} else {
		message = fmt.Sprintf("Role '%s' was not assigned to %s", req.Role, subject)
	}

	s.logger.Info("Role removal completed",
		slog.String("subject", subject),
		slog.String("role", req.Role),
		slog.Bool("was_removed", removed))

	return &RoleAssignmentResponse{
		Success:   true,
		Message:   message,
		UserID:    req.UserID,
		Role:      req.Role,
		Domain:    domain,
		Timestamp: time.Now(),
	}, nil
}

// AssignPolicy assigns a permission policy to a subject (user, role, character, etc.)
func (s *RoleAssignmentService) AssignPolicy(ctx context.Context, req *PolicyAssignmentRequest) (*PolicyAssignmentResponse, error) {
	s.logger.Info("Assigning policy",
		slog.String("subject", req.Subject),
		slog.String("resource", req.Resource),
		slog.String("action", req.Action),
		slog.String("effect", req.Effect))

	// Default domain to global if not specified
	domain := req.Domain
	if domain == "" {
		domain = "global"
	}

	// Build permission string
	permission := fmt.Sprintf("%s.%s", req.Resource, req.Action)

	// Add policy using CASBIN (5-field format: sub, obj, act, dom, eft)
	added, err := s.enforcer.AddPolicy(req.Subject, permission, req.Action, domain, req.Effect)
	if err != nil {
		s.logger.Error("Failed to assign policy",
			slog.String("subject", req.Subject),
			slog.String("permission", permission),
			slog.String("domain", domain),
			slog.String("effect", req.Effect),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to assign policy: %w", err)
	}

	var message string
	if added {
		message = fmt.Sprintf("Permission '%s' (%s) successfully assigned to %s", permission, req.Effect, req.Subject)
	} else {
		message = fmt.Sprintf("Permission '%s' (%s) was already assigned to %s", permission, req.Effect, req.Subject)
	}

	s.logger.Info("Policy assignment completed",
		slog.String("subject", req.Subject),
		slog.String("permission", permission),
		slog.String("effect", req.Effect),
		slog.Bool("newly_added", added))

	return &PolicyAssignmentResponse{
		Success:   true,
		Message:   message,
		Subject:   req.Subject,
		Resource:  req.Resource,
		Action:    req.Action,
		Effect:    req.Effect,
		Timestamp: time.Now(),
	}, nil
}

// RemovePolicy removes a permission policy from a subject
func (s *RoleAssignmentService) RemovePolicy(ctx context.Context, req *PolicyRemovalRequest) (*PolicyAssignmentResponse, error) {
	s.logger.Info("Removing policy",
		slog.String("subject", req.Subject),
		slog.String("resource", req.Resource),
		slog.String("action", req.Action),
		slog.String("effect", req.Effect))

	// Default domain to global if not specified
	domain := req.Domain
	if domain == "" {
		domain = "global"
	}

	// Build permission string
	permission := fmt.Sprintf("%s.%s", req.Resource, req.Action)

	// Remove policy using CASBIN
	removed, err := s.enforcer.RemovePolicy(req.Subject, permission, req.Action, domain, req.Effect)
	if err != nil {
		s.logger.Error("Failed to remove policy",
			slog.String("subject", req.Subject),
			slog.String("permission", permission),
			slog.String("domain", domain),
			slog.String("effect", req.Effect),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to remove policy: %w", err)
	}

	var message string
	if removed {
		message = fmt.Sprintf("Permission '%s' (%s) successfully removed from %s", permission, req.Effect, req.Subject)
	} else {
		message = fmt.Sprintf("Permission '%s' (%s) was not assigned to %s", permission, req.Effect, req.Subject)
	}

	s.logger.Info("Policy removal completed",
		slog.String("subject", req.Subject),
		slog.String("permission", permission),
		slog.String("effect", req.Effect),
		slog.Bool("was_removed", removed))

	return &PolicyAssignmentResponse{
		Success:   true,
		Message:   message,
		Subject:   req.Subject,
		Resource:  req.Resource,
		Action:    req.Action,
		Effect:    req.Effect,
		Timestamp: time.Now(),
	}, nil
}

// CheckPermission checks if a user has a specific permission
func (s *RoleAssignmentService) CheckPermission(ctx context.Context, req *PermissionCheckRequestWithChar) (*PermissionCheckResponseWithChar, error) {
	s.logger.Debug("Checking permission",
		slog.String("user_id", req.UserID),
		slog.String("resource", req.Resource),
		slog.String("action", req.Action))

	// Build subjects to check in hierarchical order
	subjects := []string{fmt.Sprintf("user:%s", req.UserID)}
	if req.CharacterID != nil {
		subjects = []string{fmt.Sprintf("character:%d", *req.CharacterID), subjects[0]}
	}

	// Default domain to global if not specified
	domain := req.Domain
	if domain == "" {
		domain = "global"
	}

	// Build permission string
	permission := fmt.Sprintf("%s.%s", req.Resource, req.Action)

	var matchedRules []string
	hasPermission := false

	// Check each subject
	for _, subject := range subjects {
		allowed, err := s.enforcer.Enforce(subject, permission, req.Action, domain)
		if err != nil {
			s.logger.Error("Error checking permission",
				slog.String("subject", subject),
				slog.String("permission", permission),
				slog.String("error", err.Error()))
			continue
		}

		if allowed {
			hasPermission = true
			matchedRules = append(matchedRules, fmt.Sprintf("%s -> %s (%s)", subject, permission, "allow"))
			break // First allow wins
		}
	}

	s.logger.Debug("Permission check completed",
		slog.String("user_id", req.UserID),
		slog.String("permission", permission),
		slog.Bool("has_permission", hasPermission),
		slog.Any("matched_rules", matchedRules))

	return &PermissionCheckResponseWithChar{
		HasPermission: hasPermission,
		UserID:        req.UserID,
		CharacterID:   req.CharacterID,
		Resource:      req.Resource,
		Action:        req.Action,
		MatchedRules:  matchedRules,
		CheckedAt:     time.Now(),
	}, nil
}

// GetUserRoles gets all roles assigned to a user
func (s *RoleAssignmentService) GetUserRoles(ctx context.Context, userID string) (*UserRolesResponse, error) {
	s.logger.Debug("Getting user roles", slog.String("user_id", userID))

	subject := fmt.Sprintf("user:%s", userID)
	
	// Get roles for user
	roles, err := s.enforcer.GetRolesForUser(subject)
	if err != nil {
		s.logger.Error("Failed to get user roles",
			slog.String("user_id", userID),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Convert to RoleInfo structs
	roleInfos := make([]RoleInfo, len(roles))
	for i, role := range roles {
		roleInfos[i] = RoleInfo{
			Role:   role,
			Domain: "global", // TODO: Support multi-domain roles
		}
	}

	return &UserRolesResponse{
		UserID: userID,
		Roles:  roleInfos,
		Total:  len(roleInfos),
	}, nil
}

// GetRolePolicies gets all policies assigned to a role
func (s *RoleAssignmentService) GetRolePolicies(ctx context.Context, role string) (*RolePoliciesResponse, error) {
	s.logger.Debug("Getting role policies", slog.String("role", role))

	// For now, return empty list until CASBIN API is clarified
	// TODO: Implement proper policy filtering when CASBIN API is stable
	rolePolicies := make([]PolicyInfo, 0)

	return &RolePoliciesResponse{
		Role:     role,
		Policies: rolePolicies,
		Total:    len(rolePolicies),
	}, nil
}

// BulkAssignRole assigns a role to multiple users at once
func (s *RoleAssignmentService) BulkAssignRole(ctx context.Context, req *BulkRoleAssignmentRequest) (*BulkRoleAssignmentResponse, error) {
	s.logger.Info("Bulk assigning role",
		slog.String("role", req.Role),
		slog.Int("user_count", len(req.UserIDs)))

	domain := req.Domain
	if domain == "" {
		domain = "global"
	}

	var success []string
	var failed []string

	for _, userID := range req.UserIDs {
		subject := fmt.Sprintf("user:%s", userID)
		
		added, err := s.enforcer.AddRoleForUser(subject, req.Role, domain)
		if err != nil {
			s.logger.Error("Failed to assign role in bulk operation",
				slog.String("user_id", userID),
				slog.String("role", req.Role),
				slog.String("error", err.Error()))
			failed = append(failed, userID)
		} else {
			success = append(success, userID)
			if added {
				s.logger.Debug("Role assigned in bulk operation",
					slog.String("user_id", userID),
					slog.String("role", req.Role))
			}
		}
	}

	return &BulkRoleAssignmentResponse{
		Success:      success,
		Failed:       failed,
		Total:        len(req.UserIDs),
		SuccessCount: len(success),
		FailureCount: len(failed),
		Role:         req.Role,
		ProcessedAt:  time.Now(),
	}, nil
}