package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// ParsedRequest represents the parsed request data for a given inference request.
//
// Note that this is not stored, and instead is computed just in time for each input
// depending on the UI.
type ParsedInferenceRequest struct {
	URL                 url.URL  `json:"url"`
	Model               string   `json:"model"`
	Seed                *int     `json:"seed,omitempty"`
	Temprature          float32  `json:"temperature,omitempty"`
	TopP                float32  `json:"top_p,omitempty"`
	MaxTokens           int      `json:"max_tokens,omitempty"`
	MaxCompletionTokens int      `json:"max_completion_tokens,omitempty"`
	StopSequences       []string `json:"stop,omitempty"`
	// TODO: Input messages and so on.
	// TODO: Tools
}

// ParsedInferenceResponse represents the parsed output for a given inference request.
type ParsedInferenceResponse struct {
	ID         string `json:"id"`
	TokensIn   int32  `json:"tokens_in"`
	TokensOut  int32  `json:"tokens_out"`
	StopReason string `json:"stop_reason"`
	// TODO: Tool use selections, parsed.
}

func ParseUnknownInput(ctx context.Context, req json.RawMessage) (ParsedInferenceRequest, error) {
	return ParsedInferenceRequest{}, fmt.Errorf("todo")
}

// Parse validates an inference request.  This checks which model, format, and URLs we should
// use for the given request which is passed through to the end provider.
//
// By default, we support the following:
//
// * OpenAI (and OpenAI compatible URLs, by changing the base URL)
// * Anthropic
// * Bedrock
// * Google Generative AI
// * Mistral
// * Cohere
// * Groq (must include URL)
func ParseInput(ctx context.Context, req Request) (ParsedInferenceRequest, error) {
	switch req.Format {
	case FormatOpenAIChat:
		// OpenAI Chat is the default format, so fall through to the default.
		fallthrough
	default:
		// Parse everything as an OpenAI Chat request.
		rf := RFOpenAIChatCompletion{}
		if err := json.Unmarshal(req.Body, &rf); err != nil {
			return ParsedInferenceRequest{}, err
		}

		return ParsedInferenceRequest{
			Model:               rf.Model,
			Seed:                rf.Seed,
			Temprature:          rf.Temperature,
			TopP:                rf.TopP,
			MaxTokens:           rf.MaxTokens,
			MaxCompletionTokens: rf.MaxCompletionTokens,
			StopSequences:       rf.Stop,
		}, nil
	}

}
