package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExamplesJSONStructure(t *testing.T) {
	// Path to examples file relative to this test
	examplesPath := filepath.Join("..", "..", "docs", "api_v2_examples.json")
	
	// Read the examples file
	data, err := os.ReadFile(examplesPath)
	if err != nil {
		t.Fatalf("Failed to read examples file at %s: %v", examplesPath, err)
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
				
				// Validate example is an object (not a primitive)
				if exampleMap, ok := example.(map[string]interface{}); !ok {
					t.Errorf("Path '%s' method '%s' status '%s' example should be an object, got %T", path, method, statusCode, example)
				} else if len(exampleMap) == 0 {
					t.Errorf("Path '%s' method '%s' status '%s' example is empty - add example data", path, method, statusCode)
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