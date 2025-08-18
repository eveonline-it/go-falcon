package middleware

import (
	"context"
	"fmt"
	"net/http"
)

// ContextHelper provides helper functions for working with authentication context
type ContextHelper struct{}

// NewContextHelper creates a new context helper
func NewContextHelper() *ContextHelper {
	return &ContextHelper{}
}

// GetUserID extracts user ID from request context
func (h *ContextHelper) GetUserID(r *http.Request) string {
	if authCtx := GetAuthContext(r.Context()); authCtx != nil {
		return authCtx.UserID
	}
	return ""
}

// GetPrimaryCharacterID extracts primary character ID from request context
func (h *ContextHelper) GetPrimaryCharacterID(r *http.Request) int64 {
	if authCtx := GetAuthContext(r.Context()); authCtx != nil {
		return authCtx.PrimaryCharID
	}
	return 0
}

// GetAllCharacterIDs extracts all character IDs from expanded context
func (h *ContextHelper) GetAllCharacterIDs(r *http.Request) []int64 {
	if expandedCtx := GetExpandedAuthContext(r.Context()); expandedCtx != nil {
		return expandedCtx.CharacterIDs
	}
	return nil
}

// GetAllCorporationIDs extracts all corporation IDs from expanded context
func (h *ContextHelper) GetAllCorporationIDs(r *http.Request) []int64 {
	if expandedCtx := GetExpandedAuthContext(r.Context()); expandedCtx != nil {
		return expandedCtx.CorporationIDs
	}
	return nil
}

// GetAllAllianceIDs extracts all alliance IDs from expanded context
func (h *ContextHelper) GetAllAllianceIDs(r *http.Request) []int64 {
	if expandedCtx := GetExpandedAuthContext(r.Context()); expandedCtx != nil {
		return expandedCtx.AllianceIDs
	}
	return nil
}

// IsAuthenticated checks if request has authenticated user
func (h *ContextHelper) IsAuthenticated(r *http.Request) bool {
	if authCtx := GetAuthContext(r.Context()); authCtx != nil {
		return authCtx.IsAuthenticated
	}
	return false
}

// GetRequestType returns the type of authentication used (cookie or bearer)
func (h *ContextHelper) GetRequestType(r *http.Request) string {
	if authCtx := GetAuthContext(r.Context()); authCtx != nil {
		return authCtx.RequestType
	}
	return ""
}

// HasExpandedContext checks if request has expanded character context
func (h *ContextHelper) HasExpandedContext(r *http.Request) bool {
	return GetExpandedAuthContext(r.Context()) != nil
}

// DebugAuthContext prints debug information about authentication context
func (h *ContextHelper) DebugAuthContext(r *http.Request) {
	fmt.Printf("[DEBUG] ContextHelper: Request URL: %s %s\n", r.Method, r.URL.Path)
	
	authCtx := GetAuthContext(r.Context())
	if authCtx == nil {
		fmt.Printf("[DEBUG] ContextHelper: No auth context found\n")
		return
	}
	
	fmt.Printf("[DEBUG] ContextHelper: Auth context found:\n")
	fmt.Printf("  - UserID: %s\n", authCtx.UserID)
	fmt.Printf("  - PrimaryCharID: %d\n", authCtx.PrimaryCharID)
	fmt.Printf("  - RequestType: %s\n", authCtx.RequestType)
	fmt.Printf("  - IsAuthenticated: %t\n", authCtx.IsAuthenticated)
	
	expandedCtx := GetExpandedAuthContext(r.Context())
	if expandedCtx == nil {
		fmt.Printf("[DEBUG] ContextHelper: No expanded context found\n")
		return
	}
	
	fmt.Printf("[DEBUG] ContextHelper: Expanded context found:\n")
	fmt.Printf("  - CharacterIDs (%d): %v\n", len(expandedCtx.CharacterIDs), expandedCtx.CharacterIDs)
	fmt.Printf("  - CorporationIDs (%d): %v\n", len(expandedCtx.CorporationIDs), expandedCtx.CorporationIDs)
	fmt.Printf("  - AllianceIDs (%d): %v\n", len(expandedCtx.AllianceIDs), expandedCtx.AllianceIDs)
	fmt.Printf("  - PrimaryCharacter: %d (%s) Corp:%d Alliance:%d\n", 
		expandedCtx.PrimaryCharacter.ID, 
		expandedCtx.PrimaryCharacter.Name, 
		expandedCtx.PrimaryCharacter.CorporationID, 
		expandedCtx.PrimaryCharacter.AllianceID)
}

// DebugMiddleware is a debugging middleware that prints auth context information
func DebugMiddleware() func(http.Handler) http.Handler {
	helper := NewContextHelper()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("\n[DEBUG] ===== DebugMiddleware START =====\n")
			fmt.Printf("[DEBUG] DebugMiddleware: Processing request %s %s\n", r.Method, r.URL.Path)
			fmt.Printf("[DEBUG] DebugMiddleware: Headers:\n")
			for name, values := range r.Header {
				if name == "Authorization" {
					if len(values) > 0 && values[0] != "" {
						fmt.Printf("  %s: Bearer [token present - length %d]\n", name, len(values[0]))
					} else {
						fmt.Printf("  %s: [empty]\n", name)
					}
				} else if name == "Cookie" {
					if len(values) > 0 && values[0] != "" {
						fmt.Printf("  %s: [cookies present - length %d]\n", name, len(values[0]))
					} else {
						fmt.Printf("  %s: [empty]\n", name)
					}
				}
			}
			
			helper.DebugAuthContext(r)
			
			fmt.Printf("[DEBUG] DebugMiddleware: Calling next handler\n")
			next.ServeHTTP(w, r)
			fmt.Printf("[DEBUG] DebugMiddleware: Request completed\n")
			fmt.Printf("[DEBUG] ===== DebugMiddleware END =====\n\n")
		})
	}
}

// AuthInfo provides comprehensive authentication information for a request
type AuthInfo struct {
	IsAuthenticated      bool     `json:"is_authenticated"`
	UserID              string   `json:"user_id,omitempty"`
	PrimaryCharacterID   int64    `json:"primary_character_id,omitempty"`
	RequestType         string   `json:"request_type,omitempty"`
	HasExpandedContext  bool     `json:"has_expanded_context"`
	CharacterIDs        []int64  `json:"character_ids,omitempty"`
	CorporationIDs      []int64  `json:"corporation_ids,omitempty"`
	AllianceIDs         []int64  `json:"alliance_ids,omitempty"`
	PrimaryCharacter    *CharacterInfo `json:"primary_character,omitempty"`
}

// CharacterInfo represents character information
type CharacterInfo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	CorporationID int64  `json:"corporation_id"`
	AllianceID    int64  `json:"alliance_id,omitempty"`
}

// GetAuthInfo extracts comprehensive authentication information from request
func (h *ContextHelper) GetAuthInfo(r *http.Request) *AuthInfo {
	info := &AuthInfo{}
	
	authCtx := GetAuthContext(r.Context())
	if authCtx != nil {
		info.IsAuthenticated = authCtx.IsAuthenticated
		info.UserID = authCtx.UserID
		info.PrimaryCharacterID = authCtx.PrimaryCharID
		info.RequestType = authCtx.RequestType
	}
	
	expandedCtx := GetExpandedAuthContext(r.Context())
	if expandedCtx != nil {
		info.HasExpandedContext = true
		info.CharacterIDs = expandedCtx.CharacterIDs
		info.CorporationIDs = expandedCtx.CorporationIDs
		info.AllianceIDs = expandedCtx.AllianceIDs
		info.PrimaryCharacter = &CharacterInfo{
			ID:            expandedCtx.PrimaryCharacter.ID,
			Name:          expandedCtx.PrimaryCharacter.Name,
			CorporationID: expandedCtx.PrimaryCharacter.CorporationID,
			AllianceID:    expandedCtx.PrimaryCharacter.AllianceID,
		}
	}
	
	return info
}

// WithAuthInfo adds authentication information to request context for easy access
func WithAuthInfo(r *http.Request) context.Context {
	helper := NewContextHelper()
	authInfo := helper.GetAuthInfo(r)
	return context.WithValue(r.Context(), "auth_info", authInfo)
}

// GetAuthInfoFromContext extracts AuthInfo from context
func GetAuthInfoFromContext(ctx context.Context) *AuthInfo {
	if info, ok := ctx.Value("auth_info").(*AuthInfo); ok {
		return info
	}
	return nil
}