package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go-falcon/pkg/app"
	pkgMigrations "go-falcon/pkg/migrations"

	// Import all migration files to register them
	localMigrations "go-falcon/migrations"
)

func main() {
	// Define command flags
	var (
		command = flag.String("command", "up", "Migration command: up, down, status, create")
		steps   = flag.Int("steps", 0, "Number of migrations to rollback (for down command)")
		name    = flag.String("name", "", "Migration name (for create command)")
		dryRun  = flag.Bool("dry-run", false, "Show what would be done without executing")
	)

	flag.Parse()

	// Initialize context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Initialize application (just for database connection)
	appCtx, err := app.InitializeApp("migrate")
	if err != nil {
		log.Fatalf("❌ Failed to initialize application: %v", err)
	}
	defer appCtx.Shutdown(ctx)

	// Create migration runner
	runner := pkgMigrations.NewRunner(appCtx.MongoDB.Database)

	// Register all migrations
	localMigrations.RegisterAll(runner)

	// Execute command
	switch *command {
	case "up":
		fmt.Println("🚀 Running database migrations...")
		if *dryRun {
			fmt.Println("⚠️  DRY RUN MODE - No changes will be made")
			if err := runner.Status(ctx); err != nil {
				log.Fatalf("❌ Failed to show status: %v", err)
			}
		} else {
			if err := runner.Run(ctx); err != nil {
				log.Fatalf("❌ Migration failed: %v", err)
			}
			fmt.Println("✅ All migrations completed successfully")
		}

	case "down":
		if *steps == 0 {
			*steps = 1 // Default to rolling back 1 migration
		}
		fmt.Printf("🔄 Rolling back %d migration(s)...\n", *steps)
		if *dryRun {
			fmt.Println("⚠️  DRY RUN MODE - No changes will be made")
			if err := runner.Status(ctx); err != nil {
				log.Fatalf("❌ Failed to show status: %v", err)
			}
		} else {
			if err := runner.Rollback(ctx, *steps); err != nil {
				log.Fatalf("❌ Rollback failed: %v", err)
			}
			fmt.Println("✅ Rollback completed successfully")
		}

	case "status":
		if err := runner.Status(ctx); err != nil {
			log.Fatalf("❌ Failed to get migration status: %v", err)
		}

	case "create":
		if *name == "" {
			log.Fatal("❌ Migration name is required for create command")
		}
		if err := createMigration(*name); err != nil {
			log.Fatalf("❌ Failed to create migration: %v", err)
		}

	default:
		log.Fatalf("❌ Unknown command: %s", *command)
	}
}

// createMigration creates a new migration file template
func createMigration(name string) error {
	// Get next version number
	version := fmt.Sprintf("%03d", getNextVersionNumber())
	filename := fmt.Sprintf("migrations/%s_%s.go", version, name)

	template := `package migrations

import (
	"context"
	
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	Register(Migration{
		Version:     "%s_%s",
		Description: "TODO: Add description",
		Up:          up%s,
		Down:        down%s,
	})
}

func up%s(ctx context.Context, db *mongo.Database) error {
	// TODO: Implement migration
	return nil
}

func down%s(ctx context.Context, db *mongo.Database) error {
	// TODO: Implement rollback
	return nil
}
`

	content := fmt.Sprintf(template, version, name, version, version, version, version)

	// Create migrations directory if it doesn't exist
	if err := os.MkdirAll("migrations", 0755); err != nil {
		return err
	}

	// Check if file already exists
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("migration file %s already exists", filename)
	}

	// Write file
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return err
	}

	fmt.Printf("✅ Created migration file: %s\n", filename)
	fmt.Println("📝 Don't forget to:")
	fmt.Println("   1. Update the Description field")
	fmt.Println("   2. Implement the up() function")
	fmt.Println("   3. Implement the down() function (if possible)")
	fmt.Println("   4. Import the migration in migrations/registry.go")

	return nil
}

// getNextVersionNumber determines the next migration version number
func getNextVersionNumber() int {
	// Read migrations directory
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return 1 // Start at 001 if directory doesn't exist
	}

	maxVersion := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Extract version number from filename (e.g., "001_create_users.go")
		var version int
		_, err := fmt.Sscanf(entry.Name(), "%03d_", &version)
		if err == nil && version > maxVersion {
			maxVersion = version
		}
	}

	return maxVersion + 1
}
