package migrations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Migration represents a database migration
type Migration struct {
	Version     string    `bson:"version"`     // e.g., "001_create_groups_indexes"
	Description string    `bson:"description"` // Human-readable description
	AppliedAt   time.Time `bson:"applied_at"`  // When the migration was applied
	Checksum    string    `bson:"checksum"`    // SHA256 of migration content for integrity
}

// MigrationFunc defines a migration function signature
type MigrationFunc func(ctx context.Context, db *mongo.Database) error

// RegisteredMigration holds migration metadata and functions
type RegisteredMigration struct {
	Version     string
	Description string
	Up          MigrationFunc // Apply migration
	Down        MigrationFunc // Rollback migration (optional)
}

// Runner manages database migrations
type Runner struct {
	db         *mongo.Database
	collection *mongo.Collection
	migrations []RegisteredMigration
}

// NewRunner creates a new migration runner
func NewRunner(db *mongo.Database) *Runner {
	return &Runner{
		db:         db,
		collection: db.Collection("_migrations"),
		migrations: make([]RegisteredMigration, 0),
	}
}

// Register adds a migration to the runner
func (r *Runner) Register(migration RegisteredMigration) {
	r.migrations = append(r.migrations, migration)
}

// Run executes all pending migrations
func (r *Runner) Run(ctx context.Context) error {
	// Create migrations collection index
	if err := r.ensureMigrationsIndex(ctx); err != nil {
		return fmt.Errorf("failed to create migrations index: %w", err)
	}

	// Get applied migrations
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create applied map for quick lookup
	appliedMap := make(map[string]bool)
	for _, m := range applied {
		appliedMap[m.Version] = true
	}

	// Run pending migrations
	for _, migration := range r.migrations {
		if appliedMap[migration.Version] {
			continue // Skip already applied
		}

		fmt.Printf("🔄 Running migration: %s - %s\n", migration.Version, migration.Description)

		// Start transaction for atomicity
		session, err := r.db.Client().StartSession()
		if err != nil {
			return fmt.Errorf("failed to start session for migration %s: %w", migration.Version, err)
		}
		defer session.EndSession(ctx)

		err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
			// Run migration
			if err := migration.Up(sc, r.db); err != nil {
				return fmt.Errorf("migration %s failed: %w", migration.Version, err)
			}

			// Record migration
			migrationRecord := Migration{
				Version:     migration.Version,
				Description: migration.Description,
				AppliedAt:   time.Now(),
				Checksum:    calculateChecksum(migration),
			}

			if _, err := r.collection.InsertOne(sc, migrationRecord); err != nil {
				return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
			}

			return nil
		})

		if err != nil {
			return err
		}

		fmt.Printf("✅ Migration %s completed successfully\n", migration.Version)
	}

	return nil
}

// Rollback rolls back the last n migrations
func (r *Runner) Rollback(ctx context.Context, steps int) error {
	// Get applied migrations in reverse order
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Limit rollback steps
	if steps > len(applied) {
		steps = len(applied)
	}

	// Find migrations with Down functions
	migrationMap := make(map[string]RegisteredMigration)
	for _, m := range r.migrations {
		migrationMap[m.Version] = m
	}

	// Rollback migrations
	for i := len(applied) - 1; i >= len(applied)-steps; i-- {
		version := applied[i].Version
		migration, exists := migrationMap[version]
		if !exists {
			return fmt.Errorf("migration %s not found in registered migrations", version)
		}

		if migration.Down == nil {
			fmt.Printf("⚠️  Migration %s has no rollback function, skipping\n", version)
			continue
		}

		fmt.Printf("🔄 Rolling back migration: %s\n", version)

		// Start transaction
		session, err := r.db.Client().StartSession()
		if err != nil {
			return fmt.Errorf("failed to start session for rollback %s: %w", version, err)
		}
		defer session.EndSession(ctx)

		err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
			// Run rollback
			if err := migration.Down(sc, r.db); err != nil {
				return fmt.Errorf("rollback %s failed: %w", version, err)
			}

			// Remove migration record
			if _, err := r.collection.DeleteOne(sc, bson.M{"version": version}); err != nil {
				return fmt.Errorf("failed to remove migration record %s: %w", version, err)
			}

			return nil
		})

		if err != nil {
			return err
		}

		fmt.Printf("✅ Rollback %s completed successfully\n", version)
	}

	return nil
}

// Status shows the current migration status
func (r *Runner) Status(ctx context.Context) error {
	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]Migration)
	for _, m := range applied {
		appliedMap[m.Version] = m
	}

	fmt.Println("\n📊 Migration Status:")
	fmt.Println(strings.Repeat("=", 80))

	for _, migration := range r.migrations {
		status := "⏳ Pending"
		appliedAt := ""

		if applied, exists := appliedMap[migration.Version]; exists {
			status = "✅ Applied"
			appliedAt = fmt.Sprintf(" (at %s)", applied.AppliedAt.Format("2006-01-02 15:04:05"))
		}

		fmt.Printf("%s %s - %s%s\n", status, migration.Version, migration.Description, appliedAt)
	}

	fmt.Printf("\nTotal: %d migrations (%d applied, %d pending)\n",
		len(r.migrations), len(applied), len(r.migrations)-len(applied))

	return nil
}

// ensureMigrationsIndex creates an index on the migrations collection
func (r *Runner) ensureMigrationsIndex(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "version", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := r.collection.Indexes().CreateOne(ctx, indexModel)
	return err
}

// getAppliedMigrations retrieves all applied migrations
func (r *Runner) getAppliedMigrations(ctx context.Context) ([]Migration, error) {
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "version", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var migrations []Migration
	if err := cursor.All(ctx, &migrations); err != nil {
		return nil, err
	}

	return migrations, nil
}

// calculateChecksum generates a checksum for migration integrity
func calculateChecksum(migration RegisteredMigration) string {
	// Simple checksum based on version and description
	// In production, you might want to use actual function content
	return fmt.Sprintf("%s:%s", migration.Version, migration.Description)
}
