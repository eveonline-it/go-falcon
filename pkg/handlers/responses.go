package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// StandardResponse represents a standard API response structure
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// JSONResponse sends a JSON response with the given data and status code
func JSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// SuccessResponse sends a successful JSON response
func SuccessResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	response := StandardResponse{
		Success: true,
		Data:    data,
	}
	JSONResponse(w, response, statusCode)
}

// ErrorResponse sends an error JSON response
func ErrorResponse(w http.ResponseWriter, message string, statusCode int, details ...interface{}) {
	response := StandardResponse{
		Success: false,
		Error:   http.StatusText(statusCode),
		Message: message,
	}

	if len(details) > 0 {
		response.Details = details[0]
	}

	JSONResponse(w, response, statusCode)
}

// ValidationErrorResponse sends a validation error response
func ValidationErrorResponse(w http.ResponseWriter, errors []string) {
	response := StandardResponse{
		Success: false,
		Error:   "Validation Failed",
		Message: "One or more validation errors occurred",
		Details: map[string]interface{}{
			"validation_errors": errors,
		},
	}
	JSONResponse(w, response, http.StatusBadRequest)
}

// MessageResponse sends a simple message response
func MessageResponse(w http.ResponseWriter, message string, statusCode int) {
	response := StandardResponse{
		Success: statusCode < 400,
		Message: message,
	}
	JSONResponse(w, response, statusCode)
}

// NotFoundResponse sends a 404 not found response
func NotFoundResponse(w http.ResponseWriter, resource string) {
	ErrorResponse(w, resource+" not found", http.StatusNotFound)
}

// UnauthorizedResponse sends a 401 unauthorized response
func UnauthorizedResponse(w http.ResponseWriter) {
	ErrorResponse(w, "Authentication required", http.StatusUnauthorized)
}

// ForbiddenResponse sends a 403 forbidden response
func ForbiddenResponse(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Access forbidden"
	}
	ErrorResponse(w, message, http.StatusForbidden)
}

// InternalErrorResponse sends a 500 internal server error response
func InternalErrorResponse(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Internal server error"
	}
	ErrorResponse(w, message, http.StatusInternalServerError)
}

// BadRequestResponse sends a 400 bad request response
func BadRequestResponse(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Bad request"
	}
	ErrorResponse(w, message, http.StatusBadRequest)
}

// CreatedResponse sends a 201 created response
func CreatedResponse(w http.ResponseWriter, data interface{}) {
	SuccessResponse(w, data, http.StatusCreated)
}

// NoContentResponse sends a 204 no content response
func NoContentResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
