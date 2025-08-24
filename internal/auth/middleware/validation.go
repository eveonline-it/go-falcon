package middleware

import (
	"encoding/json"
	"net/http"

	"go-falcon/internal/auth/dto"
	"go-falcon/pkg/handlers"

	"github.com/go-playground/validator/v10"
)

// ValidationMiddleware provides request validation functionality
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

// ValidateJSON validates JSON request body against the provided struct
func (m *ValidationMiddleware) ValidateJSON(target interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(target); err != nil {
				handlers.BadRequestResponse(w, "Invalid JSON format")
				return
			}

			if errors := dto.ValidateStruct(m.validator, target); len(errors) > 0 {
				handlers.ValidationErrorResponse(w, errors)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateLoginRequest validates login request
// TODO: Implement when LoginRequest DTO is defined
// func (m *ValidationMiddleware) ValidateLoginRequest(next http.Handler) http.Handler {
// 	var req dto.LoginRequest
// 	return m.ValidateJSON(&req)(next)
// }

// ValidateRegisterRequest validates registration request
// TODO: Implement when RegisterRequest DTO is defined
// func (m *ValidationMiddleware) ValidateRegisterRequest(next http.Handler) http.Handler {
// 	var req dto.RegisterRequest
// 	return m.ValidateJSON(&req)(next)
// }

// ValidateEVETokenRequest validates EVE token exchange request
func (m *ValidationMiddleware) ValidateEVETokenRequest(next http.Handler) http.Handler {
	var req dto.EVETokenExchangeRequest
	return m.ValidateJSON(&req)(next)
}

// ValidateRefreshTokenRequest validates refresh token request
func (m *ValidationMiddleware) ValidateRefreshTokenRequest(next http.Handler) http.Handler {
	var req dto.RefreshTokenRequest
	return m.ValidateJSON(&req)(next)
}
