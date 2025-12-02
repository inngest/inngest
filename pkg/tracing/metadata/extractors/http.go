package extractors

import (
	"context"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/util"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

const (
	KindInngestHTTP metadata.Kind = "inngest.http"
)

type HTTPMetadata struct {
	ResponseContentType *string `json:"response_content_type,omitempty"`
	RequestContentType  *string `json:"request_content_type,omitempty"`
	Method              string  `json:"method"`
	RequestSize         *int64  `json:"request_size,omitempty"`
	ResponseSize        *int64  `json:"response_size,omitempty"`
	ResponseStatus      int64   `json:"response_status"`
	Domain              *string `json:"domain,omitempty"`
	Path                *string `json:"path,omitempty"`
}

func (m HTTPMetadata) Kind() metadata.Kind {
	return KindInngestHTTP
}

func (m HTTPMetadata) Op() metadata.Opcode {
	return enums.MetadataOpcodeMerge
}

func (m HTTPMetadata) Serialize() (metadata.Values, error) {
	var rawMetadata metadata.Values
	err := rawMetadata.FromStruct(m)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}

type HTTPMetadataExtractor struct{}

func NewHTTPMetadataExtractor() *HTTPMetadataExtractor {
	return &HTTPMetadataExtractor{}
}

func (e *HTTPMetadataExtractor) ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]metadata.Structured, error) {
	if !e.isLikelyHTTPSpan(span) {
		return nil, nil
	}

	httpMetadata := e.extractHTTPMetadata(span)
	return []metadata.Structured{httpMetadata}, nil
}

var httpAttributeKeys = map[string]bool{
	"http.request.header.content-type":  true,
	"http.response.header.content-type": true,
	"http.request.method":               true,
	"http.request.size":                 true,
	"http.response.size":                true,
	"http.response.status_code":         true,
}

func (e *HTTPMetadataExtractor) isLikelyHTTPSpan(span *tracev1.Span) bool {
	for _, attr := range span.Attributes {
		if httpAttributeKeys[attr.Key] {
			return true
		}
	}
	return false
}

func (e *HTTPMetadataExtractor) extractHTTPMetadata(span *tracev1.Span) HTTPMetadata {
	var metadata HTTPMetadata

	for _, attr := range span.Attributes {
		switch attr.Key {
		case "http.request.header.content-type":
			metadata.RequestContentType = util.ToPtr(attr.Value.GetStringValue())
		case "http.response.header.content-type":
			metadata.ResponseContentType = util.ToPtr(attr.Value.GetStringValue())
		case "http.request.method":
			metadata.Method = attr.Value.GetStringValue()
		case "http.request.size":
			metadata.RequestSize = util.ToPtr(attr.Value.GetIntValue())
		case "http.response.size":
			metadata.ResponseSize = util.ToPtr(attr.Value.GetIntValue())
		case "http.response.status_code":
			metadata.ResponseStatus = attr.Value.GetIntValue()
		case "url.domain":
			metadata.Domain = util.ToPtr(attr.Value.GetStringValue())
		case "server.address":
			metadata.Domain = util.ToPtr(attr.Value.GetStringValue())
		case "url.path":
			metadata.Path = util.ToPtr(attr.Value.GetStringValue())
		}
	}

	return metadata
}
