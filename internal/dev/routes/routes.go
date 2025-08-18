package routes

import (
	"context"

	"go-falcon/internal/dev/dto"
	"go-falcon/internal/dev/services"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// Routes handles Huma-based HTTP routing for the Dev module
type Routes struct {
	service *services.Service
	api     huma.API
}

// NewRoutes creates a new Huma Dev routes handler
func NewRoutes(service *services.Service, router chi.Router) *Routes {
	// Create Huma API with Chi adapter
	config := huma.DefaultConfig("Go Falcon Dev Module", "1.0.0")
	config.Info.Description = "Development tools and testing utilities for EVE Online integration"
	
	api := humachi.New(router, config)

	hr := &Routes{
		service: service,
		api:     api,
	}

	// Register all routes
	hr.registerRoutes()

	return hr
}

// RegisterDevRoutes registers dev routes on a shared Huma API
func RegisterDevRoutes(api huma.API, basePath string, service *services.Service) {
	// Public health check endpoint
	huma.Get(api, basePath+"/health", func(ctx context.Context, input *dto.HealthCheckInput) (*dto.HealthCheckOutput, error) {
		response, err := service.GetHealthStatus(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get health status", err)
		}
		return &dto.HealthCheckOutput{Body: *response}, nil
	})

	// Service discovery endpoint
	huma.Get(api, basePath+"/services", func(ctx context.Context, input *dto.ServiceDiscoveryInput) (*dto.ServiceDiscoveryOutput, error) {
		response, err := service.GetServices(ctx, input.ServiceName, input.Detailed)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get services", err)
		}
		return &dto.ServiceDiscoveryOutput{Body: *response}, nil
	})

	// ESI testing endpoints (protected)
	huma.Get(api, basePath+"/esi/status", func(ctx context.Context, input *dto.ESIStatusInput) (*dto.ESIStatusOutput, error) {
		response, err := service.GetESIStatus(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get ESI status", err)
		}
		return &dto.ESIStatusOutput{Body: *response}, nil
	})

	huma.Get(api, basePath+"/character/{character_id}", func(ctx context.Context, input *dto.CharacterInfoInput) (*dto.CharacterInfoOutput, error) {
		charReq := &dto.CharacterRequest{CharacterID: input.CharacterID}
		response, err := service.GetCharacterInfo(ctx, charReq)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to get character info", err)
		}
		return &dto.CharacterInfoOutput{Body: *response}, nil
	})

	huma.Get(api, basePath+"/alliance/{alliance_id}", func(ctx context.Context, input *dto.AllianceInfoInput) (*dto.AllianceInfoOutput, error) {
		allianceReq := &dto.AllianceRequest{AllianceID: input.AllianceID}
		response, err := service.GetAllianceInfo(ctx, allianceReq)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to get alliance info", err)
		}
		return &dto.AllianceInfoOutput{Body: *response}, nil
	})

	huma.Get(api, basePath+"/corporation/{corporation_id}", func(ctx context.Context, input *dto.CorporationInfoInput) (*dto.CorporationInfoOutput, error) {
		corpReq := &dto.CorporationRequest{CorporationID: input.CorporationID}
		response, err := service.GetCorporationInfo(ctx, corpReq)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to get corporation info", err)
		}
		return &dto.CorporationInfoOutput{Body: *response}, nil
	})

	huma.Get(api, basePath+"/system/{system_id}", func(ctx context.Context, input *dto.SystemInfoInput) (*dto.SystemInfoOutput, error) {
		systemReq := &dto.SystemRequest{SystemID: input.SystemID}
		response, err := service.GetSystemInfo(ctx, systemReq)
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to get system info", err)
		}
		return &dto.SystemInfoOutput{Body: *response}, nil
	})

	// SDE testing endpoints (protected)
	huma.Get(api, basePath+"/sde/status", func(ctx context.Context, input *dto.SDEStatusInput) (*dto.SDEStatusOutput, error) {
		response, err := service.GetSDEStatus(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get SDE status", err)
		}
		return &dto.SDEStatusOutput{Body: *response}, nil
	})

	huma.Get(api, basePath+"/sde/types", func(ctx context.Context, input *dto.SDETypesInput) (*dto.SDETypesOutput, error) {
		var publishedPtr *bool
		if input.Published {
			publishedPtr = &input.Published
		}
		typeReq := &dto.SDETypeRequest{
			TypeID:    0, // Get all types
			Published: publishedPtr,
		}
		response, err := service.GetSDETypes(ctx, typeReq)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get SDE types", err)
		}
		return &dto.SDETypesOutput{Body: *response}, nil
	})

	huma.Get(api, basePath+"/sde/entity/{type}/{id}", func(ctx context.Context, input *dto.SDEEntityInput) (*dto.SDEEntityOutput, error) {
		entityReq := &dto.SDEEntityRequest{Type: input.Type, ID: input.ID}
		response, err := service.GetSDEEntity(ctx, entityReq)
		if err != nil {
			return nil, huma.Error404NotFound("Entity not found", err)
		}
		return &dto.SDEEntityOutput{Body: *response}, nil
	})

	// Additional universe and Redis SDE endpoints
	huma.Get(api, basePath+"/universe/{type}/{region}/systems", func(ctx context.Context, input *dto.UniverseSystemsInput) (*dto.UniverseSystemsOutput, error) {
		universeReq := &dto.UniverseRequest{
			Type:          input.Type,
			Region:        input.Region,
			Constellation: input.Constellation,
		}
		response, err := service.GetUniverseSystems(ctx, universeReq)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get universe systems", err)
		}
		return &dto.UniverseSystemsOutput{Body: *response}, nil
	})

	huma.Get(api, basePath+"/sde/redis/{type}/{id}", func(ctx context.Context, input *dto.RedisSDEEntityInput) (*dto.RedisSDEEntityOutput, error) {
		redisReq := &dto.RedisSDERequest{Type: input.Type, ID: input.ID}
		response, err := service.GetRedisSDEEntity(ctx, redisReq)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get Redis SDE entity", err)
		}
		return &dto.RedisSDEEntityOutput{Body: *response}, nil
	})
}

// registerRoutes registers all Dev module routes with Huma
func (hr *Routes) registerRoutes() {
	// Public health check endpoint
	huma.Get(hr.api, "/health", hr.healthCheck)

	// Service discovery endpoint
	huma.Get(hr.api, "/services", hr.getServices)

	// ESI testing endpoints (protected)
	huma.Get(hr.api, "/esi/status", hr.getESIStatus)
	huma.Get(hr.api, "/character/{character_id}", hr.getCharacterInfo)
	huma.Get(hr.api, "/alliance/{alliance_id}", hr.getAllianceInfo)
	huma.Get(hr.api, "/corporation/{corporation_id}", hr.getCorporationInfo)
	huma.Get(hr.api, "/system/{system_id}", hr.getSystemInfo)

	// SDE testing endpoints (protected)
	huma.Get(hr.api, "/sde/status", hr.getSDEStatus)
	huma.Get(hr.api, "/sde/types", hr.getSDETypes)
	huma.Get(hr.api, "/sde/entity/{type}/{id}", hr.getSDEEntity)

	// Universe data endpoints
	huma.Get(hr.api, "/universe/{type}/{region}/systems", hr.getUniverseSystems)

	// Redis SDE endpoints  
	huma.Get(hr.api, "/sde/redis/{type}/{id}", hr.getRedisSDEEntity)
}

// Health check handlers

func (hr *Routes) healthCheck(ctx context.Context, input *dto.HealthCheckInput) (*dto.HealthCheckOutput, error) {
	response, err := hr.service.GetHealthStatus(ctx)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to get health status", err)
	}

	return &dto.HealthCheckOutput{Body: *response}, nil
}

func (hr *Routes) getServices(ctx context.Context, input *dto.ServiceDiscoveryInput) (*dto.ServiceDiscoveryOutput, error) {
	response, err := hr.service.GetServices(ctx, input.ServiceName, input.Detailed)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get services", err)
	}

	return &dto.ServiceDiscoveryOutput{Body: *response}, nil
}

// ESI endpoints

func (hr *Routes) getESIStatus(ctx context.Context, input *dto.ESIStatusInput) (*dto.ESIStatusOutput, error) {
	response, err := hr.service.GetESIStatus(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get ESI status", err)
	}

	return &dto.ESIStatusOutput{Body: *response}, nil
}

func (hr *Routes) getCharacterInfo(ctx context.Context, input *dto.CharacterInfoInput) (*dto.CharacterInfoOutput, error) {
	// Example of how to use authentication in Huma operations
	// Note: This would require auth service integration in the dev module
	// For now, this serves as a pattern demonstration
	
	// Validate authentication if auth middleware was available
	// humaAuth := middleware.NewHumaAuthMiddleware(authService)
	// user, err := humaAuth.ValidateAuthFromHeaders(input.Authorization, input.Cookie)
	// if err != nil {
	//     return nil, err // Returns proper Huma error response
	// }
	// 
	// Add authenticated user to context for downstream operations
	// ctx = context.WithValue(ctx, middleware.HumaAuthContextKeyUser, user)
	
	charReq := &dto.CharacterRequest{CharacterID: input.CharacterID}
	response, err := hr.service.GetCharacterInfo(ctx, charReq)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get character info", err)
	}

	return &dto.CharacterInfoOutput{Body: *response}, nil
}

func (hr *Routes) getAllianceInfo(ctx context.Context, input *dto.AllianceInfoInput) (*dto.AllianceInfoOutput, error) {
	allianceReq := &dto.AllianceRequest{AllianceID: input.AllianceID}
	response, err := hr.service.GetAllianceInfo(ctx, allianceReq)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get alliance info", err)
	}

	return &dto.AllianceInfoOutput{Body: *response}, nil
}

func (hr *Routes) getCorporationInfo(ctx context.Context, input *dto.CorporationInfoInput) (*dto.CorporationInfoOutput, error) {
	corpReq := &dto.CorporationRequest{CorporationID: input.CorporationID}
	response, err := hr.service.GetCorporationInfo(ctx, corpReq)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get corporation info", err)
	}

	return &dto.CorporationInfoOutput{Body: *response}, nil
}

func (hr *Routes) getSystemInfo(ctx context.Context, input *dto.SystemInfoInput) (*dto.SystemInfoOutput, error) {
	systemReq := &dto.SystemRequest{SystemID: input.SystemID}
	response, err := hr.service.GetSystemInfo(ctx, systemReq)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get system info", err)
	}

	return &dto.SystemInfoOutput{Body: *response}, nil
}

// SDE endpoints

func (hr *Routes) getSDEStatus(ctx context.Context, input *dto.SDEStatusInput) (*dto.SDEStatusOutput, error) {
	response, err := hr.service.GetSDEStatus(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get SDE status", err)
	}

	return &dto.SDEStatusOutput{Body: *response}, nil
}

func (hr *Routes) getSDETypes(ctx context.Context, input *dto.SDETypesInput) (*dto.SDETypesOutput, error) {
	var publishedPtr *bool
	if input.Published {
		publishedPtr = &input.Published
	}

	typeReq := &dto.SDETypeRequest{
		TypeID:    0, // Get all types
		Published: publishedPtr,
	}

	response, err := hr.service.GetSDETypes(ctx, typeReq)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get SDE types", err)
	}

	return &dto.SDETypesOutput{Body: *response}, nil
}

func (hr *Routes) getSDEEntity(ctx context.Context, input *dto.SDEEntityInput) (*dto.SDEEntityOutput, error) {
	entityReq := &dto.SDEEntityRequest{Type: input.Type, ID: input.ID}
	response, err := hr.service.GetSDEEntity(ctx, entityReq)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get SDE entity", err)
	}

	return &dto.SDEEntityOutput{Body: *response}, nil
}

func (hr *Routes) getUniverseSystems(ctx context.Context, input *dto.UniverseSystemsInput) (*dto.UniverseSystemsOutput, error) {
	universeReq := &dto.UniverseRequest{
		Type:          input.Type,
		Region:        input.Region,
		Constellation: input.Constellation,
	}

	response, err := hr.service.GetUniverseSystems(ctx, universeReq)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get universe systems", err)
	}

	return &dto.UniverseSystemsOutput{Body: *response}, nil
}

func (hr *Routes) getRedisSDEEntity(ctx context.Context, input *dto.RedisSDEEntityInput) (*dto.RedisSDEEntityOutput, error) {
	redisReq := &dto.RedisSDERequest{Type: input.Type, ID: input.ID}
	response, err := hr.service.GetRedisSDEEntity(ctx, redisReq)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to get Redis SDE entity", err)
	}

	return &dto.RedisSDEEntityOutput{Body: *response}, nil
}