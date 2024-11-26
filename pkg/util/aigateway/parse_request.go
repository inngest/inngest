package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

var (
	errInvalidFormat = fmt.Errorf("invalid api format")

	inputParser = []func(json.RawMessage) (ParsedInferenceRequest, error){
		parseOpenAIRequest,
		parseAnthropicRequest,
		parseVercelGenerateTextArgs,
	}
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
	TopK                int      `json:"top_k,omitempty"`
	MaxTokens           int      `json:"max_tokens,omitempty"`
	MaxCompletionTokens int      `json:"max_completion_tokens,omitempty"`
	StopSequences       []string `json:"stop,omitempty"`
	// TODO: Input messages and so on.
	// TODO: Tools
}

// ParseUnknownInput attempts to parse the given request without prior knowledge
// of the request format.  This is used when wrapping AI calls.
//
// Note that the input here may be a request body, or it may be the arguments passed
// to a JS function.  Because of this, there are multiple formats per model API that
// we need to hit.
func ParseUnknownInput(ctx context.Context, req json.RawMessage) (ParsedInferenceRequest, error) {
	for _, parser := range inputParser {
		if parsed, err := parser(req); err == nil {
			return parsed, nil
		}
	}
	return ParsedInferenceRequest{}, fmt.Errorf("unable to parse input")
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
		return parseOpenAIRequest(req.Body)
	default:
		// OpenAI Chat is the default format
		return parseOpenAIRequest(req.Body)
	}

}

// parseOpenAIRequest assumes that the body is a fully-defined OpenAI request.
func parseOpenAIRequest(body json.RawMessage) (ParsedInferenceRequest, error) {
	// Parse everything as an OpenAI Chat request.
	rf := RFOpenAIChatCompletion{}
	if err := json.Unmarshal(body, &rf); err != nil {
		return ParsedInferenceRequest{}, err
	}
	if rf.Model == "" {
		return ParsedInferenceRequest{}, errInvalidFormat
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

func parseAnthropicRequest(body json.RawMessage) (ParsedInferenceRequest, error) {
	return ParsedInferenceRequest{}, errInvalidFormat
}

//
// step.ai.wrap parsing.
//

type vercelGenerateText struct {
	Model  vercelGenerateTextModel
	System string
	Prompt string

	Seed          *int     `json:"seed,omitempty"`
	Temprature    float32  `json:"temperature,omitempty"`
	TopP          float32  `json:"topP,omitempty"`
	TopK          int      `json:"topK,omitempty"`
	MaxTokens     int      `json:"maxTokens,omitempty"`
	StopSequences []string `json:"stopSequences,omitempty"`
}

type vercelGenerateTextModel struct {
	Config               vercelGenerateTextModelConfig
	ModelID              string `json:"modelId"`
	Settings             struct{}
	SpecificationVersion string
}

type vercelGenerateTextModelConfig struct {
	Compatibility string
	Provider      string
}

// parseVercelGenerateTextArgs attempts to parse the `generateText` arguments from
// step.ai.wrap.
func parseVercelGenerateTextArgs(body json.RawMessage) (ParsedInferenceRequest, error) {
	if len(body) <= 2 {
		return ParsedInferenceRequest{}, errInvalidFormat
	}

	var req vercelGenerateText
	if body[0] == '[' {
		// Unwrap this into an array of structs, as the JS SDK sends every argument as an array.
		reqs := []vercelGenerateText{}
		if err := json.Unmarshal(body, &reqs); err != nil || len(reqs) == 0 {
			return ParsedInferenceRequest{}, err
		}
		req = reqs[0]
	} else {
		if err := json.Unmarshal(body, &req); err != nil {
			return ParsedInferenceRequest{}, err
		}
	}

	if req.Model.ModelID == "" {
		return ParsedInferenceRequest{}, errInvalidFormat
	}

	parsed := ParsedInferenceRequest{
		Model:         req.Model.ModelID,
		Seed:          req.Seed,
		Temprature:    req.Temprature,
		TopP:          req.TopP,
		TopK:          req.TopK,
		MaxTokens:     req.MaxTokens,
		StopSequences: req.StopSequences,
	}
	return parsed, nil
}
