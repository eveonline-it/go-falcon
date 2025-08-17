#!/bin/bash

# Setup Granular Permissions for Go-Falcon
# This script creates the service definitions for the new granular permission system
# Run this script after starting the gateway with a super admin user

set -e

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
SUPER_ADMIN_JWT="${SUPER_ADMIN_JWT:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if JWT token is provided
if [ -z "$SUPER_ADMIN_JWT" ]; then
    print_error "SUPER_ADMIN_JWT environment variable is required"
    print_info "Set the super admin JWT token: export SUPER_ADMIN_JWT=your_jwt_token"
    exit 1
fi

# Function to create a service
create_service() {
    local service_data="$1"
    local service_name=$(echo "$service_data" | jq -r '.name')
    
    print_info "Creating service: $service_name"
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "${API_BASE_URL}/admin/permissions/services" \
        -H "Authorization: Bearer $SUPER_ADMIN_JWT" \
        -H "Content-Type: application/json" \
        -d "$service_data")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq 201 ]; then
        print_success "Service '$service_name' created successfully"
    elif [ "$http_code" -eq 409 ]; then
        print_warning "Service '$service_name' already exists"
    else
        print_error "Failed to create service '$service_name' (HTTP $http_code): $body"
        return 1
    fi
}

print_info "Setting up granular permissions for Go-Falcon modules..."

# Create Scheduler Service
print_info "Setting up Scheduler service..."
create_service '{
  "name": "scheduler",
  "display_name": "Task Scheduler",
  "description": "Task scheduling and management system with cron scheduling and distributed locking",
  "resources": [
    {
      "name": "tasks",
      "display_name": "Scheduled Tasks",
      "description": "Task definitions, management, and lifecycle operations",
      "actions": ["read", "write", "delete", "execute", "admin"],
      "enabled": true
    },
    {
      "name": "executions",
      "display_name": "Task Executions",
      "description": "Task execution history and runtime details",
      "actions": ["read"],
      "enabled": true
    }
  ]
}'

# Create SDE Service
print_info "Setting up SDE service..."
create_service '{
  "name": "sde",
  "display_name": "Static Data Export",
  "description": "EVE Online static data management with automated processing and scheduler integration",
  "resources": [
    {
      "name": "entities",
      "display_name": "SDE Entities",
      "description": "EVE Online static data entities including agents, blueprints, types, and universe data",
      "actions": ["read"],
      "enabled": true
    },
    {
      "name": "management",
      "display_name": "SDE Management",
      "description": "SDE update processes, index rebuilding, and administrative operations",
      "actions": ["read", "write", "admin"],
      "enabled": true
    }
  ]
}'

# Create Dev Service
print_info "Setting up Dev service..."
create_service '{
  "name": "dev",
  "display_name": "Development Tools",
  "description": "ESI testing and SDE data access tools for development and debugging",
  "resources": [
    {
      "name": "tools",
      "display_name": "Development Tools",
      "description": "ESI testing endpoints, SDE data access, and development utilities",
      "actions": ["read", "write"],
      "enabled": true
    }
  ]
}'

# Create Users Service
print_info "Setting up Users service..."
create_service '{
  "name": "users",
  "display_name": "User Management",
  "description": "User profile management and character administration",
  "resources": [
    {
      "name": "profiles",
      "display_name": "User Profiles",
      "description": "User profiles, character management, and account administration",
      "actions": ["read", "write", "delete"],
      "enabled": true
    }
  ]
}'

# Create Notifications Service
print_info "Setting up Notifications service..."
create_service '{
  "name": "notifications",
  "display_name": "Notification System",
  "description": "User notification management and messaging system",
  "resources": [
    {
      "name": "messages",
      "display_name": "Notification Messages",
      "description": "User notifications, alerts, and messaging functionality",
      "actions": ["read", "write", "delete"],
      "enabled": true
    }
  ]
}'

print_success "All services created successfully!"

print_info "Next steps:"
print_info "1. Grant permissions to appropriate groups using the admin API"
print_info "2. Test the permission system with different user groups"
print_info "3. Monitor the system for proper access control"

print_info "Example permission grant commands:"
echo ""
print_info "Grant SDE read access to authenticated users:"
echo "curl -X POST \"\${API_BASE_URL}/admin/permissions/assignments\" \\"
echo "  -H \"Authorization: Bearer \$SUPER_ADMIN_JWT\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"service\": \"sde\", \"resource\": \"entities\", \"action\": \"read\", \"subject_type\": \"group\", \"subject_id\": \"full_group_id\", \"reason\": \"Allow authenticated users to read SDE data\"}'"

echo ""
print_info "Grant scheduler admin access to administrators:"
echo "curl -X POST \"\${API_BASE_URL}/admin/permissions/assignments\" \\"
echo "  -H \"Authorization: Bearer \$SUPER_ADMIN_JWT\" \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"service\": \"scheduler\", \"resource\": \"tasks\", \"action\": \"admin\", \"subject_type\": \"group\", \"subject_id\": \"administrators_group_id\", \"reason\": \"Allow administrators to manage scheduled tasks\"}'"

print_success "Granular permission system setup complete!"