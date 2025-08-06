package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go-falcon/pkg/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(ctx context.Context, serviceName string) (*MongoDB, error) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://admin:password123@localhost:27017/" + serviceName + "?authSource=admin"
	}

	// Create client options
	opts := options.Client().ApplyURI(uri)

	// Only add OpenTelemetry instrumentation if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		opts.SetMonitor(otelmongo.NewMonitor())
	}

	// Set connection timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	// Extract database name from URI or use service name
	dbName := extractDatabaseName(uri, serviceName)
	database := client.Database(dbName)

	log.Printf("Connected to MongoDB database: %s", dbName)

	return &MongoDB{
		Client:   client,
		Database: database,
	}, nil
}

func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

func (m *MongoDB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return m.Client.Ping(ctx, nil)
}

func extractDatabaseName(uri, fallback string) string {
	// Simple extraction - in production, use proper URI parsing
	// For now, return fallback (service name)
	return fallback
}