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
	"go-falcon/internal/scheduler"
	"go-falcon/internal/users"
	"go-falcon/pkg/app"
	"go-falcon/pkg/config"
	evegateway "go-falcon/pkg/evegateway"
	"go-falcon/pkg/module"
	"go-falcon/pkg/version"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/danielgtaylor/huma/v2"
	falconMiddleware "go-falcon/pkg/middleware"
	casbinPkg "go-falcon/pkg/middleware/casbin"
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
	log.Printf("🏷️  Version: %s | Build: %s", version.GetVersionString(), versionInfo.BuildDate)
	log.Printf("🖥️  CPUs: %d | GOMAXPROCS: %d", runtime.NumCPU(), runtime.GOMAXPROCS(0))

	ctx := context.Background()

	// Initialize application with shared components
	appCtx, err := app.InitializeApp("falcon")
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer appCtx.Shutdown(ctx)

	// Print memory stats (compact)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("💾 Memory: %s heap | %s total", formatBytes(m.HeapAlloc), formatBytes(m.Sys))
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
	r.Use(falconMiddleware.TracingMiddleware) // Add our custom tracing middleware

	// Health check endpoint with version info
	r.Get("/health", enhancedHealthHandler)

	// Initialize EVE Online ESI client as shared package
	evegateClient := evegateway.NewClient()
	
	// Initialize modules
	var modules []module.Module
	authModule := auth.New(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService, evegateClient)
	
	// Initialize CASBIN middleware factory for authorization
	var casbinFactory *casbinPkg.CasbinMiddlewareFactory
	characterResolver := falconMiddleware.NewUserCharacterResolver(appCtx.MongoDB, appCtx.Redis)
	casbinFactory, err = casbinPkg.NewCasbinMiddlewareFactory(
		authModule.GetAuthService(),
		characterResolver,
		appCtx.MongoDB.Client,
		"falcon",
	)
	if err != nil {
		log.Printf("⚠️  CASBIN disabled: %v", err)
	} else {
		// Set CASBIN service in auth module for hierarchy sync
		authModule.GetAuthService().SetCasbinService(casbinFactory.GetCasbinService())
		
		if setupErr := setupInitialCasbinPolicies(casbinFactory); setupErr != nil {
			log.Printf("⚠️  CASBIN policies setup failed: %v", setupErr)
		} else {
			log.Printf("🔒 CASBIN authorization enabled")
		}
	}
	
	usersModule := users.New(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService, authModule, casbinFactory)
	schedulerModule := scheduler.New(appCtx.MongoDB, appCtx.Redis, appCtx.SDEService, authModule, casbinFactory)
	
	modules = append(modules, authModule, usersModule, schedulerModule)

	// Mount module routes with configurable API prefix
	apiPrefix := config.GetAPIPrefix()
	
	// Create unified Huma API configuration
	humaConfig := huma.DefaultConfig("Go Falcon API Server", "1.0.0")
	humaConfig.Info.Description = "Unified EVE Online API Server with modular architecture"
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
	
	// Register all module routes on the unified API
	authModule.RegisterUnifiedRoutes(unifiedAPI, "/auth")
	usersModule.RegisterUnifiedRoutes(unifiedAPI, "/users")
	schedulerModule.RegisterUnifiedRoutes(unifiedAPI, "/scheduler")
	
	// Register role management routes if CASBIN is available
	if casbinFactory != nil {
		roleManagementRoutes := casbinPkg.NewRoleManagementRoutes(
			casbinPkg.NewRoleAssignmentService(casbinFactory.GetEnhanced().GetCasbinAuth().GetEnforcer()),
			casbinFactory.GetEnhanced().GetCasbinAuth(),
		)
		// Register on unified API with empty basePath since unifiedAPI already has the prefix
		roleManagementRoutes.RegisterRoleManagementRoutes(unifiedAPI, "")
	}
	
	// Register CASBIN management API if initialized
	if casbinFactory != nil {
		apiHandler := casbinFactory.GetAPIHandler()
		r.Route(apiPrefix+"/admin/permissions", func(adminRouter chi.Router) {
			adminRouter.Use(func(next http.Handler) http.Handler {
				return casbinFactory.GetConvenience().SuperAdminOnly()(next)
			})
			apiHandler.RegisterRoutes(adminRouter)
		})
	}
	
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
	
	
	// Main server
	srv := &http.Server{
		Addr:         host + ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Display server configuration
	serverAddr := host + ":" + port
	if host == "0.0.0.0" {
		log.Printf("🚀 Server: http://localhost:%s%s | OpenAPI: %s/openapi.json", port, apiPrefix, apiPrefix)
	} else {
		log.Printf("🚀 Server: http://%s%s | OpenAPI: %s/openapi.json", serverAddr, apiPrefix, apiPrefix)
	}

	// Start main server
	go func() {
		slog.Info("Starting main API falcon server", slog.String("addr", srv.Addr))
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

func displayBanner() {
	file, err := os.Open("banner.txt")
	if err != nil {
		// Fallback to inline banner if file not found
		fmt.Print("\033[38;5;33m")
		fmt.Print("GO-FALCON API Server\n")
		fmt.Print("\033[0m")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		fmt.Print("\033[38;5;33m")
		fmt.Print("GO-FALCON API Server\n")
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
		log.Printf("📦 Container limit: %s", formatBytes(uint64(limit)))
		return
	}
	
	// Try cgroups v1 (older systems)
	if limit := readCgroupV1MemoryLimit(); limit > 0 {
		log.Printf("📦 Container limit: %s", formatBytes(uint64(limit)))
		return
	}
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

// setupInitialCasbinPolicies sets up initial roles and permissions for CASBIN
func setupInitialCasbinPolicies(factory *casbinPkg.CasbinMiddlewareFactory) error {
	casbinAuth := factory.GetEnhanced().GetCasbinAuth()
	
	// Define basic roles and their permissions based on user journey
	rolePolicies := map[string][][]string{
		// Super Admin role - full system access (assigned to first user)
		"role:super_admin": {
			{"system", "super_admin", "allow"},
			{"system", "admin", "allow"},
			{"users", "admin", "allow"},
			{"scheduler", "admin", "allow"},
			{"auth", "admin", "allow"},
			{"alliance", "admin", "allow"},
			{"corporation", "admin", "allow"},
		},
		
		// Admin role - general administration
		"role:admin": {
			{"system", "admin", "allow"},
			{"users", "read", "allow"},
			{"users", "write", "allow"},
			{"users", "admin", "allow"},
			{"scheduler", "admin", "allow"},
			{"auth", "admin", "allow"},
		},
		
		// Alliance Member role - alliance-level permissions
		"role:alliance_member": {
			{"alliance", "read", "allow"},
			{"users.alliance", "read", "allow"},
			{"scheduler.alliance", "read", "allow"},
			{"auth.profile", "read", "allow"},
			{"auth.profile", "write", "allow"},
		},
		
		// Corporation Member role - corporation-level permissions
		"role:corporation_member": {
			{"corporation", "read", "allow"},
			{"users.corporation", "read", "allow"},
			{"scheduler.corporation", "read", "allow"},
			{"auth.profile", "read", "allow"},
			{"auth.profile", "write", "allow"},
		},
		
		// Member role - basic authenticated member
		"role:member": {
			{"users.profiles", "read", "allow"},
			{"scheduler.tasks", "read", "allow"},
			{"scheduler.tasks", "write", "allow"},
			{"auth.profile", "read", "allow"},
			{"auth.profile", "write", "allow"},
		},
		
		// Registered role - newly registered user
		"role:registered": {
			{"auth.profile", "read", "allow"},
			{"auth.profile", "write", "allow"},
			{"users.profiles", "read", "allow"},
			{"public", "read", "allow"},
		},
		
		// Login role - basic login access
		"role:login": {
			{"auth.status", "read", "allow"},
			{"auth.profile", "read", "allow"},
			{"public", "read", "allow"},
		},
	}
	
	// Add all role policies
	for role, policies := range rolePolicies {
		for _, policy := range policies {
			resource := policy[0]
			action := policy[1]
			effect := policy[2]
			
			err := casbinAuth.AddPolicy(role, resource, action, effect)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to add policy for %s: %w", role, err)
			}
		}
	}
	
	// Note: Super admin role is assigned to the first user that registers in the system
	// This is handled in the user registration logic, not here
	
	return nil
}