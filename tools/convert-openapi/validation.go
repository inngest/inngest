// Package main provides validation functions for OpenAPI examples against proto schemas.
// This file contains dynamic schema validation logic that ensures API examples
// match their corresponding protobuf message definitions.
package main

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

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