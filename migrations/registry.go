package migrations

import (
	"go-falcon/pkg/migrations"
)

// registeredMigrations holds all registered migrations
var registeredMigrations []migrations.RegisteredMigration

// Register adds a migration to the registry
func Register(migration Migration) {
	registeredMigrations = append(registeredMigrations, migrations.RegisteredMigration{
		Version:     migration.Version,
		Description: migration.Description,
		Up:          migration.Up,
		Down:        migration.Down,
	})
}

// Migration is a convenience type for registering migrations
type Migration struct {
	Version     string
	Description string
	Up          migrations.MigrationFunc
	Down        migrations.MigrationFunc
}

// RegisterAll registers all migrations with the runner
func RegisterAll(runner *migrations.Runner) {
	for _, m := range registeredMigrations {
		runner.Register(m)
	}
}
