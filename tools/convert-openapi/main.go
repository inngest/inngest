package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

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

		// Remove internal schema-only paths used only to force schema generation
		removeInternalPaths(&v2Doc)
		removeInternalOperations(&v2Doc)

		// Convert to OpenAPI v3
		v3Doc, err := openapi2conv.ToV3(&v2Doc)
		if err != nil {
			return fmt.Errorf("failed to convert %s to OpenAPI v3: %w", path, err)
		}

		// Handle basePath conversion to servers for OpenAPI v3
		handleBasePath(&v2Doc, v3Doc)

		// Match the public REST API enum names produced by the v2 HTTP gateway.
		shortenPublicEnumNames(v3Doc)

		// Add parameter constraints for OpenAPI v3
		addParameterConstraints(v3Doc)

		// Apply examples from external JSON file
		if err := applyExamples(v3Doc, inputDir); err != nil {
			return fmt.Errorf("failed to apply examples to %s: %w", path, err)
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

// removeInternalPaths removes paths prefixed with /_internal/ — these are only
// present to force protoc-gen-openapiv2 to emit schema definitions and should
// never appear in the public API docs.
func removeInternalPaths(doc *openapi2.T) {
	if doc.Paths == nil {
		return
	}
	for path := range doc.Paths {
		if strings.HasPrefix(path, "/_internal/") {
			delete(doc.Paths, path)
		}
	}
}

func removeInternalOperations(doc *openapi2.T) {
	if doc.Paths == nil {
		return
	}

	for path, pathItem := range doc.Paths {
		if pathItem == nil {
			continue
		}

		if hasTag(pathItem.Get, "Internal") {
			pathItem.Get = nil
		}
		if hasTag(pathItem.Post, "Internal") {
			pathItem.Post = nil
		}
		if hasTag(pathItem.Put, "Internal") {
			pathItem.Put = nil
		}
		if hasTag(pathItem.Patch, "Internal") {
			pathItem.Patch = nil
		}
		if hasTag(pathItem.Delete, "Internal") {
			pathItem.Delete = nil
		}
		if hasTag(pathItem.Options, "Internal") {
			pathItem.Options = nil
		}
		if hasTag(pathItem.Head, "Internal") {
			pathItem.Head = nil
		}

		if pathItem.Get == nil &&
			pathItem.Post == nil &&
			pathItem.Put == nil &&
			pathItem.Patch == nil &&
			pathItem.Delete == nil &&
			pathItem.Options == nil &&
			pathItem.Head == nil {
			delete(doc.Paths, path)
			continue
		}

		doc.Paths[path] = pathItem
	}
}

func hasTag(op *openapi2.Operation, tag string) bool {
	if op == nil {
		return false
	}
	for _, t := range op.Tags {
		if t == tag {
			return true
		}
	}
	return false
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

func shortenPublicEnumNames(v3Doc *openapi3.T) {
	if v3Doc == nil {
		return
	}

	publicEnumPrefixes := derivePublicEnumPrefixes(v3Doc.Components)
	visited := map[*openapi3.Schema]bool{}
	if v3Doc.Components != nil {
		for _, schemaRef := range v3Doc.Components.Schemas {
			shortenPublicEnumSchemaRef(schemaRef, publicEnumPrefixes, visited)
		}
	}

	if v3Doc.Paths == nil {
		return
	}
	for _, pathItem := range v3Doc.Paths.Map() {
		if pathItem == nil {
			continue
		}
		shortenPublicEnumParameters(pathItem.Parameters, publicEnumPrefixes, visited)
		for _, operation := range pathItem.Operations() {
			shortenPublicEnumOperation(operation, publicEnumPrefixes, visited)
		}
	}
}

func derivePublicEnumPrefixes(components *openapi3.Components) []string {
	if components == nil {
		return nil
	}

	prefixes := map[string]bool{}
	for schemaName, schemaRef := range components.Schemas {
		if schemaRef == nil || schemaRef.Value == nil || len(schemaRef.Value.Enum) == 0 {
			continue
		}

		prefix := enumPrefixFromSchemaName(schemaName)
		if prefix == "" || !schemaHasEnumPrefix(schemaRef.Value, prefix) {
			continue
		}
		prefixes[prefix] = true
	}

	return sortedPublicEnumPrefixes(prefixes)
}

func enumPrefixFromSchemaName(schemaName string) string {
	typeName := schemaName
	for i, r := range schemaName {
		if unicode.IsUpper(r) {
			typeName = schemaName[i:]
			break
		}
	}
	if typeName == "" {
		return ""
	}
	return upperSnake(typeName) + "_"
}

func upperSnake(value string) string {
	runes := []rune(value)
	var out strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			prev := runes[i-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || unicode.IsUpper(prev) && nextIsLower {
				out.WriteByte('_')
			}
		}
		out.WriteRune(unicode.ToUpper(r))
	}
	return out.String()
}

func schemaHasEnumPrefix(schema *openapi3.Schema, prefix string) bool {
	for _, value := range schema.Enum {
		str, ok := value.(string)
		if ok && strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}

func sortedPublicEnumPrefixes(prefixes map[string]bool) []string {
	result := make([]string, 0, len(prefixes))
	for prefix := range prefixes {
		result = append(result, prefix)
	}
	sort.Slice(result, func(i, j int) bool {
		if len(result[i]) != len(result[j]) {
			return len(result[i]) > len(result[j])
		}
		return result[i] < result[j]
	})
	return result
}

func shortenPublicEnumOperation(operation *openapi3.Operation, prefixes []string, visited map[*openapi3.Schema]bool) {
	shortenPublicEnumParameters(operation.Parameters, prefixes, visited)
	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		shortenPublicEnumContent(operation.RequestBody.Value.Content, prefixes, visited)
	}
	if operation.Responses == nil {
		return
	}
	for _, responseRef := range operation.Responses.Map() {
		if responseRef != nil && responseRef.Value != nil {
			shortenPublicEnumContent(responseRef.Value.Content, prefixes, visited)
		}
	}
}

func shortenPublicEnumParameters(parameters openapi3.Parameters, prefixes []string, visited map[*openapi3.Schema]bool) {
	for _, parameterRef := range parameters {
		if parameterRef == nil || parameterRef.Value == nil {
			continue
		}
		shortenPublicEnumSchemaRef(parameterRef.Value.Schema, prefixes, visited)
		shortenPublicEnumContent(parameterRef.Value.Content, prefixes, visited)
	}
}

func shortenPublicEnumContent(content openapi3.Content, prefixes []string, visited map[*openapi3.Schema]bool) {
	for _, mediaType := range content {
		if mediaType != nil {
			shortenPublicEnumSchemaRef(mediaType.Schema, prefixes, visited)
		}
	}
}

func shortenPublicEnumSchemaRef(schemaRef *openapi3.SchemaRef, prefixes []string, visited map[*openapi3.Schema]bool) {
	if schemaRef == nil || schemaRef.Value == nil {
		return
	}

	schema := schemaRef.Value
	if visited[schema] {
		return
	}
	visited[schema] = true

	if str, ok := schema.Default.(string); ok {
		schema.Default = shortenPublicEnumString(str, prefixes)
	}
	for i, value := range schema.Enum {
		if str, ok := value.(string); ok {
			schema.Enum[i] = shortenPublicEnumString(str, prefixes)
		}
	}

	for _, group := range []openapi3.SchemaRefs{schema.OneOf, schema.AnyOf, schema.AllOf} {
		for _, childRef := range group {
			shortenPublicEnumSchemaRef(childRef, prefixes, visited)
		}
	}
	shortenPublicEnumSchemaRef(schema.Not, prefixes, visited)
	shortenPublicEnumSchemaRef(schema.Items, prefixes, visited)
	shortenPublicEnumSchemaRef(schema.AdditionalProperties.Schema, prefixes, visited)
	for _, childRef := range schema.Properties {
		shortenPublicEnumSchemaRef(childRef, prefixes, visited)
	}
}

func shortenPublicEnumString(value string, prefixes []string) string {
	for _, prefix := range prefixes {
		if trimmed, ok := strings.CutPrefix(value, prefix); ok {
			return trimmed
		}
	}
	return value
}

// applyExamples reads authored examples and applies them to OpenAPI v3 responses.
func applyExamples(v3Doc *openapi3.T, inputDir string) error {
	// Construct path to examples file (go up from docs/openapi/v2 to docs/)
	examplesPath := filepath.Join(filepath.Dir(filepath.Dir(inputDir)), "api_v2_examples.json")

	examplesData, err := os.ReadFile(examplesPath)
	if err != nil {
		return fmt.Errorf("read examples file %s: %w", examplesPath, err)
	}

	var examples map[string]map[string]map[string]interface{}
	if err := json.Unmarshal(examplesData, &examples); err != nil {
		return fmt.Errorf("parse examples file %s: %w", examplesPath, err)
	}

	if v3Doc.Paths == nil && len(examples) > 0 {
		return fmt.Errorf("examples file contains entries but OpenAPI document has no paths")
	}

	for pathKey, pathExamples := range examples {
		pathItem := v3Doc.Paths.Find(pathKey)
		if pathItem == nil {
			return fmt.Errorf("example references unknown path %s", pathKey)
		}

		for method, methodExamples := range pathExamples {
			operation := operationForMethod(pathItem, method)
			if operation == nil {
				return fmt.Errorf("example references unknown operation %s %s", strings.ToUpper(method), pathKey)
			}

			for statusCode, exampleData := range methodExamples {
				responseRef, exists := operation.Responses.Map()[statusCode]
				if !exists || responseRef == nil || responseRef.Value == nil || len(responseRef.Value.Content) == 0 {
					return fmt.Errorf("example references unknown response %s for %s %s", statusCode, strings.ToUpper(method), pathKey)
				}

				if isTodoExample(exampleData) {
					return fmt.Errorf("example for %s %s response %s is a TODO placeholder", strings.ToUpper(method), pathKey, statusCode)
				}

				for _, mediaType := range responseRef.Value.Content {
					if mediaType == nil {
						continue
					}

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

func operationForMethod(pathItem *openapi3.PathItem, method string) *openapi3.Operation {
	switch method {
	case "get":
		return pathItem.Get
	case "post":
		return pathItem.Post
	case "put":
		return pathItem.Put
	case "patch":
		return pathItem.Patch
	case "delete":
		return pathItem.Delete
	case "head":
		return pathItem.Head
	case "options":
		return pathItem.Options
	case "trace":
		return pathItem.Trace
	default:
		return nil
	}
}

// isTodoExample checks whether an example is only a TODO/comment placeholder.
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
