.PHONY: dev build build-all build-utils clean test install-tools help version postman postman-build lint fmt tidy dev-setup quick-test

# Version variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_USER ?= $(shell whoami)@$(shell hostname)

# Build flags for version injection
LDFLAGS = -ldflags "\
	-X 'go-falcon/pkg/version.Version=$(VERSION)' \
	-X 'go-falcon/pkg/version.GitCommit=$(GIT_COMMIT)' \
	-X 'go-falcon/pkg/version.GitBranch=$(GIT_BRANCH)' \
	-X 'go-falcon/pkg/version.BuildDate=$(BUILD_DATE)' \
	-X 'go-falcon/pkg/version.BuildUser=$(BUILD_USER)'"

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

dev: ## Start development server with hot reload
	@echo "🚀 Starting development server with hot reload..."
	@./scripts/dev.sh

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Branch: $(GIT_BRANCH)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Build User: $(BUILD_USER)"

build: ## Build the falcon application
	@echo "🔨 Building falcon application..."
	@go build $(LDFLAGS) -o falcon ./cmd/gateway
	@echo "✅ Build complete: ./falcon"

build-all: ## Build all applications (gateway, backup, restore, postman)
	@echo "🔨 Building all applications..."
	@go build $(LDFLAGS) -o falcon ./cmd/gateway
	@go build $(LDFLAGS) -o backup ./cmd/backup
	@go build $(LDFLAGS) -o restore ./cmd/restore
	@go build $(LDFLAGS) -o postman ./cmd/postman
	@echo "✅ Build complete: ./falcon, ./backup, ./restore, ./postman"

build-utils: ## Build utility applications (backup, restore, postman)
	@echo "🔨 Building utility applications..."
	@go build $(LDFLAGS) -o backup ./cmd/backup
	@go build $(LDFLAGS) -o restore ./cmd/restore
	@go build $(LDFLAGS) -o postman ./cmd/postman
	@echo "✅ Build complete: ./backup, ./restore, ./postman"

clean: ## Clean build artifacts and temporary files
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf tmp/
	@rm -f falcon gateway backup restore postman
	@rm -f *.postman_collection.json
	@echo "✅ Clean complete"

test: ## Run tests
	@echo "🧪 Running tests..."
	@go test ./...

install-tools: ## Install development tools
	@echo "📦 Installing development tools..."
	@go install github.com/air-verse/air@latest
	@echo "✅ Tools installed"

docker-infra: ## Start infrastructure services only
	@echo "🐳 Starting infrastructure services (MongoDB + Redis)..."
	@docker-compose -f docker-compose.infra.yml up -d

docker-logs: ## View infrastructure logs
	@echo "📋 Viewing infrastructure logs..."
	@docker-compose -f docker-compose.infra.yml logs -f

docker-logs-app: ## View production application logs
	@echo "📋 Viewing production application logs..."
	@docker-compose -f docker-compose.prod.yml logs -f gateway

docker-stop: ## Stop infrastructure services
	@echo "🛑 Stopping infrastructure services..."
	@docker-compose -f docker-compose.infra.yml down

docker-stop-all: ## Stop all services (infrastructure + production)
	@echo "🛑 Stopping all services..."
	@docker-compose -f docker-compose.infra.yml -f docker-compose.prod.yml down

# Database commands
db-up: ## Start only database services
	@echo "🗄️ Starting database services..."
	@docker-compose -f docker-compose.infra.yml up -d

db-down: ## Stop database services
	@echo "🗄️ Stopping database services..."
	@docker-compose -f docker-compose.infra.yml down

# Production deployment
deploy-prod: ## Deploy production environment (infrastructure + application)
	@echo "🚀 Deploying production environment..."
	@docker-compose -f docker-compose.infra.yml -f docker-compose.prod.yml up -d

stop-prod: ## Stop production environment
	@echo "🛑 Stopping production environment..."
	@docker-compose -f docker-compose.infra.yml -f docker-compose.prod.yml down

# Health check
health: ## Check application health
	@echo "🏥 Checking application health..."
	@curl -s http://localhost:8080/health | jq .

health-infra: ## Check infrastructure health
	@echo "🏥 Checking infrastructure health..."
	@echo "MongoDB:"
	@docker exec -it go-falcon-mongodb mongosh --eval "db.adminCommand('ping')" --quiet || echo "MongoDB not running"
	@echo "Redis:"
	@docker exec -it go-falcon-redis redis-cli ping || echo "Redis not running"

# Development tools
postman: ## Generate Postman collection for all gateway endpoints
	@echo "📋 Generating Postman collection..."
	@go run ./cmd/postman
	@echo "✅ Postman collection generated: go-falcon-gateway-endpoints.postman_collection.json"

postman-build: ## Build and run postman exporter
	@echo "🔨 Building postman exporter..."
	@go build $(LDFLAGS) -o postman ./cmd/postman
	@echo "📋 Generating Postman collection..."
	@./postman
	@rm ./postman
	@echo "✅ Postman collection generated: go-falcon-gateway-endpoints.postman_collection.json"

# Linting and code quality
lint: ## Run linter
	@echo "🔍 Running linter..."
	@golangci-lint run ./...

fmt: ## Format code
	@echo "✨ Formatting code..."
	@go fmt ./...

tidy: ## Tidy dependencies
	@echo "🧹 Tidying dependencies..."
	@go mod tidy

# Quick development workflow
dev-setup: install-tools docker-infra ## Set up development environment
	@echo "🚀 Development environment ready!"
	@echo "Run 'make dev' to start the development server"

quick-test: fmt lint test ## Run formatting, linting, and tests
	@echo "✅ Quick test suite completed successfully!"