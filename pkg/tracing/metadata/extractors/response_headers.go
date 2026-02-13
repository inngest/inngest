package extractors

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

//tygo:generate
const (
	KindInngestResponseHeaders metadata.Kind = "inngest.response_headers"
)

//tygo:generate
type ResponseHeaderMetadata map[string]string

func (m ResponseHeaderMetadata) Kind() metadata.Kind {
	return KindInngestResponseHeaders
}

func (m ResponseHeaderMetadata) Op() metadata.Opcode {
	return enums.MetadataOpcodeMerge
}

func (m ResponseHeaderMetadata) Serialize() (metadata.Values, error) {
	ret := make(metadata.Values)
	for key, value := range m {
		ret[key], _ = json.Marshal(value)
	}
	return ret, nil
}

type ResponseHeaderMetadataExtractor struct{}

func NewResponseHeaderMetadataExtractor() *ResponseHeaderMetadataExtractor {
	return &ResponseHeaderMetadataExtractor{}
}

func (e *ResponseHeaderMetadataExtractor) ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]metadata.Structured, error) {
	var headersJSON string
	var found bool

	for _, attr := range span.Attributes {
		if attr.Key == "_inngest.response.headers" {
			headersJSON = attr.Value.GetStringValue()
			found = true
			break
		}
	}

	if !found {
		return nil, nil
	}

	// Parse JSON into http.Header shape (map[string][]string)
	var rawHeaders map[string][]string
	if err := json.Unmarshal([]byte(headersJSON), &rawHeaders); err != nil {
		return nil, nil
	}

	if len(rawHeaders) == 0 {
		return nil, nil
	}

	// Flatten multi-value headers to comma-separated strings
	result := make(ResponseHeaderMetadata, len(rawHeaders))
	for key, values := range rawHeaders {
		result[key] = strings.Join(values, ", ")
	}

	return []metadata.Structured{result}, nil
}

// NewResponseHeaderMetadataFromHTTPHeader converts an http.Header and status code
// to ResponseHeaderMetadata, flattening multi-value headers to comma-separated strings.
// The status code is included as the "Status Code" key.
func NewResponseHeaderMetadataFromHTTPHeader(header http.Header, statusCode int) ResponseHeaderMetadata {
	if len(header) == 0 && statusCode == 0 {
		return nil
	}
	result := make(ResponseHeaderMetadata, len(header)+1)
	if statusCode != 0 {
		result["Status Code"] = strconv.Itoa(statusCode)
	}
	for key, values := range header {
		result[key] = strings.Join(values, ", ")
	}
	return result
}
