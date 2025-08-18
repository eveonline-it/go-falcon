#!/bin/bash

# Development script with hot reload

echo "🚀 Starting development server with hot reload..."

# Check if Air is installed
AIR_PATH=$(go env GOPATH)/bin/air
if [ ! -f "$AIR_PATH" ]; then
    echo "❌ Air is not installed. Installing..."
    go install github.com/air-verse/air@latest
fi

# Load environment variables if .env exists
if [ -f .env ]; then
    echo "📄 Loading environment variables from .env..."
    set -o allexport
    source .env
    set +o allexport
else
    echo "❌ Error: .env file not found"
    echo "Please create a .env file in the project root"
    exit 1
fi

# Create tmp directory if it doesn't exist and set permissions
mkdir -p tmp
chmod 755 tmp
rm -f tmp/falcon 2>/dev/null || true

echo "🔥 Starting Air with hot reload..."
echo "📝 Watching for changes in:"
echo "   - *.go files"
echo "   - internal/ directory"
echo "   - pkg/ directory"
echo "   - cmd/ directory"
echo ""
echo "🌐 Application will be available at: http://localhost:8080"
echo "🔄 Press Ctrl+C to stop"
echo ""

# Start Air
$AIR_PATH