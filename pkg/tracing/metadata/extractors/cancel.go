package extractors

import (
	"context"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

const KindInngestCancel metadata.Kind = "inngest.cancel"

type CancelMetadata struct {
	Reason string `json:"reason"`
}

func (m CancelMetadata) Kind() metadata.Kind { return KindInngestCancel }
func (m CancelMetadata) Op() metadata.Opcode { return enums.MetadataOpcodeSet }
func (m CancelMetadata) Serialize() (metadata.Values, error) {
	var v metadata.Values
	return v, v.FromStruct(m)
}

type CancelMetadataExtractor struct{}

func NewCancelMetadataExtractor() *CancelMetadataExtractor {
	return &CancelMetadataExtractor{}
}

func (e *CancelMetadataExtractor) ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]metadata.Structured, error) {
	for _, attr := range span.Attributes {
		if attr.Key == consts.OtelSysFunctionCancelReason {
			return []metadata.Structured{
				CancelMetadata{Reason: attr.Value.GetStringValue()},
			}, nil
		}
	}
	return nil, nil
}