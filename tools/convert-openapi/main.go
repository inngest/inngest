// Package main provides a tool for converting OpenAPI v2 specifications to v3
// with enhanced example validation against protobuf schemas.
//
// Usage: convert-openapi <input-dir> <output-dir>
//
// This tool performs the following operations:
//   - Converts OpenAPI v2 files to v3 format
//   - Applies examples from docs/api_v2_examples.json
//   - Validates examples against proto schema definitions
//   - Handles schema reference resolution
//   - Generates missing example templates
package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage: convert-openapi <input-dir> <output-dir>")
	}

	inputDir := os.Args[1]
	outputDir := os.Args[2]

	if err := convertOpenAPIFiles(inputDir, outputDir); err != nil {
		log.Fatalf("Error converting OpenAPI files: %v", err)
	}

	fmt.Printf("Successfully converted OpenAPI v2 files from %s to OpenAPI v3 in %s\n", inputDir, outputDir)
}

func convertOpenAPIFiles(inputDir, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Walk through input directory and find all .json files
	return filepath.WalkDir(inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".json") {
			return nil
		}

		// Read the OpenAPI v2 file
		v2Data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Parse as OpenAPI v2
		var v2Doc openapi2.T
		if err := json.Unmarshal(v2Data, &v2Doc); err != nil {
			fmt.Printf("Warning: skipping file %s (not a valid OpenAPI v2 file): %v\n", path, err)
			return nil
		}

		// Remove default responses from v2 doc before conversion
		removeDefaultResponses(&v2Doc)

		// Convert to OpenAPI v3
		v3Doc, err := openapi2conv.ToV3(&v2Doc)
		if err != nil {
			return fmt.Errorf("failed to convert %s to OpenAPI v3: %w", path, err)
		}

		// Handle basePath conversion to servers for OpenAPI v3
		handleBasePath(&v2Doc, v3Doc)
		
		// Add parameter constraints for OpenAPI v3
		addParameterConstraints(v3Doc)
		
		// Apply examples from external JSON file with validation
		if err := applyExamples(v3Doc, inputDir); err != nil {
			return fmt.Errorf("failed to apply examples for %s: %w", path, err)
		}

		// Generate output filename
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		outputPath := filepath.Join(outputDir, relPath)

		// Create output subdirectories if needed
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output subdirectory: %w", err)
		}

		// Marshal OpenAPI v3 to JSON
		v3Data, err := json.MarshalIndent(v3Doc, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal OpenAPI v3 for %s: %w", path, err)
		}

		// Write OpenAPI v3 file
		if err := os.WriteFile(outputPath, v3Data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", outputPath, err)
		}

		fmt.Printf("Converted %s -> %s\n", path, outputPath)
		return nil
	})
}

// removeDefaultResponses removes "default" responses from all operations
func removeDefaultResponses(doc *openapi2.T) {
	if doc.Paths == nil {
		return
	}

	for path, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		// Check all HTTP methods
		operations := []*openapi2.Operation{
			pathItem.Get,
			pathItem.Post,
			pathItem.Put,
			pathItem.Patch,
			pathItem.Delete,
			pathItem.Options,
			pathItem.Head,
		}

		for _, op := range operations {
			if op != nil && op.Responses != nil {
				// Remove the default response
				delete(op.Responses, "default")
				// Also remove automatic 200 response if we have custom status codes
				if hasCustomStatusCodes(op.Responses) {
					delete(op.Responses, "200")
				}
			}
		}

		// Update the path item back to the map
		doc.Paths[path] = pathItem
	}
}

// hasCustomStatusCodes checks if an operation has custom success status codes (2xx, non-200)
func hasCustomStatusCodes(responses map[string]*openapi2.Response) bool {
	for code := range responses {
		if len(code) == 3 && code[0] == '2' && code != "200" {
			return true // Has custom 2xx success code (like 201, 204, etc.)
		}
	}
	return false
}

// handleBasePath converts OpenAPI v2 basePath to OpenAPI v3 servers with multiple environments
func handleBasePath(v2Doc *openapi2.T, v3Doc *openapi3.T) {
	// Define multiple servers for different environments
	servers := []*openapi3.Server{
		{
			URL:         "https://api.inngest.com/v2",
			Description: "Production server",
		},
		{
			URL:         "http://localhost:8288/api/v2",
			Description: "Development server",
		},
	}
	v3Doc.Servers = servers
}

// addParameterConstraints adds validation constraints to specific parameters
func addParameterConstraints(v3Doc *openapi3.T) {
	if v3Doc.Paths == nil {
		return
	}

	// Target the specific /partner/accounts endpoint
	accountsPath := v3Doc.Paths.Find("/partner/accounts")
	if accountsPath != nil && accountsPath.Get != nil {
		for _, param := range accountsPath.Get.Parameters {
			if param.Value != nil && param.Value.Name == "limit" {
				// Add min/max constraints for limit parameter
				if param.Value.Schema != nil && param.Value.Schema.Value != nil {
					min := float64(1)
					max := float64(1000)
					param.Value.Schema.Value.Min = &min
					param.Value.Schema.Value.Max = &max
				}
			}
		}
	}
}

