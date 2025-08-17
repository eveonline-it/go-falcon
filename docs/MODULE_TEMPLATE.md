# Module Template Guide

## Overview

This document defines the standardized structure for internal modules in the Go Falcon monolithic API gateway. All modules in `internal/` MUST follow this exact structure.

## Directory Structure

```
internal/modulename/
├── dto/                    # Data Transfer Objects
│   ├── requests.go        # Request DTOs with validation
│   ├── responses.go       # Response DTOs
│   └── validators.go      # Custom validation logic
├── middleware/            # Module-specific middleware
│   ├── auth.go           # Authentication middleware
│   ├── validation.go     # Request validation
│   └── ratelimit.go      # Rate limiting (if needed)
├── routes/               # Route definitions
│   ├── routes.go         # Main route registration
│   ├── health.go         # Health check endpoints
│   └── api.go            # API endpoint handlers
├── services/             # Business logic
│   ├── service.go        # Main service implementation
│   └── repository.go     # Database operations
├── models/               # Database models
│   └── models.go         # MongoDB/Redis schemas
├── module.go             # Module initialization
└── CLAUDE.md             # Module documentation
```

## File Templates

### module.go
```go
package modulename

import (
	"context"
	"net/http"

	"go-falcon/internal/modulename/middleware"
	"go-falcon/internal/modulename/routes"
	"go-falcon/internal/modulename/services"
	"go-falcon/pkg/database"
	"go-falcon/pkg/module"
	"go-falcon/pkg/sde"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	*module.BaseModule
	service    *services.Service
	repository *services.Repository
	middleware *middleware.Middleware
	routes     *routes.Routes
}

func New(mongodb *database.MongoDB, redis *database.Redis, sdeService sde.SDEService) *Module {
	baseModule := module.NewBaseModule("modulename", mongodb, redis, sdeService)
	repository := services.NewRepository(mongodb)
	service := services.NewService(repository, redis, sdeService)
	middlewareLayer := middleware.New(service)
	routeHandlers := routes.New(service, middlewareLayer)

	return &Module{
		BaseModule: baseModule,
		service:    service,
		repository: repository,
		middleware: middlewareLayer,
		routes:     routeHandlers,
	}
}

func (m *Module) Routes(r chi.Router) {
	m.routes.RegisterRoutes(r)
}

func (m *Module) StartBackgroundTasks(ctx context.Context) {
	// Override if module has specific background tasks
	m.BaseModule.StartBackgroundTasks(ctx)
}
```

### dto/requests.go
```go
package dto

import "time"

// CreateItemRequest represents a request to create a new item
type CreateItemRequest struct {
	Name        string            `json:"name" validate:"required,min=3,max=100"`
	Description string            `json:"description" validate:"max=500"`
	Tags        []string          `json:"tags" validate:"dive,min=1,max=50"`
	Metadata    map[string]string `json:"metadata" validate:"dive,keys,min=1,max=50,endkeys,min=1,max=500"`
}

// UpdateItemRequest represents a request to update an existing item
type UpdateItemRequest struct {
	Name        *string           `json:"name,omitempty" validate:"omitempty,min=3,max=100"`
	Description *string           `json:"description,omitempty" validate:"omitempty,max=500"`
	Tags        []string          `json:"tags,omitempty" validate:"omitempty,dive,min=1,max=50"`
	Metadata    map[string]string `json:"metadata,omitempty" validate:"omitempty,dive,keys,min=1,max=50,endkeys,min=1,max=500"`
}

// ListItemsRequest represents parameters for listing items
type ListItemsRequest struct {
	Page     int      `json:"page" validate:"min=1"`
	PageSize int      `json:"page_size" validate:"min=1,max=100"`
	Search   string   `json:"search" validate:"max=100"`
	Tags     []string `json:"tags" validate:"dive,min=1,max=50"`
	SortBy   string   `json:"sort_by" validate:"oneof=name created_at updated_at"`
	SortDir  string   `json:"sort_dir" validate:"oneof=asc desc"`
}
```

### dto/responses.go
```go
package dto

import "time"

// ItemResponse represents an item in API responses
type ItemResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CreatedBy   string            `json:"created_by"`
}

// ListItemsResponse represents a paginated list of items
type ListItemsResponse struct {
	Items      []ItemResponse `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// CreateItemResponse represents the response after creating an item
type CreateItemResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}
```

### dto/validators.go
```go
package dto

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// RegisterCustomValidators registers custom validation rules
func RegisterCustomValidators(validate *validator.Validate) error {
	// Register custom tag validator
	if err := validate.RegisterValidation("custom_tag", validateCustomTag); err != nil {
		return fmt.Errorf("failed to register custom_tag validator: %w", err)
	}

	// Register custom name validator
	if err := validate.RegisterValidation("custom_name", validateCustomName); err != nil {
		return fmt.Errorf("failed to register custom_name validator: %w", err)
	}

	return nil
}

// validateCustomTag validates custom tag format
func validateCustomTag(fl validator.FieldLevel) bool {
	tag := fl.Field().String()
	// Tags must be alphanumeric with dashes and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, tag)
	return matched
}

// validateCustomName validates custom name format
func validateCustomName(fl validator.FieldLevel) bool {
	name := fl.Field().String()
	// Names must start with letter and contain only letters, numbers, spaces, dashes
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9\s\-]*$`, name)
	return matched
}

// ValidateStruct validates a struct using the validator instance
func ValidateStruct(validate *validator.Validate, s interface{}) []string {
	var errors []string
	
	if err := validate.Struct(s); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			errors = append(errors, formatValidationError(err))
		}
	}
	
	return errors
}

// formatValidationError formats validation errors for user-friendly messages
func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", err.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", err.Field(), err.Param())
	case "custom_tag":
		return fmt.Sprintf("%s must contain only letters, numbers, dashes, and underscores", err.Field())
	case "custom_name":
		return fmt.Sprintf("%s must start with a letter and contain only letters, numbers, spaces, and dashes", err.Field())
	default:
		return fmt.Sprintf("%s is invalid", err.Field())
	}
}
```

### middleware/auth.go
```go
package middleware

import (
	"net/http"

	"go-falcon/internal/modulename/services"
)

type AuthMiddleware struct {
	service *services.Service
}

func NewAuthMiddleware(service *services.Service) *AuthMiddleware {
	return &AuthMiddleware{
		service: service,
	}
}

// RequireAuth ensures the user is authenticated
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add authentication logic here
		// This would typically check JWT tokens or session cookies
		next.ServeHTTP(w, r)
	})
}

// RequirePermission ensures the user has specific permissions
func (m *AuthMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add permission checking logic here
			next.ServeHTTP(w, r)
		})
	}
}
```

### middleware/validation.go
```go
package middleware

import (
	"encoding/json"
	"net/http"

	"go-falcon/internal/modulename/dto"
	"go-falcon/pkg/handlers"

	"github.com/go-playground/validator/v10"
)

type ValidationMiddleware struct {
	validator *validator.Validate
}

func NewValidationMiddleware() *ValidationMiddleware {
	validate := validator.New()
	dto.RegisterCustomValidators(validate)
	
	return &ValidationMiddleware{
		validator: validate,
	}
}

// ValidateJSON validates JSON request body against the provided struct
func (m *ValidationMiddleware) ValidateJSON(target interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(target); err != nil {
				handlers.ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if errors := dto.ValidateStruct(m.validator, target); len(errors) > 0 {
				handlers.ErrorResponse(w, "Validation failed", http.StatusBadRequest, map[string]interface{}{
					"validation_errors": errors,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

### routes/routes.go
```go
package routes

import (
	"go-falcon/internal/modulename/middleware"
	"go-falcon/internal/modulename/services"

	"github.com/go-chi/chi/v5"
)

type Routes struct {
	service    *services.Service
	middleware *middleware.Middleware
}

func New(service *services.Service, middleware *middleware.Middleware) *Routes {
	return &Routes{
		service:    service,
		middleware: middleware,
	}
}

// RegisterRoutes registers all routes for this module
func (r *Routes) RegisterRoutes(router chi.Router) {
	// Health check endpoint
	router.Get("/health", r.service.HealthCheck)

	// Public routes
	router.Group(func(router chi.Router) {
		router.Get("/info", r.GetInfo)
	})

	// Protected routes
	router.Group(func(router chi.Router) {
		router.Use(r.middleware.RequireAuth)
		
		router.Get("/items", r.ListItems)
		router.Post("/items", r.CreateItem)
		router.Get("/items/{id}", r.GetItem)
		router.Put("/items/{id}", r.UpdateItem)
		router.Delete("/items/{id}", r.DeleteItem)
	})

	// Admin routes
	router.Group(func(router chi.Router) {
		router.Use(r.middleware.RequireAuth)
		router.Use(r.middleware.RequirePermission("items", "admin"))
		
		router.Post("/items/bulk", r.BulkCreateItems)
		router.Delete("/items/bulk", r.BulkDeleteItems)
	})
}
```

### routes/api.go
```go
package routes

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-falcon/internal/modulename/dto"
	"go-falcon/pkg/handlers"

	"github.com/go-chi/chi/v5"
)

// GetInfo returns module information
func (r *Routes) GetInfo(w http.ResponseWriter, req *http.Request) {
	info := map[string]interface{}{
		"module":  "modulename",
		"version": "1.0.0",
		"status":  "active",
	}
	
	handlers.JSONResponse(w, info, http.StatusOK)
}

// ListItems handles GET /items
func (r *Routes) ListItems(w http.ResponseWriter, req *http.Request) {
	// Parse query parameters
	page, _ := strconv.Atoi(req.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	
	pageSize, _ := strconv.Atoi(req.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	listReq := &dto.ListItemsRequest{
		Page:     page,
		PageSize: pageSize,
		Search:   req.URL.Query().Get("search"),
		SortBy:   req.URL.Query().Get("sort_by"),
		SortDir:  req.URL.Query().Get("sort_dir"),
	}

	result, err := r.service.ListItems(req.Context(), listReq)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to list items", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, result, http.StatusOK)
}

// CreateItem handles POST /items
func (r *Routes) CreateItem(w http.ResponseWriter, req *http.Request) {
	var createReq dto.CreateItemRequest
	if err := json.NewDecoder(req.Body).Decode(&createReq); err != nil {
		handlers.ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := r.service.CreateItem(req.Context(), &createReq)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to create item", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, result, http.StatusCreated)
}

// GetItem handles GET /items/{id}
func (r *Routes) GetItem(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if id == "" {
		handlers.ErrorResponse(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	item, err := r.service.GetItem(req.Context(), id)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to get item", http.StatusInternalServerError)
		return
	}

	if item == nil {
		handlers.ErrorResponse(w, "Item not found", http.StatusNotFound)
		return
	}

	handlers.JSONResponse(w, item, http.StatusOK)
}

// UpdateItem handles PUT /items/{id}
func (r *Routes) UpdateItem(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if id == "" {
		handlers.ErrorResponse(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	var updateReq dto.UpdateItemRequest
	if err := json.NewDecoder(req.Body).Decode(&updateReq); err != nil {
		handlers.ErrorResponse(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	item, err := r.service.UpdateItem(req.Context(), id, &updateReq)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to update item", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, item, http.StatusOK)
}

// DeleteItem handles DELETE /items/{id}
func (r *Routes) DeleteItem(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if id == "" {
		handlers.ErrorResponse(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	err := r.service.DeleteItem(req.Context(), id)
	if err != nil {
		handlers.ErrorResponse(w, "Failed to delete item", http.StatusInternalServerError)
		return
	}

	handlers.JSONResponse(w, map[string]string{"message": "Item deleted successfully"}, http.StatusOK)
}
```

### services/service.go
```go
package services

import (
	"context"
	"net/http"

	"go-falcon/internal/modulename/dto"
	"go-falcon/internal/modulename/models"
	"go-falcon/pkg/database"
	"go-falcon/pkg/handlers"
	"go-falcon/pkg/sde"
)

type Service struct {
	repository *Repository
	redis      *database.Redis
	sdeService sde.SDEService
}

func NewService(repository *Repository, redis *database.Redis, sdeService sde.SDEService) *Service {
	return &Service{
		repository: repository,
		redis:      redis,
		sdeService: sdeService,
	}
}

// HealthCheck handles health check requests
func (s *Service) HealthCheck(w http.ResponseWriter, r *http.Request) {
	handlers.HealthHandler("modulename")(w, r)
}

// ListItems retrieves a paginated list of items
func (s *Service) ListItems(ctx context.Context, req *dto.ListItemsRequest) (*dto.ListItemsResponse, error) {
	items, total, err := s.repository.FindItems(ctx, req)
	if err != nil {
		return nil, err
	}

	// Convert models to DTOs
	itemResponses := make([]dto.ItemResponse, len(items))
	for i, item := range items {
		itemResponses[i] = s.modelToDTO(item)
	}

	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &dto.ListItemsResponse{
		Items:      itemResponses,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// CreateItem creates a new item
func (s *Service) CreateItem(ctx context.Context, req *dto.CreateItemRequest) (*dto.CreateItemResponse, error) {
	item := &models.Item{
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
	}

	createdItem, err := s.repository.CreateItem(ctx, item)
	if err != nil {
		return nil, err
	}

	return &dto.CreateItemResponse{
		ID:        createdItem.ID,
		CreatedAt: createdItem.CreatedAt,
		Message:   "Item created successfully",
	}, nil
}

// GetItem retrieves an item by ID
func (s *Service) GetItem(ctx context.Context, id string) (*dto.ItemResponse, error) {
	item, err := s.repository.GetItem(ctx, id)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, nil
	}

	response := s.modelToDTO(*item)
	return &response, nil
}

// UpdateItem updates an existing item
func (s *Service) UpdateItem(ctx context.Context, id string, req *dto.UpdateItemRequest) (*dto.ItemResponse, error) {
	item, err := s.repository.UpdateItem(ctx, id, req)
	if err != nil {
		return nil, err
	}

	response := s.modelToDTO(*item)
	return &response, nil
}

// DeleteItem deletes an item by ID
func (s *Service) DeleteItem(ctx context.Context, id string) error {
	return s.repository.DeleteItem(ctx, id)
}

// modelToDTO converts a model to a DTO
func (s *Service) modelToDTO(item models.Item) dto.ItemResponse {
	return dto.ItemResponse{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Tags:        item.Tags,
		Metadata:    item.Metadata,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
		CreatedBy:   item.CreatedBy,
	}
}
```

### services/repository.go
```go
package services

import (
	"context"
	"time"

	"go-falcon/internal/modulename/dto"
	"go-falcon/internal/modulename/models"
	"go-falcon/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	mongodb *database.MongoDB
}

func NewRepository(mongodb *database.MongoDB) *Repository {
	return &Repository{
		mongodb: mongodb,
	}
}

// FindItems retrieves items with pagination and filtering
func (r *Repository) FindItems(ctx context.Context, req *dto.ListItemsRequest) ([]models.Item, int64, error) {
	collection := r.mongodb.Collection("modulename_items")

	// Build filter
	filter := bson.M{}
	if req.Search != "" {
		filter["$or"] = []bson.M{
			{"name": primitive.Regex{Pattern: req.Search, Options: "i"}},
			{"description": primitive.Regex{Pattern: req.Search, Options: "i"}},
		}
	}
	if len(req.Tags) > 0 {
		filter["tags"] = bson.M{"$in": req.Tags}
	}

	// Count total documents
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Build options
	opts := options.Find()
	opts.SetSkip(int64((req.Page - 1) * req.PageSize))
	opts.SetLimit(int64(req.PageSize))

	// Set sort order
	sortField := "created_at"
	if req.SortBy != "" {
		sortField = req.SortBy
	}
	sortOrder := 1
	if req.SortDir == "desc" {
		sortOrder = -1
	}
	opts.SetSort(bson.M{sortField: sortOrder})

	// Execute query
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var items []models.Item
	if err := cursor.All(ctx, &items); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// CreateItem creates a new item in the database
func (r *Repository) CreateItem(ctx context.Context, item *models.Item) (*models.Item, error) {
	collection := r.mongodb.Collection("modulename_items")

	// Set metadata
	now := time.Now()
	item.ID = primitive.NewObjectID().Hex()
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := collection.InsertOne(ctx, item)
	if err != nil {
		return nil, err
	}

	return item, nil
}

// GetItem retrieves an item by ID
func (r *Repository) GetItem(ctx context.Context, id string) (*models.Item, error) {
	collection := r.mongodb.Collection("modulename_items")

	var item models.Item
	err := collection.FindOne(ctx, bson.M{"id": id}).Decode(&item)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		return nil, err
	}

	return &item, nil
}

// UpdateItem updates an existing item
func (r *Repository) UpdateItem(ctx context.Context, id string, req *dto.UpdateItemRequest) (*models.Item, error) {
	collection := r.mongodb.Collection("modulename_items")

	// Build update document
	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	if req.Name != nil {
		update["$set"].(bson.M)["name"] = *req.Name
	}
	if req.Description != nil {
		update["$set"].(bson.M)["description"] = *req.Description
	}
	if req.Tags != nil {
		update["$set"].(bson.M)["tags"] = req.Tags
	}
	if req.Metadata != nil {
		update["$set"].(bson.M)["metadata"] = req.Metadata
	}

	// Update and return the updated document
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var item models.Item
	err := collection.FindOneAndUpdate(ctx, bson.M{"id": id}, update, opts).Decode(&item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// DeleteItem removes an item by ID
func (r *Repository) DeleteItem(ctx context.Context, id string) error {
	collection := r.mongodb.Collection("modulename_items")

	_, err := collection.DeleteOne(ctx, bson.M{"id": id})
	return err
}
```

### models/models.go
```go
package models

import "time"

// Item represents the database model for an item
type Item struct {
	ID          string            `bson:"id" json:"id"`
	Name        string            `bson:"name" json:"name"`
	Description string            `bson:"description" json:"description"`
	Tags        []string          `bson:"tags" json:"tags"`
	Metadata    map[string]string `bson:"metadata" json:"metadata"`
	CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `bson:"updated_at" json:"updated_at"`
	CreatedBy   string            `bson:"created_by" json:"created_by"`
}
```

## Implementation Guidelines

### 1. Naming Conventions
- **Package names**: Use the exact module name (e.g., `auth`, `scheduler`)
- **File names**: Use snake_case for multi-word concepts (e.g., `auth_service.go`)
- **Type names**: Use PascalCase (e.g., `AuthService`, `CreateUserRequest`)
- **Function names**: Use PascalCase for exported, camelCase for internal
- **Variable names**: Use camelCase

### 2. Import Organization
```go
import (
	// Standard library
	"context"
	"fmt"
	"net/http"

	// Internal packages (project)
	"go-falcon/internal/modulename/dto"
	"go-falcon/pkg/database"

	// External packages (third-party)
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
)
```

### 3. Error Handling
- Always return meaningful errors
- Use `pkg/handlers` for consistent HTTP error responses
- Log errors with appropriate context
- Never expose internal errors to API responses

### 4. Validation
- Use struct tags for basic validation
- Implement custom validators in `dto/validators.go`
- Validate at the DTO level, not in services
- Return user-friendly validation error messages

### 5. Testing
- Write unit tests for all services
- Create integration tests for routes
- Mock external dependencies
- Test validation rules thoroughly

### 6. Documentation
- Each module MUST have a comprehensive CLAUDE.md file
- Document all public APIs
- Include usage examples
- Document configuration requirements

## Migration Strategy

When refactoring existing modules:

1. **Create new structure alongside existing code**
2. **Migrate functionality piece by piece**
3. **Update imports as you go**
4. **Test thoroughly at each step**
5. **Remove old files only after everything works**
6. **Update documentation to reflect changes**

## Quality Standards

- **Code Coverage**: Minimum 80% for services and repositories
- **Linting**: All code must pass golangci-lint
- **Documentation**: All public functions must have godoc comments
- **Performance**: Database queries must be optimized with proper indexes
- **Security**: All inputs must be validated and sanitized