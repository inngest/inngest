package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestExamplesJSONStructure(t *testing.T) {
	// Path to examples file relative to this test
	examplesPath := filepath.Join("..", "..", "docs", "api_v2_examples.json")

	// Read the examples file
	data, err := os.ReadFile(examplesPath)
	if err != nil {
		// File doesn't exist - this is okay, we can skip validation
		t.Logf("Examples file at %s doesn't exist, skipping validation", examplesPath)
		return
	}

	// If file is empty, that's also okay
	if len(data) == 0 {
		t.Logf("Examples file at %s is empty, skipping validation", examplesPath)
		return
	}

	// Parse JSON with expected structure: path -> method -> statusCode -> example
	var examples map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(data, &examples); err != nil {
		t.Fatalf("Examples JSON is not valid JSON: %v", err)
	}

	// Validate structure
	for path, pathExamples := range examples {
		if pathExamples == nil {
			t.Errorf("Path '%s' has null value", path)
			continue
		}

		// Validate path starts with /
		if path == "" || path[0] != '/' {
			t.Errorf("Path '%s' should start with '/'", path)
		}

		for method, methodExamples := range pathExamples {
			if methodExamples == nil {
				t.Errorf("Path '%s' method '%s' has null value", path, method)
				continue
			}

			// Validate HTTP method
			validMethods := map[string]bool{
				"get": true, "post": true, "put": true, "patch": true, "delete": true,
				"head": true, "options": true,
			}
			if !validMethods[method] {
				t.Errorf("Path '%s' has invalid HTTP method '%s'", path, method)
			}

			for statusCode, example := range methodExamples {
				if example == nil {
					t.Errorf("Path '%s' method '%s' status '%s' has null value", path, method, statusCode)
					continue
				}

				// Validate status code format (3 digits)
				if len(statusCode) != 3 {
					t.Errorf("Path '%s' method '%s' has invalid status code '%s' (should be 3 digits)", path, method, statusCode)
					continue
				}

				// Check if it's a valid HTTP status code range
				firstDigit := statusCode[0]
				if firstDigit < '1' || firstDigit > '5' {
					t.Errorf("Path '%s' method '%s' has invalid status code '%s' (should be 1xx-5xx)", path, method, statusCode)
				}

				// Validate example is an object or non-empty string
				switch v := example.(type) {
				case map[string]interface{}:
					if len(v) == 0 {
						t.Errorf("Path '%s' method '%s' status '%s' example is empty - add example data", path, method, statusCode)
					}
				case string:
					if v == "" {
						t.Errorf("Path '%s' method '%s' status '%s' example is empty string - add example data", path, method, statusCode)
					}
				default:
					t.Errorf("Path '%s' method '%s' status '%s' example should be an object or string, got %T", path, method, statusCode, example)
				}
			}

			if len(methodExamples) == 0 {
				t.Errorf("Path '%s' method '%s' has no status code examples", path, method)
			}
		}

		if len(pathExamples) == 0 {
			t.Errorf("Path '%s' has no HTTP method examples", path)
		}
	}

	if len(examples) == 0 {
		t.Error("Examples file is empty - should contain at least one endpoint example")
	}
}

func TestExamplesMatchOpenAPISpec(t *testing.T) {
	// This test could be enhanced to actually load the OpenAPI spec and validate
	// that all examples correspond to real endpoints, but for now we'll do basic validation

	examplesPath := filepath.Join("..", "..", "docs", "api_v2_examples.json")
	data, err := os.ReadFile(examplesPath)
	if err != nil {
		t.Fatalf("Failed to read examples file: %v", err)
	}

	var examples map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(data, &examples); err != nil {
		t.Fatalf("Examples JSON is invalid: %v", err)
	}

	// Basic validation that common endpoints exist
	expectedPaths := []string{"/health", "/account", "/partner/accounts", "/envs"}

	for _, expectedPath := range expectedPaths {
		if _, exists := examples[expectedPath]; !exists {
			t.Errorf("Expected path '%s' not found in examples", expectedPath)
		}
	}

	// Validate that GET /health has required status codes
	if healthExamples, exists := examples["/health"]; exists {
		if getExamples, exists := healthExamples["get"]; exists {
			expectedStatusCodes := []string{"200", "401", "500"}
			for _, statusCode := range expectedStatusCodes {
				if _, exists := getExamples[statusCode]; !exists {
					t.Errorf("Expected status code '%s' not found for GET /health", statusCode)
				}
			}
		} else {
			t.Error("GET method not found for /health endpoint")
		}
	}
}

func TestShortenPublicEnumNames(t *testing.T) {
	v3Doc := &openapi3.T{
		Components: &openapi3.Components{
			Schemas: openapi3.Schemas{
				"v2FunctionRunStatus": {
					Value: &openapi3.Schema{
						Default: "FUNCTION_RUN_STATUS_UNSPECIFIED",
						Enum: []any{
							"FUNCTION_RUN_STATUS_UNSPECIFIED",
							"FUNCTION_RUN_STATUS_QUEUED",
							"FUNCTION_RUN_STATUS_CANCELLED",
						},
					},
				},
				"v2TraceSpanStatus": {
					Value: &openapi3.Schema{
						Default: "TRACE_SPAN_STATUS_UNKNOWN",
						Enum: []any{
							"TRACE_SPAN_STATUS_UNKNOWN",
							"TRACE_SPAN_STATUS_COMPLETED",
						},
					},
				},
				"v2TraceStepOp": {
					Value: &openapi3.Schema{
						Default: "TRACE_STEP_OP_UNSPECIFIED",
						Enum: []any{
							"TRACE_STEP_OP_UNSPECIFIED",
							"TRACE_STEP_OP_SEND_EVENT",
						},
					},
				},
				"TraceSpan": {
					Value: &openapi3.Schema{
						Properties: openapi3.Schemas{
							"status": {
								Value: &openapi3.Schema{
									Default: "TRACE_SPAN_STATUS_UNKNOWN",
									Enum: []any{
										"TRACE_SPAN_STATUS_UNKNOWN",
										"TRACE_SPAN_STATUS_COMPLETED",
									},
								},
							},
							"stepOp": {
								Value: &openapi3.Schema{
									Default: "TRACE_STEP_OP_UNSPECIFIED",
									Enum: []any{
										"TRACE_STEP_OP_UNSPECIFIED",
										"TRACE_STEP_OP_SEND_EVENT",
									},
								},
							},
						},
					},
				},
				"v2EnvType": {
					Value: &openapi3.Schema{
						Default: "PRODUCTION",
						Enum:    []any{"PRODUCTION", "TEST", "BRANCH"},
					},
				},
			},
		},
		Paths: openapi3.NewPaths(openapi3.WithPath("/runs/{runId}", &openapi3.PathItem{
			Parameters: openapi3.Parameters{
				{
					Value: &openapi3.Parameter{
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{
								Enum: []any{"FUNCTION_RUN_STATUS_FAILED"},
							},
						},
					},
				},
			},
			Get: &openapi3.Operation{
				Parameters: openapi3.Parameters{
					{
						Value: &openapi3.Parameter{
							Content: openapi3.Content{
								"application/json": {
									Schema: &openapi3.SchemaRef{
										Value: &openapi3.Schema{
											Enum: []any{"TRACE_STEP_OP_WAIT_FOR_EVENT"},
										},
									},
								},
							},
						},
					},
				},
				RequestBody: &openapi3.RequestBodyRef{
					Value: &openapi3.RequestBody{
						Content: openapi3.Content{
							"application/json": {
								Schema: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Enum: []any{"TRACE_SPAN_STATUS_WAITING"},
									},
								},
							},
						},
					},
				},
				Responses: openapi3.NewResponses(openapi3.WithStatus(200, &openapi3.ResponseRef{
					Value: &openapi3.Response{
						Content: openapi3.Content{
							"application/json": {
								Schema: &openapi3.SchemaRef{
									Value: &openapi3.Schema{
										Enum: []any{"TRACE_SPAN_STATUS_FAILED"},
									},
								},
							},
						},
					},
				})),
			},
		})),
	}

	shortenPublicEnumNames(v3Doc)

	assertEqual(t, "UNSPECIFIED", v3Doc.Components.Schemas["v2FunctionRunStatus"].Value.Default)
	assertEqual(t, []any{"UNSPECIFIED", "QUEUED", "CANCELLED"}, v3Doc.Components.Schemas["v2FunctionRunStatus"].Value.Enum)
	assertEqual(t, "UNKNOWN", v3Doc.Components.Schemas["v2TraceSpanStatus"].Value.Default)
	assertEqual(t, []any{"UNKNOWN", "COMPLETED"}, v3Doc.Components.Schemas["v2TraceSpanStatus"].Value.Enum)
	assertEqual(t, "UNSPECIFIED", v3Doc.Components.Schemas["v2TraceStepOp"].Value.Default)
	assertEqual(t, []any{"UNSPECIFIED", "SEND_EVENT"}, v3Doc.Components.Schemas["v2TraceStepOp"].Value.Enum)
	assertEqual(t, "UNKNOWN", v3Doc.Components.Schemas["TraceSpan"].Value.Properties["status"].Value.Default)
	assertEqual(t, []any{"UNKNOWN", "COMPLETED"}, v3Doc.Components.Schemas["TraceSpan"].Value.Properties["status"].Value.Enum)
	assertEqual(t, "UNSPECIFIED", v3Doc.Components.Schemas["TraceSpan"].Value.Properties["stepOp"].Value.Default)
	assertEqual(t, []any{"UNSPECIFIED", "SEND_EVENT"}, v3Doc.Components.Schemas["TraceSpan"].Value.Properties["stepOp"].Value.Enum)
	assertEqual(t, "PRODUCTION", v3Doc.Components.Schemas["v2EnvType"].Value.Default)
	assertEqual(t, []any{"PRODUCTION", "TEST", "BRANCH"}, v3Doc.Components.Schemas["v2EnvType"].Value.Enum)

	pathItem := v3Doc.Paths.Find("/runs/{runId}")
	assertEqual(t, []any{"FAILED"}, pathItem.Parameters[0].Value.Schema.Value.Enum)
	assertEqual(t, []any{"WAIT_FOR_EVENT"}, pathItem.Get.Parameters[0].Value.Content["application/json"].Schema.Value.Enum)
	assertEqual(t, []any{"WAITING"}, pathItem.Get.RequestBody.Value.Content["application/json"].Schema.Value.Enum)
	assertEqual(t, []any{"FAILED"}, pathItem.Get.Responses.Value("200").Value.Content["application/json"].Schema.Value.Enum)
}

func TestDerivePublicEnumPrefixes(t *testing.T) {
	prefixes := derivePublicEnumPrefixes(&openapi3.Components{
		Schemas: openapi3.Schemas{
			"v2FunctionRunStatus": {
				Value: &openapi3.Schema{
					Enum: []any{"FUNCTION_RUN_STATUS_QUEUED"},
				},
			},
			"v2AIGatewayStatus": {
				Value: &openapi3.Schema{
					Enum: []any{"AI_GATEWAY_STATUS_COMPLETED"},
				},
			},
			"v2EnvType": {
				Value: &openapi3.Schema{
					Enum: []any{"PRODUCTION"},
				},
			},
			"protobufNullValue": {
				Value: &openapi3.Schema{
					Enum: []any{"NULL_VALUE"},
				},
			},
		},
	})

	assertEqual(t, []string{"FUNCTION_RUN_STATUS_", "AI_GATEWAY_STATUS_"}, prefixes)
}

func assertEqual(t *testing.T, expected any, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("expected %#v, got %#v", expected, actual)
	}
}
