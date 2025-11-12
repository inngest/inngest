package extractors

import (
	"context"

	"github.com/inngest/inngest/pkg/tracing/meta"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

type AIMetadata struct {
	InputTokens   int64  `json:"input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	Model         string `json:"model"`
	System        string `json:"system"`
	OperationName string `json:"operation_name"`
}

func (ms AIMetadata) Kind() meta.MetadataKind {
	return "inngest.ai"
}

func (ms AIMetadata) Op() meta.MetadataOp {
	return meta.MetadataOpMerge
}

func (ms AIMetadata) Serialize() (meta.RawMetadata, error) {
	var rawMetadata meta.RawMetadata
	err := rawMetadata.FromStruct(ms)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}

type AITokenExtractor struct{}

func NewAITokenExtractor() *AITokenExtractor {
	return &AITokenExtractor{}
}

func (e *AITokenExtractor) ExtractMetadata(ctx context.Context, span *tracev1.Span) ([]meta.StructuredMetadata, error) {
	if !e.isLikelyAISpan(span) {
		return nil, nil // TODO: should this be an explicit "nah, didn't find any" return?
	}

	aiMetadata := e.extractAIMetadata(span)
	return []meta.StructuredMetadata{aiMetadata}, nil
}

var aiAttributeKeys = map[string]bool{
	"gen_ai.usage.input_tokens":  true,
	"gen_ai.usage.output_tokens": true,
	"gen_ai.request.model":       true,
	"gen_ai.system":              true,
	"gen_ai.operation.name":      true,
}

func (e *AITokenExtractor) isLikelyAISpan(span *tracev1.Span) bool {
	for _, attr := range span.Attributes {
		if aiAttributeKeys[attr.Key] {
			return true
		}
	}
	return false
}

func (e *AITokenExtractor) extractAIMetadata(span *tracev1.Span) AIMetadata {
	var metadata AIMetadata

	for _, attr := range span.Attributes {
		switch attr.Key {
		case "gen_ai.usage.input_tokens":
			metadata.InputTokens = attr.Value.GetIntValue()
		case "gen_ai.usage.output_tokens":
			metadata.OutputTokens = attr.Value.GetIntValue()
		case "gen_ai.request.model":
			metadata.Model = attr.Value.GetStringValue()
		case "gen_ai.system":
			metadata.System = attr.Value.GetStringValue()
		case "gen_ai.operation.name":
			metadata.OperationName = attr.Value.GetStringValue()
		}
	}

	return metadata
<<<<<<< HEAD:pkg/tracing/meta/extractors/ai_tokens.go
}
=======
}
>>>>>>> 52996a499 (tweak):pkg/tracing/meta/extractors/ai.go
