package apiv2base

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ValidateJSONForProto validates JSON data against expected protobuf field types
// and checks for required fields
func ValidateJSONForProto(jsonData []byte, msg proto.Message) error {
	if len(jsonData) == 0 {
		return NewError(http.StatusBadRequest, ErrorInvalidRequest, "Request body is required")
	}

	// Parse JSON into generic map
	var data map[string]interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return NewError(http.StatusBadRequest, ErrorInvalidRequest, fmt.Sprintf("Invalid JSON: %v", err))
	}

	// Get protobuf message descriptor to understand expected types
	msgDesc := msg.ProtoReflect().Descriptor()
	fields := msgDesc.Fields()

	var errors []ErrorItem

	// First, check for required fields that are missing
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.Name())

		// Skip optional fields
		if field.HasOptionalKeyword() {
			continue
		}

		// Check if required field is present in JSON
		jsonName := field.JSONName()
		if jsonName == "" {
			jsonName = fieldName
		}

		if _, exists := data[jsonName]; !exists {
			// Also check snake_case version of field name
			if _, exists := data[fieldName]; !exists {
				errors = append(errors, ErrorItem{
					Code:    ErrorMissingField,
					Message: fmt.Sprintf("Field '%s' is required", fieldName),
				})
			}
		}
	}

	// Then, check each field in the JSON against expected protobuf types
	for key, value := range data {
		// Find the corresponding protobuf field
		field := fields.ByJSONName(key)
		if field == nil {
			field = fields.ByName(protoreflect.Name(key))
		}

		if field == nil {
			// Unknown field - could warn but not error for now
			continue
		}

		// Validate the value type matches the expected protobuf type
		if err := validateJSONFieldType(string(field.Name()), value, field); err != nil {
			errors = append(errors, *err)
		}

		// For required string fields, also check they're not empty
		if !field.HasOptionalKeyword() && field.Kind() == protoreflect.StringKind {
			if str, ok := value.(string); ok && str == "" {
				errors = append(errors, ErrorItem{
					Code:    ErrorMissingField,
					Message: fmt.Sprintf("Field '%s' is required", string(field.Name())),
				})
			}
		}
	}

	if len(errors) > 0 {
		return NewErrors(http.StatusBadRequest, errors...)
	}

	return nil
}

// validateJSONFieldType checks if a JSON value matches the expected protobuf field type
func validateJSONFieldType(fieldName string, value interface{}, field protoreflect.FieldDescriptor) *ErrorItem {
	switch field.Kind() {
	case protoreflect.StringKind:
		if _, ok := value.(string); !ok {
			return &ErrorItem{
				Code:    ErrorInvalidFieldFormat,
				Message: fmt.Sprintf("Field '%s' must be a string, got %T", fieldName, value),
			}
		}

	case protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		// JSON numbers can be int or float64, but should be whole numbers for integer fields
		switch v := value.(type) {
		case float64:
			// Check if it's a whole number
			if v != float64(int64(v)) {
				return &ErrorItem{
					Code:    ErrorInvalidFieldFormat,
					Message: fmt.Sprintf("Field '%s' must be a whole number, got %v", fieldName, v),
				}
			}
		case int, int32, int64:
			// These are fine
		default:
			return &ErrorItem{
				Code:    ErrorInvalidFieldFormat,
				Message: fmt.Sprintf("Field '%s' must be a number, got %T", fieldName, value),
			}
		}

	case protoreflect.FloatKind, protoreflect.DoubleKind:
		if _, ok := value.(float64); !ok {
			// Also accept integers for float fields
			if _, ok := value.(int); !ok {
				return &ErrorItem{
					Code:    ErrorInvalidFieldFormat,
					Message: fmt.Sprintf("Field '%s' must be a number, got %T", fieldName, value),
				}
			}
		}

	case protoreflect.BoolKind:
		if _, ok := value.(bool); !ok {
			return &ErrorItem{
				Code:    ErrorInvalidFieldFormat,
				Message: fmt.Sprintf("Field '%s' must be a boolean, got %T", fieldName, value),
			}
		}

	case protoreflect.EnumKind:
		// Enums can be strings (enum name) or numbers (enum value)
		switch value.(type) {
		case string, float64, int, int32, int64:
			// These are acceptable for enums
		default:
			return &ErrorItem{
				Code:    ErrorInvalidFieldFormat,
				Message: fmt.Sprintf("Field '%s' must be a string or number for enum, got %T", fieldName, value),
			}
		}

	case protoreflect.BytesKind:
		// Bytes can be base64 strings or byte arrays
		switch value.(type) {
		case string:
			// Base64 encoded bytes - could add actual base64 validation here
		case []byte:
			// Raw bytes (less common in JSON)
		case []interface{}:
			// Array of numbers representing bytes
			for _, item := range value.([]interface{}) {
				if num, ok := item.(float64); ok {
					if num < 0 || num > 255 || num != float64(int(num)) {
						return &ErrorItem{
							Code:    ErrorInvalidFieldFormat,
							Message: fmt.Sprintf("Field '%s' contains invalid byte value %v (must be 0-255)", fieldName, num),
						}
					}
				} else {
					return &ErrorItem{
						Code:    ErrorInvalidFieldFormat,
						Message: fmt.Sprintf("Field '%s' byte array must contain only numbers", fieldName),
					}
				}
			}
		default:
			return &ErrorItem{
				Code:    ErrorInvalidFieldFormat,
				Message: fmt.Sprintf("Field '%s' must be a string (base64) or array for bytes, got %T", fieldName, value),
			}
		}

	case protoreflect.MessageKind:
		// Nested messages should be objects (maps)
		if _, ok := value.(map[string]interface{}); !ok {
			return &ErrorItem{
				Code:    ErrorInvalidFieldFormat,
				Message: fmt.Sprintf("Field '%s' must be an object, got %T", fieldName, value),
			}
		}

	case protoreflect.GroupKind:
		// Groups are deprecated but still supported - treat like messages
		if _, ok := value.(map[string]interface{}); !ok {
			return &ErrorItem{
				Code:    ErrorInvalidFieldFormat,
				Message: fmt.Sprintf("Field '%s' must be an object, got %T", fieldName, value),
			}
		}
	}

	// Handle repeated (array) fields
	if field.IsList() {
		array, ok := value.([]interface{})
		if !ok {
			return &ErrorItem{
				Code:    ErrorInvalidFieldFormat,
				Message: fmt.Sprintf("Field '%s' must be an array, got %T", fieldName, value),
			}
		}

		// Validate each item in the array matches the field type
		for i, item := range array {
			// Create a temporary field descriptor for the list element
			// Note: This is a simplified approach - in a complete implementation
			// you'd need to validate each array element against the field's element type
			if field.Kind() == protoreflect.StringKind {
				if _, ok := item.(string); !ok {
					return &ErrorItem{
						Code:    ErrorInvalidFieldFormat,
						Message: fmt.Sprintf("Field '%s'[%d] must be a string, got %T", fieldName, i, item),
					}
				}
			}
			// Add more array element type checking as needed
		}
	}

	// Handle map fields
	if field.IsMap() {
		mapValue, ok := value.(map[string]interface{})
		if !ok {
			return &ErrorItem{
				Code:    ErrorInvalidFieldFormat,
				Message: fmt.Sprintf("Field '%s' must be an object (map), got %T", fieldName, value),
			}
		}

		// Validate map keys and values against their expected types
		// This would require more complex logic to check the map's key/value types
		_ = mapValue // For now, just accept any valid JSON object
	}

	return nil
}

// JSONTypeValidationMiddleware creates HTTP middleware that validates JSON types before grpc-gateway processing
func JSONTypeValidationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip validation for GET requests and requests without body
			if r.Method == http.MethodGet || r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Read the request body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeHTTPError(w, http.StatusBadRequest, ErrorInvalidRequest, "Failed to read request body")
				return
			}
			r.Body.Close()

			// Restore the body for downstream handlers
			r.Body = io.NopCloser(bytes.NewReader(body))

			// Get the protobuf message type for this endpoint
			protoMsg := getProtoMessageForPath(r.URL.Path, r.Method)
			if protoMsg == nil {
				// No validation configured for this path, continue
				next.ServeHTTP(w, r)
				return
			}

			// Validate JSON types against protobuf schema
			if err := ValidateJSONForProto(body, protoMsg); err != nil {
				// Extract error details and write HTTP response
				if apiErr, ok := extractAPIError(err); ok {
					writeHTTPErrors(w, apiErr.statusCode, apiErr.errors...)
				} else {
					writeHTTPError(w, http.StatusBadRequest, ErrorInvalidRequest, err.Error())
				}
				return
			}

			// Continue to the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// getProtoMessageForPath returns the appropriate protobuf message type for a given path and method
// This function dynamically discovers all endpoints from the V2 service protobuf definition
func getProtoMessageForPath(path, method string) proto.Message {
	// Get the V2 service descriptor
	serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if serviceDesc == nil {
		return nil
	}

	// Iterate through all methods in the V2 service
	methods := serviceDesc.Methods()
	for i := 0; i < methods.Len(); i++ {
		methodDesc := methods.Get(i)

		// Get the HTTP method and path from protobuf annotations
		httpMethod := getHTTPMethod(methodDesc)
		httpPath := getHTTPPath(methodDesc)

		// Skip if no HTTP annotations or path doesn't match
		if httpPath == "" {
			continue
		}

		// Add /api/v2 prefix to the path for comparison
		fullPath := "/api/v2" + httpPath

		// Check if this matches the requested path and method
		if matchesHTTPPath(fullPath, path) && httpMethod == method {
			// Get the input message type for this method
			inputDesc := methodDesc.Input()

			// Create a new instance of the input message
			inputMsg := createMessageFromDescriptor(inputDesc)
			if inputMsg != nil {
				return inputMsg
			}
		}
	}

	return nil
}

// matchesHTTPPath checks if the given paths match, handling path parameters
func matchesHTTPPath(templatePath, requestPath string) bool {
	// For exact matches (no path parameters)
	if templatePath == requestPath {
		return true
	}

	// TODO: Add path parameter matching logic here if needed
	// For now, just do exact matching
	return false
}

// createMessageFromDescriptor creates a new instance of a message type from its descriptor
// This uses dynamicpb to create message instances without hardcoding types
func createMessageFromDescriptor(desc protoreflect.MessageDescriptor) proto.Message {
	// Use dynamicpb to create a dynamic message instance
	return dynamicpb.NewMessage(desc)
}

// apiError represents an internal error structure for passing error details
type apiError struct {
	statusCode int
	errors     []ErrorItem
}

// extractAPIError attempts to extract error details from a validation error
func extractAPIError(err error) (*apiError, bool) {
	errMsg := err.Error()

	// Check if the error message contains our JSON error format
	if strings.Contains(errMsg, `{"errors":`) {
		// Extract the JSON portion
		startIdx := strings.Index(errMsg, `{"errors":`)
		if startIdx >= 0 {
			jsonStr := errMsg[startIdx:]

			var errResp ErrorResponse
			if json.Unmarshal([]byte(jsonStr), &errResp) == nil {
				return &apiError{
					statusCode: http.StatusBadRequest,
					errors:     errResp.Errors,
				}, true
			}
		}
	}

	return nil, false
}

// writeHTTPError writes a single error response in the v2 API format
func writeHTTPError(w http.ResponseWriter, statusCode int, errorCode, message string) {
	writeHTTPErrors(w, statusCode, ErrorItem{Code: errorCode, Message: message})
}

// writeHTTPErrors writes multiple errors in the v2 API format
func writeHTTPErrors(w http.ResponseWriter, statusCode int, errors ...ErrorItem) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Errors: errors,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback error response
		fallback := `{"errors":[{"code":"internal_error","message":"Failed to encode error response"}]}`
		_, _ = w.Write([]byte(fallback))
	}
}
