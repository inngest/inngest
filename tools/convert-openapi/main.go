package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
		
		// Apply examples from external JSON file
		applyExamples(v3Doc, inputDir)

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

// applyExamples reads examples from external JSON file and applies them to OpenAPI v3 responses
func applyExamples(v3Doc *openapi3.T, inputDir string) {
	// Construct path to examples file (go up from docs/openapi/v2 to docs/)
	examplesPath := filepath.Join(filepath.Dir(filepath.Dir(inputDir)), "api_v2_examples.json")
	
	// Read examples file
	examplesData, err := os.ReadFile(examplesPath)
	if err != nil {
		fmt.Printf("Warning: Could not read examples file %s: %v\n", examplesPath, err)
		return
	}
	
	// Parse examples JSON with new structure: path -> method -> statusCode -> example
	var examples map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(examplesData, &examples); err != nil {
		fmt.Printf("Warning: Could not parse examples file: %v\n", err)
		return
	}
	
	// Generate missing entries in examples structure
	generateMissingExamples(v3Doc, &examples)
	
	// Sort examples structure for better maintainability
	sortedExamples := sortExamplesStructure(examples)
	
	// Write updated examples back to file
	updatedExamplesData, err := json.MarshalIndent(sortedExamples, "", "  ")
	if err != nil {
		fmt.Printf("Warning: Could not marshal updated examples: %v\n", err)
	} else if err := os.WriteFile(examplesPath, updatedExamplesData, 0644); err != nil {
		fmt.Printf("Warning: Could not write updated examples file: %v\n", err)
	} else {
		fmt.Printf("Updated examples file with missing entries: %s\n", examplesPath)
	}
	
	if v3Doc.Paths == nil {
		return
	}
	
	// Apply examples to each path and operation
	for pathKey, pathItem := range v3Doc.Paths.Map() {
		if pathItem == nil {
			continue
		}
		
		// Find examples for this path
		pathExamples, pathExists := examples[pathKey]
		if !pathExists {
			continue
		}
		
		// Check each HTTP method
		operations := map[string]*openapi3.Operation{
			"get":    pathItem.Get,
			"post":   pathItem.Post,
			"put":    pathItem.Put,
			"patch":  pathItem.Patch,
			"delete": pathItem.Delete,
		}
		
		for method, operation := range operations {
			if operation == nil || operation.Responses == nil {
				continue
			}
			
			// Find examples for this method
			methodExamples, methodExists := pathExamples[method]
			if !methodExists {
				continue
			}
			
			// Apply examples to each response status code
			for statusCode, exampleData := range methodExamples {
				responseRef, exists := operation.Responses.Map()[statusCode]
				if !exists || responseRef == nil || responseRef.Value == nil || responseRef.Value.Content == nil {
					continue
				}
				
				// Skip TODO entries - don't add them to the generated documentation
				if isTodoExample(exampleData) {
					continue
				}
				
				// Add example to each content type
				for _, mediaType := range responseRef.Value.Content {
					if mediaType == nil {
						continue
					}
					
					// Add the example
					if mediaType.Examples == nil {
						mediaType.Examples = make(map[string]*openapi3.ExampleRef)
					}
					
					mediaType.Examples["default"] = &openapi3.ExampleRef{
						Value: &openapi3.Example{
							Value: exampleData,
						},
					}
				}
			}
		}
	}
}

// generateMissingExamples creates empty example entries for any missing path/method/statusCode combinations
func generateMissingExamples(v3Doc *openapi3.T, examples *map[string]map[string]map[string]interface{}) {
	if v3Doc.Paths == nil {
		return
	}
	
	// Initialize examples map if nil
	if *examples == nil {
		*examples = make(map[string]map[string]map[string]interface{})
	}
	
	// Scan all paths and operations in the OpenAPI spec
	for pathKey, pathItem := range v3Doc.Paths.Map() {
		if pathItem == nil {
			continue
		}
		
		// Initialize path entry if missing
		if (*examples)[pathKey] == nil {
			(*examples)[pathKey] = make(map[string]map[string]interface{})
		}
		
		// Check each HTTP method
		operations := map[string]*openapi3.Operation{
			"get":    pathItem.Get,
			"post":   pathItem.Post,
			"put":    pathItem.Put,
			"patch":  pathItem.Patch,
			"delete": pathItem.Delete,
		}
		
		for method, operation := range operations {
			if operation == nil || operation.Responses == nil {
				continue
			}
			
			// Initialize method entry if missing
			if (*examples)[pathKey][method] == nil {
				(*examples)[pathKey][method] = make(map[string]interface{})
			}
			
			// Add missing status codes with empty objects
			for statusCode := range operation.Responses.Map() {
				if (*examples)[pathKey][method][statusCode] == nil {
					(*examples)[pathKey][method][statusCode] = map[string]interface{}{
						"// TODO": "Add example data for " + method + " " + pathKey + " " + statusCode,
					}
				}
			}
		}
	}
}

// isTodoExample checks if an example is a TODO placeholder that should not be included in documentation
func isTodoExample(exampleData interface{}) bool {
	if exampleMap, ok := exampleData.(map[string]interface{}); ok {
		// Check if it has a TODO field
		if _, hasTodo := exampleMap["// TODO"]; hasTodo {
			return true
		}
		
		// Check if it only has TODO fields (any key starting with //)
		nonTodoFields := 0
		for key := range exampleMap {
			if !strings.HasPrefix(key, "//") {
				nonTodoFields++
			}
		}
		
		// If all fields are TODO/comment fields, consider it a TODO example
		return nonTodoFields == 0
	}
	
	return false
}

// sortExamplesStructure sorts the examples structure for better maintainability
// Sorts: paths alphabetically, then methods (get, post, put, patch, delete), then status codes numerically
func sortExamplesStructure(examples map[string]map[string]map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Sort paths
	paths := make([]string, 0, len(examples))
	for path := range examples {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	
	// Method order preference
	methodOrder := map[string]int{
		"get": 1, "post": 2, "put": 3, "patch": 4, "delete": 5, "head": 6, "options": 7,
	}
	
	for _, path := range paths {
		pathMethods := examples[path]
		sortedPath := make(map[string]interface{})
		
		// Sort methods by preferred order
		methods := make([]string, 0, len(pathMethods))
		for method := range pathMethods {
			methods = append(methods, method)
		}
		sort.Slice(methods, func(i, j int) bool {
			orderI, okI := methodOrder[methods[i]]
			orderJ, okJ := methodOrder[methods[j]]
			if okI && okJ {
				return orderI < orderJ
			}
			if okI {
				return true
			}
			if okJ {
				return false
			}
			return methods[i] < methods[j]
		})
		
		for _, method := range methods {
			methodStatuses := pathMethods[method]
			sortedMethod := make(map[string]interface{})
			
			// Sort status codes numerically
			statusCodes := make([]string, 0, len(methodStatuses))
			for status := range methodStatuses {
				statusCodes = append(statusCodes, status)
			}
			sort.Slice(statusCodes, func(i, j int) bool {
				// Convert to int for proper numerical sorting
				iVal := 0
				jVal := 0
				if val, err := strconv.Atoi(statusCodes[i]); err == nil {
					iVal = val
				}
				if val, err := strconv.Atoi(statusCodes[j]); err == nil {
					jVal = val
				}
				return iVal < jVal
			})
			
			for _, status := range statusCodes {
				sortedMethod[status] = methodStatuses[status]
			}
			
			sortedPath[method] = sortedMethod
		}
		
		result[path] = sortedPath
	}
	
	return result
}
