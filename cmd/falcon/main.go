package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go-falcon/internal/alliance"
	"go-falcon/internal/auth"
	"go-falcon/internal/character"
	"go-falcon/internal/corporation"
	"go-falcon/internal/groups"
	groupsDto "go-falcon/internal/groups/dto"
	"go-falcon/internal/killmails"
	"go-falcon/internal/scheduler"
	"go-falcon/internal/sde_admin"
	"go-falcon/internal/site_settings"
	"go-falcon/internal/sitemap"
	sitemapServices "go-falcon/internal/sitemap/services"
	"go-falcon/internal/users"
	"go-falcon/internal/websocket"
	"go-falcon/internal/zkillboard"
	"go-falcon/pkg/app"
	"go-falcon/pkg/config"
	evegateway "go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"
	"go-falcon/pkg/permissions"
	"go-falcon/pkg/version"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "go.uber.org/automaxprocs"
)

// customLoggerMiddleware logs requests but excludes health check endpoints
func customLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for health check endpoints
		if strings.HasSuffix(r.URL.Path, "/health") {
			next.ServeHTTP(w, r)
			return
		}

		// Use the default chi logger for all other requests
		middleware.Logger(next).ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers for cross-subdomain requests
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow requests from any subdomain of eveonline.it or localhost for development
		if strings.HasSuffix(origin, ".eveonline.it") || origin == "https://eveonline.it" ||
			strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "https://localhost") {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// rawRequestDebugMiddleware captures raw HTTP requests before Huma processing
func rawRequestDebugMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only debug sitemap PUT requests to reduce noise
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/admin/sitemap/") {
			log.Printf("🚀 RAW HTTP REQUEST (BEFORE HUMA) 🚀")
			log.Printf("Method: %s", r.Method)
			log.Printf("URL: %s", r.URL.String())
			log.Printf("Content-Type: %s", r.Header.Get("Content-Type"))
			log.Printf("Content-Length: %s", r.Header.Get("Content-Length"))
			log.Printf("User-Agent: %s", r.Header.Get("User-Agent"))

			// Read and log the raw body
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil {
					log.Printf("Raw Body Length: %d bytes", len(bodyBytes))
					log.Printf("Raw Body Content: %s", string(bodyBytes))
					log.Printf("Raw Body Hex: %x", bodyBytes)

					// Restore the body for further processing
					r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
				} else {
					log.Printf("❌ Failed to read raw body: %v", err)
				}
			} else {
				log.Printf("❌ No request body found")
			}
			log.Printf("🚀 END RAW HTTP DEBUG 🚀")
		}

		// Continue with normal processing
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Display startup banner
	displayBanner()

	// Display version information
	versionInfo := version.Get()
	log.Printf("🏷️  Version: %s", version.GetVersionString())
	log.Printf("🔧 Build: %s (%s)", versionInfo.BuildDate, versionInfo.Platform)

	// Print CPU information (automaxprocs automatically adjusts GOMAXPROCS)
	numCPU := runtime.NumCPU()
	maxProcs := runtime.GOMAXPROCS(0)
	log.Printf("🖥️  CPU Configuration:")
	log.Printf(" - System CPUs: %d", numCPU)
	log.Printf(" - GOMAXPROCS: %d", maxProcs)
	log.Printf(" - automaxprocs: Automatically adjusting based on container limits")

	ctx := context.Background()

	// Initialize application with shared components
	appCtx, err := app.InitializeApp("falcon")
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer appCtx.Shutdown(ctx)

	// Print memory information after app initialization (more accurate)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	runtime.GC() // Force GC to get more accurate stats
	runtime.ReadMemStats(&m)

	log.Printf("💾 Memory Configuration:")
	log.Printf(" - Heap allocated: %s", formatBytes(m.HeapAlloc))
	log.Printf(" - Heap system: %s", formatBytes(m.HeapSys))
	log.Printf(" - Total system: %s", formatBytes(m.Sys))
	log.Printf(" - Stack: %s", formatBytes(m.StackSys))
	log.Printf(" - GC cycles: %d", m.NumGC)
	log.Printf(" - Next GC target: %s", formatBytes(m.NextGC))

	// Print memory limits if available (cgroups v1/v2)
	printMemoryLimits()

	// Initialize Chi router
	r := chi.NewRouter()

	// Global middleware
	r.Use(customLoggerMiddleware) // Custom logger that excludes health checks
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	// Apply timeout middleware but exclude WebSocket endpoints
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip timeout for WebSocket endpoints
			if strings.HasPrefix(r.URL.Path, "/websocket/") {
				next.ServeHTTP(w, r)
				return
			}
			// Apply timeout for all other endpoints
			middleware.Timeout(60*time.Second)(next).ServeHTTP(w, r)
		})
	})
	r.Use(corsMiddleware) // Add CORS support for cross-subdomain requests

	// Health check endpoint with version info
	r.Get("/health", enhancedHealthHandler)

	// Note: WebSocket handler registration will be done after WebSocket module initialization

	// Initialize EVE Online ESI client with Redis caching
	evegateClient := evegateway.NewClientWithRedis(appCtx.Redis)

	// Initialize modules in dependency order
	var modules []module.Module

	// 1. Initialize base modules without dependencies (corporation needs auth, will be moved later)

	// 2. Initialize site settings module (no dependencies)
	siteSettingsModule, err := site_settings.NewModule(appCtx.MongoDB, nil, nil)
	if err != nil {
		log.Fatalf("Failed to initialize site settings module: %v", err)
	}

	// Initialize site settings first
	if err := siteSettingsModule.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize site settings: %v", err)
	}

	// 3. Initialize groups module with site settings dependency
	groupsModule, err := groups.NewModule(appCtx.MongoDB, nil, siteSettingsModule.GetService())
	if err != nil {
		log.Fatalf("Failed to initialize groups module: %v", err)
	}

	// Initialize groups module
	if err := groupsModule.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize groups module: %v", err)
	}

	// 4. Initialize auth module and set groups service dependency
	authModule := auth.New(appCtx.MongoDB, appCtx.Redis, evegateClient)
	authModule.GetAuthService().SetGroupsService(groupsModule.GetService())

	// 5. Update groups module with auth dependencies
	if err := groupsModule.SetAuthModule(authModule); err != nil {
		log.Fatalf("Failed to set auth dependencies on groups module: %v", err)
	}

	// 6. Initialize permission manager
	log.Printf("🔐 Initializing permission management system")
	permissionManager := permissions.NewPermissionManager(appCtx.MongoDB.Database)

	// Set permission manager in groups module
	if err := groupsModule.SetPermissionManager(permissionManager); err != nil {
		log.Fatalf("Failed to set permission manager on groups module: %v", err)
	}

	// 7. Initialize character module with auth dependency
	characterModule := character.New(appCtx.MongoDB, appCtx.Redis, evegateClient, authModule)
	characterModule.SetGroupService(groupsModule.GetService())

	// 8. Initialize corporation module with auth, character and SDE dependencies
	corporationModule := corporation.NewModule(appCtx.MongoDB, appCtx.Redis, evegateClient, authModule, characterModule.GetService(), appCtx.SDEService)
	corporationModule.SetGroupService(groupsModule.GetService())

	// Update groups service with permission manager
	groupsModule.GetService().SetPermissionManager(permissionManager)

	// 6.5. Initialize sitemap module with auth service, permission manager and groups service
	// Create adapter functions to bridge groups service to sitemap interface
	groupsService := groupsModule.GetService()
	groupsAdapter := sitemapServices.NewGroupsServiceAdapter(
		// getUserGroupsFunc
		func(ctx context.Context, userID string) ([]sitemapServices.GroupInfo, error) {
			output, err := groupsService.GetUserGroups(ctx, &groupsDto.GetUserGroupsInput{
				UserID: userID,
			})
			if err != nil {
				return nil, err
			}

			// Convert groups service output to our interface format
			groups := make([]sitemapServices.GroupInfo, len(output.Body.Groups))
			for i, group := range output.Body.Groups {
				groups[i] = sitemapServices.GroupInfo{
					ID:          group.ID,
					Name:        group.Name,
					Type:        group.Type,
					SystemName:  group.SystemName,
					EVEEntityID: group.EVEEntityID,
					IsActive:    group.IsActive,
				}
			}
			return groups, nil
		},
		// getCharacterGroupsFunc
		func(ctx context.Context, characterID int64) ([]sitemapServices.GroupInfo, error) {
			output, err := groupsService.GetCharacterGroups(ctx, &groupsDto.GetCharacterGroupsInput{
				CharacterID: fmt.Sprintf("%d", characterID),
			})
			if err != nil {
				return nil, err
			}

			// Convert groups service output to our interface format
			groups := make([]sitemapServices.GroupInfo, len(output.Body.Groups))
			for i, group := range output.Body.Groups {
				groups[i] = sitemapServices.GroupInfo{
					ID:          group.ID,
					Name:        group.Name,
					Type:        group.Type,
					SystemName:  group.SystemName,
					EVEEntityID: group.EVEEntityID,
					IsActive:    group.IsActive,
				}
			}
			return groups, nil
		},
	)

	// Create corporation service adapter for sitemap
	corporationAdapter := sitemapServices.NewCorporationServiceAdapter(
		func(ctx context.Context, corporationID int) (*sitemapServices.CorporationInfo, error) {
			// Bridge to corporation module service
			corpService := corporationModule.GetService()
			corpInfo, err := corpService.GetCorporationInfo(ctx, corporationID)
			if err != nil {
				return nil, err
			}
			return &sitemapServices.CorporationInfo{
				CorporationID: corporationID, // Use the parameter since the DTO doesn't include ID
				Name:          corpInfo.Body.Name,
				Ticker:        corpInfo.Body.Ticker,
			}, nil
		},
	)

	// Create site settings service adapter for sitemap
	siteSettingsAdapter := sitemapServices.NewSiteSettingsServiceAdapter(
		func(ctx context.Context) ([]sitemapServices.ManagedCorporation, error) {
			// Bridge to site settings module service
			settingsService := siteSettingsModule.GetService()

			// Get all managed corporations (no filter, high limit)
			managedCorps, _, err := settingsService.GetManagedCorporations(ctx, "", 1, 1000)
			if err != nil {
				return nil, err
			}

			// Convert to sitemap interface format
			result := make([]sitemapServices.ManagedCorporation, len(managedCorps))
			for i, corp := range managedCorps {
				result[i] = sitemapServices.ManagedCorporation{
					CorporationID: corp.CorporationID,
					Name:          corp.Name,
					Ticker:        corp.Ticker,
					Enabled:       corp.Enabled,
					Position:      corp.Position,
				}
			}
			return result, nil
		},
		func(ctx context.Context) ([]sitemapServices.ManagedAlliance, error) {
			// Bridge to site settings module service
			settingsService := siteSettingsModule.GetService()

			// Get all managed alliances (no filter, high limit)
			managedAlliances, _, err := settingsService.GetManagedAlliances(ctx, "", 1, 1000)
			if err != nil {
				return nil, err
			}

			// Convert to sitemap interface format
			result := make([]sitemapServices.ManagedAlliance, len(managedAlliances))
			for i, alliance := range managedAlliances {
				result[i] = sitemapServices.ManagedAlliance{
					AllianceID: alliance.AllianceID,
					Name:       alliance.Name,
					Ticker:     alliance.Ticker,
					Enabled:    alliance.Enabled,
					Position:   alliance.Position,
				}
			}
			return result, nil
		},
	)

	sitemapModule, err := sitemap.NewModule(appCtx.MongoDB, appCtx.Redis, authModule.GetAuthService(), permissionManager, groupsAdapter, corporationAdapter, siteSettingsAdapter)
	if err != nil {
		log.Fatalf("Failed to initialize sitemap module: %v", err)
	}

	// 7. Initialize remaining modules that depend on auth
	allianceModule := alliance.NewModule(appCtx.MongoDB, appCtx.Redis, evegateClient, authModule.GetAuthService(), permissionManager)
	killmailsModule := killmails.New(appCtx.MongoDB, appCtx.Redis, evegateClient)

	// Initialize killmails module to create database indexes
	if err := killmailsModule.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize killmails module: %v", err)
	}

	usersModule := users.New(appCtx.MongoDB, appCtx.Redis, authModule, evegateClient, appCtx.SDEService)
	usersModule.SetGroupService(groupsModule.GetService())
	schedulerModule := scheduler.New(appCtx.MongoDB, appCtx.Redis, authModule, characterModule, allianceModule.GetService(), corporationModule)
	schedulerModule.SetGroupService(groupsModule.GetService())
	sdeAdminModule := sde_admin.New(appCtx.MongoDB, appCtx.Redis, authModule, permissionManager, appCtx.SDEService)

	// 8. Initialize WebSocket module
	log.Printf("🔌 Initializing WebSocket module")
	websocketModule, err := websocket.NewModule(appCtx.MongoDB.Database, appCtx.Redis.Client, authModule.GetAuthService())
	if err != nil {
		log.Fatalf("Failed to initialize WebSocket module: %v", err)
	}

	// Start WebSocket service
	if err := websocketModule.Initialize(ctx); err != nil {
		log.Printf("❌ Failed to initialize WebSocket service: %v", err)
	} else {
		log.Printf("✅ WebSocket module initialized successfully")
	}

	// 9. Initialize zkillboard module with websocket dependency
	log.Printf("📡 Initializing ZKillboard module")
	zkillboardModule, err := zkillboard.NewModule(
		appCtx.MongoDB,
		appCtx.Redis,
		killmailsModule.GetRepository(),
		evegateClient,
		websocketModule.GetService(),
		appCtx.SDEService,
	)
	if err != nil {
		log.Fatalf("Failed to create zkillboard module: %v", err)
	}

	// Initialize zkillboard module
	if err := zkillboardModule.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize zkillboard module: %v", err)
	}

	// Register WebSocket HTTP handler on main router (must be outside Huma API for WebSocket upgrades)
	log.Printf("🔌 Registering WebSocket HTTP handler")
	websocketModule.RegisterHTTPHandler(r)

	// 10. Register service permissions
	log.Printf("📝 Registering service permissions")

	// Register permissions in background to avoid startup hang
	go func() {
		log.Printf("🔄 Starting permission registration in background...")

		// Register scheduler permissions
		if err := schedulerModule.RegisterPermissions(ctx, permissionManager); err != nil {
			log.Printf("❌ Failed to register scheduler permissions: %v", err)
		} else {
			log.Printf("   ⏰ Scheduler permissions registered successfully")
		}

		// Register character permissions
		if err := characterModule.RegisterPermissions(ctx, permissionManager); err != nil {
			log.Printf("❌ Failed to register character permissions: %v", err)
		} else {
			log.Printf("   🚀 Character permissions registered successfully")
		}

		// Register sitemap permissions
		if err := sitemapModule.RegisterPermissions(ctx, permissionManager); err != nil {
			log.Printf("❌ Failed to register sitemap permissions: %v", err)
		} else {
			log.Printf("   🗺️  Sitemap permissions registered successfully")
		}

		// Seed default routes for sitemap
		if err := sitemapModule.SeedDefaultRoutes(ctx); err != nil {
			log.Printf("❌ Failed to seed default routes: %v", err)
		}

		// Register corporation permissions
		if err := corporationModule.RegisterPermissions(ctx, permissionManager); err != nil {
			log.Printf("❌ Failed to register corporation permissions: %v", err)
		} else {
			log.Printf("   🏢 Corporation permissions registered successfully")
		}

		// Register alliance permissions
		if err := allianceModule.RegisterPermissions(ctx, permissionManager); err != nil {
			log.Printf("❌ Failed to register alliance permissions: %v", err)
		} else {
			log.Printf("   🌟 Alliance permissions registered successfully")
		}

		log.Printf("✅ Background permission registration completed")
	}()

	// 9. Initialize system group permissions (must be after service permissions are registered)
	log.Printf("🔐 Initializing system group permissions")

	// Initialize system group permissions in background after a delay
	go func() {
		time.Sleep(3 * time.Second) // Wait for service to start
		log.Printf("🔄 Starting system group permission initialization in background...")

		if err := permissionManager.InitializeSystemGroupPermissions(ctx); err != nil {
			log.Printf("❌ Failed to initialize system group permissions: %v", err)
		} else {
			log.Printf("✅ System group permissions initialized successfully")
		}
	}()

	log.Printf("✅ Permission system initialized (background registration enabled)")

	// Update site settings with auth, groups services, and permission manager
	siteSettingsModule.SetDependenciesWithPermissions(authModule.GetAuthService(), groupsModule.GetService(), permissionManager)

	modules = append(modules, authModule, usersModule, schedulerModule, characterModule, corporationModule, allianceModule, killmailsModule, zkillboardModule, groupsModule, sitemapModule, siteSettingsModule, sdeAdminModule, websocketModule)

	// Initialize remaining modules
	// Initialize character module in background to avoid index creation hang during startup
	go func() {
		time.Sleep(5 * time.Second) // Wait for main service to be fully operational
		log.Printf("🔄 Starting character module initialization in background...")

		if err := characterModule.Initialize(ctx); err != nil {
			log.Printf("❌ Failed to initialize character module: %v", err)
		} else {
			log.Printf("✅ Character module initialized successfully (indexes created)")
		}
	}()

	log.Printf("🚀 Character module initialization scheduled for background")

	log.Printf("🚀 EVE Online ESI client initialized")

	// Mount module routes with configurable API prefix
	apiPrefix := config.GetAPIPrefix()
	log.Printf("🔗 Using API prefix: '%s'", apiPrefix)

	// Scalar API Documentation
	r.Get("/docs", scalarDocsHandler(apiPrefix))

	// Create unified Huma v2 API for integrated mode
	log.Printf("🚀 Creating unified Huma v2 API (type-safe APIs with single OpenAPI specification)")

	// Create unified Huma API configuration
	humaConfig := huma.DefaultConfig("Go Falcon API", "1.0.0")
	humaConfig.Info.Description = "Unified EVE Online API with modular architecture"
	humaConfig.Info.Contact = &huma.Contact{
		Name: "Go Falcon",
		URL:  "https://github.com/your-org/go-falcon",
	}

	// Disable default docs path since we're using Scalar
	humaConfig.DocsPath = ""

	// Add security schemes for authentication
	humaConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearerAuth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "JWT Bearer token authentication",
		},
		"cookieAuth": {
			Type:        "apiKey",
			In:          "cookie",
			Name:        "falcon_auth_token",
			Description: "JWT authentication cookie",
		},
	}

	// Add tags for better organization in Scalar docs
	humaConfig.Tags = []*huma.Tag{
		{Name: "Health", Description: "Module health checks and system status"},
		{Name: "Auth", Description: "EVE Online SSO authentication and JWT management"},
		{Name: "Auth / EVE", Description: "EVE Online SSO integration endpoints"},
		{Name: "Auth / Profile", Description: "User profile management and character information"},
		{Name: "Users", Description: "User management and character administration"},
		{Name: "Users / Management", Description: "Administrative user management operations"},
		{Name: "Users / Characters", Description: "Character listing and management"},
		{Name: "Character", Description: "EVE Online character profiles and information"},
		{Name: "Corporations", Description: "EVE Online corporation information and management"},
		{Name: "Alliances", Description: "EVE Online alliance information and management"},
		{Name: "Groups", Description: "Group and role-based access control management"},
		{Name: "Groups / Management", Description: "Group creation, modification, and deletion"},
		{Name: "Groups / Memberships", Description: "Character group membership operations"},
		{Name: "Groups / Characters", Description: "Character-centric group operations"},
		{Name: "Permissions", Description: "Permission management and checking"},
		{Name: "Group Permissions", Description: "Group permission assignment and management"},
		{Name: "Scheduler", Description: "Task scheduling, execution, and monitoring"},
		{Name: "Scheduler / Status", Description: "Task scheduler status and statistics"},
		{Name: "Scheduler / Tasks", Description: "Scheduled task management and configuration"},
		{Name: "Scheduler / Executions", Description: "Task execution history and monitoring"},
		{Name: "Site Settings", Description: "Application configuration and site settings management"},
		{Name: "Site Settings / Public", Description: "Public site settings accessible without authentication"},
		{Name: "Site Settings / Management", Description: "Administrative site settings management operations"},
		{Name: "SDE Admin", Description: "EVE Online Static Data Export administration and Redis import management"},
		{Name: "WebSocket", Description: "Real-time WebSocket communication and connection management"},
		{Name: "WebSocket Admin", Description: "Administrative WebSocket connection and room management"},
		{Name: "ZKillboard", Description: "ZKillboard RedisQ consumer service and killmail statistics"},
		{Name: "Module Status", Description: "Module health status and statistics endpoints"},
	}

	// Add servers based on environment configuration or defaults
	customServers := config.GetOpenAPIServers()
	if customServers != nil {
		// Use custom servers from environment variable
		humaConfig.Servers = make([]*huma.Server, len(customServers))
		for i, server := range customServers {
			serverURL := server.URL
			if apiPrefix != "" && !strings.HasSuffix(serverURL, apiPrefix) {
				serverURL = serverURL + apiPrefix
			}
			humaConfig.Servers[i] = &huma.Server{
				URL:         serverURL,
				Description: server.Description,
			}
		}
	} else {
		// Use default server configuration
		frontendURL := config.GetFrontendURL()
		if apiPrefix == "" {
			humaConfig.Servers = []*huma.Server{
				{URL: frontendURL, Description: "Production server"},
				{URL: "http://localhost:3000", Description: "Local development"},
			}
		} else {
			humaConfig.Servers = []*huma.Server{
				{URL: frontendURL + apiPrefix, Description: "Production server"},
				{URL: "http://localhost:3000" + apiPrefix, Description: "Local development"},
			}
		}
	}

	// Create the unified API on main router
	var unifiedAPI huma.API
	if apiPrefix == "" {
		unifiedAPI = humachi.New(r, humaConfig)
	} else {
		// Mount the API under the prefix
		r.Route(apiPrefix, func(prefixRouter chi.Router) {
			unifiedAPI = humachi.New(prefixRouter, humaConfig)
		})
	}

	log.Printf("✅ Unified Huma v2 API created")
	log.Printf("🔧 Single OpenAPI 3.1.1 specification will be available at %s/openapi.json", apiPrefix)
	log.Printf("📚 Scalar API Documentation available at /docs")

	// Register all module routes on the unified API
	log.Printf("📝 Registering module routes on unified API:")

	// Register auth module routes
	log.Printf("   🔐 Auth module: /auth/*")
	authModule.RegisterUnifiedRoutes(unifiedAPI, "/auth")

	// Register users module routes
	log.Printf("   👥 Users module: /users/*")
	usersModule.RegisterUnifiedRoutes(unifiedAPI, "/users")

	// Register scheduler module routes
	log.Printf("   ⏰ Scheduler module: /scheduler/*")
	schedulerModule.RegisterUnifiedRoutes(unifiedAPI, "/scheduler")

	// Register character module routes
	log.Printf("   🚀 Character module: /character/*")
	characterModule.RegisterUnifiedRoutes(unifiedAPI, "/character")

	// Register corporation module routes
	log.Printf("   🏢 Corporation module: /corporations/*")
	corporationModule.RegisterUnifiedRoutes(unifiedAPI, "/corporations")

	// Register alliance module routes
	log.Printf("   🤝 Alliance module: /alliances/*")
	allianceModule.RegisterUnifiedRoutes(unifiedAPI, "/alliances")

	// Register killmails module routes
	log.Printf("   ⚔️  Killmails module: /killmails/*")
	killmailsModule.RegisterUnifiedRoutes(unifiedAPI, "/killmails")

	// Register zkillboard module routes
	log.Printf("   📡 ZKillboard module: /zkillboard/*")
	if err := zkillboardModule.RegisterRoutes(unifiedAPI); err != nil {
		log.Fatalf("Failed to register zkillboard routes: %v", err)
	}

	// Register groups module routes
	log.Printf("   👥 Groups module: /groups/*")
	groupsModule.RegisterUnifiedRoutes(unifiedAPI)

	// Register sitemap module routes
	log.Printf("   🗺️  Sitemap module: /sitemap/*")
	sitemapModule.RegisterUnifiedRoutes(unifiedAPI)

	// Register site settings module routes
	log.Printf("   ⚙️  Site Settings module: /site-settings/*")
	siteSettingsModule.RegisterUnifiedRoutes(unifiedAPI)

	// Register SDE admin module routes
	log.Printf("   📊 SDE Admin module: /sde/*")
	sdeAdminModule.RegisterUnifiedRoutes(unifiedAPI, "/sde")

	// Register WebSocket module routes
	log.Printf("   🔌 WebSocket module: /websocket/*")
	websocketModule.RegisterUnifiedRoutes(unifiedAPI)

	log.Printf("✅ All modules registered on unified API")

	// Note: evegateway is now a shared package for EVE Online ESI integration
	// Other services can import and use: evegateway.NewClient().GetServerStatus(ctx)
	_ = evegateClient // Available for modules to use

	// Start background services for all modules
	for _, mod := range modules {
		go mod.StartBackgroundTasks(ctx)
	}

	// HTTP server setup
	port := app.GetPort("8080")
	host := config.GetHost()
	humaPort := config.GetHumaPort()
	humaHost := config.GetHumaHost()
	separateHumaServer := config.GetHumaSeparateServer()

	// Display HUMA configuration
	log.Printf("🔧 HUMA Configuration:")
	log.Printf(" - Separate Server: %v", separateHumaServer)
	log.Printf(" - Main Server: %s:%s", host, port)
	if humaPort != "" {
		log.Printf(" - HUMA Server: %s:%s", humaHost, humaPort)
	} else {
		log.Printf(" - HUMA Port: Not specified (would use main server)")
	}
	if separateHumaServer && humaPort != "" {
		log.Printf(" - Mode: Separate server - HUMA APIs on %s:%s", humaHost, humaPort)
	} else if separateHumaServer && humaPort == "" {
		log.Printf(" - Mode: Separate server DISABLED - HUMA_PORT not set")
	} else {
		log.Printf(" - Mode: Integrated - HUMA APIs on main server %s:%s", host, port)
	}

	// Main server
	srv := &http.Server{
		Addr:         host + ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Optional separate HUMA server
	var humaSrv *http.Server
	if separateHumaServer && humaPort != "" {
		log.Printf("🚀 Separate HUMA server mode is currently disabled in unified architecture")
		log.Printf("⚠️  This feature will be reimplemented with unified OpenAPI support")
		log.Printf("⚠️  For now, all routes are served from the main server")
	}

	// Display final configuration
	if separateHumaServer && humaPort != "" {
		log.Printf("⚠️  HUMA_SEPARATE_SERVER=true but feature disabled - using integrated mode")
	}

	if humaPort != "" && !separateHumaServer {
		log.Printf("⚠️  HUMA_PORT=%s set but HUMA_SEPARATE_SERVER=false - using integrated mode", humaPort)
	}

	log.Printf("✅ Unified HUMA API available on main server: %s:%s", host, port)
	if host == "0.0.0.0" {
		log.Printf("📋 Single OpenAPI specification: http://localhost:%s%s/openapi.json", port, apiPrefix)
		log.Printf("📚 Scalar API Documentation: http://localhost:%s/docs", port)
		log.Printf("🌐 Access all modules via unified API")
	} else {
		log.Printf("📋 Single OpenAPI specification: http://%s:%s%s/openapi.json", host, port, apiPrefix)
		log.Printf("📚 Scalar API Documentation: http://%s:%s/docs", host, port)
		log.Printf("🌐 Access all modules via unified API")
	}

	// Start main server
	go func() {
		slog.Info("Starting main Falcon API server", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Main server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Received shutdown signal, initiating graceful shutdown...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP servers
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Main server forced to shutdown", "error", err)
	}

	// Shutdown separate HUMA server if running
	if humaSrv != nil {
		if err := humaSrv.Shutdown(shutdownCtx); err != nil {
			slog.Error("HUMA server forced to shutdown", "error", err)
		}
	}

	// Stop background services for all modules
	for _, mod := range modules {
		mod.Stop()
	}

	// Application context will handle database and telemetry shutdown
	appCtx.Shutdown(shutdownCtx)

	slog.Info("Falcon shutdown completed successfully")
}

func enhancedHealthHandler(w http.ResponseWriter, r *http.Request) {
	// Health checks are excluded from logging to reduce noise
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	versionInfo := version.Get()
	response := fmt.Sprintf(`{
		"status": "healthy",
		"architecture": "falcon",
		"version": "%s",
		"git_commit": "%s",
		"build_date": "%s",
		"go_version": "%s",
		"platform": "%s"
	}`, versionInfo.Version, versionInfo.GitCommit, versionInfo.BuildDate, versionInfo.GoVersion, versionInfo.Platform)

	w.Write([]byte(response))
}

// scalarDocsHandler returns a handler that serves the Scalar API documentation interface
func scalarDocsHandler(apiPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Build the full OpenAPI URL based on the current request
		scheme := "http"
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}

		host := r.Host
		if host == "" {
			host = r.Header.Get("Host")
		}

		openAPIPath := "/openapi.json"
		if apiPrefix != "" {
			openAPIPath = apiPrefix + "/openapi.json"
		}

		// Build absolute URL for OpenAPI spec
		openAPIURL := fmt.Sprintf("%s://%s%s", scheme, host, openAPIPath)

		// Serve the Scalar documentation HTML
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		// Scalar HTML with simpler configuration
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Go Falcon API Documentation</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
    <script id="api-reference" data-url="%s"></script>
    <script>
        var configuration = {
            theme: 'kepler',
						layout: 'classic',
            darkMode: true,
						hideModels: false
        }
    </script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`, openAPIURL)

		w.Write([]byte(html))
	}
}

func displayBanner() {
	file, err := os.Open("banner.txt")
	if err != nil {
		// Fallback to inline banner if file not found
		fmt.Print("\033[38;5;33m")
		fmt.Print("GO-FALCON API\n")
		fmt.Print("\033[0m")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Print("\033[38;5;33m")
		fmt.Print("GO-FALCON API\n")
		fmt.Print("\033[0m")
		return
	}

	lines := strings.Split(string(content), "\n")
	colors := []string{
		"\033[38;5;33m", // Bright blue
		"\033[38;5;39m", // Cyan
		"\033[38;5;75m", // Light blue
		"\033[38;5;51m", // Bright cyan
		"\033[38;5;33m", // Bright blue
		"\033[38;5;39m", // Cyan
	}

	fmt.Print("\n")
	for i, line := range lines {
		if line != "" && i < len(colors) {
			fmt.Print(colors[i])
			fmt.Println(line)
		}
	}
	fmt.Print("\033[0m") // Reset colors
	fmt.Print("\n")
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// printMemoryLimits reads and displays container memory limits
func printMemoryLimits() {
	// Try cgroups v2 first (newer systems)
	if limit := readCgroupV2MemoryLimit(); limit > 0 {
		log.Printf(" - Container limit: %s (cgroups v2)", formatBytes(uint64(limit)))
		return
	}

	// Try cgroups v1 (older systems)
	if limit := readCgroupV1MemoryLimit(); limit > 0 {
		log.Printf(" - Container limit: %s (cgroups v1)", formatBytes(uint64(limit)))
		return
	}

	log.Printf(" - Container limit: Not detected (running outside container or unsupported)")
}

// readCgroupV2MemoryLimit reads memory limit from cgroups v2
func readCgroupV2MemoryLimit() int64 {
	data, err := os.ReadFile("/sys/fs/cgroup/memory.max")
	if err != nil {
		return 0
	}

	limitStr := strings.TrimSpace(string(data))
	if limitStr == "max" {
		return 0 // No limit set
	}

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		return 0
	}

	return limit
}

// readCgroupV1MemoryLimit reads memory limit from cgroups v1
func readCgroupV1MemoryLimit() int64 {
	data, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	if err != nil {
		return 0
	}

	limitStr := strings.TrimSpace(string(data))
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		return 0
	}

	// cgroups v1 sometimes returns very large values when no limit is set
	// Anything larger than 1TB is probably "unlimited"
	if limit > 1024*1024*1024*1024 {
		return 0
	}

	return limit
}
