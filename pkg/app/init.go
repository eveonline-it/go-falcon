package app

import (
	"context"
	"log"
	"log/slog"

	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
	"go-falcon/pkg/logging"
	"go-falcon/pkg/sde"

	"github.com/joho/godotenv"
)

// AppContext holds the shared application context and dependencies
type AppContext struct {
	MongoDB          *database.MongoDB
	Redis            *database.Redis
	SDEService       sde.SDEService
	TelemetryManager *logging.TelemetryManager
	ServiceName      string
	shutdownFuncs    []func(context.Context) error
}

// InitializeApp initializes common application dependencies
func InitializeApp(serviceName string) (*AppContext, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}

	ctx := context.Background()

	// Initialize telemetry
	telemetryManager := logging.NewTelemetryManager()
	if err := telemetryManager.Initialize(ctx); err != nil {
		log.Printf("Warning: Failed to initialize telemetry: %v", err)
		// Continue without telemetry rather than failing
	}

	// Initialize databases
	mongodb, err := database.NewMongoDB(ctx, "falcon")
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		// Continue without MongoDB for now - some applications might not need it
	} else {
		slog.Info("Connected to MongoDB")
	}

	redis, err := database.NewRedis(ctx)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		// Continue without Redis for now - some applications might not need it
	} else {
		slog.Info("Connected to Redis")
	}

	// Initialize SDE service
	sdeService := sde.NewService("data/sde")
	slog.Info("SDE service initialized", "data_dir", "data/sde")

	appCtx := &AppContext{
		MongoDB:          mongodb,
		Redis:            redis,
		SDEService:       sdeService,
		TelemetryManager: telemetryManager,
		ServiceName:      serviceName,
	}

	// Register shutdown functions
	if mongodb != nil {
		appCtx.shutdownFuncs = append(appCtx.shutdownFuncs, mongodb.Close)
	}
	if redis != nil {
		appCtx.shutdownFuncs = append(appCtx.shutdownFuncs, func(ctx context.Context) error {
			return redis.Close()
		})
	}
	if telemetryManager != nil {
		appCtx.shutdownFuncs = append(appCtx.shutdownFuncs, telemetryManager.Shutdown)
	}

	return appCtx, nil
}

// Shutdown gracefully shuts down all application dependencies
func (a *AppContext) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down application", "service", a.ServiceName)

	for _, shutdown := range a.shutdownFuncs {
		if err := shutdown(ctx); err != nil {
			slog.Error("Error during shutdown", "error", err)
		}
	}

	slog.Info("Application shutdown completed", "service", a.ServiceName)
	return nil
}

// GetPort returns the port from environment or default
func GetPort(defaultPort string) string {
	return config.GetEnv("PORT", defaultPort)
}

// IsProduction returns true if running in production environment
func IsProduction() bool {
	env := config.GetEnv("NODE_ENV", "development")
	return env == "production"
}

// IsDevelopment returns true if running in development environment
func IsDevelopment() bool {
	return !IsProduction()
}
