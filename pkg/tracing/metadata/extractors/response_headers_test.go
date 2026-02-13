package extractors

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

func TestResponseHeaderMetadataExtractor_FullHeaders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	span := &tracev1.Span{
		SpanId: []byte("test-span-id"),
		Attributes: []*commonv1.KeyValue{
			{
				Key: "_inngest.response.headers",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{
						StringValue: `{"Content-Type":["application/json"],"Cache-Control":["no-cache"],"X-Custom":["custom-value"]}`,
					},
				},
			},
		},
	}

	extractor := NewResponseHeaderMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	require.NotNil(t, md, "Expected metadata for span with response headers")
	require.Len(t, md, 1, "Expected exactly one metadata item")

	assert.Equal(t, metadata.Kind("inngest.response_headers"), md[0].Kind())
	assert.Equal(t, enums.MetadataOpcodeMerge, md[0].Op())

	raw, err := md[0].Serialize()
	require.NoError(t, err)

	// Verify each header was extracted and flattened correctly
	data := make(map[string]string)
	for k, v := range raw {
		var s string
		err := json.Unmarshal(v, &s)
		require.NoError(t, err)
		data[k] = s
	}

	assert.Equal(t, "application/json", data["Content-Type"])
	assert.Equal(t, "no-cache", data["Cache-Control"])
	assert.Equal(t, "custom-value", data["X-Custom"])
}

func TestResponseHeaderMetadataExtractor_NonMatchingSpan(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

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
		},
	}

	extractor := NewResponseHeaderMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	assert.Nil(t, md, "Non-matching span should not produce metadata")
}

func TestResponseHeaderMetadataExtractor_PartialHeaders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	span := &tracev1.Span{
		SpanId: []byte("partial-span"),
		Attributes: []*commonv1.KeyValue{
			{
				Key: "_inngest.response.headers",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{
						StringValue: `{"Content-Type":["text/html"]}`,
					},
				},
			},
		},
	}

	extractor := NewResponseHeaderMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	require.NotNil(t, md, "Expected metadata for span with partial headers")
	require.Len(t, md, 1, "Expected exactly one metadata item")

	raw, err := md[0].Serialize()
	require.NoError(t, err)

	assert.Len(t, raw, 1, "Should only have one header")

	var contentType string
	err = json.Unmarshal(raw["Content-Type"], &contentType)
	require.NoError(t, err)
	assert.Equal(t, "text/html", contentType)
}

func TestResponseHeaderMetadataExtractor_MultiValueHeaders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	span := &tracev1.Span{
		SpanId: []byte("multi-value-span"),
		Attributes: []*commonv1.KeyValue{
			{
				Key: "_inngest.response.headers",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{
						StringValue: `{"X-Custom":["val1","val2"],"Accept":["text/html","application/json"]}`,
					},
				},
			},
		},
	}

	extractor := NewResponseHeaderMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	require.NotNil(t, md)
	require.Len(t, md, 1)

	raw, err := md[0].Serialize()
	require.NoError(t, err)

	data := make(map[string]string)
	for k, v := range raw {
		var s string
		err := json.Unmarshal(v, &s)
		require.NoError(t, err)
		data[k] = s
	}

	assert.Equal(t, "val1, val2", data["X-Custom"], "Multi-value headers should be comma-separated")
	assert.Equal(t, "text/html, application/json", data["Accept"], "Multi-value headers should be comma-separated")
}

func TestResponseHeaderMetadataExtractor_EmptyHeaders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	span := &tracev1.Span{
		SpanId: []byte("empty-headers-span"),
		Attributes: []*commonv1.KeyValue{
			{
				Key: "_inngest.response.headers",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{
						StringValue: `{}`,
					},
				},
			},
		},
	}

	extractor := NewResponseHeaderMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	assert.Nil(t, md, "Empty headers should return nil metadata")
}

func TestNewResponseHeaderMetadataFromHTTPHeader(t *testing.T) {
	t.Parallel()

	header := http.Header{
		"Content-Type":  {"application/json"},
		"Cache-Control": {"no-cache"},
		"X-Request-Id":  {"abc-123"},
	}

	result := NewResponseHeaderMetadataFromHTTPHeader(header, 200)

	require.NotNil(t, result)
	assert.Len(t, result, 4)
	assert.Equal(t, "200", result["Status Code"])
	assert.Equal(t, "application/json", result["Content-Type"])
	assert.Equal(t, "no-cache", result["Cache-Control"])
	assert.Equal(t, "abc-123", result["X-Request-Id"])
}

func TestNewResponseHeaderMetadataFromHTTPHeader_Nil(t *testing.T) {
	t.Parallel()

	result := NewResponseHeaderMetadataFromHTTPHeader(nil, 0)
	assert.Nil(t, result)
}

func TestNewResponseHeaderMetadataFromHTTPHeader_Empty(t *testing.T) {
	t.Parallel()

	result := NewResponseHeaderMetadataFromHTTPHeader(http.Header{}, 0)
	assert.Nil(t, result)
}

func TestNewResponseHeaderMetadataFromHTTPHeader_RedactsSensitiveHeaders(t *testing.T) {
	t.Parallel()

	header := http.Header{
		"Content-Type":    {"application/json"},
		"Set-Cookie":      {"session=abc123"},
		"Authorization":   {"Bearer secret-token"},
		"X-Api-Key":       {"my-api-key"},
		"Cookie":          {"session=abc123"},
		"Proxy-Authorization": {"Basic creds"},
		"X-Forwarded-For": {"192.168.1.1"},
		"Cache-Control":   {"no-cache"},
	}

	result := NewResponseHeaderMetadataFromHTTPHeader(header, 200)

	require.NotNil(t, result)
	// Safe headers are passed through
	assert.Equal(t, "application/json", result["Content-Type"])
	assert.Equal(t, "no-cache", result["Cache-Control"])
	assert.Equal(t, "192.168.1.1", result["X-Forwarded-For"])
	assert.Equal(t, "200", result["Status Code"])

	// Sensitive headers are redacted
	assert.Equal(t, "[REDACTED]", result["Set-Cookie"])
	assert.Equal(t, "[REDACTED]", result["Authorization"])
	assert.Equal(t, "[REDACTED]", result["X-Api-Key"])
	assert.Equal(t, "[REDACTED]", result["Cookie"])
	assert.Equal(t, "[REDACTED]", result["Proxy-Authorization"])
}

func TestNewResponseHeaderMetadataFromHTTPHeader_RedactsCaseInsensitive(t *testing.T) {
	t.Parallel()

	header := http.Header{
		"AUTHORIZATION": {"Bearer token"},
		"set-cookie":    {"session=xyz"},
		"X-API-KEY":     {"key123"},
	}

	result := NewResponseHeaderMetadataFromHTTPHeader(header, 0)

	require.NotNil(t, result)
	assert.Equal(t, "[REDACTED]", result["AUTHORIZATION"])
	assert.Equal(t, "[REDACTED]", result["set-cookie"])
	assert.Equal(t, "[REDACTED]", result["X-API-KEY"])
}

func TestResponseHeaderMetadataExtractor_RedactsSensitiveHeaders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	span := &tracev1.Span{
		SpanId: []byte("sensitive-span"),
		Attributes: []*commonv1.KeyValue{
			{
				Key: "_inngest.response.headers",
				Value: &commonv1.AnyValue{
					Value: &commonv1.AnyValue_StringValue{
						StringValue: `{"Content-Type":["application/json"],"Set-Cookie":["session=abc"],"Authorization":["Bearer token"]}`,
					},
				},
			},
		},
	}

	extractor := NewResponseHeaderMetadataExtractor()
	md, err := extractor.ExtractSpanMetadata(ctx, span)

	require.NoError(t, err)
	require.NotNil(t, md)
	require.Len(t, md, 1)

	raw, err := md[0].Serialize()
	require.NoError(t, err)

	data := make(map[string]string)
	for k, v := range raw {
		var s string
		err := json.Unmarshal(v, &s)
		require.NoError(t, err)
		data[k] = s
	}

	assert.Equal(t, "application/json", data["Content-Type"])
	assert.Equal(t, "[REDACTED]", data["Set-Cookie"])
	assert.Equal(t, "[REDACTED]", data["Authorization"])
}
