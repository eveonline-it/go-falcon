package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"go-falcon/pkg/database"
	"go-falcon/pkg/logging"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}

	ctx := context.Background()

	// Initialize telemetry
	telemetryManager := logging.NewTelemetryManager()
	if err := telemetryManager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer telemetryManager.Shutdown(ctx)

	slog.Info("Starting backup utility...")

	// Initialize databases
	mongodb, err := database.NewMongoDB(ctx, "gateway")
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "error", err)
		os.Exit(1)
	}
	defer mongodb.Close(ctx)

	redis, err := database.NewRedis(ctx)
	if err != nil {
		slog.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()

	// TODO: Implement backup logic
	fmt.Println("Backup utility - Implementation pending")
	slog.Info("Backup completed successfully")
}