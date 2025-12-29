package meta

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
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

		t.Run("StringishAttr", func(t *testing.T) {
			type S string
			key := "stringish-test"
			value := S("test-stringish")
			stringishAttr := StringishAttr[S](key)

			AddAttr(attrs, stringishAttr, &value)
			serialized := attrs.Serialize()

			expectedKey := withPrefix(key)
			found := false
			for _, kv := range serialized {
				if string(kv.Key) == expectedKey && kv.Value.AsString() == string(value) {
					found = true
					break
				}
			}
			require.True(t, found, "StringishAttr should be serialized correctly")
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

		t.Run("TruncatedStringAttr", func(t *testing.T) {
			attrs := NewAttrSet()
			key := "slice-test"
			value := "1234567"
			expected := "12345"
			truncatedStrAttr := TruncatedStringAttr(key, 5)

			AddAttr(attrs, truncatedStrAttr, &value)
			require.Len(t, attrs.Attrs, 1)
			require.Equal(t, &value, attrs.Attrs[0].value)

			serialized := attrs.Serialize()
			require.Len(t, serialized, 1)
			require.Equal(t, expected, serialized[0].Value.AsString())
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

	t.Run("works with custom Text attribute", func(t *testing.T) {
		attrs := NewAttrSet()
		key := "json-test"
		value := hexInt(42)
		expected := "0x2a"
		jsonAttr := TextAttr[hexInt](key)

		AddAttr(attrs, jsonAttr, &value)
		require.Len(t, attrs.Attrs, 1)
		require.Equal(t, &value, attrs.Attrs[0].value)

		serialized := attrs.Serialize()
		require.Len(t, serialized, 1)
		require.Equal(t, expected, serialized[0].Value.AsString())
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

func TestExtractTypedValues(t *testing.T) {
	// Test basic extraction with the existing attributes
	t.Run("extracts existing attributes correctly", func(t *testing.T) {
		attrs := map[string]any{
			"_inngest.step.attempt":         "2",
			"_inngest.step.has_output":      "true",
			"_inngest.ended_at":             "1761452234278",
			"_inngest.request.url":          "http://localhost:1234/?fnId=foo",
			"_inngest.response.output_size": "208",
			"_inngest.dynamic.status":       "Completed",
			"_inngest.step.op":              "StepRun",
			"_inngest.response.headers":     "{\"Content-Length\":[\"208\"],\"Content-Type\":[\"application/json\"],\"Date\":[\"Sun, 26 Oct 2025 04:17:14 GMT\"],\"User-Agent\":[\"go:v0.14.0\"],\"X-Inngest-Req-Version\":[\"2\"],\"X-Inngest-Request-State\":[\"{\\\"DNSLookup\\\":164541,\\\"TCPConnection\\\":734625,\\\"TLSHandshake\\\":0,\\\"ServerProcessing\\\":952375,\\\"IsIPv6\\\":false,\\\"Addresses\\\":[{\\\"IP\\\":\\\"192.168.65.254\\\",\\\"Zone\\\":\\\"\\\"}],\\\"ConnectedTo\\\":{\\\"IP\\\":\\\"192.168.65.254\\\",\\\"Port\\\":49345,\\\"Zone\\\":\\\"\\\"},\\\"NameLookup\\\":164541,\\\"Connect\\\":1015083,\\\"Pretransfer\\\":1015083,\\\"StartTransfer\\\":2143333,\\\"Total\\\":2247541,\\\"HostID\\\":\\\"\\\"}\"],\"X-Inngest-Sdk\":[\"go:v0.14.0\"]}",
			"_inngest.step.input":           "executor.Execute",
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)
		require.NotNil(t, ev)

		assert.NotNil(t, ev.StepAttempt)
		assert.NotNil(t, ev.ResponseOutputSize)
		assert.NotNil(t, ev.StepHasOutput)
		assert.NotNil(t, ev.EndedAt)
		assert.NotNil(t, ev.RequestURL)
		assert.NotNil(t, ev.DynamicStatus)
		assert.NotNil(t, ev.StepOp)
		assert.NotNil(t, ev.ResponseHeaders)
		assert.NotNil(t, ev.StepInput)

		assert.EqualValues(t, true, *ev.StepHasOutput)
		assert.EqualValues(t, 2, *ev.StepAttempt)
		assert.EqualValues(t, 208, *ev.ResponseOutputSize)
		assert.Equal(t, "http://localhost:1234/?fnId=foo", *ev.RequestURL)
		assert.Equal(t, "executor.Execute", *ev.StepInput)
	})

	// Test all string attributes
	t.Run("string attributes", func(t *testing.T) {
		testUUID := uuid.New()
		attrs := map[string]any{
			"_inngest.cron.schedule":            testUUID.String(),
			"_inngest.events.input":             "test-events-input",
			"_inngest.event.trigger.name":       "user.created",
			"_inngest.internal.location":        "file.go:123",
			"_inngest.step.id":                  "step-1",
			"_inngest.step.name":                "process-user",
			"_inngest.step.code_location":       "handler.go:45",
			"_inngest.step.input":               "input-data",
			"_inngest.step.output":              "output-data",
			"_inngest.step.output_ref":          "ref-12345",
			"_inngest.step.userland.id":         "user-step-1",
			"_inngest.step.run.type":            "sync",
			"_inngest.step.invoke.function.id":  "fn-123",
			"_inngest.step.wait_for_event.if":   "event.data.status == 'completed'",
			"_inngest.step.wait_for_event.name": "process.complete",
			"_inngest.step.signal.name":         "user.signal",
			"_inngest.request.url":              "https://example.com/webhook",
			"_inngest.dynamic.span.id":          "span-123",
			"_inngest.dynamic.trace.id":         "trace-456",
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, testUUID.String(), *ev.CronSchedule)
		assert.Equal(t, "test-events-input", *ev.EventsInput)
		assert.Equal(t, "user.created", *ev.TriggeringEventName)
		assert.Equal(t, "file.go:123", *ev.InternalLocation)
		assert.Equal(t, "step-1", *ev.StepID)
		assert.Equal(t, "process-user", *ev.StepName)
		assert.Equal(t, "handler.go:45", *ev.StepCodeLocation)
		assert.Equal(t, "input-data", *ev.StepInput)
		assert.Equal(t, "output-data", *ev.StepOutput)
		assert.Equal(t, "ref-12345", *ev.StepOutputRef)
		assert.Equal(t, "user-step-1", *ev.StepUserlandID)
		assert.Equal(t, "sync", *ev.StepRunType)
		assert.Equal(t, "fn-123", *ev.StepInvokeFunctionID)
		assert.Equal(t, "event.data.status == 'completed'", *ev.StepWaitForEventIf)
		assert.Equal(t, "process.complete", *ev.StepWaitForEventName)
		assert.Equal(t, "user.signal", *ev.StepSignalName)
		assert.Equal(t, "https://example.com/webhook", *ev.RequestURL)
		assert.Equal(t, "span-123", *ev.DynamicSpanID)
		assert.Equal(t, "trace-456", *ev.DynamicTraceID)
	})

	// Test integer attributes
	t.Run("integer attributes", func(t *testing.T) {
		attrs := map[string]any{
			"_inngest.function.version":                        "5",
			"_inngest.step.attempt":                            "3",
			"_inngest.step.max_attempts":                       "10",
			"_inngest.step.userland.index":                     "2",
			"_inngest.step.gateway.response.status_code":       "200",
			"_inngest.step.gateway.response.output_size_bytes": "1024",
			"_inngest.response.status_code":                    "201",
			"_inngest.response.output_size":                    "512",
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, 5, *ev.FunctionVersion)
		assert.Equal(t, 3, *ev.StepAttempt)
		assert.Equal(t, 10, *ev.StepMaxAttempts)
		assert.Equal(t, 2, *ev.StepUserlandIndex)
		assert.Equal(t, 200, *ev.StepGatewayResponseStatusCode)
		assert.Equal(t, 1024, *ev.StepGatewayResponseOutputSizeBytes)
		assert.Equal(t, 201, *ev.ResponseStatusCode)
		assert.Equal(t, 512, *ev.ResponseOutputSize)
	})

	// Test boolean attributes
	t.Run("boolean attributes", func(t *testing.T) {
		attrs := map[string]any{
			"_inngest.executor.drop":      "true",
			"_inngest.is.function.output": "false",
			"_inngest.step.has_output":    "true",
			"_inngest.step.wait.expired":  "false",
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, true, *ev.DropSpan)
		assert.Equal(t, false, *ev.IsFunctionOutput)
		assert.Equal(t, true, *ev.StepHasOutput)
		assert.Equal(t, false, *ev.StepWaitExpired)
	})

	// Test time attributes
	t.Run("time attributes", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-5 * time.Minute)
		queueTime := now.Add(-10 * time.Minute)
		batchTime := now.Add(-1 * time.Hour)
		waitExpiry := now.Add(30 * time.Minute)

		attrs := map[string]any{
			"_inngest.started_at":       fmt.Sprintf("%d", startTime.UnixMilli()),
			"_inngest.queued_at":        fmt.Sprintf("%d", queueTime.UnixMilli()),
			"_inngest.ended_at":         fmt.Sprintf("%d", now.UnixMilli()),
			"_inngest.batch.ts":         fmt.Sprintf("%d", batchTime.UnixMilli()),
			"_inngest.step.wait.expiry": fmt.Sprintf("%d", waitExpiry.UnixMilli()),
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, startTime.UnixMilli(), ev.StartedAt.UnixMilli())
		assert.Equal(t, queueTime.UnixMilli(), ev.QueuedAt.UnixMilli())
		assert.Equal(t, now.UnixMilli(), ev.EndedAt.UnixMilli())
		assert.Equal(t, batchTime.UnixMilli(), ev.BatchTimestamp.UnixMilli())
		assert.Equal(t, waitExpiry.UnixMilli(), ev.StepWaitExpiry.UnixMilli())
	})

	// Test duration attributes
	t.Run("duration attributes", func(t *testing.T) {
		attrs := map[string]any{
			"_inngest.step.sleep.duration": "5000", // 5 seconds in milliseconds
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, 5*time.Second, *ev.StepSleepDuration)
	})

	// Test UUID attributes
	t.Run("UUID attributes", func(t *testing.T) {
		accountID := uuid.New()
		appID := uuid.New()
		envID := uuid.New()
		functionID := uuid.New()

		attrs := map[string]any{
			"_inngest.account.id":  accountID.String(),
			"_inngest.app.id":      appID.String(),
			"_inngest.env.id":      envID.String(),
			"_inngest.function.id": functionID.String(),
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, accountID, *ev.AccountID)
		assert.Equal(t, appID, *ev.AppID)
		assert.Equal(t, envID, *ev.EnvID)
		assert.Equal(t, functionID, *ev.FunctionID)
	})

	// Test ULID attributes
	t.Run("ULID attributes", func(t *testing.T) {
		batchID := ulid.Make()
		runID := ulid.Make()
		triggerEventID := ulid.Make()
		finishEventID := ulid.Make()
		invokeRunID := ulid.Make()
		matchedEventID := ulid.Make()
		debugSessionID := ulid.Make()
		debugRunID := ulid.Make()

		attrs := map[string]any{
			"_inngest.batch.id":                       batchID.String(),
			"_inngest.run.id":                         runID.String(),
			"_inngest.step.invoke.trigger.event.id":   triggerEventID.String(),
			"_inngest.step.invoke.finish.event.id":    finishEventID.String(),
			"_inngest.step.invoke.run.id":             invokeRunID.String(),
			"_inngest.step.wait_for_event.matched_id": matchedEventID.String(),
			"_inngest.debug.session.id":               debugSessionID.String(),
			"_inngest.debug.run.id":                   debugRunID.String(),
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, batchID, *ev.BatchID)
		assert.Equal(t, runID, *ev.RunID)
		assert.Equal(t, triggerEventID, *ev.StepInvokeTriggerEventID)
		assert.Equal(t, finishEventID, *ev.StepInvokeFinishEventID)
		assert.Equal(t, invokeRunID, *ev.StepInvokeRunID)
		assert.Equal(t, matchedEventID, *ev.StepWaitForEventMatchedID)
		assert.Equal(t, debugSessionID, *ev.DebugSessionID)
		assert.Equal(t, debugRunID, *ev.DebugRunID)
	})

	// Test string slice attributes
	t.Run("string slice attributes", func(t *testing.T) {
		eventIDs := []string{"event-1", "event-2", "event-3"}

		// StringSlice attributes are serialized as actual string slices
		attrs := map[string]any{
			"_inngest.event.ids": eventIDs,
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, eventIDs, *ev.EventIDs)
	})

	// Test enum attributes
	t.Run("enum attributes", func(t *testing.T) {
		attrs := map[string]any{
			"_inngest.dynamic.status": "Completed",
			"_inngest.step.op":        "StepRun",
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, "Completed", ev.DynamicStatus.String())
		assert.Equal(t, "StepRun", ev.StepOp.String())
	})

	// Test JSON attributes (ResponseHeaders)
	t.Run("JSON attributes", func(t *testing.T) {
		headers := map[string][]string{
			"Content-Type":   {"application/json"},
			"Content-Length": {"256"},
			"X-Custom":       {"value1", "value2"},
		}
		headersJSON := "{\"Content-Type\":[\"application/json\"],\"Content-Length\":[\"256\"],\"X-Custom\":[\"value1\",\"value2\"]}"

		attrs := map[string]any{
			"_inngest.response.headers": headersJSON,
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, headers, map[string][]string(*ev.ResponseHeaders))
	})

	// Test missing attributes don't cause errors
	t.Run("missing attributes are nil", func(t *testing.T) {
		attrs := map[string]any{
			"_inngest.step.attempt": "1", // Only set one attribute
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)
		require.NotNil(t, ev)

		// Only StepAttempt should be set
		assert.NotNil(t, ev.StepAttempt)
		assert.Equal(t, 1, *ev.StepAttempt)

		// All others should be nil
		assert.Nil(t, ev.StartedAt)
		assert.Nil(t, ev.EndedAt)
		assert.Nil(t, ev.AccountID)
		assert.Nil(t, ev.StepHasOutput)
		assert.Nil(t, ev.RequestURL)
		assert.Nil(t, ev.DynamicStatus)
		// ... etc - they should all be nil
	})

	// Test invalid values are ignored gracefully
	t.Run("invalid values are ignored", func(t *testing.T) {
		attrs := map[string]any{
			"_inngest.step.attempt":     "invalid-int",
			"_inngest.step.has_output":  "not-a-bool",
			"_inngest.ended_at":         "not-a-timestamp",
			"_inngest.account.id":       "invalid-uuid",
			"_inngest.run.id":           "invalid-ulid",
			"_inngest.dynamic.status":   "InvalidStatus",
			"_inngest.step.op":          "InvalidOpcode",
			"_inngest.response.headers": "invalid-json",
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)
		require.NotNil(t, ev)

		// All should be nil since deserialization should fail gracefully
		assert.Nil(t, ev.StepAttempt)
		assert.Nil(t, ev.StepHasOutput)
		assert.Nil(t, ev.EndedAt)
		assert.Nil(t, ev.AccountID)
		assert.Nil(t, ev.RunID)
		assert.Nil(t, ev.DynamicStatus)
		assert.Nil(t, ev.StepOp)
		assert.Nil(t, ev.ResponseHeaders)
	})

	t.Run("Metadata Attributes", func(t *testing.T) {
		values := metadata.Values{
			"key": []byte(`"value"`),
		}
		kind := extractors.KindInngestAI
		op := enums.MetadataOpcodeSet
		attrs := map[string]any{
			"_inngest.metadata.values": "{\"key\":\"value\"}",
			"_inngest.metadata.kind":   "inngest.ai",
			"_inngest.metadata.op":     "set",
		}

		ev, err := ExtractTypedValues(context.Background(), attrs)
		require.NoError(t, err)

		assert.Equal(t, op, *ev.MetadataOp)
		assert.Equal(t, values, *ev.Metadata)
		assert.Equal(t, kind, *ev.MetadataKind)
	})
}

type hexInt int64

func (t hexInt) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "0x%x", int64(t)), nil
}

func (t *hexInt) UnmarshalText(b []byte) error {
	i, err := strconv.ParseInt(string(b), 16, 64)
	if err != nil {
		return err
	}

	*t = hexInt(i)
	return nil
}
