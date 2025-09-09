package apiv2base

import (
	"encoding/json"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewSingleError(t *testing.T) {
	tests := []struct {
		name         string
		httpCode     int
		errorCode    string
		message      string
		wantGRPCCode codes.Code
		wantJSON     string
	}{
		{
			name:         "400 bad request error",
			httpCode:     http.StatusBadRequest,
			errorCode:    ErrorInvalidRequest,
			message:      "Invalid request format",
			wantGRPCCode: codes.InvalidArgument,
			wantJSON:     `{"errors":[{"code":"invalid_request","message":"Invalid request format"}]}`,
		},
		{
			name:         "401 unauthorized error",
			httpCode:     http.StatusUnauthorized,
			errorCode:    ErrorAuthorizationHeaderMissing,
			message:      "Authorization header is required",
			wantGRPCCode: codes.Unauthenticated,
			wantJSON:     `{"errors":[{"code":"authorization_header_missing","message":"Authorization header is required"}]}`,
		},
		{
			name:         "403 forbidden error",
			httpCode:     http.StatusForbidden,
			errorCode:    ErrorAccessDenied,
			message:      "Access denied",
			wantGRPCCode: codes.PermissionDenied,
			wantJSON:     `{"errors":[{"code":"access_denied","message":"Access denied"}]}`,
		},
		{
			name:         "409 conflict error",
			httpCode:     http.StatusConflict,
			errorCode:    ErrorResourceAlreadyExists,
			message:      "Resource already exists",
			wantGRPCCode: codes.AlreadyExists,
			wantJSON:     `{"errors":[{"code":"resource_already_exists","message":"Resource already exists"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.httpCode, tt.errorCode, tt.message)

			// Check that it's a gRPC status error
			grpcStatus, ok := status.FromError(err)
			if !ok {
				t.Fatal("Expected gRPC status error")
			}

			// Check gRPC code
			if grpcStatus.Code() != tt.wantGRPCCode {
				t.Errorf("Expected gRPC code %v, got %v", tt.wantGRPCCode, grpcStatus.Code())
			}

			// Check JSON message
			if grpcStatus.Message() != tt.wantJSON {
				t.Errorf("Expected JSON %v, got %v", tt.wantJSON, grpcStatus.Message())
			}

			// Verify it's valid JSON by unmarshaling
			var response ErrorResponse
			if err := json.Unmarshal([]byte(grpcStatus.Message()), &response); err != nil {
				t.Errorf("gRPC message is not valid JSON: %v", err)
			}

			if len(response.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(response.Errors))
			}

			if response.Errors[0].Code != tt.errorCode {
				t.Errorf("Expected error code %v, got %v", tt.errorCode, response.Errors[0].Code)
			}

			if response.Errors[0].Message != tt.message {
				t.Errorf("Expected message %v, got %v", tt.message, response.Errors[0].Message)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		err := NewErrors(http.StatusBadRequest, ErrorItem{Code: ErrorMissingField, Message: "Field required"})

		grpcStatus, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}

		if grpcStatus.Code() != codes.InvalidArgument {
			t.Errorf("Expected InvalidArgument, got %v", grpcStatus.Code())
		}

		var response ErrorResponse
		if err := json.Unmarshal([]byte(grpcStatus.Message()), &response); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if len(response.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(response.Errors))
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		err := NewErrors(http.StatusBadRequest,
			ErrorItem{Code: ErrorMissingField, Message: "Name required"},
			ErrorItem{Code: ErrorInvalidFieldFormat, Message: "Invalid timeout"},
		)

		grpcStatus, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}

		var response ErrorResponse
		if err := json.Unmarshal([]byte(grpcStatus.Message()), &response); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if len(response.Errors) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(response.Errors))
		}

		expectedJSON := `{"errors":[{"code":"missing_field","message":"Name required"},{"code":"invalid_field_format","message":"Invalid timeout"}]}`
		if grpcStatus.Message() != expectedJSON {
			t.Errorf("Expected %v, got %v", expectedJSON, grpcStatus.Message())
		}
	})

	t.Run("no errors provided", func(t *testing.T) {
		err := NewErrors(http.StatusBadRequest)

		grpcStatus, ok := status.FromError(err)
		if !ok {
			t.Fatal("Expected gRPC status error")
		}

		var response ErrorResponse
		if err := json.Unmarshal([]byte(grpcStatus.Message()), &response); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if len(response.Errors) != 1 {
			t.Errorf("Expected 1 default error, got %d", len(response.Errors))
		}

		if response.Errors[0].Code != ErrorInvalidRequest {
			t.Errorf("Expected default error code %v, got %v", ErrorInvalidRequest, response.Errors[0].Code)
		}
	})
}

func TestHTTPToGRPCStatus(t *testing.T) {
	tests := []struct {
		httpCode int
		grpcCode codes.Code
	}{
		{http.StatusBadRequest, codes.InvalidArgument},
		{http.StatusUnauthorized, codes.Unauthenticated},
		{http.StatusForbidden, codes.PermissionDenied},
		{http.StatusNotFound, codes.NotFound},
		{http.StatusConflict, codes.AlreadyExists},
		{http.StatusUnprocessableEntity, codes.InvalidArgument},
		{http.StatusTooManyRequests, codes.ResourceExhausted},
		{http.StatusInternalServerError, codes.Internal},
		{http.StatusNotImplemented, codes.Unimplemented},
		{http.StatusServiceUnavailable, codes.Unavailable},
		{999, codes.Internal}, // Unknown status should default to Internal
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.httpCode), func(t *testing.T) {
			result := httpToGRPCStatus(tt.httpCode)
			if result != tt.grpcCode {
				t.Errorf("httpToGRPCStatus(%d) = %v, want %v", tt.httpCode, result, tt.grpcCode)
			}
		})
	}
}

func TestErrorConstants(t *testing.T) {
	// Test that all constants follow the expected snake_case format
	expectedConstants := map[string]string{
		"invalid_request":              ErrorInvalidRequest,
		"missing_field":                ErrorMissingField,
		"invalid_field_format":         ErrorInvalidFieldFormat,
		"authorization_header_missing": ErrorAuthorizationHeaderMissing,
		"invalid_signing_key":          ErrorInvalidSigningKey,
		"access_denied":                ErrorAccessDenied,
		"resource_already_exists":      ErrorResourceAlreadyExists,
		"validation_error":             ErrorValidationError,
		"not_implemented":              ErrorNotImplemented,
	}

	for expected, actual := range expectedConstants {
		if actual != expected {
			t.Errorf("Error constant mismatch: got %v, want %v", actual, expected)
		}
	}
}

func TestGRPCGatewayIntegration(t *testing.T) {
	// Test that the error format is exactly what grpc-gateway expects
	err := NewError(http.StatusBadRequest, ErrorMissingField, "Field 'name' is required")

	grpcStatus, ok := status.FromError(err)
	if !ok {
		t.Fatal("Expected gRPC status error")
	}

	// The message should be valid JSON that grpc-gateway can parse
	message := grpcStatus.Message()

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(message), &result); err != nil {
		t.Fatalf("gRPC message is not valid JSON: %v", err)
	}

	// Verify structure matches API spec
	errors, exists := result["errors"]
	if !exists {
		t.Fatal("Missing 'errors' field")
	}

	errorSlice, ok := errors.([]interface{})
	if !ok {
		t.Fatal("'errors' field is not an array")
	}

	if len(errorSlice) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errorSlice))
	}

	errorObj, ok := errorSlice[0].(map[string]interface{})
	if !ok {
		t.Fatal("Error item is not an object")
	}

	// Verify only expected fields exist
	expectedFields := map[string]bool{"code": true, "message": true}
	for field := range errorObj {
		if !expectedFields[field] {
			t.Errorf("Unexpected field in error object: %s", field)
		}
	}
}
