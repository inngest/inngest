package devserver

import (
	"encoding/json"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/require"
)

// TestMCPInputSchemaValidation verifies that the MCP tool input schemas
// are valid JSON Schema objects that comply with the MCP spec.
// This is a regression test for GitHub issue #3305.
//
// The issue was that using `any` type with `jsonschema:"true"` tag created
// schemas like: {"data": {"description": "true"}} which lacks a proper type.
//
// The fix changes Data and User fields from `any` to `map[string]any`,
// producing valid schemas like: {"data": {"type": "object", "additionalProperties": true}}
func TestMCPInputSchemaValidation(t *testing.T) {
	t.Run("SendEventArgs schema is valid", func(t *testing.T) {
		// Generate schema from SendEventArgs
		schema, err := jsonschema.For[SendEventArgs](nil)
		require.NoError(t, err)

		// Verify the schema has type "object"
		require.Equal(t, "object", schema.Type)

		// Marshal to JSON to inspect the structure
		schemaBytes, err := json.Marshal(schema)
		require.NoError(t, err)

		// Unmarshal to a map to verify properties are proper objects
		var schemaMap map[string]any
		err = json.Unmarshal(schemaBytes, &schemaMap)
		require.NoError(t, err)

		// Check that properties exist and are valid objects
		props, ok := schemaMap["properties"].(map[string]any)
		require.True(t, ok, "properties should be an object")

		// Verify each property is an object, not a boolean
		for propName, propValue := range props {
			propMap, ok := propValue.(map[string]any)
			require.True(t, ok, "property %q should be an object, not %T", propName, propValue)

			// For map[string]any fields, verify they have type "object"
			if propName == "data" || propName == "user" {
				require.Equal(t, "object", propMap["type"], "property %q should have type 'object'", propName)
				// Also verify description is not set to "true" (the old bug)
				desc, hasDesc := propMap["description"]
				if hasDesc {
					require.NotEqual(t, "true", desc, "property %q should not have description 'true' (old bug)", propName)
				}
			}
		}
	})

	t.Run("InvokeFunctionArgs schema is valid", func(t *testing.T) {
		// Generate schema from InvokeFunctionArgs
		schema, err := jsonschema.For[InvokeFunctionArgs](nil)
		require.NoError(t, err)

		// Verify the schema has type "object"
		require.Equal(t, "object", schema.Type)

		// Marshal to JSON to inspect the structure
		schemaBytes, err := json.Marshal(schema)
		require.NoError(t, err)

		// Unmarshal to a map to verify properties are proper objects
		var schemaMap map[string]any
		err = json.Unmarshal(schemaBytes, &schemaMap)
		require.NoError(t, err)

		// Check that properties exist and are valid objects
		props, ok := schemaMap["properties"].(map[string]any)
		require.True(t, ok, "properties should be an object")

		// Verify each property is an object, not a boolean
		for propName, propValue := range props {
			propMap, ok := propValue.(map[string]any)
			require.True(t, ok, "property %q should be an object, not %T", propName, propValue)

			// For map[string]any fields, verify they have type "object"
			if propName == "data" || propName == "user" {
				require.Equal(t, "object", propMap["type"], "property %q should have type 'object'", propName)
				// Also verify description is not set to "true" (the old bug)
				desc, hasDesc := propMap["description"]
				if hasDesc {
					require.NotEqual(t, "true", desc, "property %q should not have description 'true' (old bug)", propName)
				}
			}
		}
	})

	t.Run("Data and User fields have explicit type definition", func(t *testing.T) {
		// This test ensures that the old jsonschema:"true" tag issue is fixed
		// The old bug produced schemas like: {"data": {"description": "true"}}
		// which lacks a "type" field and has meaningless description
		//
		// The MCP spec requires that inputSchema properties are well-formed
		// JSON Schema objects with explicit types

		for _, testCase := range []struct {
			name   string
			schema any
		}{
			{"SendEventArgs", func() any {
				s, _ := jsonschema.For[SendEventArgs](nil)
				return s
			}()},
			{"InvokeFunctionArgs", func() any {
				s, _ := jsonschema.For[InvokeFunctionArgs](nil)
				return s
			}()},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				schemaBytes, err := json.Marshal(testCase.schema)
				require.NoError(t, err)

				var schemaMap map[string]any
				err = json.Unmarshal(schemaBytes, &schemaMap)
				require.NoError(t, err)

				props, ok := schemaMap["properties"].(map[string]any)
				require.True(t, ok, "properties should be an object")

				// Check data and user properties specifically
				for _, fieldName := range []string{"data", "user"} {
					propValue, exists := props[fieldName]
					require.True(t, exists, "property %q should exist", fieldName)

					propMap, ok := propValue.(map[string]any)
					require.True(t, ok, "property %q must be an object schema", fieldName)

					// Ensure the property has a type field (the fix ensures type:"object")
					typeVal, hasType := propMap["type"]
					require.True(t, hasType, "property %q must have a 'type' field (old bug had missing type)", fieldName)
					require.Equal(t, "object", typeVal, "property %q should have type 'object'", fieldName)

					// Ensure description is not "true" (the old bug)
					if desc, hasDesc := propMap["description"]; hasDesc {
						require.NotEqual(t, "true", desc, "property %q must not have description 'true' (old bug)", fieldName)
					}
				}
			})
		}
	})
}
