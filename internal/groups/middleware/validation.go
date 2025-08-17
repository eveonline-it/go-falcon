package middleware

import (
	"encoding/json"
	"net/http"

	"go-falcon/internal/groups/dto"
	"go-falcon/pkg/handlers"

	"github.com/go-playground/validator/v10"
)

// ValidationMiddleware provides request validation for groups endpoints
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

// ValidateGroupCreateRequest validates group creation requests
func (m *ValidationMiddleware) ValidateGroupCreateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req dto.GroupCreateRequest
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

// ValidateGroupUpdateRequest validates group update requests
func (m *ValidationMiddleware) ValidateGroupUpdateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req dto.GroupUpdateRequest
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

// ValidateMembershipRequest validates membership requests
func (m *ValidationMiddleware) ValidateMembershipRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req dto.MembershipRequest
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

// ValidateServiceCreateRequest validates service creation requests
func (m *ValidationMiddleware) ValidateServiceCreateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req dto.ServiceCreateRequest
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

// ValidateServiceUpdateRequest validates service update requests
func (m *ValidationMiddleware) ValidateServiceUpdateRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req dto.ServiceUpdateRequest
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

// ValidatePermissionAssignmentRequest validates permission assignment requests
func (m *ValidationMiddleware) ValidatePermissionAssignmentRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req dto.PermissionAssignmentRequest
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

// ValidatePermissionCheckRequest validates permission check requests
func (m *ValidationMiddleware) ValidatePermissionCheckRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req dto.PermissionCheckGranularRequest
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

// ValidateQueryParams validates query parameters for list endpoints
func (m *ValidationMiddleware) ValidateQueryParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse and validate query parameters based on the endpoint
		query := dto.GroupListQuery{
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
		if isDefaultStr := r.URL.Query().Get("is_default"); isDefaultStr != "" {
			if isDefaultStr == "true" {
				isDefault := true
				query.IsDefault = &isDefault
			} else if isDefaultStr == "false" {
				isDefault := false
				query.IsDefault = &isDefault
			}
		}

		query.Search = r.URL.Query().Get("search")
		query.ShowMembers = r.URL.Query().Get("show_members") == "true"

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