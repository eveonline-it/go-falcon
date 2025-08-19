package app

import (
	"context"
	"log/slog"

	"go-falcon/pkg/config"
	"go-falcon/pkg/database"
	"go-falcon/pkg/logging"
	"go-falcon/pkg/sde"

	"github.com/joho/godotenv"
)

// AppContext holds the shared application context and dependencies
type AppContext struct {
	MongoDB           *database.MongoDB
	Redis             *database.Redis
	TelemetryManager  *logging.TelemetryManager
	SDEService        sde.SDEService
	ServiceName       string
	shutdownFuncs     []func(context.Context) error
}

// InitializeApp initializes common application dependencies
func InitializeApp(serviceName string) (*AppContext, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Silent - .env is optional
	}

	ctx := context.Background()

	// Initialize telemetry
	telemetryManager := logging.NewTelemetryManager()
	if err := telemetryManager.Initialize(ctx); err != nil {
		// Continue without telemetry rather than failing
	}

	// Initialize databases
	mongodb, err := database.NewMongoDB(ctx, "falcon")
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		// Continue without MongoDB for now - some applications might not need it
	}

	redis, err := database.NewRedis(ctx)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		// Continue without Redis for now - some applications might not need it
	}

	// Initialize SDE service
	sdeService := sde.NewService("data/sde")

	appCtx := &AppContext{
		MongoDB:          mongodb,
		Redis:            redis,
		TelemetryManager: telemetryManager,
		SDEService:       sdeService,
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
	for _, shutdown := range a.shutdownFuncs {
		if err := shutdown(ctx); err != nil {
			slog.Error("Error during shutdown", "error", err)
		}
	}
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