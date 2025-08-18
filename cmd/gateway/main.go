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
	
	// Register Huma v2 routes as primary routing system
	log.Printf("🚀 Registering Huma v2 routes (type-safe APIs with automatic OpenAPI)")
	
	// Handle empty prefix case
	if apiPrefix == "" {
		// Huma v2 routes (now primary)
		r.Route("/auth", authModule.RegisterHumaRoutes)
		groupsModule.Routes(r) // Groups routes are mounted at /groups via sub-router
		r.Route("/dev", devModule.RegisterHumaRoutes)
		r.Route("/users", usersModule.RegisterHumaRoutes)
		r.Route("/notifications", notificationsModule.RegisterHumaRoutes)
		r.Route("/sde", sdeModule.RegisterHumaRoutes)
		r.Route("/scheduler", schedulerModule.RegisterHumaRoutes)
	} else {
		// Huma v2 routes (now primary)
		r.Route(apiPrefix+"/auth", authModule.RegisterHumaRoutes)
		r.Route(apiPrefix, groupsModule.Routes) // Groups routes are mounted at /api/groups via sub-router
		r.Route(apiPrefix+"/dev", devModule.RegisterHumaRoutes)
		r.Route(apiPrefix+"/users", usersModule.RegisterHumaRoutes)
		r.Route(apiPrefix+"/notifications", notificationsModule.RegisterHumaRoutes)
		r.Route(apiPrefix+"/sde", sdeModule.RegisterHumaRoutes)
		r.Route(apiPrefix+"/scheduler", schedulerModule.RegisterHumaRoutes)
	}
	
	log.Printf("✅ Huma v2 routes: /auth, /dev, /users, /notifications, /sde, /scheduler")
	log.Printf("🔧 Each module provides automatic OpenAPI 3.1.1 specification at /{module}/openapi.json")
	
	// Note: evegateway is now a shared package for EVE Online ESI integration
	// Other services can import and use: evegateway.NewClient().GetServerStatus(ctx)
	_ = evegateClient // Available for modules to use

	// Start background services for all modules
	for _, mod := range modules {
		go mod.StartBackgroundTasks(ctx)
	}

	// HTTP server setup
	port := app.GetPort("8080")
	humaPort := config.GetHumaPort()
	separateHumaServer := config.GetHumaSeparateServer()
	
	// Display HUMA configuration
	log.Printf("🔧 HUMA Configuration:")
	log.Printf(" - Separate Server: %v", separateHumaServer)
	if humaPort != "" {
		log.Printf(" - HUMA Port: %s", humaPort)
	} else {
		log.Printf(" - HUMA Port: Not specified (would default to main port)")
	}
	if separateHumaServer && humaPort != "" {
		log.Printf(" - Mode: Separate server - HUMA APIs on port %s", humaPort)
	} else if separateHumaServer && humaPort == "" {
		log.Printf(" - Mode: Separate server DISABLED - HUMA_PORT not set")
	} else {
		log.Printf(" - Mode: Integrated - HUMA APIs on main port %s", port)
	}
	
	// Main server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Optional separate HUMA server
	var humaSrv *http.Server
	if separateHumaServer && humaPort != "" {
		// Create separate router for HUMA APIs only
		humaRouter := chi.NewRouter()
		humaRouter.Use(customLoggerMiddleware)
		humaRouter.Use(middleware.Recoverer)
		humaRouter.Use(middleware.RequestID)
		humaRouter.Use(middleware.RealIP)
		humaRouter.Use(middleware.Timeout(60 * time.Second))
		humaRouter.Use(corsMiddleware)

		// Register HUMA routes on separate server
		log.Printf("🚀 Starting separate HUMA server on port %s", humaPort)
		
		if apiPrefix == "" {
			humaRouter.Route("/auth", authModule.RegisterHumaRoutes)
			humaRouter.Route("/dev", devModule.RegisterHumaRoutes)
			humaRouter.Route("/users", usersModule.RegisterHumaRoutes)
			humaRouter.Route("/notifications", notificationsModule.RegisterHumaRoutes)
			humaRouter.Route("/sde", sdeModule.RegisterHumaRoutes)
			humaRouter.Route("/scheduler", schedulerModule.RegisterHumaRoutes)
		} else {
			humaRouter.Route(apiPrefix+"/auth", authModule.RegisterHumaRoutes)
			humaRouter.Route(apiPrefix+"/dev", devModule.RegisterHumaRoutes)
			humaRouter.Route(apiPrefix+"/users", usersModule.RegisterHumaRoutes)
			humaRouter.Route(apiPrefix+"/notifications", notificationsModule.RegisterHumaRoutes)
			humaRouter.Route(apiPrefix+"/sde", sdeModule.RegisterHumaRoutes)
			humaRouter.Route(apiPrefix+"/scheduler", schedulerModule.RegisterHumaRoutes)
		}

		humaSrv = &http.Server{
			Addr:         ":" + humaPort,
			Handler:      humaRouter,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		// Start separate HUMA server
		go func() {
			slog.Info("Starting separate HUMA API server", slog.String("addr", humaSrv.Addr))
			if err := humaSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				slog.Error("HUMA server failed to start", "error", err)
				os.Exit(1)
			}
		}()
		
		log.Printf("✅ HUMA APIs running on separate server: port %s", humaPort)
		log.Printf("📋 OpenAPI specs: http://localhost:%s/{module}/openapi.json", humaPort)
		log.Printf("🌐 Example: http://localhost:%s/auth/openapi.json", humaPort)
	} else {
		if humaPort != "" && !separateHumaServer {
			log.Printf("⚠️  HUMA_PORT=%s set but HUMA_SEPARATE_SERVER=false - using integrated mode", humaPort)
		}
		log.Printf("✅ HUMA APIs integrated with main server: port %s", port)
		log.Printf("📋 OpenAPI specs: http://localhost:%s/{module}/openapi.json", port)
		log.Printf("🌐 Example: http://localhost:%s/auth/openapi.json", port)
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