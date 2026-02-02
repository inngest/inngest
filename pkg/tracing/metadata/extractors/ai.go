package extractors

import (
	"context"
	"net/http"
	"net/url"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/util"
	"github.com/inngest/inngest/pkg/util/aigateway"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
)

//tygo:generate
const (
	KindInngestAI metadata.Kind = "inngest.ai"
)

//tygo:generate
type AIMetadata struct {
	InputTokens   int64  `json:"input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	Model         string `json:"model"`
	System        string `json:"system"`
	OperationName string `json:"operation_name"`
}

func (ms AIMetadata) Kind() metadata.Kind {
	return KindInngestAI
}

func (ms AIMetadata) Op() metadata.Opcode {
	return enums.MetadataOpcodeMerge
}

func (ms AIMetadata) Serialize() (metadata.Values, error) {
	var rawMetadata metadata.Values
	err := rawMetadata.FromStruct(ms)
	if err != nil {
		return nil, err
	}

	return rawMetadata, nil
}

type AIMetadataExtractor struct{}

func NewAIMetadataExtractor() *AIMetadataExtractor {
	return &AIMetadataExtractor{}
}

func (e *AIMetadataExtractor) ExtractSpanMetadata(ctx context.Context, span *tracev1.Span) ([]metadata.Structured, error) {
	if !e.isLikelyAISpan(span) {
		return nil, nil // TODO: should this be an explicit "nah, didn't find any" return?
	}

	aiMetadata := e.extractAIMetadata(span)
	return []metadata.Structured{aiMetadata}, nil
}

var aiAttributeKeys = map[string]bool{
	"gen_ai.usage.input_tokens":  true,
	"gen_ai.usage.output_tokens": true,
	"gen_ai.request.model":       true,
	"gen_ai.system":              true,
	"gen_ai.operation.name":      true,
}

func (e *AIMetadataExtractor) isLikelyAISpan(span *tracev1.Span) bool {
	for _, attr := range span.Attributes {
		if aiAttributeKeys[attr.Key] {
			return true
		}
	}
	return false
}

func (e *AIMetadataExtractor) extractAIMetadata(span *tracev1.Span) AIMetadata {
	var md AIMetadata

	for _, attr := range span.Attributes {
		switch attr.Key {
		case "gen_ai.usage.input_tokens":
			md.InputTokens = attr.Value.GetIntValue()
		case "gen_ai.usage.output_tokens":
			md.OutputTokens = attr.Value.GetIntValue()
		case "gen_ai.request.model":
			md.Model = attr.Value.GetStringValue()
		case "gen_ai.system":
			md.System = attr.Value.GetStringValue()
		case "gen_ai.operation.name":
			md.OperationName = attr.Value.GetStringValue()
		}
	}

	return md
}

func ExtractAIGatewayMetadata(req aigateway.Request, respStatus int, resp []byte) ([]metadata.Structured, error) {
	parsedInput, err := aigateway.ParseInput(req)
	if err != nil {
		return nil, &metadata.WarningError{
			Key: "inngest.ai.request.parsing.failed",
			Err: err,
		}
	}

	u, err := url.Parse(parsedInput.URL)
	if err != nil {
		return nil, &metadata.WarningError{
			Key: "inngest.ai.request.parsing.failed",
			Err: err,
		}
	}

	parsedOutput, err := aigateway.ParseOutput(req.Format, resp)
	if err != nil {
		return nil, &metadata.WarningError{
			Key: "inngest.ai.response.parsing.failed",
			Err: err,
		}
	}

	return []metadata.Structured{
		&AIMetadata{
			Model:         parsedInput.Model,
			System:        req.Format, // TODO: make sure this is reasonable
			OperationName: "",         // TODO: figure this out

			InputTokens:  int64(parsedOutput.TokensIn),
			OutputTokens: int64(parsedOutput.TokensOut),
		},
		&HTTPMetadata{
			Method:             http.MethodPost,
			Domain:             util.ToPtr(u.Host),
			Path:               util.ToPtr(u.Path),
			RequestContentType: util.ToPtr("application/json"),
			RequestSize:        util.ToPtr(int64(len(req.Body))),

			ResponseContentType: util.ToPtr("application/json"),
			ResponseSize:        util.ToPtr(int64(len(resp))),
			ResponseStatus:      int64(respStatus),
		},
	}, nil
}
