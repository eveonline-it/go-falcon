package middleware

import (
	"fmt"
	"net/http"
)

// DebugTestHandler creates a simple test handler that shows auth context information
func DebugTestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("\n[DEBUG] ===== DebugTestHandler START =====\n")
		fmt.Printf("[DEBUG] DebugTestHandler: Handling %s %s\n", r.Method, r.URL.Path)
		
		helper := NewContextHelper()
		authInfo := helper.GetAuthInfo(r)
		
		fmt.Printf("[DEBUG] DebugTestHandler: Auth Info:\n")
		fmt.Printf("  - IsAuthenticated: %t\n", authInfo.IsAuthenticated)
		fmt.Printf("  - UserID: %s\n", authInfo.UserID)
		fmt.Printf("  - PrimaryCharacterID: %d\n", authInfo.PrimaryCharacterID)
		fmt.Printf("  - RequestType: %s\n", authInfo.RequestType)
		fmt.Printf("  - HasExpandedContext: %t\n", authInfo.HasExpandedContext)
		
		if authInfo.HasExpandedContext {
			fmt.Printf("  - CharacterIDs: %v\n", authInfo.CharacterIDs)
			fmt.Printf("  - CorporationIDs: %v\n", authInfo.CorporationIDs)
			fmt.Printf("  - AllianceIDs: %v\n", authInfo.AllianceIDs)
			if authInfo.PrimaryCharacter != nil {
				fmt.Printf("  - PrimaryCharacter: %d (%s) Corp:%d Alliance:%d\n", 
					authInfo.PrimaryCharacter.ID,
					authInfo.PrimaryCharacter.Name,
					authInfo.PrimaryCharacter.CorporationID,
					authInfo.PrimaryCharacter.AllianceID)
			}
		}
		
		// Create response (for future use)
		_ = map[string]interface{}{
			"message": "Debug test endpoint accessed successfully",
			"auth_info": authInfo,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		if authInfo.IsAuthenticated {
			w.Write([]byte(fmt.Sprintf(`{
	"message": "Debug test endpoint accessed successfully",
	"authenticated": true,
	"user_id": "%s",
	"primary_character_id": %d,
	"request_type": "%s",
	"has_expanded_context": %t
}`, authInfo.UserID, authInfo.PrimaryCharacterID, authInfo.RequestType, authInfo.HasExpandedContext)))
		} else {
			w.Write([]byte(`{
	"message": "Debug test endpoint accessed successfully",
	"authenticated": false
}`))
		}
		
		fmt.Printf("[DEBUG] DebugTestHandler: Response sent\n")
		fmt.Printf("[DEBUG] ===== DebugTestHandler END =====\n\n")
	}
}

// SetupDebugRoutes adds debug routes to the router for testing middleware
func SetupDebugRoutes(factory *MiddlewareFactory, mux *http.ServeMux) {
	fmt.Printf("[DEBUG] SetupDebugRoutes: Adding debug test routes\n")
	
	// Public endpoint with optional auth
	mux.Handle("/debug/public", factory.PublicWithOptionalAuth()(DebugTestHandler()))
	fmt.Printf("[DEBUG] SetupDebugRoutes: Added /debug/public (optional auth)\n")
	
	// Authenticated endpoint
	mux.Handle("/debug/auth", factory.RequireBasicAuth()(DebugTestHandler()))
	fmt.Printf("[DEBUG] SetupDebugRoutes: Added /debug/auth (require auth)\n")
	
	// Authenticated endpoint with character resolution
	mux.Handle("/debug/characters", factory.RequireAuthWithCharacters()(DebugTestHandler()))
	fmt.Printf("[DEBUG] SetupDebugRoutes: Added /debug/characters (require auth + characters)\n")
	
	fmt.Printf("[DEBUG] SetupDebugRoutes: All debug routes added\n")
}

// AddDebugRoutesToChi adds debug routes to a Chi router
func AddDebugRoutesToChi(factory *MiddlewareFactory, r interface{}) {
	// This would be for Chi router integration
	fmt.Printf("[DEBUG] AddDebugRoutesToChi: Debug routes would be added to Chi router\n")
	fmt.Printf("[DEBUG] Available debug endpoints:\n")
	fmt.Printf("  - GET /debug/public (optional auth)\n")
	fmt.Printf("  - GET /debug/auth (require auth)\n") 
	fmt.Printf("  - GET /debug/characters (require auth + characters)\n")
}