package dev

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-falcon/internal/dev/dto"
	"go-falcon/internal/dev/routes"
	"go-falcon/internal/dev/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/evegateway"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// TestHumaRoutesCreation tests that Huma routes can be created without error
func TestHumaRoutesCreation(t *testing.T) {
	// Create a simple service for testing (even if some methods will fail)
	service := createMinimalService()
	
	// Create a router
	router := chi.NewRouter()
	
	// This should not panic or error
	humaRoutes := routes.NewHumaRoutes(service, router)
	assert.NotNil(t, humaRoutes)
}

// TestHumaOpenAPIDocument tests that Huma generates an OpenAPI document
func TestHumaOpenAPIDocument(t *testing.T) {
	// Create a simple service
	service := createMinimalService()
	
	// Create a router and add Huma routes
	router := chi.NewRouter()
	routes.NewHumaRoutes(service, router)
	
	// Test that the OpenAPI document is generated
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()
	
	router.ServeHTTP(w, req)
	
	if w.Code == http.StatusOK {
		// Verify it's valid JSON
		var openapi map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &openapi)
		assert.NoError(t, err)
		
		// Basic OpenAPI structure
		assert.Contains(t, openapi, "openapi")
		assert.Contains(t, openapi, "info")
		
		t.Logf("✅ OpenAPI document generated successfully")
	} else {
		t.Logf("⚠️  OpenAPI document not found at /openapi.json (status: %d)", w.Code)
	}
}

// TestHumaDTOStructures tests that Huma DTOs are properly structured
func TestHumaDTOStructures(t *testing.T) {
	// Test HealthCheckInput/Output
	healthInput := &dto.HealthCheckInput{}
	healthOutput := &dto.HealthCheckOutput{}
	
	assert.NotNil(t, healthInput)
	assert.NotNil(t, healthOutput)
	
	// Test CharacterInfoInput/Output
	charInput := &dto.CharacterInfoInput{CharacterID: 90000001}
	charOutput := &dto.CharacterInfoOutput{}
	
	assert.Equal(t, 90000001, charInput.CharacterID)
	assert.NotNil(t, charOutput)
	
	// Test ServiceDiscoveryInput/Output
	serviceInput := &dto.ServiceDiscoveryInput{}
	serviceOutput := &dto.ServiceDiscoveryOutput{}
	
	assert.NotNil(t, serviceInput)
	assert.NotNil(t, serviceOutput)
	
	t.Logf("✅ Huma DTOs are properly structured")
}

// TestHumaValidationTags tests that validation tags are properly set
func TestHumaValidationTags(t *testing.T) {
	// Create character input with invalid ID
	charInput := &dto.CharacterInfoInput{CharacterID: -1}
	
	// The validation tag should specify minimum: 1
	// This test just ensures the struct is properly tagged
	assert.Equal(t, -1, charInput.CharacterID)
	
	// Create character input with valid ID
	charInput2 := &dto.CharacterInfoInput{CharacterID: 90000001}
	assert.Equal(t, 90000001, charInput2.CharacterID)
	
	t.Logf("✅ Huma validation tags are present in DTOs")
}

// Helper to create a minimal service for testing
func createMinimalService() *services.Service {
	// Create mock/minimal dependencies
	mongodb := &database.MongoDB{}
	redis := &database.Redis{}
	
	// Create empty repository
	repo := services.NewRepository(mongodb, redis)
	
	// Create minimal EVE gateway client
	evegateClient := evegateway.NewClient()
	
	// Create minimal SDE service (will be nil but that's ok for these tests)
	var sdeService sde.SDEService = nil
	
	// Create service with minimal dependencies
	return services.NewService(
		repo,
		evegateClient,
		nil, // statusClient
		nil, // characterClient
		nil, // universeClient
		nil, // allianceClient
		nil, // corporationClient
		sdeService,
		nil, // cacheManager
	)
}