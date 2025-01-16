package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/sashabaranov/go-openai"
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
	// Tools represents the tools available in the request
	Tools []ToolUseRequest `json:"tools,omitempty"`
	// ToolChoice indicates whether the model should automatically choose to use tools,
	// is being forced to use a tool, or is being forced to use a _specific_ tool.
	ToolChoice string `json:"tool_choice,omitempty"`
	// AgentName represents the name of the agent that made this request, if
	// provided as a request header.
	AgentName string `json:"agent_name"`
}

// ParsedInferenceResponse represents the parsed output for a given inference request.
type ParsedInferenceResponse struct {
	ID         string            `json:"id"`
	TokensIn   int32             `json:"tokens_in"`
	TokensOut  int32             `json:"tokens_out"`
	StopReason string            `json:"stop_reason,omitempty"`
	Tools      []ToolUseResponse `json:"tools,omitempty"`
	// XXX: We do not yet extract content, just like we do not yet extract prompts.
	Error string `json:"error,omitempty"`
}

type Choice struct {
}

// ToolUseRequest represents a tool provided to a model in the request.
type ToolUseRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolResponse indicates that the model is asking to invoke a specific tool, using the
// given ID to track the tool's output.
type ToolUseResponse struct {
	// ID represents the ID assigned to this tool
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
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
		rf := OpenAIChatCompletionRequest{}
		if err := json.Unmarshal(req.Body, &rf); err != nil {
			return ParsedInferenceRequest{}, err
		}

		// Parse the tool choice.
		var toolChoice string
		switch t := rf.ToolChoice.(type) {
		case string:
			toolChoice = t
		case openai.ToolChoice:
			toolChoice = t.Function.Name
		}

		// Parse the tools.
		tools := make([]ToolUseRequest, len(rf.Tools))
		for i, t := range rf.Tools {
			if t.Function == nil {
				continue
			}

			var params json.RawMessage
			switch typ := t.Function.Parameters.(type) {
			case json.RawMessage:
				params = typ
			case []byte:
				params = typ
			case string:
				params = json.RawMessage(typ)
			}

			tools[i] = ToolUseRequest{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  params,
			}
		}

		return ParsedInferenceRequest{
			Model:               rf.Model,
			Seed:                rf.Seed,
			Temprature:          rf.Temperature,
			TopP:                rf.TopP,
			MaxTokens:           rf.MaxTokens,
			MaxCompletionTokens: rf.MaxCompletionTokens,
			StopSequences:       rf.Stop,
			ToolChoice:          toolChoice,
			Tools:               tools,
		}, nil
	}

}

func ParseOutput(ctx context.Context, format string, response []byte) (ParsedInferenceResponse, error) {
	switch format {
	case FormatAnthropic:
		r := anthropic.MessagesResponse{}
		err := json.Unmarshal(response, &r)
		if err != nil {
			return ParsedInferenceResponse{}, fmt.Errorf("error parsing openai response: %w", err)
		}

		if r.Type == anthropic.MessagesResponseTypeError {
			r := anthropic.ErrorResponse{}
			err := json.Unmarshal(response, &r)
			if err != nil {
				return ParsedInferenceResponse{Error: "anthropic API error"}, fmt.Errorf("error parsing openai response: %w", err)
			}
			msg := "anthropic API error"
			if r.Error != nil {
				msg = string(r.Error.Type)
			}
			return ParsedInferenceResponse{
				Error: msg,
			}, fmt.Errorf("anthropic api error: %s", msg)
		}

		tools := []ToolUseResponse{}
		for _, m := range r.Content {
			switch m.Type {
			case "text":
				// ignore, for now
			case "tool_use":
				tools = append(tools, ToolUseResponse{
					ID:        m.ID,
					Name:      m.Name,
					Arguments: string(m.Input),
				})
			}
		}

		return ParsedInferenceResponse{
			ID:         r.ID,
			TokensIn:   int32(r.Usage.InputTokens),
			TokensOut:  int32(r.Usage.OutputTokens),
			StopReason: string(r.StopReason),
			Tools:      tools,
		}, nil
	case FormatOpenAIChat:
		fallthrough
	default:
		// OpenAI Chat is the default format, so fall through to the default.
		r := openai.ChatCompletionResponse{}
		err := json.Unmarshal(response, &r)
		if err != nil {
			return ParsedInferenceResponse{}, fmt.Errorf("error parsing openai response: %w", err)
		}

		if len(r.Choices) == 0 {
			return ParsedInferenceResponse{
				ID:        r.ID,
				TokensIn:  int32(r.Usage.PromptTokens),
				TokensOut: int32(r.Usage.CompletionTokens),
			}, fmt.Errorf("no choices returned in openai api response")
		}

		choice := r.Choices[0]
		// XXX: We do not support n>1 in OpenAI requests just yet

		tools := make([]ToolUseResponse, len(choice.Message.ToolCalls))
		for i, t := range choice.Message.ToolCalls {
			tools[i] = ToolUseResponse{
				ID:        t.ID,
				Name:      t.Function.Name,
				Arguments: t.Function.Arguments,
			}
		}

		return ParsedInferenceResponse{
			ID:         r.ID,
			TokensIn:   int32(r.Usage.PromptTokens),
			TokensOut:  int32(r.Usage.CompletionTokens),
			StopReason: string(choice.FinishReason),
			Tools:      tools,
		}, nil
	}
}
