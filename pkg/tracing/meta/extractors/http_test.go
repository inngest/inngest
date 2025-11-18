package extractors

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

func TestHTTPMetadataExtractor_HTTPSpan(t *testing.T) {
	span := &tracev1.Span{
		SpanId: []byte("test-span-id"),
		Name:   "POST /api/users",
		Attributes: []*commonv1.KeyValue{
			{
				Key: "http.request.header.content-type",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "application/json"},
				},
			},
			{
				Key: "http.response.header.content-type",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "application/json; charset=utf-8"},
				},
			},
			{
				Key: "http.request.method",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "POST"},
				},
			},
			{
				Key: "http.request.size",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_IntValue{IntValue: 1024},
				},
			},
			{
				Key: "http.response.size",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_IntValue{IntValue: 2048},
				},
			},
			{
				Key: "http.response.status_code",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_IntValue{IntValue: 201},
				},
			},
			{
				Key: "http.route",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "/api/users"},
				},
			},
		},
	}

	extractor := NewHTTPMetadataExtractor()
	metadata, err := extractor.ExtractMetadata(context.Background(), span)

	require.NoError(t, err)

	require.NotNil(t, metadata, "Expected metadata for HTTP span")
	require.Len(t, metadata, 1, "Expected exactly one metadata item")

	assert.Equal(t, meta.MetadataKind("inngest.http"), metadata[0].Kind())
	assert.Equal(t, meta.MetadataOpMerge, metadata[0].Op())

	// Verify the extracted data content
	raw, err := metadata[0].Serialize()
	require.NoError(t, err)

	var data map[string]any
	if dataBytes, ok := raw["data"]; ok {
		err = json.Unmarshal(dataBytes, &data)
		require.NoError(t, err)
	} else {
		// Or the data might be directly in the raw map
		data = make(map[string]any)
		for k, v := range raw {
			var value any
			if err := json.Unmarshal(v, &value); err == nil {
				data[k] = value
			}
		}
	}

	// Verify header data was extracted correctly
	assert.Equal(t, "application/json", data["request_content_type"], "Should extract request content type")
	assert.Equal(t, "application/json; charset=utf-8", data["response_content_type"], "Should extract response content type")

	// Verify method and size data
	assert.Equal(t, "POST", data["method"], "Should extract request method")
	assert.Equal(t, 1024.0, data["request_size"], "Should extract request size")
	assert.Equal(t, 2048.0, data["response_size"], "Should extract response size")
	assert.Equal(t, 201.0, data["response_status"], "Should extract response status code")
}

func TestHTTPMetadataExtractor_NonHTTPSpan(t *testing.T) {
	span := &tracev1.Span{
		SpanId: []byte("database-span-id"),
		Name:   "SELECT FROM users",
		Attributes: []*commonv1.KeyValue{
			{
				Key: "db.system",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "postgresql"},
				},
			},
			{
				Key: "db.name",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "users_db"},
				},
			},
			{
				Key: "db.operation",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "SELECT"},
				},
			},
		},
	}

	extractor := NewHTTPMetadataExtractor()
	metadata, err := extractor.ExtractMetadata(context.Background(), span)

	require.NoError(t, err)

	assert.Nil(t, metadata, "Non-HTTP span should not produce metadata")
}

func TestHTTPMetadataExtractor_PartialHTTPSpan(t *testing.T) {
	span := &tracev1.Span{
		SpanId: []byte("partial-http-span"),
		Name:   "GET /api/health",
		Attributes: []*commonv1.KeyValue{
			{
				Key: "http.request.method",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{StringValue: "GET"},
				},
			},
			{
				Key: "http.response.status_code",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_IntValue{IntValue: 200},
				},
			},
			// Deliberately missing some fields to test partial extraction
		},
	}

	extractor := NewHTTPMetadataExtractor()
	metadata, err := extractor.ExtractMetadata(context.Background(), span)

	require.NoError(t, err)

	require.NotNil(t, metadata, "Expected metadata for partial HTTP span")
	require.Len(t, metadata, 1, "Expected exactly one metadata item")

	// Verify the extracted data content
	raw, err := metadata[0].Serialize()
	require.NoError(t, err)

	var data map[string]any
	if dataBytes, ok := raw["data"]; ok {
		err = json.Unmarshal(dataBytes, &data)
		require.NoError(t, err)
	} else {
		// Or the data might be directly in the raw map
		data = make(map[string]any)
		for k, v := range raw {
			var value any
			if err := json.Unmarshal(v, &value); err == nil {
				data[k] = value
			}
		}
	}

	// Verify only the fields that were present were extracted
	assert.Equal(t, "GET", data["method"], "Should extract request method")
	assert.Equal(t, 200.0, data["response_status"], "Should extract response status code")

	// Verify missing fields are not set (or are zero values)
	assert.Empty(t, data["request_content_type"], "Should not have request content type")
	assert.Empty(t, data["response_content_type"], "Should not have response content type")
	assert.Equal(t, 0.0, data["request_size"], "Should have zero request size")
	assert.Equal(t, 0.0, data["response_size"], "Should have zero response size")
}