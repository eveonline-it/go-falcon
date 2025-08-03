package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"go-falcon/internal/auth"
	"go-falcon/internal/notifications"
	"go-falcon/internal/users"
	"go-falcon/pkg/app"
	"go-falcon/pkg/module"
	"go-falcon/pkg/version"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "go.uber.org/automaxprocs"
)

func main() {
	// Display startup banner
	fmt.Print(`
 ██████╗  ██████╗       ███████╗ █████╗ ██╗      ██████╗ ██████╗ ███╗   ██╗
██╔════╝ ██╔═══██╗      ██╔════╝██╔══██╗██║     ██╔════╝██╔═══██╗████╗  ██║
██║  ███╗██║   ██║█████╗█████╗  ███████║██║     ██║     ██║   ██║██╔██╗ ██║
██║   ██║██║   ██║╚════╝██╔══╝  ██╔══██║██║     ██║     ██║   ██║██║╚██╗██║
╚██████╔╝╚██████╔╝      ██║     ██║  ██║███████╗╚██████╗╚██████╔╝██║ ╚████║
 ╚═════╝  ╚═════╝       ╚═╝     ╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝

`)
	
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

	// Initialize Chi router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoint with version info
	r.Get("/health", enhancedHealthHandler)

	// Initialize modules
	var modules []module.Module
	authModule := auth.New(appCtx.MongoDB, appCtx.Redis)
	usersModule := users.New(appCtx.MongoDB, appCtx.Redis)
	notificationsModule := notifications.New(appCtx.MongoDB, appCtx.Redis)
	
	modules = append(modules, authModule, usersModule, notificationsModule)

	// Mount module routes
	r.Route("/api/auth", authModule.Routes)
	r.Route("/api/users", usersModule.Routes)
	r.Route("/api/notifications", notificationsModule.Routes)

	// Start background services for all modules
	for _, mod := range modules {
		go mod.StartBackgroundTasks(ctx)
	}

	// HTTP server setup
	port := app.GetPort("8080")
	
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		slog.Info("Starting gateway server", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
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

	// Shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
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
	slog.Info("Gateway health check requested",
		slog.String("remote_addr", r.RemoteAddr),
		slog.String("user_agent", r.UserAgent()),
	)
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