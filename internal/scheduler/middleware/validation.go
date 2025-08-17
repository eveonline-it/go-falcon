package middleware

import (
	"encoding/json"
	"net/http"

	"go-falcon/internal/scheduler/dto"
	"go-falcon/pkg/handlers"

	"github.com/go-playground/validator/v10"
)

// ValidationMiddleware provides request validation for scheduler endpoints
type ValidationMiddleware struct {
	validator *validator.Validate
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware() *ValidationMiddleware {
	validate := validator.New()
	dto.RegisterCustomValidators(validate)
	
	return &ValidationMiddleware{
		validator: validate,
	}
}

// ValidateTaskCreateRequest validates task creation requests
func (m *ValidationMiddleware) ValidateTaskCreateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req dto.TaskCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := m.validator.Struct(&req); err != nil {
			handlers.ValidationErrorResponseFromError(w, err)
			return
		}

		// Store the validated request in context for the handler
		ctx := r.Context()
		ctx = handlers.WithValidatedRequest(ctx, &req)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateTaskUpdateRequest validates task update requests
func (m *ValidationMiddleware) ValidateTaskUpdateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req dto.TaskUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handlers.ErrorResponse(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := m.validator.Struct(&req); err != nil {
			handlers.ValidationErrorResponseFromError(w, err)
			return
		}

		// Store the validated request in context for the handler
		ctx := r.Context()
		ctx = handlers.WithValidatedRequest(ctx, &req)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateQueryParams validates query parameters
func (m *ValidationMiddleware) ValidateQueryParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse and validate query parameters
		query := dto.TaskListQuery{
			Page:     1,  // default page
			PageSize: 20, // default page size
		}

		// Parse page parameter
		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if page, err := handlers.ParseIntQuery(pageStr, 1); err == nil {
				query.Page = page
			}
		}

		// Parse page_size parameter
		if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
			if pageSize, err := handlers.ParseIntQuery(pageSizeStr, 20); err == nil {
				if pageSize > 100 {
					pageSize = 100 // enforce max limit
				}
				query.PageSize = pageSize
			}
		}

		// Parse other parameters
		query.Status = r.URL.Query().Get("status")
		query.Type = r.URL.Query().Get("type")
		query.Enabled = r.URL.Query().Get("enabled")
		
		// Parse tags
		if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
			query.Tags = handlers.ParseCommaSeparated(tagsStr)
		}

		// Validate the query
		if err := m.validator.Struct(&query); err != nil {
			handlers.ValidationErrorResponseFromError(w, err)
			return
		}

		// Store the validated query in context
		ctx := r.Context()
		ctx = handlers.WithValidatedQuery(ctx, &query)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}