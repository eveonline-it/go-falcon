#!/bin/bash

# Script to show available API routes in the Go Falcon gateway

echo "=== Go Falcon API Routes ==="
echo ""

echo "🔐 Authentication Routes (via /auth):"
echo "  GET  /auth/eve/login         - Initiate EVE SSO login"
echo "  GET  /auth/eve/callback      - OAuth2 callback handler"
echo "  GET  /auth/eve/verify        - Verify JWT token"
echo "  POST /auth/eve/refresh       - Refresh access tokens"
echo "  POST /auth/eve/token         - Exchange EVE token for JWT (mobile)"
echo "  GET  /auth/status            - Quick auth status check"
echo "  GET  /auth/user              - Get current user info"
echo "  POST /auth/logout            - Clear auth cookie"
echo "  GET  /auth/profile           - Get full user profile (requires auth)"
echo "  POST /auth/profile/refresh   - Refresh profile from ESI (requires auth)"
echo "  GET  /auth/profile/public    - Get public profile by ID"
echo ""


echo "👥 Users Routes (via /users):"
echo "  GET  /users/health           - Users module health check"
echo ""


echo "⏰ Scheduler Routes (via /scheduler):"
echo "  GET  /scheduler/health       - Scheduler module health check"
echo "  GET  /scheduler/tasks        - List all scheduled tasks (requires auth)"
echo "  POST /scheduler/tasks        - Create new task (admin only)"
echo "  GET  /scheduler/stats        - Get scheduler statistics (admin only)"
echo ""

echo "🏥 System Routes:"
echo "  GET  /health                     - Gateway health check with version info"
echo ""


echo "=== Authentication Examples ==="
echo ""
echo "# Web Authentication Flow:"
echo "1. GET  /auth/eve/login      - Get EVE SSO URL"
echo "2. [User authenticates with EVE Online]"
echo "3. GET  /auth/eve/callback   - Process callback (sets cookie)"
echo "4. GET  /auth/user           - Get user info from cookie"
echo ""
echo "# Mobile Authentication Flow:"
echo "1. [App handles EVE SSO directly]"
echo "2. POST /auth/eve/token      - Exchange EVE token for JWT"
echo "3. Use 'Authorization: Bearer <jwt>' header for API calls"
echo ""

echo "=== Configuration Environment Variables ==="
echo ""
echo "# Required for EVE SSO:"
echo "EVE_CLIENT_ID=your_client_id"
echo "EVE_CLIENT_SECRET=your_client_secret"
echo "JWT_SECRET=your_jwt_secret"
echo ""
echo "# Optional:"
echo "API_PREFIX=api                   # API prefix (default: 'api')"
echo "SUPER_ADMIN_CHARACTER_ID=123456789  # Super admin character ID"
echo ""