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

// validateExample validates an example against the expected response schema
func validateExample(exampleData interface{}, response *openapi3.Response, path, method, statusCode string, doc *openapi3.T) error {
	// Get the schema from the response
	var schema *openapi3.Schema
	for _, mediaType := range response.Content {
		if mediaType != nil && mediaType.Schema != nil {
			if mediaType.Schema.Value != nil {
				schema = mediaType.Schema.Value
				break
			} else if mediaType.Schema.Ref != "" {
				// Resolve the schema reference
				resolvedSchema := resolveSchemaReference(mediaType.Schema.Ref, doc)
				if resolvedSchema != nil {
					schema = resolvedSchema
					break
				} else {
					fmt.Printf("Warning: Could not resolve schema reference '%s'\n", mediaType.Schema.Ref)
					return nil
				}
			}
		}
	}
	
	if schema == nil {
		// No schema found, skip validation
		return nil
	}
	
	return validateAgainstSchema(exampleData, schema, "", doc)
}

// resolveSchemaReference resolves a schema reference to the actual schema
func resolveSchemaReference(ref string, doc *openapi3.T) *openapi3.Schema {
	if doc == nil || doc.Components == nil || doc.Components.Schemas == nil {
		return nil
	}
	
	// Handle both v2 and v3 reference formats
	var schemaName string
	if strings.HasPrefix(ref, "#/definitions/") {
		schemaName = strings.TrimPrefix(ref, "#/definitions/")
	} else if strings.HasPrefix(ref, "#/components/schemas/") {
		schemaName = strings.TrimPrefix(ref, "#/components/schemas/")
	} else {
		return nil
	}
	
	if schemaRef, exists := doc.Components.Schemas[schemaName]; exists && schemaRef.Value != nil {
		return schemaRef.Value
	}
	
	return nil
}

// validateAgainstSchema dynamically validates data against an OpenAPI schema
func validateAgainstSchema(data interface{}, schema *openapi3.Schema, fieldPath string, doc *openapi3.T) error {
	if schema == nil {
		return nil
	}
	
	// Handle different schema types
	schemaType := ""
	if schema.Type != nil && len(*schema.Type) > 0 {
		schemaType = (*schema.Type)[0] // Get the first type
	}
	
	switch {
	case schemaType == "object":
		return validateObjectSchema(data, schema, fieldPath, doc)
	case schemaType == "array":
		return validateArraySchema(data, schema, fieldPath, doc)
	case schemaType == "string":
		return validateStringSchema(data, fieldPath)
	case schemaType == "number":
		return validateNumberSchema(data, fieldPath)
	case schemaType == "integer":
		return validateIntegerSchema(data, fieldPath)
	case schemaType == "boolean":
		return validateBooleanSchema(data, fieldPath)
	case schemaType == "" && len(schema.OneOf) > 0:
		// Handle oneOf schemas (like error responses that can be string or object)
		return validateOneOfSchema(data, schema, fieldPath, doc)
	case schemaType == "":
		// No type specified, assume any type is valid
		return nil
	default:
		fmt.Printf("Warning: Unknown schema type '%s' at path '%s', skipping validation\n", schemaType, fieldPath)
		return nil
	}
}

// validateObjectSchema validates an object against an object schema
func validateObjectSchema(data interface{}, schema *openapi3.Schema, fieldPath string, doc *openapi3.T) error {
	obj, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected object at path '%s', got %T", fieldPath, data)
	}
	
	// Check required fields from schema
	for _, requiredField := range schema.Required {
		if _, exists := obj[requiredField]; !exists {
			path := fieldPath
			if path != "" {
				path += "."
			}
			return fmt.Errorf("missing required field: %s%s", path, requiredField)
		}
	}
	
	// Additional validation for known proto types that don't properly export required fields
	extraRequired := getExtraRequiredFields(schema, doc)
	for _, requiredField := range extraRequired {
		if _, exists := obj[requiredField]; !exists {
			path := fieldPath
			if path != "" {
				path += "."
			}
			return fmt.Errorf("missing required field: %s%s", path, requiredField)
		}
	}
	
	// Validate each property in the object
	if schema.Properties != nil {
		for fieldName, fieldValue := range obj {
			if propSchema, exists := schema.Properties[fieldName]; exists {
				newPath := fieldPath
				if newPath != "" {
					newPath += "."
				}
				newPath += fieldName
				
				if err := validateAgainstSchema(fieldValue, propSchema.Value, newPath, doc); err != nil {
					return err
				}
			}
			// Note: We don't fail on additional properties not defined in the schema
			// This allows for some flexibility in examples
		}
	}
	
	return nil
}

// validateArraySchema validates an array against an array schema
func validateArraySchema(data interface{}, schema *openapi3.Schema, fieldPath string, doc *openapi3.T) error {
	arr, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("expected array at path '%s', got %T", fieldPath, data)
	}
	
	// Validate each item in the array
	if schema.Items != nil && schema.Items.Value != nil {
		for i, item := range arr {
			indexPath := fmt.Sprintf("%s[%d]", fieldPath, i)
			if err := validateAgainstSchema(item, schema.Items.Value, indexPath, doc); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// validateStringSchema validates a string value
func validateStringSchema(data interface{}, fieldPath string) error {
	// Allow null values (represented as nil in Go)
	if data == nil {
		return nil
	}
	
	_, ok := data.(string)
	if !ok {
		return fmt.Errorf("expected string at path '%s', got %T", fieldPath, data)
	}
	return nil
}

// validateNumberSchema validates a number value
func validateNumberSchema(data interface{}, fieldPath string) error {
	// Allow null values
	if data == nil {
		return nil
	}
	
	switch data.(type) {
	case float64, float32, int, int32, int64:
		return nil
	default:
		return fmt.Errorf("expected number at path '%s', got %T", fieldPath, data)
	}
}

// validateIntegerSchema validates an integer value
func validateIntegerSchema(data interface{}, fieldPath string) error {
	// Allow null values
	if data == nil {
		return nil
	}
	
	switch data.(type) {
	case int, int32, int64:
		return nil
	case float64:
		// JSON numbers are parsed as float64, check if it's actually an integer
		if f, ok := data.(float64); ok && f == float64(int64(f)) {
			return nil
		}
		return fmt.Errorf("expected integer at path '%s', got float", fieldPath)
	default:
		return fmt.Errorf("expected integer at path '%s', got %T", fieldPath, data)
	}
}

// validateBooleanSchema validates a boolean value
func validateBooleanSchema(data interface{}, fieldPath string) error {
	// Allow null values
	if data == nil {
		return nil
	}
	
	_, ok := data.(bool)
	if !ok {
		return fmt.Errorf("expected boolean at path '%s', got %T", fieldPath, data)
	}
	return nil
}

// validateOneOfSchema validates data against a oneOf schema
func validateOneOfSchema(data interface{}, schema *openapi3.Schema, fieldPath string, doc *openapi3.T) error {
	var errors []error
	
	// Try validating against each schema in oneOf
	for i, oneOfSchema := range schema.OneOf {
		if oneOfSchema.Value != nil {
			if err := validateAgainstSchema(data, oneOfSchema.Value, fieldPath, doc); err == nil {
				// Validation succeeded against this schema
				return nil
			} else {
				errors = append(errors, fmt.Errorf("oneOf[%d]: %w", i, err))
			}
		}
	}
	
	// If we get here, validation failed against all schemas
	if len(errors) > 0 {
		return fmt.Errorf("value at path '%s' does not match any of the oneOf schemas: %v", fieldPath, errors)
	}
	
	return fmt.Errorf("no valid oneOf schemas found for path '%s'", fieldPath)
}

// getExtraRequiredFields returns additional required fields for known proto schemas
// that don't properly export their required field information to OpenAPI
func getExtraRequiredFields(schema *openapi3.Schema, doc *openapi3.T) []string {
	// Check if this schema matches a known type by examining its properties
	if schema.Properties != nil {
		// Check for v2Error schema (has code and message properties)
		if hasProperty(schema, "code") && hasProperty(schema, "message") && len(schema.Properties) == 2 {
			// This looks like a v2Error schema - both code and message are required in proto
			return []string{"code", "message"}
		}
		
		// Check for other known schemas that should have required fields
		// Add more as needed based on proto definitions
	}
	
	return nil
}

// hasProperty checks if a schema has a specific property
func hasProperty(schema *openapi3.Schema, propertyName string) bool {
	if schema.Properties == nil {
		return false
	}
	_, exists := schema.Properties[propertyName]
	return exists
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
