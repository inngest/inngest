package apiv2base

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Error codes
const (
	// 400 Bad Request errors
	ErrorInvalidRequest     = "invalid_request"
	ErrorMissingField       = "missing_field"
	ErrorInvalidFieldFormat = "invalid_field_format"

	// 401 Unauthorized errors
	ErrorAuthorizationHeaderMissing = "authorization_header_missing"
	ErrorInvalidSigningKey          = "invalid_signing_key"

	// 403 Forbidden errors
	ErrorAccessDenied = "access_denied"

	// 409 Conflict errors
	ErrorResourceAlreadyExists = "resource_already_exists"

	// 404 Not Found errors
	ErrorAccountNotFound = "account_not_found"
	ErrorEnvNotFound     = "env_not_found"

	// 422 Unprocessable Entity errors
	ErrorValidationError = "validation_error"

	// 500 Not Implemented errors
	ErrorInternalError = "internal_error"

	// 501 Not Implemented errors
	ErrorNotImplemented = "not_implemented"
)

// ErrorItem represents a single error in the API response
type ErrorItem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represents the standard API error response format
type ErrorResponse struct {
	Errors []ErrorItem `json:"errors"`
}

// NewErrors creates a gRPC error that will be properly formatted by grpc-gateway
// Takes one or more ErrorItem and returns a gRPC error
func NewErrors(httpCode int, errors ...ErrorItem) error {
	if len(errors) == 0 {
		errors = []ErrorItem{{Code: ErrorInvalidRequest, Message: "No error details provided"}}
	}

	response := ErrorResponse{
		Errors: errors,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		// Fallback error
		fallback := `{"errors":[{"code":"invalid_request","message":"Failed to format error response"}]}`
		jsonData = []byte(fallback)
	}

	// Create gRPC error with JSON in the message
	// Our custom error handler will extract and format this properly
	grpcCode := httpToGRPCStatus(httpCode)
	return status.Error(grpcCode, string(jsonData))
}

// NewSingleError creates a gRPC error for a single error condition
func NewError(httpCode int, errorCode, message string) error {
	return NewErrors(httpCode, ErrorItem{Code: errorCode, Message: message})
}

// httpToGRPCStatus maps HTTP status codes to gRPC status codes
func httpToGRPCStatus(httpCode int) codes.Code {
	switch httpCode {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusUnprocessableEntity:
		return codes.InvalidArgument
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

// CustomErrorHandler converts gRPC errors to our API error format
func CustomErrorHandler(base *Base) func(context.Context, *runtime.ServeMux, runtime.Marshaler, http.ResponseWriter, *http.Request, error) {
	return func(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
		const fallback = `{"errors":[{"code":"internal_server_error","message":"An unexpected error occurred"}]}`

		w.Header().Set("Content-Type", "application/json")

		// Extract gRPC status from error
		st, ok := status.FromError(err)
		if !ok {
			// Not a gRPC error, return 500
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(fallback))
			return
		}

		// Map gRPC codes to HTTP status codes
		httpCode := base.GRPCToHTTPStatus(st.Code())

		// Try to parse the error message as our error format
		message := st.Message()

		// If the message looks like our JSON format, use it directly
		if strings.HasPrefix(message, `{"errors":`) {
			w.WriteHeader(httpCode)
			_, _ = w.Write([]byte(message))
			return
		}

		// Otherwise, create a single error response
		errorResponse := ErrorResponse{
			Errors: []ErrorItem{
				{
					Code:    "api_error", // Generic code for non-structured errors
					Message: message,
				},
			},
		}

		jsonData, jsonErr := json.Marshal(errorResponse)
		if jsonErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(fallback))
			return
		}

		w.WriteHeader(httpCode)
		_, _ = w.Write(jsonData)
	}
}
