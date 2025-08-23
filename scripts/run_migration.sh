#!/bin/bash

# Go Falcon - Groups and Site Settings Migration Runner
# This script builds and runs the database migration for the new group auto-join system

set -e  # Exit on any error

echo "🚀 Go Falcon Migration Runner"
echo "============================="
echo

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Must be run from the go-falcon project root directory"
    echo "   Current directory: $(pwd)"
    echo "   Please cd to your go-falcon directory and run: ./scripts/run_migration.sh"
    exit 1
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "❌ Error: .env file not found"
    echo "   Please create a .env file with your database configuration"
    echo "   You can copy from .env.example if available"
    exit 1
fi

# Load environment variables from .env file
echo "📋 Loading environment configuration..."
export $(grep -v '^#' .env | grep -E '^[A-Z_]+=.*' | xargs)

# Check required environment variables
if [ -z "$MONGO_URI" ]; then
    echo "❌ Error: MONGO_URI environment variable is required"
    echo "   Please set MONGO_URI in your .env file"
    echo "   Example: MONGO_URI=mongodb://admin:password@localhost:27017"
    exit 1
fi

if [ -z "$MONGO_DATABASE" ]; then
    echo "❌ Error: MONGO_DATABASE environment variable is required"  
    echo "   Please set MONGO_DATABASE in your .env file"
    echo "   Example: MONGO_DATABASE=falcon"
    exit 1
fi

echo "   ✅ Database URI: $MONGO_URI"
echo "   ✅ Database Name: $MONGO_DATABASE"
echo

# Build the migration binary
echo "🔨 Building migration binary..."
go build -o ./tmp/migrate_groups_and_site_settings ./scripts/migrate_groups_and_site_settings.go

if [ $? -ne 0 ]; then
    echo "❌ Failed to build migration binary"
    exit 1
fi

echo "   ✅ Migration binary built successfully"
echo

# Run the migration
echo "🏃 Running database migration..."
echo "⚠️  IMPORTANT: This will modify your database!"
echo

./tmp/migrate_groups_and_site_settings

# Clean up
if [ -f "./tmp/migrate_groups_and_site_settings" ]; then
    rm ./tmp/migrate_groups_and_site_settings
    echo "🧹 Cleaned up migration binary"
fi

echo
echo "🎉 Migration script completed!"
echo
echo "📋 Next Steps:"
echo "1. Start your Go Falcon server: make dev (or your preferred method)"
echo "2. Use the Site Settings API to add corporations and alliances"
echo "3. Enable the entities you want for auto-join groups"
echo "4. Test character login to verify auto-join functionality"
echo
echo "🔗 Useful API endpoints:"
echo "   POST /site-settings/corporations - Add a managed corporation"
echo "   POST /site-settings/alliances   - Add a managed alliance"
echo "   GET  /site-settings/corporations - List managed corporations"
echo "   GET  /site-settings/alliances   - List managed alliances"
echo "   GET  /groups                     - List all groups (including auto-created ones)"
echo