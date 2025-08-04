package dev

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"go-falcon/pkg/database"
	"go-falcon/pkg/evegate"
	"go-falcon/pkg/evegate/alliance"
	"go-falcon/pkg/evegate/character"
	"go-falcon/pkg/evegate/status"
	"go-falcon/pkg/evegate/universe"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	*module.BaseModule
	evegateClient    *evegate.Client
	statusClient     status.Client
	characterClient  character.Client
	universeClient   universe.Client
	allianceClient   alliance.Client
	cacheManager     evegate.CacheManager
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *Module {
	evegateClient := evegate.NewClient()
	
	// Create shared cache manager for consistency
	cacheManager := evegate.NewDefaultCacheManager()
	httpClient := &http.Client{Timeout: 30 * time.Second}
	baseURL := "https://esi.evetech.net"
	userAgent := "go-falcon/1.0.0 contact@example.com"
	
	errorLimits := &evegate.ESIErrorLimits{}
	limitsMutex := &sync.RWMutex{}
	retryClient := evegate.NewDefaultRetryClient(httpClient, errorLimits, limitsMutex)
	
	statusClient := status.NewStatusClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	characterClient := character.NewCharacterClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	universeClient := universe.NewUniverseClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	allianceClient := alliance.NewAllianceClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	
	return &Module{
		BaseModule:       module.NewBaseModule("dev", mongodb, redis, sdeService),
		evegateClient:    evegateClient,
		statusClient:     statusClient,
		characterClient:  characterClient,
		universeClient:   universeClient,
		allianceClient:   allianceClient,
		cacheManager:     cacheManager,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	r.Get("/esi-status", m.esiStatusHandler)
	r.Get("/character/{characterID}", m.characterInfoHandler)
	r.Get("/character/{characterID}/portrait", m.characterPortraitHandler)
	r.Get("/universe/system/{systemID}", m.systemInfoHandler)
	r.Get("/universe/station/{stationID}", m.stationInfoHandler)
	r.Get("/alliances", m.alliancesHandler)
	r.Get("/alliance/{allianceID}", m.allianceInfoHandler)
	r.Get("/alliance/{allianceID}/corporations", m.allianceCorporationsHandler)
	r.Get("/alliance/{allianceID}/icons", m.allianceIconsHandler)
	r.Get("/sde/status", m.sdeStatusHandler)
	r.Get("/sde/agent/{agentID}", m.sdeAgentHandler)
	r.Get("/sde/category/{categoryID}", m.sdeCategoryHandler)
	r.Get("/sde/blueprint/{blueprintID}", m.sdeBlueprintHandler)
	r.Get("/sde/agents/location/{locationID}", m.sdeAgentsByLocationHandler)
	r.Get("/sde/blueprints", m.sdeBlueprintIdsHandler)
	r.Get("/services", m.servicesHandler)
	r.Get("/status", m.statusHandler)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	slog.Info("Starting dev module background tasks")
	
	// Call base implementation for common functionality
	go m.BaseModule.StartBackgroundTasks(ctx)
	
	// Add dev-specific background processing here
	for {
		select {
		case <-ctx.Done():
			slog.Info("Dev background tasks stopped due to context cancellation")
			return
		case <-m.StopChannel():
			slog.Info("Dev background tasks stopped")
			return
		default:
			// Dev-specific background work would go here
			select {
			case <-ctx.Done():
				return
			case <-m.StopChannel():
				return
			}
		}
	}
}