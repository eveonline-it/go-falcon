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

	"go-falcon/internal/auth"
	"go-falcon/internal/dev"
	"go-falcon/internal/groups"
	"go-falcon/internal/notifications"
	"go-falcon/internal/scheduler"
	"go-falcon/internal/sde"
	"go-falcon/internal/users"
	"go-falcon/pkg/app"
	"go-falcon/pkg/config"
	evegateway "go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"
	pkgsde "go-falcon/pkg/sde"
	"go-falcon/pkg/version"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
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
		
		// Allow requests from any subdomain of eveonline.it
		if strings.HasSuffix(origin, ".eveonline.it") || origin == "https://eveonline.it" {
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
	appCtx, err := app.InitializeApp("gateway")
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
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(corsMiddleware) // Add CORS support for cross-subdomain requests

	// Health check endpoint with version info
	r.Get("/health", enhancedHealthHandler)

	// Initialize EVE Online ESI client as shared package
	evegateClient := evegateway.NewClient()
	
	// Initialize modules
	var modules []module.Module
	authModule := auth.New(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService, evegateClient)
	groupsModule := groups.New(appCtx.MongoDB, appCtx.Redis)
	
	// Set auth module dependency for groups module
	groupsModule.SetAuthModule(authModule)
	
	devModule, err := dev.NewModule(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService)
	if err != nil {
		log.Fatalf("Failed to initialize dev module: %v", err)
	}
	usersModule := users.New(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService, authModule, groupsModule)
	notificationsModule := notifications.New(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService, authModule, groupsModule)
	// Initialize SDE module - need to type assert the interface
	sdeService, ok := appCtx.SDEService.(*pkgsde.Service)
	if !ok {
		log.Fatalf("SDE Service is not the expected type")
	}
	sdeModule := sde.NewModule(appCtx.MongoDB, appCtx.Redis, sdeService)
	schedulerModule := scheduler.New(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService, authModule, sdeModule, groupsModule)
	
	modules = append(modules, authModule, groupsModule, devModule, usersModule, notificationsModule, sdeModule, schedulerModule)
	log.Printf("🚀 EVE Online ESI client initialized")

	// Mount module routes with configurable API prefix
	apiPrefix := config.GetAPIPrefix()
	log.Printf("🔗 Using API prefix: '%s'", apiPrefix)
	
	// Create unified Huma v2 API for integrated mode
	log.Printf("🚀 Creating unified Huma v2 API (type-safe APIs with single OpenAPI specification)")
	
	// Create unified Huma API configuration
	humaConfig := huma.DefaultConfig("Go Falcon API Gateway", "1.0.0")
	humaConfig.Info.Description = "Unified EVE Online API Gateway with modular architecture"
	humaConfig.Info.Contact = &huma.Contact{
		Name: "Go Falcon",
		URL:  "https://github.com/your-org/go-falcon",
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
	
	// Register all module routes on the unified API
	log.Printf("📝 Registering module routes on unified API:")
	
	// Register auth module routes
	log.Printf("   🔐 Auth module: /auth/*")
	authModule.RegisterUnifiedRoutes(unifiedAPI, "/auth")
	
	// Register dev module routes  
	log.Printf("   🔧 Dev module: /dev/*")
	devModule.RegisterUnifiedRoutes(unifiedAPI, "/dev")
	
	// Register users module routes
	log.Printf("   👥 Users module: /users/*")
	usersModule.RegisterUnifiedRoutes(unifiedAPI, "/users")
	
	// Register notifications module routes
	log.Printf("   📬 Notifications module: /notifications/*")
	notificationsModule.RegisterUnifiedRoutes(unifiedAPI, "/notifications")
	
	// Register SDE module routes
	log.Printf("   📊 SDE module: /sde/*")
	sdeModule.RegisterUnifiedRoutes(unifiedAPI, "/sde")
	
	// Register scheduler module routes
	log.Printf("   ⏰ Scheduler module: /scheduler/*")
	schedulerModule.RegisterUnifiedRoutes(unifiedAPI, "/scheduler")
	
	// Register groups module routes
	log.Printf("   👤 Groups module: /groups/*")
	groupsModule.RegisterUnifiedRoutes(unifiedAPI, "/groups")
	
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
		log.Printf("🌐 Access all modules via unified API")
	} else {
		log.Printf("📋 Single OpenAPI specification: http://%s:%s%s/openapi.json", host, port, apiPrefix)
		log.Printf("🌐 Access all modules via unified API")
	}

	// Start main server
	go func() {
		slog.Info("Starting main API gateway server", slog.String("addr", srv.Addr))
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

	slog.Info("Gateway shutdown completed successfully")
}

func enhancedHealthHandler(w http.ResponseWriter, r *http.Request) {
	// Health checks are excluded from logging to reduce noise
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	versionInfo := version.Get()
	response := fmt.Sprintf(`{
		"status": "healthy",
		"architecture": "gateway",
		"version": "%s",
		"git_commit": "%s",
		"build_date": "%s",
		"go_version": "%s",
		"platform": "%s"
	}`, versionInfo.Version, versionInfo.GitCommit, versionInfo.BuildDate, versionInfo.GoVersion, versionInfo.Platform)
	
	w.Write([]byte(response))
}

func displayBanner() {
	file, err := os.Open("banner.txt")
	if err != nil {
		// Fallback to inline banner if file not found
		fmt.Print("\033[38;5;33m")
		fmt.Print("GO-FALCON API Gateway\n")
		fmt.Print("\033[0m")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Print("\033[38;5;33m")
		fmt.Print("GO-FALCON API Gateway\n")
		fmt.Print("\033[0m")
		return
	}

	lines := strings.Split(string(content), "\n")
	colors := []string{
		"\033[38;5;33m",  // Bright blue
		"\033[38;5;39m",  // Cyan
		"\033[38;5;75m",  // Light blue
		"\033[38;5;51m",  // Bright cyan
		"\033[38;5;33m",  // Bright blue
		"\033[38;5;39m",  // Cyan
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