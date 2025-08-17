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

// GroupsModule interface defines the methods needed from the groups module
type GroupsModule interface {
	RequireGranularPermission(service, resource, action string) func(http.Handler) http.Handler
}

// AuthModule interface defines the methods needed from the auth module
type AuthModule interface {
	JWTMiddleware(next http.Handler) http.Handler
	OptionalJWTMiddleware(next http.Handler) http.Handler
}

type Module struct {
	*module.BaseModule
	evegateClient     *evegateway.Client
	statusClient      status.Client
	characterClient   character.Client
	universeClient    universe.Client
	allianceClient    alliance.Client
	corporationClient corporation.Client
	cacheManager      evegateway.CacheManager
	groupsModule      GroupsModule
	authModule        AuthModule
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService, groupsModule GroupsModule, authModule AuthModule) *Module {
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
		groupsModule:      groupsModule,
		authModule:        authModule,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.RegisterHealthRoute(r) // Use the base module health handler
	
	// Public status endpoints
	r.Get("/status", m.statusHandler)
	r.Get("/services", m.servicesHandler)
	
	// Protected ESI testing endpoints (require authentication and dev tools access)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/esi-status", m.esiStatusHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/character/{characterID}", m.characterInfoHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/character/{characterID}/portrait", m.characterPortraitHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/universe/system/{systemID}", m.systemInfoHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/universe/station/{stationID}", m.stationInfoHandler)
	
	// Protected alliance endpoints
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/alliances", m.alliancesHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/alliance/{allianceID}", m.allianceInfoHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/alliance/{allianceID}/contacts", m.allianceContactsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/alliance/{allianceID}/contacts/labels", m.allianceContactLabelsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/alliance/{allianceID}/corporations", m.allianceCorporationsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/alliance/{allianceID}/icons", m.allianceIconsHandler)
	
	// Protected corporation endpoints
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/corporation/{corporationID}", m.corporationInfoHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/corporation/{corporationID}/icons", m.corporationIconsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/corporation/{corporationID}/alliancehistory", m.corporationAllianceHistoryHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/corporation/{corporationID}/members", m.corporationMembersHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/corporation/{corporationID}/membertracking", m.corporationMemberTrackingHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/corporation/{corporationID}/roles", m.corporationMemberRolesHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/corporation/{corporationID}/structures", m.corporationStructuresHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/corporation/{corporationID}/standings", m.corporationStandingsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/corporation/{corporationID}/wallets", m.corporationWalletsHandler)
	
	// Protected SDE data access endpoints
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/status", m.sdeStatusHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/agent/{agentID}", m.sdeAgentHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/category/{categoryID}", m.sdeCategoryHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/blueprint/{blueprintID}", m.sdeBlueprintHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/agents/location/{locationID}", m.sdeAgentsByLocationHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/blueprints", m.sdeBlueprintIdsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/marketgroup/{marketGroupID}", m.sdeMarketGroupHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/marketgroups", m.sdeMarketGroupsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/metagroup/{metaGroupID}", m.sdeMetaGroupHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/metagroups", m.sdeMetaGroupsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/npccorp/{corpID}", m.sdeNPCCorpHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/npccorps", m.sdeNPCCorpsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/npccorps/faction/{factionID}", m.sdeNPCCorpsByFactionHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/typeid/{typeID}", m.sdeTypeIDHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/type/{typeID}", m.sdeTypeHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/types", m.sdeTypesHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/types/published", m.sdePublishedTypesHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/types/group/{groupID}", m.sdeTypesByGroupHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/typematerials/{typeID}", m.sdeTypeMaterialsHandler)
	
	// Protected Redis SDE endpoints
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/redis/{type}/{id}", m.sdeRedisEntityHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/redis/{type}", m.sdeRedisEntitiesByTypeHandler)
	
	// Protected universe endpoints
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/universe/{universeType}/{regionName}/systems", m.sdeUniverseRegionSystemsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/universe/{universeType}/{regionName}/{constellationName}/systems", m.sdeUniverseConstellationSystemsHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/universe/{universeType}/{regionName}", m.sdeUniverseDataHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/universe/{universeType}/{regionName}/{constellationName}", m.sdeUniverseDataHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/universe/{universeType}/{regionName}/{constellationName}/{systemName}", m.sdeUniverseDataHandler)
	
	// Protected SDE management endpoints
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/management/status", m.sdeManagementStatusHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/comprehensive/status", m.sdeComprehensiveStatusHandler)
	r.With(m.authModule.JWTMiddleware, m.groupsModule.RequireGranularPermission("dev", "tools", "read")).Get("/sde/entity-types", m.sdeAllEntityTypesHandler)
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