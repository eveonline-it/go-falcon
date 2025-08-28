# Database Migrations

## Overview

This directory contains database migrations for the Go Falcon project. Migrations provide version control for your database schema and seed data, ensuring consistent database setup across all environments.

## Migration System Features

- **Version Control**: Track database schema changes over time
- **Atomic Operations**: Migrations run in transactions for safety
- **Rollback Support**: Undo migrations when needed
- **Status Tracking**: See which migrations have been applied
- **Integrity Checks**: Checksums ensure migration consistency

## Usage

### Run All Pending Migrations
```bash
go run cmd/migrate/main.go -command=up
```

### Check Migration Status
```bash
go run cmd/migrate/main.go -command=status
```

### Rollback Last Migration
```bash
go run cmd/migrate/main.go -command=down -steps=1
```

### Rollback Multiple Migrations
```bash
go run cmd/migrate/main.go -command=down -steps=3
```

### Create New Migration
```bash
go run cmd/migrate/main.go -command=create -name=add_new_feature
```

### Dry Run (Preview Changes)
```bash
go run cmd/migrate/main.go -command=up -dry-run
```

## Migration Files

### Naming Convention

Migrations follow the pattern: `{version}_{description}.go`

- **Version**: Three-digit number (e.g., `001`, `002`, `003`)
- **Description**: Snake_case description of the migration

Examples:
- `001_create_groups_indexes.go`
- `002_create_scheduler_indexes.go`
- `003_seed_system_groups.go`

### Structure

Each migration file contains:

```go
package migrations

import (
    "context"
    "go.mongodb.org/mongo-driver/mongo"
)

func init() {
    Register(Migration{
        Version:     "001_create_users_table",
        Description: "Create users table with indexes",
        Up:          up001,
        Down:        down001,
    })
}

func up001(ctx context.Context, db *mongo.Database) error {
    // Apply migration
    return nil
}

func down001(ctx context.Context, db *mongo.Database) error {
    // Rollback migration
    return nil
}
```

## Current Migrations

| Version | Description | Purpose |
|---------|-------------|---------|
| 001 | create_groups_indexes | Creates indexes for groups and group_memberships collections |
| 002 | create_scheduler_indexes | Creates indexes for scheduler_tasks and scheduler_executions |
| 003 | seed_system_groups | Seeds initial system groups (super_admin, authenticated, guest) |
| 004 | create_character_indexes | Creates indexes for characters collection including text search |
| 005 | create_users_indexes | Creates indexes for users collection |
| 006 | create_user_profiles_indexes | Creates indexes for user_profiles collection (auth system) |
| 007 | create_auth_states_indexes | Creates indexes for auth_states collection (EVE SSO states) |
| 008 | create_permissions_indexes | Creates indexes for permissions collection (permission system) |
| 009 | create_group_permissions_indexes | Creates indexes for group_permissions collection (assignments) |
| 010 | create_alliances_indexes | Creates indexes for alliances collection (EVE alliance data) |
| 011 | create_corporations_indexes | Creates indexes for corporations collection (EVE corporation data) |
| 012 | create_routes_indexes | Creates indexes for routes collection (dynamic routing system) |
| 013 | create_site_settings_indexes_and_seed | Creates indexes and seed data for site_settings |

## Integration with Application

### Remove Runtime Index Creation

Update modules to remove `CreateIndexes()` calls from initialization:

```go
// BEFORE (in module.Initialize())
if err := s.repo.CreateIndexes(ctx); err != nil {
    return fmt.Errorf("failed to create indexes: %w", err)
}

// AFTER - Remove index creation, rely on migrations
// Indexes are created via migrations before app starts
```

### Deployment Process

1. **Development**: Run migrations manually during development
2. **CI/CD**: Run migrations as part of deployment pipeline
3. **Production**: Run migrations before starting the application

Example Docker Compose:

```yaml
services:
  migrate:
    build: .
    command: ["/app/migrate", "-command=up"]
    depends_on:
      - mongodb
    environment:
      - MONGODB_URI=mongodb://mongodb:27017
      - MONGODB_DATABASE=go_falcon
    
  app:
    build: .
    command: ["/app/falcon"]
    depends_on:
      migrate:
        condition: service_completed_successfully
```

## Best Practices

### Do's
- ✅ Test migrations in development first
- ✅ Keep migrations small and focused
- ✅ Always implement rollback (Down) functions when possible
- ✅ Use transactions for data consistency
- ✅ Document complex migrations with comments
- ✅ Version control all migration files

### Don'ts
- ❌ Don't modify existing migration files after deployment
- ❌ Don't skip version numbers
- ❌ Don't mix schema and large data migrations
- ❌ Don't use migrations for runtime configuration
- ❌ Don't delete migration files

## Troubleshooting

### Migration Failed

If a migration fails:
1. Check the error message for details
2. Fix the issue in the migration code
3. Rollback if partially applied: `go run cmd/migrate/main.go -command=down`
4. Retry the migration: `go run cmd/migrate/main.go -command=up`

### Duplicate Key Errors

For seed data migrations, use `InsertMany` with `SetOrdered(false)` to ignore duplicates:

```go
opts := options.InsertMany().SetOrdered(false)
_, err := collection.InsertMany(ctx, documents, opts)
if err != nil && !mongo.IsDuplicateKeyError(err) {
    return err
}
```

### Index Creation Timeout

For large collections, consider:
- Creating indexes in background: `options.Index().SetBackground(true)`
- Increasing context timeout in migration runner
- Running index creation separately during maintenance windows

## Adding Module Migrations

When adding a new module:

1. Create migration file: `go run cmd/migrate/main.go -command=create -name=create_module_indexes`
2. Implement index creation in Up function
3. Remove `CreateIndexes()` from module initialization
4. Test migration: `go run cmd/migrate/main.go -command=up -dry-run`
5. Apply migration: `go run cmd/migrate/main.go -command=up`

## Migration Tracking

Migrations are tracked in the `_migrations` collection:

```json
{
  "version": "001_create_groups_indexes",
  "description": "Create indexes for groups and group_memberships collections",
  "applied_at": "2025-01-20T10:30:00Z",
  "checksum": "001_create_groups_indexes:Create indexes..."
}
```

This ensures:
- Migrations run only once
- Application knows current schema version
- Rollbacks can be tracked
- Migration integrity is maintained