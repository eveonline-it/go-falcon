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
	"go-falcon/pkg/database"
	"go-falcon/pkg/logging"
	"go-falcon/pkg/version"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	_ "go.uber.org/automaxprocs"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}

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

	// Initialize telemetry
	telemetryManager := logging.NewTelemetryManager()
	if err := telemetryManager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer telemetryManager.Shutdown(ctx)

	// Initialize databases
	mongodb, err := database.NewMongoDB(ctx, "gateway")
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		// Continue without MongoDB for now
	} else {
		defer mongodb.Close(ctx)
		slog.Info("Connected to MongoDB")
	}

	redis, err := database.NewRedis(ctx)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		// Continue without Redis for now
	} else {
		defer redis.Close()
		slog.Info("Connected to Redis")
	}

	// Initialize Chi router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check endpoint
	r.Get("/health", healthHandler)

	// Initialize modules
	authModule := auth.New(mongodb, redis)
	usersModule := users.New(mongodb, redis)
	notificationsModule := notifications.New(mongodb, redis)

	// Mount module routes
	r.Route("/api/auth", func(r chi.Router) {
		authModule.Routes(r)
	})

	r.Route("/api/users", func(r chi.Router) {
		usersModule.Routes(r)
	})

	r.Route("/api/notifications", func(r chi.Router) {
		notificationsModule.Routes(r)
	})

	// Start background services
	go authModule.StartBackgroundTasks(ctx)
	go usersModule.StartBackgroundTasks(ctx)
	go notificationsModule.StartBackgroundTasks(ctx)

	// HTTP server setup
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
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

	// Stop background services
	authModule.Stop()
	usersModule.Stop()
	notificationsModule.Stop()

	// Close database connections
	if mongodb != nil {
		if err := mongodb.Close(shutdownCtx); err != nil {
			slog.Error("Error closing MongoDB connection", "error", err)
		}
	}

	if redis != nil {
		if err := redis.Close(); err != nil {
			slog.Error("Error closing Redis connection", "error", err)
		}
	}

	// Shutdown telemetry
	telemetryManager.Shutdown(shutdownCtx)

	slog.Info("Gateway shutdown completed successfully")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Health check requested",
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