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
	Client      *mongo.Client
	Database    *mongo.Database
	uri         string
	serviceName string
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
		Client:      client,
		Database:    database,
		uri:         uri,
		serviceName: serviceName,
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
	
	err := m.Client.Ping(ctx, nil)
	if err != nil {
		log.Printf("MongoDB health check failed: %v", err)
		
		// Try to reconnect automatically
		log.Printf("Attempting to reconnect to MongoDB...")
		if reconnErr := m.reconnect(ctx); reconnErr != nil {
			log.Printf("Failed to reconnect to MongoDB: %v", reconnErr)
			return fmt.Errorf("mongodb ping failed and reconnect failed: ping=%v, reconnect=%v", err, reconnErr)
		}
		
		// Try ping again after reconnection
		if pingErr := m.Client.Ping(ctx, nil); pingErr != nil {
			log.Printf("MongoDB ping still failing after reconnect: %v", pingErr)
			return fmt.Errorf("mongodb still unhealthy after reconnect: %v", pingErr)
		}
		
		log.Printf("Successfully reconnected to MongoDB")
	}
	return nil
}

// reconnect attempts to reconnect to MongoDB
func (m *MongoDB) reconnect(ctx context.Context) error {
	// Close existing connection if it exists
	if m.Client != nil {
		if err := m.Client.Disconnect(ctx); err != nil {
			log.Printf("Warning: failed to disconnect existing MongoDB client: %v", err)
		}
	}
	
	// Create client options
	opts := options.Client().ApplyURI(m.uri)

	// Only add OpenTelemetry instrumentation if telemetry is enabled
	if config.GetBoolEnv("ENABLE_TELEMETRY", false) {
		opts.SetMonitor(otelmongo.NewMonitor())
	}

	// Set connection timeout
	connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connCtx, opts)
	if err != nil {
		return fmt.Errorf("failed to create new MongoDB client: %v", err)
	}

	// Ping to verify connection
	if err = client.Ping(connCtx, nil); err != nil {
		client.Disconnect(connCtx)
		return fmt.Errorf("failed to ping new MongoDB connection: %v", err)
	}

	// Update client and database references
	m.Client = client
	dbName := extractDatabaseName(m.uri, m.serviceName)
	m.Database = client.Database(dbName)
	
	return nil
}

func extractDatabaseName(uri, fallback string) string {
	// Simple extraction - in production, use proper URI parsing
	// For now, return fallback (service name)
	return fallback
}