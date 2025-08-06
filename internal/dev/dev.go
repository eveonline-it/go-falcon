package dev

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"go-falcon/pkg/database"
	evegateway "go-falcon/pkg/evegateway"
	"go-falcon/pkg/evegateway/alliance"
	"go-falcon/pkg/evegateway/character"
	"go-falcon/pkg/evegateway/corporation"
	"go-falcon/pkg/evegateway/status"
	"go-falcon/pkg/evegateway/universe"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	*module.BaseModule
	evegateClient     *evegateway.Client
	statusClient      status.Client
	characterClient   character.Client
	universeClient    universe.Client
	allianceClient    alliance.Client
	corporationClient corporation.Client
	cacheManager      evegateway.CacheManager
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *Module {
	evegateClient := evegateway.NewClient()
	
	// Create shared cache manager for consistency
	cacheManager := evegateway.NewDefaultCacheManager()
	httpClient := &http.Client{Timeout: 30 * time.Second}
	baseURL := "https://esi.evetech.net"
	userAgent := "go-falcon/1.0.0 contact@example.com"
	
	errorLimits := &evegateway.ESIErrorLimits{}
	limitsMutex := &sync.RWMutex{}
	retryClient := evegateway.NewDefaultRetryClient(httpClient, errorLimits, limitsMutex)
	
	statusClient := status.NewStatusClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	characterClient := character.NewCharacterClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	universeClient := universe.NewUniverseClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	allianceClient := alliance.NewAllianceClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	corporationClient := corporation.NewCorporationClient(httpClient, baseURL, userAgent, cacheManager, retryClient)
	
	return &Module{
		BaseModule:        module.NewBaseModule("dev", mongodb, redis, sdeService),
		evegateClient:     evegateClient,
		statusClient:      statusClient,
		characterClient:   characterClient,
		universeClient:    universeClient,
		allianceClient:    allianceClient,
		corporationClient: corporationClient,
		cacheManager:      cacheManager,
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
	r.Get("/alliance/{allianceID}/contacts", m.allianceContactsHandler)
	r.Get("/alliance/{allianceID}/contacts/labels", m.allianceContactLabelsHandler)
	r.Get("/alliance/{allianceID}/corporations", m.allianceCorporationsHandler)
	r.Get("/alliance/{allianceID}/icons", m.allianceIconsHandler)
	// Corporation endpoints
	r.Get("/corporation/{corporationID}", m.corporationInfoHandler)
	r.Get("/corporation/{corporationID}/icons", m.corporationIconsHandler)
	r.Get("/corporation/{corporationID}/alliancehistory", m.corporationAllianceHistoryHandler)
	r.Get("/corporation/{corporationID}/members", m.corporationMembersHandler)
	r.Get("/corporation/{corporationID}/membertracking", m.corporationMemberTrackingHandler)
	r.Get("/corporation/{corporationID}/roles", m.corporationMemberRolesHandler)
	r.Get("/corporation/{corporationID}/structures", m.corporationStructuresHandler)
	r.Get("/corporation/{corporationID}/standings", m.corporationStandingsHandler)
	r.Get("/corporation/{corporationID}/wallets", m.corporationWalletsHandler)
	r.Get("/sde/status", m.sdeStatusHandler)
	r.Get("/sde/agent/{agentID}", m.sdeAgentHandler)
	r.Get("/sde/category/{categoryID}", m.sdeCategoryHandler)
	r.Get("/sde/blueprint/{blueprintID}", m.sdeBlueprintHandler)
	r.Get("/sde/agents/location/{locationID}", m.sdeAgentsByLocationHandler)
	r.Get("/sde/blueprints", m.sdeBlueprintIdsHandler)
	r.Get("/sde/marketgroup/{marketGroupID}", m.sdeMarketGroupHandler)
	r.Get("/sde/marketgroups", m.sdeMarketGroupsHandler)
	r.Get("/sde/metagroup/{metaGroupID}", m.sdeMetaGroupHandler)
	r.Get("/sde/metagroups", m.sdeMetaGroupsHandler)
	r.Get("/sde/npccorp/{corpID}", m.sdeNPCCorpHandler)
	r.Get("/sde/npccorps", m.sdeNPCCorpsHandler)
	r.Get("/sde/npccorps/faction/{factionID}", m.sdeNPCCorpsByFactionHandler)
	r.Get("/sde/typeid/{typeID}", m.sdeTypeIDHandler)
	r.Get("/sde/type/{typeID}", m.sdeTypeHandler)
	r.Get("/sde/types", m.sdeTypesHandler)
	r.Get("/sde/types/published", m.sdePublishedTypesHandler)
	r.Get("/sde/types/group/{groupID}", m.sdeTypesByGroupHandler)
	r.Get("/sde/typematerials/{typeID}", m.sdeTypeMaterialsHandler)
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