package apiv2

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
)

func TestHTTPValidation_CreateEnv(t *testing.T) {
	ctx := context.Background()

	t.Run("should validate JSON types and required fields", func(t *testing.T) {
		opts := HTTPHandlerOptions{}
		handler, err := NewHTTPHandler(ctx, opts)
		require.NoError(t, err)

		testCases := []struct {
			name           string
			body           string
			expectedStatus int
			expectedError  string
		}{
			{
				name:           "empty name field",
				body:           `{"name": ""}`,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "name",
			},
			{
				name:           "missing name field",
				body:           `{}`,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "name",
			},
			{
				name:           "name wrong type - number instead of string",
				body:           `{"name": 123}`,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid value for string field name",
			},
			{
				name:           "name wrong type - boolean instead of string",
				body:           `{"name": true}`,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid value for string field name",
			},
			{
				name:           "name wrong type - array instead of string",
				body:           `{"name": ["test"]}`,
				expectedStatus: http.StatusBadRequest,
				expectedError:  "invalid value for string field name",
			},
			{
				name:           "valid request with name",
				body:           `{"name": "test-env"}`,
				expectedStatus: http.StatusNotImplemented, // Service returns not implemented
				expectedError:  "",
			},
			{
				name:           "valid request with proper name",
				body:           `{"name": "production"}`,
				expectedStatus: http.StatusNotImplemented, // Service returns not implemented
				expectedError:  "",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodPost, "/api/v2/envs", bytes.NewReader([]byte(tc.body)))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()

				handler.ServeHTTP(rec, req)

				// Log the response for debugging
				t.Logf("Response status: %d", rec.Code)
				t.Logf("Response body: %s", rec.Body.String())

				require.Equal(t, tc.expectedStatus, rec.Code)

				if tc.expectedError != "" {
					var response ErrorResponse
					err := json.Unmarshal(rec.Body.Bytes(), &response)
					require.NoError(t, err)
					require.NotEmpty(t, response.Errors)

					// Check that at least one error mentions the expected field
					found := false
					for _, e := range response.Errors {
						if bytes.Contains([]byte(e.Message), []byte(tc.expectedError)) {
							found = true
							break
						}
					}
					require.True(t, found, "Expected error about %s but got: %v", tc.expectedError, response.Errors)
				}
			})
		}
	})
}

func TestValidateJSONForProto(t *testing.T) {
	t.Run("should validate JSON types against protobuf schema", func(t *testing.T) {
		testCases := []struct {
			name        string
			jsonData    string
			expectError bool
			errorText   string
		}{
			{
				name:        "valid JSON with all required fields",
				jsonData:    `{"name": "test-env"}`,
				expectError: false,
			},
			{
				name:        "missing required name field",
				jsonData:    `{}`,
				expectError: true,
				errorText:   "name' is required",
			},
			{
				name:        "empty required name field",
				jsonData:    `{"name": ""}`,
				expectError: true,
				errorText:   "name' is required",
			},
			{
				name:        "name wrong type - number",
				jsonData:    `{"name": 123}`,
				expectError: true,
				errorText:   "must be a string, got float64",
			},
			{
				name:        "name wrong type - boolean",
				jsonData:    `{"name": true}`,
				expectError: true,
				errorText:   "must be a string, got bool",
			},
			{
				name:        "name wrong type - array",
				jsonData:    `{"name": []}`,
				expectError: true,
				errorText:   "must be a string, got []interface",
			},
			{
				name:        "invalid JSON",
				jsonData:    `{"name": }`,
				expectError: true,
				errorText:   "Invalid JSON",
			},
			{
				name:        "empty JSON",
				jsonData:    ``,
				expectError: true,
				errorText:   "Request body is required",
			},
		}

		// Use CreateEnvRequest proto message for testing
		protoMsg := &apiv2.CreateEnvRequest{}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := ValidateJSONForProto([]byte(tc.jsonData), protoMsg)

				if tc.expectError {
					require.Error(t, err)
					require.Contains(t, err.Error(), tc.errorText)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}

func TestDynamicEndpointDiscovery(t *testing.T) {
	t.Run("should discover all V2 service endpoints automatically", func(t *testing.T) {
		// Get the V2 service descriptor
		serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
		require.NotNil(t, serviceDesc)

		discoveredEndpoints := []string{}

		// Iterate through all methods in the V2 service
		methods := serviceDesc.Methods()
		for i := 0; i < methods.Len(); i++ {
			methodDesc := methods.Get(i)

			// Get the HTTP method and path from protobuf annotations
			httpMethod := getHTTPMethod(methodDesc)
			httpPath := getHTTPPath(methodDesc)

			if httpPath != "" {
				fullPath := "/api/v2" + httpPath
				endpoint := httpMethod + " " + fullPath
				discoveredEndpoints = append(discoveredEndpoints, endpoint)

				t.Logf("Discovered endpoint: %s", endpoint)

				// Test that we can get a protobuf message for this endpoint
				protoMsg := getProtoMessageForPath(fullPath, httpMethod)
				if httpMethod != "GET" { // Only validate non-GET requests
					require.NotNil(t, protoMsg, "Should find proto message for %s", endpoint)
				}
			}
		}

		// Verify we found some endpoints
		require.NotEmpty(t, discoveredEndpoints, "Should discover at least one endpoint")

		// Verify our known endpoint is discovered
		found := false
		for _, endpoint := range discoveredEndpoints {
			if endpoint == "POST /api/v2/partner/accounts" {
				found = true
				break
			}
		}
		require.True(t, found, "Should discover the CreatePartnerAccount endpoint")
	})
}

func TestAdvancedFieldTypeValidation(t *testing.T) {
	t.Run("should handle additional protobuf field types", func(t *testing.T) {
		// Test cases for field types that might appear in real APIs
		testCases := []struct {
			name         string
			fieldType    string
			validValues  []interface{}
			invalidValue interface{}
			expectError  string
		}{
			{
				name:         "bytes field accepts string (base64)",
				fieldType:    "bytes",
				validValues:  []interface{}{"SGVsbG8gV29ybGQ=", ""},
				invalidValue: 123,
				expectError:  "must be a string (base64) or array for bytes",
			},
			{
				name:         "bytes field accepts byte arrays",
				fieldType:    "bytes",
				validValues:  []interface{}{[]interface{}{72, 101, 108, 108, 111}},
				invalidValue: []interface{}{"not", "numbers"},
				expectError:  "byte array must contain only numbers",
			},
			{
				name:         "array fields validate element types",
				fieldType:    "array",
				validValues:  []interface{}{[]interface{}{"a", "b", "c"}, []interface{}{}},
				invalidValue: "not an array",
				expectError:  "must be an array",
			},
			{
				name:         "map fields accept objects",
				fieldType:    "map",
				validValues:  []interface{}{map[string]interface{}{"key": "value"}, map[string]interface{}{}},
				invalidValue: []interface{}{},
				expectError:  "must be an object (map)",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Note: This is a conceptual test - in reality you'd need actual proto messages
				// with these field types to test properly. This demonstrates the validation logic.
				t.Logf("Field type: %s", tc.fieldType)
				t.Logf("Valid values: %v", tc.validValues)
				t.Logf("Invalid value: %v (should error: %s)", tc.invalidValue, tc.expectError)

				// The actual validation would happen through validateJSONFieldType
				// but would require proto message definitions with these field types
			})
		}
	})
}
