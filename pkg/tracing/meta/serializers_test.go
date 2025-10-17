package meta

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestAddAttr(t *testing.T) {
	t.Run("adds new attribute when key doesn't exist", func(t *testing.T) {
		attrs := NewAttrSet()
		testKey := "test-key"
		testValue := "test-value"
		stringAttr := StringAttr(testKey)

		AddAttr(attrs, stringAttr, &testValue)

		require.Len(t, attrs.Attrs, 1)
		require.Equal(t, withPrefix(testKey), attrs.Attrs[0].key)
		require.Equal(t, &testValue, attrs.Attrs[0].value)
	})

	t.Run("updates existing attribute value when key exists", func(t *testing.T) {
		attrs := NewAttrSet()
		testKey := "test-key"
		originalValue := "original-value"
		updatedValue := "updated-value"
		stringAttr := StringAttr(testKey)

		// Add initial attribute
		AddAttr(attrs, stringAttr, &originalValue)
		require.Len(t, attrs.Attrs, 1)
		require.Equal(t, &originalValue, attrs.Attrs[0].value)

		// Update the same attribute
		AddAttr(attrs, stringAttr, &updatedValue)
		require.Len(t, attrs.Attrs, 1, "should still have only one attribute")
		require.Equal(t, &updatedValue, attrs.Attrs[0].value, "value should be updated")
		require.Equal(t, withPrefix(testKey), attrs.Attrs[0].key, "key should remain the same")
	})

	t.Run("handles multiple attributes with different keys", func(t *testing.T) {
		attrs := NewAttrSet()
		
		// Add string attribute
		stringKey := "string-key"
		stringValue := "string-value"
		stringAttr := StringAttr(stringKey)
		AddAttr(attrs, stringAttr, &stringValue)

		// Add int attribute
		intKey := "int-key"
		intValue := 42
		intAttr := IntAttr(intKey)
		AddAttr(attrs, intAttr, &intValue)

		// Add bool attribute
		boolKey := "bool-key"
		boolValue := true
		boolAttr := BoolAttr(boolKey)
		AddAttr(attrs, boolAttr, &boolValue)

		require.Len(t, attrs.Attrs, 3)
		
		// Verify each attribute was added correctly
		keys := make([]string, len(attrs.Attrs))
		for i, attr := range attrs.Attrs {
			keys[i] = attr.key
		}
		require.Contains(t, keys, withPrefix(stringKey))
		require.Contains(t, keys, withPrefix(intKey))
		require.Contains(t, keys, withPrefix(boolKey))
	})

	t.Run("updates only the correct attribute when multiple exist", func(t *testing.T) {
		attrs := NewAttrSet()
		
		// Add first attribute
		key1 := "key1"
		value1 := "value1"
		attr1 := StringAttr(key1)
		AddAttr(attrs, attr1, &value1)

		// Add second attribute
		key2 := "key2"
		value2 := "value2"
		attr2 := StringAttr(key2)
		AddAttr(attrs, attr2, &value2)

		require.Len(t, attrs.Attrs, 2)

		// Update first attribute
		newValue1 := "updated-value1"
		AddAttr(attrs, attr1, &newValue1)

		require.Len(t, attrs.Attrs, 2, "should still have two attributes")
		
		// Find and verify the updated attribute
		var foundAttr1, foundAttr2 *SerializableAttr
		for i := range attrs.Attrs {
			if attrs.Attrs[i].key == withPrefix(key1) {
				foundAttr1 = &attrs.Attrs[i]
			} else if attrs.Attrs[i].key == withPrefix(key2) {
				foundAttr2 = &attrs.Attrs[i]
			}
		}

		require.NotNil(t, foundAttr1, "first attribute should exist")
		require.NotNil(t, foundAttr2, "second attribute should exist")
		require.Equal(t, &newValue1, foundAttr1.value, "first attribute should be updated")
		require.Equal(t, &value2, foundAttr2.value, "second attribute should remain unchanged")
	})

	t.Run("works with different attribute types", func(t *testing.T) {
		attrs := NewAttrSet()

		t.Run("StringAttr", func(t *testing.T) {
			key := "string-test"
			value := "test-string"
			stringAttr := StringAttr(key)
			
			AddAttr(attrs, stringAttr, &value)
			serialized := attrs.Serialize()
			
			expectedKey := withPrefix(key)
			found := false
			for _, kv := range serialized {
				if string(kv.Key) == expectedKey && kv.Value.AsString() == value {
					found = true
					break
				}
			}
			require.True(t, found, "StringAttr should be serialized correctly")
		})

		t.Run("IntAttr", func(t *testing.T) {
			key := "int-test"
			value := 123
			intAttr := IntAttr(key)
			
			AddAttr(attrs, intAttr, &value)
			serialized := attrs.Serialize()
			
			expectedKey := withPrefix(key)
			found := false
			for _, kv := range serialized {
				if string(kv.Key) == expectedKey && kv.Value.AsInt64() == int64(value) {
					found = true
					break
				}
			}
			require.True(t, found, "IntAttr should be serialized correctly")
		})

		t.Run("BoolAttr", func(t *testing.T) {
			key := "bool-test"
			value := true
			boolAttr := BoolAttr(key)
			
			AddAttr(attrs, boolAttr, &value)
			serialized := attrs.Serialize()
			
			expectedKey := withPrefix(key)
			found := false
			for _, kv := range serialized {
				if string(kv.Key) == expectedKey && kv.Value.AsBool() == value {
					found = true
					break
				}
			}
			require.True(t, found, "BoolAttr should be serialized correctly")
		})
	})

	t.Run("handles nil values appropriately", func(t *testing.T) {
		attrs := NewAttrSet()
		key := "nil-test"
		stringAttr := StringAttr(key)
		
		// Add nil value
		AddAttr(attrs, stringAttr, (*string)(nil))
		
		require.Len(t, attrs.Attrs, 1)
		require.Nil(t, attrs.Attrs[0].value)
		
		// Serialization should handle nil appropriately (returns BlankAttr with empty key)
		serialized := attrs.Serialize()
		// BlankAttr has an empty key, so we shouldn't find our expected key
		expectedKey := withPrefix(key)
		found := false
		for _, kv := range serialized {
			if string(kv.Key) == expectedKey {
				found = true
				break
			}
		}
		// When nil, BlankAttr is returned which has empty key, so we shouldn't find our key
		require.False(t, found, "nil value should serialize to BlankAttr with empty key")
		
		// Verify that the attribute was stored internally even if serialization returns BlankAttr
		require.Equal(t, withPrefix(key), attrs.Attrs[0].key, "internal key should be preserved")
		require.Nil(t, attrs.Attrs[0].value, "internal value should be nil")
	})

	t.Run("preserves attribute order when updating", func(t *testing.T) {
		attrs := NewAttrSet()
		
		// Add multiple attributes in specific order
		key1, key2, key3 := "first", "second", "third"
		value1, value2, value3 := "val1", "val2", "val3"
		
		AddAttr(attrs, StringAttr(key1), &value1)
		AddAttr(attrs, StringAttr(key2), &value2)
		AddAttr(attrs, StringAttr(key3), &value3)
		
		// Update middle attribute
		newValue2 := "updated-val2"
		AddAttr(attrs, StringAttr(key2), &newValue2)
		
		require.Len(t, attrs.Attrs, 3)
		
		// Check that the order is preserved and only the middle value changed
		require.Equal(t, withPrefix(key1), attrs.Attrs[0].key)
		require.Equal(t, &value1, attrs.Attrs[0].value)
		
		require.Equal(t, withPrefix(key2), attrs.Attrs[1].key)
		require.Equal(t, &newValue2, attrs.Attrs[1].value, "middle attribute should be updated")
		
		require.Equal(t, withPrefix(key3), attrs.Attrs[2].key)
		require.Equal(t, &value3, attrs.Attrs[2].value)
	})

	t.Run("works with complex attribute types", func(t *testing.T) {
		t.Run("TimeAttr", func(t *testing.T) {
			attrs := NewAttrSet()
			key := "time-test"
			value := time.Now().UTC()
			timeAttr := TimeAttr(key)
			
			AddAttr(attrs, timeAttr, &value)
			require.Len(t, attrs.Attrs, 1)
			require.Equal(t, &value, attrs.Attrs[0].value)
		})

		t.Run("UUIDAttr", func(t *testing.T) {
			attrs := NewAttrSet()
			key := "uuid-test"
			value := uuid.New()
			uuidAttr := UUIDAttr(key)
			
			AddAttr(attrs, uuidAttr, &value)
			require.Len(t, attrs.Attrs, 1)
			require.Equal(t, &value, attrs.Attrs[0].value)
		})

		t.Run("ULIDAttr", func(t *testing.T) {
			attrs := NewAttrSet()
			key := "ulid-test"
			value := ulid.Make()
			ulidAttr := ULIDAttr(key)
			
			AddAttr(attrs, ulidAttr, &value)
			require.Len(t, attrs.Attrs, 1)
			require.Equal(t, &value, attrs.Attrs[0].value)
		})

		t.Run("StringSliceAttr", func(t *testing.T) {
			attrs := NewAttrSet()
			key := "slice-test"
			value := []string{"item1", "item2", "item3"}
			sliceAttr := StringSliceAttr(key)
			
			AddAttr(attrs, sliceAttr, &value)
			require.Len(t, attrs.Attrs, 1)
			require.Equal(t, &value, attrs.Attrs[0].value)
		})
	})

	t.Run("works with custom JSON attribute", func(t *testing.T) {
		type testStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		attrs := NewAttrSet()
		key := "json-test"
		value := testStruct{Name: "test", Value: 42}
		jsonAttr := JsonAttr[testStruct](key)
		
		AddAttr(attrs, jsonAttr, &value)
		require.Len(t, attrs.Attrs, 1)
		require.Equal(t, &value, attrs.Attrs[0].value)

		// Update with new value
		newValue := testStruct{Name: "updated", Value: 99}
		AddAttr(attrs, jsonAttr, &newValue)
		require.Len(t, attrs.Attrs, 1, "should still have only one attribute")
		require.Equal(t, &newValue, attrs.Attrs[0].value, "JSON value should be updated")
	})

	t.Run("empty attribute set works correctly", func(t *testing.T) {
		attrs := NewAttrSet()
		require.Len(t, attrs.Attrs, 0, "new attr set should be empty")
		
		key := "first-attr"
		value := "first-value"
		stringAttr := StringAttr(key)
		
		AddAttr(attrs, stringAttr, &value)
		require.Len(t, attrs.Attrs, 1, "should have one attribute after adding")
		require.Equal(t, withPrefix(key), attrs.Attrs[0].key)
		require.Equal(t, &value, attrs.Attrs[0].value)
	})
}

func TestAddAttrIfUnset(t *testing.T) {
	t.Run("adds attribute when key doesn't exist", func(t *testing.T) {
		attrs := NewAttrSet()
		key := "test-key"
		value := "test-value"
		stringAttr := StringAttr(key)

		AddAttrIfUnset(attrs, stringAttr, &value)

		require.Len(t, attrs.Attrs, 1)
		require.Equal(t, withPrefix(key), attrs.Attrs[0].key)
		require.Equal(t, &value, attrs.Attrs[0].value)
	})

	t.Run("does not add attribute when key exists", func(t *testing.T) {
		attrs := NewAttrSet()
		key := "test-key"
		originalValue := "original-value"
		newValue := "new-value"
		stringAttr := StringAttr(key)

		// Add initial attribute
		AddAttrIfUnset(attrs, stringAttr, &originalValue)
		require.Len(t, attrs.Attrs, 1)
		require.Equal(t, &originalValue, attrs.Attrs[0].value)

		// Try to add same key with different value - should be ignored
		AddAttrIfUnset(attrs, stringAttr, &newValue)
		require.Len(t, attrs.Attrs, 1, "should still have only one attribute")
		require.Equal(t, &originalValue, attrs.Attrs[0].value, "value should remain unchanged")
	})

	t.Run("works with attributes added by AddAttr", func(t *testing.T) {
		attrs := NewAttrSet()
		key := "test-key"
		originalValue := "original-value"
		newValue := "new-value"
		stringAttr := StringAttr(key)

		// Add using AddAttr
		AddAttr(attrs, stringAttr, &originalValue)
		require.Len(t, attrs.Attrs, 1)

		// Try to add same key using AddAttrIfUnset - should be ignored
		AddAttrIfUnset(attrs, stringAttr, &newValue)
		require.Len(t, attrs.Attrs, 1, "should still have only one attribute")
		require.Equal(t, &originalValue, attrs.Attrs[0].value, "value should remain unchanged")
	})

	t.Run("works with multiple different keys", func(t *testing.T) {
		attrs := NewAttrSet()
		
		key1, key2, key3 := "key1", "key2", "key3"
		value1, value2, value3 := "value1", "value2", "value3"
		
		AddAttrIfUnset(attrs, StringAttr(key1), &value1)
		AddAttrIfUnset(attrs, StringAttr(key2), &value2)
		AddAttrIfUnset(attrs, StringAttr(key3), &value3)
		
		require.Len(t, attrs.Attrs, 3)
		
		// Verify all keys were added
		keys := make([]string, len(attrs.Attrs))
		for i, attr := range attrs.Attrs {
			keys[i] = attr.key
		}
		require.Contains(t, keys, withPrefix(key1))
		require.Contains(t, keys, withPrefix(key2))
		require.Contains(t, keys, withPrefix(key3))
	})

	t.Run("handles nil keyMap initialization", func(t *testing.T) {
		// Create attrs without using NewAttrSet to test nil keyMap handling
		attrs := &SerializableAttrs{Attrs: []SerializableAttr{}}
		key := "test-key"
		value := "test-value"
		stringAttr := StringAttr(key)

		AddAttrIfUnset(attrs, stringAttr, &value)

		require.Len(t, attrs.Attrs, 1)
		require.NotNil(t, attrs.keyMap, "keyMap should be initialized")
		require.Equal(t, withPrefix(key), attrs.Attrs[0].key)
		require.Equal(t, &value, attrs.Attrs[0].value)
	})
}