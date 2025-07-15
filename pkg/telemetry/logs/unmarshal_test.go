package logs

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	api "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestLogRecordRoundTrip(t *testing.T) {
	now := time.Now()
	observedTime := now.Add(time.Millisecond)

	tests := []struct {
		name   string
		record api.Record
	}{
		{
			name: "basic record",
			record: createAPIRecord(func(r *api.Record) {
				r.SetTimestamp(now)
				r.SetObservedTimestamp(observedTime)
				r.SetSeverity(api.SeverityInfo)
				r.SetSeverityText("INFO")
				r.SetBody(api.StringValue("test message"))
			}),
		},
		{
			name: "record with attributes",
			record: createAPIRecord(func(r *api.Record) {
				r.SetTimestamp(now)
				r.SetObservedTimestamp(observedTime)
				r.SetSeverity(api.SeverityError)
				r.SetSeverityText("ERROR")
				r.SetBody(api.StringValue("error message"))
				r.AddAttributes(
					api.KeyValue{Key: "string_attr", Value: api.StringValue("value")},
					api.KeyValue{Key: "int_attr", Value: api.Int64Value(42)},
					api.KeyValue{Key: "bool_attr", Value: api.BoolValue(true)},
					api.KeyValue{Key: "float_attr", Value: api.Float64Value(3.14)},
				)
			}),
		},
		{
			name: "record with complex body types",
			record: createAPIRecord(func(r *api.Record) {
				r.SetTimestamp(now)
				r.SetObservedTimestamp(observedTime)
				r.SetSeverity(api.SeverityDebug)
				r.SetSeverityText("DEBUG")
				r.SetBody(api.MapValue(
					api.KeyValue{Key: "nested_string", Value: api.StringValue("nested")},
					api.KeyValue{Key: "nested_int", Value: api.Int64Value(123)},
				))
			}),
		},
		{
			name: "record with slice body",
			record: createAPIRecord(func(r *api.Record) {
				r.SetTimestamp(now)
				r.SetObservedTimestamp(observedTime)
				r.SetSeverity(api.SeverityTrace)
				r.SetSeverityText("TRACE")
				r.SetBody(api.SliceValue(
					api.StringValue("item1"),
					api.StringValue("item2"),
					api.Int64Value(42),
				))
			}),
		},
		{
			name: "record with bytes body",
			record: createAPIRecord(func(r *api.Record) {
				r.SetTimestamp(now)
				r.SetObservedTimestamp(observedTime)
				r.SetSeverity(api.SeverityFatal)
				r.SetSeverityText("FATAL")
				r.SetBody(api.BytesValue([]byte("binary data")))
			}),
		},
		{
			name: "record with event name",
			record: createAPIRecord(func(r *api.Record) {
				r.SetTimestamp(now)
				r.SetObservedTimestamp(observedTime)
				r.SetSeverity(api.SeverityInfo)
				r.SetSeverityText("INFO")
				r.SetEventName("test.event")
				r.SetBody(api.StringValue("event message"))
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to protobuf using the existing LogRecord function
			// But first we need to convert api.Record to log.Record
			logRecord := createLogRecord(t, tt.record)
			protoRecord := LogRecord(logRecord)

			// Convert back to api record
			roundTripRecord := LogRecordFromProto(protoRecord)

			// Compare the records
			compareAPIRecords(t, tt.record, roundTripRecord.Record)
		})
	}
}

func createAPIRecord(setup func(*api.Record)) api.Record {
	record := api.Record{}
	setup(&record)
	return record
}

func createLogRecord(t *testing.T, apiRecord api.Record) log.Record {
	t.Helper()

	// Create a resource for the record
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", "test-service"),
			attribute.String("service.version", "1.0.0"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Create test exporter to capture records
	exporter := &captureExporter{}

	// Create logger with processor and resource
	processor := log.NewSimpleProcessor(exporter)
	provider := log.NewLoggerProvider(
		log.WithProcessor(processor),
		log.WithResource(res),
	)

	// Create scoped logger
	logger := provider.Logger("test-scope", api.WithInstrumentationVersion("1.0.0"))

	// Emit the record
	logger.Emit(context.Background(), apiRecord)

	// Force flush to ensure record is captured
	processor.ForceFlush(context.Background())

	// Return the captured record
	if len(exporter.records) == 0 {
		t.Fatal("No records captured")
	}

	return exporter.records[0]
}

// captureExporter is a simple exporter that captures records for testing
type captureExporter struct {
	records []log.Record
}

func (e *captureExporter) Export(ctx context.Context, records []log.Record) error {
	e.records = append(e.records, records...)
	return nil
}

func (e *captureExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *captureExporter) ForceFlush(ctx context.Context) error {
	return nil
}

func compareAPIRecords(t *testing.T, expected, actual api.Record) {
	t.Helper()

	// Compare timestamps
	if !expected.Timestamp().Equal(actual.Timestamp()) {
		t.Errorf("Timestamp mismatch: expected %v, got %v", expected.Timestamp(), actual.Timestamp())
	}

	if !expected.ObservedTimestamp().Equal(actual.ObservedTimestamp()) {
		t.Errorf("ObservedTimestamp mismatch: expected %v, got %v", expected.ObservedTimestamp(), actual.ObservedTimestamp())
	}

	// Compare severity
	if expected.Severity() != actual.Severity() {
		t.Errorf("Severity mismatch: expected %v, got %v", expected.Severity(), actual.Severity())
	}

	// Compare severity text
	if expected.SeverityText() != actual.SeverityText() {
		t.Errorf("SeverityText mismatch: expected %q, got %q", expected.SeverityText(), actual.SeverityText())
	}

	// Compare event name
	if expected.EventName() != actual.EventName() {
		t.Errorf("EventName mismatch: expected %q, got %q", expected.EventName(), actual.EventName())
	}

	// Compare body
	if !compareLogValues(expected.Body(), actual.Body()) {
		t.Errorf("Body mismatch: expected %v (kind: %v), got %v (kind: %v)",
			expected.Body(), expected.Body().Kind(), actual.Body(), actual.Body().Kind())
	}

	// Compare attributes
	expectedAttrs := collectAPIAttributes(expected)
	actualAttrs := collectAPIAttributes(actual)

	if len(expectedAttrs) != len(actualAttrs) {
		t.Errorf("Attributes count mismatch: expected %d, got %d", len(expectedAttrs), len(actualAttrs))
		return
	}

	for key, expectedVal := range expectedAttrs {
		actualVal, exists := actualAttrs[key]
		if !exists {
			t.Errorf("Missing attribute %q", key)
			continue
		}

		if !compareLogValues(expectedVal, actualVal) {
			t.Errorf("Attribute %q mismatch: expected %v (kind: %v), got %v (kind: %v)",
				key, expectedVal, expectedVal.Kind(), actualVal, actualVal.Kind())
		}
	}
}

func collectAPIAttributes(record api.Record) map[string]api.Value {
	attrs := make(map[string]api.Value)
	record.WalkAttributes(func(kv api.KeyValue) bool {
		attrs[kv.Key] = kv.Value
		return true
	})
	return attrs
}

func compareLogValues(expected, actual api.Value) bool {
	if expected.Kind() != actual.Kind() {
		return false
	}

	switch expected.Kind() {
	case api.KindBool:
		return expected.AsBool() == actual.AsBool()
	case api.KindInt64:
		return expected.AsInt64() == actual.AsInt64()
	case api.KindFloat64:
		return expected.AsFloat64() == actual.AsFloat64()
	case api.KindString:
		return expected.AsString() == actual.AsString()
	case api.KindBytes:
		expectedBytes := expected.AsBytes()
		actualBytes := actual.AsBytes()
		if len(expectedBytes) != len(actualBytes) {
			return false
		}
		for i := range expectedBytes {
			if expectedBytes[i] != actualBytes[i] {
				return false
			}
		}
		return true
	case api.KindSlice:
		expectedSlice := expected.AsSlice()
		actualSlice := actual.AsSlice()
		if len(expectedSlice) != len(actualSlice) {
			return false
		}
		for i := range expectedSlice {
			if !compareLogValues(expectedSlice[i], actualSlice[i]) {
				return false
			}
		}
		return true
	case api.KindMap:
		expectedMap := expected.AsMap()
		actualMap := actual.AsMap()
		if len(expectedMap) != len(actualMap) {
			return false
		}

		expectedKVs := make(map[string]api.Value)
		for _, kv := range expectedMap {
			expectedKVs[kv.Key] = kv.Value
		}

		actualKVs := make(map[string]api.Value)
		for _, kv := range actualMap {
			actualKVs[kv.Key] = kv.Value
		}

		for key, expectedVal := range expectedKVs {
			actualVal, exists := actualKVs[key]
			if !exists || !compareLogValues(expectedVal, actualVal) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
