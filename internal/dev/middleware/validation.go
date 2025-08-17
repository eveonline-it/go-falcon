package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go-falcon/internal/dev/dto"
	"go-falcon/pkg/handlers"

	"github.com/go-playground/validator/v10"
	"github.com/go-chi/chi/v5"
)

// ValidationMiddleware provides request validation for Dev endpoints
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

// ValidateCharacterID validates character ID from URL parameters
func (m *ValidationMiddleware) ValidateCharacterID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		charIDStr := chi.URLParam(r, "characterID")
		if charIDStr == "" {
			charIDStr = chi.URLParam(r, "character_id")
		}
		
		if charIDStr == "" {
			handlers.ErrorResponse(w, "Character ID is required", http.StatusBadRequest)
			return
		}
		
		charID, err := strconv.Atoi(charIDStr)
		if err != nil {
			handlers.ErrorResponse(w, "Invalid character ID format", http.StatusBadRequest)
			return
		}
		
		if !dto.ValidateEVEID(charID, "character") {
			handlers.ErrorResponse(w, "Invalid character ID range", http.StatusBadRequest)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateAllianceID validates alliance ID from URL parameters
func (m *ValidationMiddleware) ValidateAllianceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allianceIDStr := chi.URLParam(r, "allianceID")
		if allianceIDStr == "" {
			allianceIDStr = chi.URLParam(r, "alliance_id")
		}
		
		if allianceIDStr == "" {
			handlers.ErrorResponse(w, "Alliance ID is required", http.StatusBadRequest)
			return
		}
		
		allianceID, err := strconv.Atoi(allianceIDStr)
		if err != nil {
			handlers.ErrorResponse(w, "Invalid alliance ID format", http.StatusBadRequest)
			return
		}
		
		if !dto.ValidateEVEID(allianceID, "alliance") {
			handlers.ErrorResponse(w, "Invalid alliance ID range", http.StatusBadRequest)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateCorporationID validates corporation ID from URL parameters
func (m *ValidationMiddleware) ValidateCorporationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpIDStr := chi.URLParam(r, "corporationID")
		if corpIDStr == "" {
			corpIDStr = chi.URLParam(r, "corporation_id")
		}
		
		if corpIDStr == "" {
			handlers.ErrorResponse(w, "Corporation ID is required", http.StatusBadRequest)
			return
		}
		
		corpID, err := strconv.Atoi(corpIDStr)
		if err != nil {
			handlers.ErrorResponse(w, "Invalid corporation ID format", http.StatusBadRequest)
			return
		}
		
		if !dto.ValidateEVEID(corpID, "corporation") {
			handlers.ErrorResponse(w, "Invalid corporation ID range", http.StatusBadRequest)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateSystemID validates system ID from URL parameters
func (m *ValidationMiddleware) ValidateSystemID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		systemIDStr := chi.URLParam(r, "systemID")
		if systemIDStr == "" {
			systemIDStr = chi.URLParam(r, "system_id")
		}
		
		if systemIDStr == "" {
			handlers.ErrorResponse(w, "System ID is required", http.StatusBadRequest)
			return
		}
		
		systemID, err := strconv.Atoi(systemIDStr)
		if err != nil {
			handlers.ErrorResponse(w, "Invalid system ID format", http.StatusBadRequest)
			return
		}
		
		if !dto.ValidateEVEID(systemID, "system") {
			handlers.ErrorResponse(w, "Invalid system ID range", http.StatusBadRequest)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateTypeID validates type ID from URL parameters
func (m *ValidationMiddleware) ValidateTypeID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		typeIDStr := chi.URLParam(r, "typeID")
		if typeIDStr == "" {
			typeIDStr = chi.URLParam(r, "type_id")
		}
		
		if typeIDStr == "" {
			handlers.ErrorResponse(w, "Type ID is required", http.StatusBadRequest)
			return
		}
		
		typeID, err := strconv.Atoi(typeIDStr)
		if err != nil {
			handlers.ErrorResponse(w, "Invalid type ID format", http.StatusBadRequest)
			return
		}
		
		if !dto.ValidateEVEID(typeID, "type") {
			handlers.ErrorResponse(w, "Invalid type ID range", http.StatusBadRequest)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateUniversePath validates universe path parameters
func (m *ValidationMiddleware) ValidateUniversePath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		universeType := chi.URLParam(r, "type")
		region := chi.URLParam(r, "region")
		constellation := chi.URLParam(r, "constellation")
		system := chi.URLParam(r, "system")
		
		if !dto.ValidateUniversePath(universeType, region, constellation, system) {
			handlers.ErrorResponse(w, "Invalid universe path components", http.StatusBadRequest)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateSDEEntityType validates SDE entity type from URL parameters
func (m *ValidationMiddleware) ValidateSDEEntityType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entityType := chi.URLParam(r, "type")
		if entityType == "" {
			entityType = chi.URLParam(r, "entity_type")
		}
		
		if entityType == "" {
			handlers.ErrorResponse(w, "Entity type is required", http.StatusBadRequest)
			return
		}
		
		// Create a temporary request to validate
		req := dto.SDEEntityRequest{Type: entityType, ID: "1"}
		if err := m.validator.Struct(req); err != nil {
			handlers.ErrorResponse(w, "Invalid entity type", http.StatusBadRequest)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateTestRequest validates test request bodies
func (m *ValidationMiddleware) ValidateTestRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != "PUT" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Determine test type from URL path
		path := strings.ToLower(r.URL.Path)
		
		switch {
		case strings.Contains(path, "/validate"):
			var req dto.ValidationTestRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				handlers.ErrorResponse(w, "Invalid JSON format", http.StatusBadRequest)
				return
			}
			
			if err := m.validator.Struct(req); err != nil {
				// Convert validator errors to string slice
				var errors []string
				if validationErrors, ok := err.(validator.ValidationErrors); ok {
					for _, fieldError := range validationErrors {
						errors = append(errors, fmt.Sprintf("Field '%s' failed validation: %s", 
							fieldError.Field(), fieldError.Tag()))
					}
				} else {
					errors = append(errors, err.Error())
				}
				handlers.ValidationErrorResponse(w, errors)
				return
			}
			
		case strings.Contains(path, "/performance"):
			var req dto.PerformanceTestRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				handlers.ErrorResponse(w, "Invalid JSON format", http.StatusBadRequest)
				return
			}
			
			if err := m.validator.Struct(req); err != nil {
				// Convert validator errors to string slice
				var errors []string
				if validationErrors, ok := err.(validator.ValidationErrors); ok {
					for _, fieldError := range validationErrors {
						errors = append(errors, fmt.Sprintf("Field '%s' failed validation: %s", 
							fieldError.Field(), fieldError.Tag()))
					}
				} else {
					errors = append(errors, err.Error())
				}
				handlers.ValidationErrorResponse(w, errors)
				return
			}
			
		case strings.Contains(path, "/cache"):
			var req dto.CacheTestRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				handlers.ErrorResponse(w, "Invalid JSON format", http.StatusBadRequest)
				return
			}
			
			if err := m.validator.Struct(req); err != nil {
				// Convert validator errors to string slice
				var errors []string
				if validationErrors, ok := err.(validator.ValidationErrors); ok {
					for _, fieldError := range validationErrors {
						errors = append(errors, fmt.Sprintf("Field '%s' failed validation: %s", 
							fieldError.Field(), fieldError.Tag()))
					}
				} else {
					errors = append(errors, err.Error())
				}
				handlers.ValidationErrorResponse(w, errors)
				return
			}
			
		case strings.Contains(path, "/mock"):
			var req dto.MockDataRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				handlers.ErrorResponse(w, "Invalid JSON format", http.StatusBadRequest)
				return
			}
			
			if err := m.validator.Struct(req); err != nil {
				// Convert validator errors to string slice
				var errors []string
				if validationErrors, ok := err.(validator.ValidationErrors); ok {
					for _, fieldError := range validationErrors {
						errors = append(errors, fmt.Sprintf("Field '%s' failed validation: %s", 
							fieldError.Field(), fieldError.Tag()))
					}
				} else {
					errors = append(errors, err.Error())
				}
				handlers.ValidationErrorResponse(w, errors)
				return
			}
			
		case strings.Contains(path, "/bulk"):
			var req dto.BulkTestRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				handlers.ErrorResponse(w, "Invalid JSON format", http.StatusBadRequest)
				return
			}
			
			if err := m.validator.Struct(req); err != nil {
				// Convert validator errors to string slice
				var errors []string
				if validationErrors, ok := err.(validator.ValidationErrors); ok {
					for _, fieldError := range validationErrors {
						errors = append(errors, fmt.Sprintf("Field '%s' failed validation: %s", 
							fieldError.Field(), fieldError.Tag()))
					}
				} else {
					errors = append(errors, err.Error())
				}
				handlers.ValidationErrorResponse(w, errors)
				return
			}
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateESITestRequest validates ESI test requests
func (m *ValidationMiddleware) ValidateESITestRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		
		var req dto.ESITestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handlers.ErrorResponse(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}
		
		if err := m.validator.Struct(req); err != nil {
			// Convert validator errors to string slice
			var errors []string
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				for _, fieldError := range validationErrors {
					errors = append(errors, fmt.Sprintf("Field '%s' failed validation: %s", 
						fieldError.Field(), fieldError.Tag()))
				}
			} else {
				errors = append(errors, err.Error())
			}
			handlers.ValidationErrorResponse(w, errors)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateAccessToken validates access tokens from headers
func (m *ValidationMiddleware) ValidateAccessToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for access token in Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// Extract token from "Bearer TOKEN" format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				token := parts[1]
				if !dto.ValidateAccessToken(token) {
					handlers.ErrorResponse(w, "Invalid access token format", http.StatusBadRequest)
					return
				}
			}
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateQueryParameters validates common query parameters
func (m *ValidationMiddleware) ValidateQueryParameters(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		
		// Validate page parameter
		if pageStr := query.Get("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err != nil || page < 1 {
				handlers.ErrorResponse(w, "Invalid page parameter", http.StatusBadRequest)
				return
			}
		}
		
		// Validate limit parameter
		if limitStr := query.Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err != nil || limit < 1 || limit > 1000 {
				handlers.ErrorResponse(w, "Invalid limit parameter (1-1000)", http.StatusBadRequest)
				return
			}
		}
		
		// Validate published parameter for SDE types
		if publishedStr := query.Get("published"); publishedStr != "" {
			if publishedStr != "true" && publishedStr != "false" {
				handlers.ErrorResponse(w, "Invalid published parameter (true/false)", http.StatusBadRequest)
				return
			}
		}
		
		// Validate detailed parameter for service discovery
		if detailedStr := query.Get("detailed"); detailedStr != "" {
			if detailedStr != "true" && detailedStr != "false" {
				handlers.ErrorResponse(w, "Invalid detailed parameter (true/false)", http.StatusBadRequest)
				return
			}
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateContentType validates Content-Type for POST/PUT requests
func (m *ValidationMiddleware) ValidateContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == "PUT" {
			contentType := r.Header.Get("Content-Type")
			if contentType != "" && !strings.Contains(contentType, "application/json") {
				handlers.ErrorResponse(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
				return
			}
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateRequestSize validates request body size
func (m *ValidationMiddleware) ValidateRequestSize(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == "PUT" {
				if r.ContentLength > maxSize {
					handlers.ErrorResponse(w, fmt.Sprintf("Request body too large (max %d bytes)", maxSize), http.StatusRequestEntityTooLarge)
					return
				}
				
				// Limit reader to prevent large payloads
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// ValidateUserAgent validates User-Agent header for ESI compliance
func (m *ValidationMiddleware) ValidateUserAgent(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			// Set default user agent if none provided
			r.Header.Set("User-Agent", "go-falcon-dev/1.0.0")
		}
		
		next.ServeHTTP(w, r)
	})
}

// ValidateRateLimit implements simple rate limiting validation
func (m *ValidationMiddleware) ValidateRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This is a placeholder for rate limiting logic
		// In a real implementation, you would check Redis or memory store
		// for rate limit counters per IP/user
		
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// Helper function to format validation errors
func formatValidationError(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, fieldError := range validationErrors {
			messages = append(messages, fmt.Sprintf("Field '%s' failed validation: %s", 
				fieldError.Field(), fieldError.Tag()))
		}
		return strings.Join(messages, "; ")
	}
	return err.Error()
}