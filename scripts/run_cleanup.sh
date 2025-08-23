#!/bin/bash

# Go Falcon - Old Groups Cleanup Script
# This script removes incorrectly created groups with old naming convention

set -e  # Exit on any error

echo "🧹 Go Falcon - Old Groups Cleanup"
echo "================================="
echo

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Must be run from the go-falcon project root directory"
    exit 1
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "❌ Error: .env file not found"
    exit 1
fi

# Load environment variables (only the ones we need)
export MONGO_URI=$(grep '^MONGO_URI=' .env | cut -d'=' -f2-)
export MONGO_DATABASE=$(grep '^MONGO_DATABASE=' .env | cut -d'=' -f2-)

# Build the cleanup binary
echo "🔨 Building cleanup binary..."
go build -o ./tmp/cleanup_old_groups ./scripts/cleanup_old_groups.go

if [ $? -ne 0 ]; then
    echo "❌ Failed to build cleanup binary"
    exit 1
fi

echo "   ✅ Cleanup binary built successfully"
echo

# Run the cleanup
echo "🏃 Running groups cleanup..."
./tmp/cleanup_old_groups

# Clean up
if [ -f "./tmp/cleanup_old_groups" ]; then
    rm ./tmp/cleanup_old_groups
    echo "🧹 Cleaned up cleanup binary"
fi

echo
echo "🎉 Cleanup script completed!"