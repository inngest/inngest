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

func TestResourceLogsRoundTrip(t *testing.T) {
	now := time.Now()
	observedTime := now.Add(time.Millisecond)

	tests := []struct {
		name    string
		records []log.Record
	}{
		{
			name:    "empty records",
			records: []log.Record{},
		},
		{
			name: "single basic record",
			records: []log.Record{
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now)
					r.SetObservedTimestamp(observedTime)
					r.SetSeverity(api.SeverityInfo)
					r.SetSeverityText("INFO")
					r.SetBody(api.StringValue("test message"))
				}),
			},
		},
		{
			name: "record with attributes",
			records: []log.Record{
				createLogRecord(t, log.Record{}, func(r *api.Record) {
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
		},
		{
			name: "record with trace context",
			records: []log.Record{
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now)
					r.SetObservedTimestamp(observedTime)
					r.SetSeverity(api.SeverityWarn)
					r.SetSeverityText("WARN")
					r.SetBody(api.StringValue("warning message"))
					// Note: TraceID and SpanID are set differently in the otel API
					// They need to be set using trace context or similar mechanisms
				}),
			},
		},
		{
			name: "record with complex body types",
			records: []log.Record{
				createLogRecord(t, log.Record{}, func(r *api.Record) {
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
		},
		{
			name: "record with slice body",
			records: []log.Record{
				createLogRecord(t, log.Record{}, func(r *api.Record) {
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
		},
		{
			name: "record with bytes body",
			records: []log.Record{
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now)
					r.SetObservedTimestamp(observedTime)
					r.SetSeverity(api.SeverityFatal)
					r.SetSeverityText("FATAL")
					r.SetBody(api.BytesValue([]byte("binary data")))
				}),
			},
		},
		{
			name: "multiple records with different severities",
			records: []log.Record{
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now)
					r.SetSeverity(api.SeverityTrace)
					r.SetBody(api.StringValue("trace message"))
				}),
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now.Add(time.Second))
					r.SetSeverity(api.SeverityDebug)
					r.SetBody(api.StringValue("debug message"))
				}),
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now.Add(2 * time.Second))
					r.SetSeverity(api.SeverityInfo)
					r.SetBody(api.StringValue("info message"))
				}),
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now.Add(3 * time.Second))
					r.SetSeverity(api.SeverityWarn)
					r.SetBody(api.StringValue("warn message"))
				}),
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now.Add(4 * time.Second))
					r.SetSeverity(api.SeverityError)
					r.SetBody(api.StringValue("error message"))
				}),
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now.Add(5 * time.Second))
					r.SetSeverity(api.SeverityFatal)
					r.SetBody(api.StringValue("fatal message"))
				}),
			},
		},
		{
			name: "record with event name",
			records: []log.Record{
				createLogRecord(t, log.Record{}, func(r *api.Record) {
					r.SetTimestamp(now)
					r.SetObservedTimestamp(observedTime)
					r.SetSeverity(api.SeverityInfo)
					r.SetSeverityText("INFO")
					r.SetEventName("test.event")
					r.SetBody(api.StringValue("event message"))
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to protobuf
			protoLogs := ResourceLogs(tt.records)
			
			// Convert back to log records
			_, roundTripRecords := ResourceLogsFromProto(protoLogs)
			
			// Compare lengths
			if len(roundTripRecords) != len(tt.records) {
				t.Fatalf("Expected %d records, got %d", len(tt.records), len(roundTripRecords))
			}
			
			// Compare each record
			for i, expected := range tt.records {
				actual := roundTripRecords[i]
				compareLogRecords(t, expected, actual)
			}
		})
	}
}


func createLogRecord(t *testing.T, base log.Record, setup func(*api.Record)) log.Record {
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
	processor := log.NewBatchProcessor(exporter)
	provider := log.NewLoggerProvider(
		log.WithProcessor(processor),
		log.WithResource(res),
	)
	
	// Create scoped logger
	logger := provider.Logger("test-scope", api.WithInstrumentationVersion("1.0.0"))
	
	// Create a record and set it up
	record := api.Record{}
	setup(&record)
	
	// Emit the record
	logger.Emit(context.Background(), record)
	
	// Force flush to ensure record is captured
	processor.ForceFlush(context.Background())
	
	// Return the captured record
	if len(exporter.records) == 0 {
		t.Fatal("No records captured")
	}
	
	return exporter.records[len(exporter.records)-1]
}

func compareLogRecords(t *testing.T, expected, actual log.Record) {
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
		t.Errorf("Body mismatch: expected %v, got %v", expected.Body(), actual.Body())
	}
	
	// Compare trace context
	if expected.TraceID() != actual.TraceID() {
		t.Errorf("TraceID mismatch: expected %v, got %v", expected.TraceID(), actual.TraceID())
	}
	
	if expected.SpanID() != actual.SpanID() {
		t.Errorf("SpanID mismatch: expected %v, got %v", expected.SpanID(), actual.SpanID())
	}
	
	// Compare attributes
	expectedAttrs := collectAttributes(expected)
	actualAttrs := collectAttributes(actual)
	
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
			t.Errorf("Attribute %q mismatch: expected %v (kind: %v), got %v (kind: %v)", key, expectedVal, expectedVal.Kind(), actualVal, actualVal.Kind())
		}
	}
}

func collectAttributes(record log.Record) map[string]api.Value {
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