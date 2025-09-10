// Package main provides example processing functions for OpenAPI documentation.
// This file handles reading, generating, and applying examples from the external
// api_v2_examples.json file to OpenAPI specifications.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// applyExamples reads examples from external JSON file and applies them to OpenAPI v3 responses
func applyExamples(v3Doc *openapi3.T, inputDir string) error {
	// Construct path to examples file (go up from docs/openapi/v2 to docs/)
	examplesPath := filepath.Join(filepath.Dir(filepath.Dir(inputDir)), "api_v2_examples.json")

	// Initialize examples structure
	var examples map[string]map[string]map[string]interface{}

	// Read examples file
	examplesData, err := os.ReadFile(examplesPath)
	if err != nil {
		// File doesn't exist or can't be read, start with empty structure
		fmt.Printf("Examples file %s doesn't exist or can't be read, creating new structure\n", examplesPath)
		examples = make(map[string]map[string]map[string]interface{})
	} else if len(examplesData) == 0 {
		// File exists but is empty, start with empty structure
		fmt.Printf("Examples file %s is empty, creating new structure\n", examplesPath)
		examples = make(map[string]map[string]map[string]interface{})
	} else {
		// Parse examples JSON with new structure: path -> method -> statusCode -> example
		if err := json.Unmarshal(examplesData, &examples); err != nil {
			fmt.Printf("Warning: Could not parse examples file: %v, creating new structure\n", err)
			examples = make(map[string]map[string]map[string]interface{})
		}
	}

	// Ensure examples is not nil
	if examples == nil {
		examples = make(map[string]map[string]map[string]interface{})
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
		return nil
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

				// Validate example against schema
				if err := validateExample(exampleData, responseRef.Value, pathKey, method, statusCode, v3Doc); err != nil {
					return fmt.Errorf("validation failed for %s %s %s: %w", method, pathKey, statusCode, err)
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

	return nil
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