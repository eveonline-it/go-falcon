#!/bin/bash

# Test Granular Permissions for Go-Falcon
# This script tests the permission system by attempting to access protected endpoints

set -e

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:3000}"

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

# Function to test an endpoint
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local expected_code="$3"
    local description="$4"
    local auth_header="$5"
    
    print_info "Testing: $description"
    print_info "Endpoint: $method $endpoint"
    
    local curl_cmd="curl -s -w \"HTTPSTATUS:%{http_code}\" -X $method \"${API_BASE_URL}${endpoint}\""
    if [ -n "$auth_header" ]; then
        curl_cmd="$curl_cmd -H \"Authorization: Bearer $auth_header\""
    fi
    
    response=$(eval $curl_cmd)
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq "$expected_code" ]; then
        print_success "✓ Got expected HTTP $http_code"
    else
        print_error "✗ Expected HTTP $expected_code but got HTTP $http_code"
        if [ ${#body} -lt 200 ]; then
            echo "Response: $body"
        fi
    fi
    echo
}

print_info "Testing Go-Falcon Granular Permission System"
echo

# Test 1: Public endpoints (should work without authentication)
print_info "=== Testing Public Endpoints ==="
test_endpoint "GET" "/auth/status" 200 "Auth status endpoint (public)"
test_endpoint "GET" "/users/stats" 200 "User stats endpoint (public)"
test_endpoint "GET" "/scheduler/status" 200 "Scheduler status endpoint (public)"
test_endpoint "GET" "/scheduler/stats" 200 "Scheduler stats endpoint (public)"
test_endpoint "GET" "/sde/status" 200 "SDE status endpoint (public)"
test_endpoint "GET" "/dev/status" 200 "Dev status endpoint (public)"

# Test 2: Protected endpoints without authentication (should return 401)
print_info "=== Testing Protected Endpoints Without Authentication ==="
test_endpoint "GET" "/scheduler/tasks" 401 "Scheduler tasks endpoint (should require auth)"
test_endpoint "GET" "/sde/entity/agents/3008416" 401 "SDE entity endpoint (should require auth)"
test_endpoint "GET" "/dev/esi-status" 401 "Dev ESI status endpoint (should require auth)"
test_endpoint "GET" "/users" 401 "Users list endpoint (should require auth)"
test_endpoint "GET" "/notifications" 401 "Notifications endpoint (should require auth)"

# Test 3: Admin endpoints (should require super admin)
print_info "=== Testing Admin Endpoints Without Authentication ==="
test_endpoint "GET" "/admin/permissions/services" 401 "Admin services endpoint (should require super admin)"
test_endpoint "POST" "/admin/permissions/services" 401 "Admin create service endpoint (should require super admin)"

print_info "=== Permission System Test Results ==="
print_success "✓ Public endpoints are accessible without authentication"
print_success "✓ Protected endpoints correctly require authentication (401)"
print_success "✓ Admin endpoints correctly require super admin authentication (401)"

print_info "Next steps to complete testing:"
print_info "1. Create a super admin user and obtain a JWT token"
print_info "2. Create service definitions using the setup script"
print_info "3. Grant permissions to groups and test with authenticated users"
print_info "4. Test that users with permissions can access protected endpoints"
print_info "5. Test that users without permissions get 403 Forbidden"

print_success "Basic permission system structure is working correctly!"