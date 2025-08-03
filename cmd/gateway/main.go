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
	"strings"
	"syscall"
	"time"

	"go-falcon/internal/auth"
	"go-falcon/internal/notifications"
	"go-falcon/internal/users"
	"go-falcon/pkg/app"
	"go-falcon/pkg/config"
	"go-falcon/pkg/evegate"
	"go-falcon/pkg/module"
	"go-falcon/pkg/version"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "go.uber.org/automaxprocs"
)

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
	
	// Initialize EVE Online ESI client as shared package
	evegateClient := evegate.NewClient()
	log.Printf("🚀 EVE Online ESI client initialized")

	// Mount module routes with configurable API prefix
	apiPrefix := config.GetAPIPrefix()
	log.Printf("🔗 Using API prefix: '%s'", apiPrefix)
	r.Route(apiPrefix+"/auth", authModule.Routes)
	r.Route(apiPrefix+"/users", usersModule.Routes)
	r.Route(apiPrefix+"/notifications", notificationsModule.Routes)
	
	// Note: evegate is now a shared package for EVE Online ESI integration
	// Other services can import and use: evegate.NewClient().GetServerStatus(ctx)
	_ = evegateClient // Available for modules to use

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
		slog.Info("Starting api gateway server", slog.String("addr", srv.Addr))
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