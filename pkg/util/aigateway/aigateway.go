package aigateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/sashabaranov/go-openai"
)

// ParsedRequest represents the parsed request data for a given inference request.
//
// Note that this is not stored, and instead is computed just in time for each input
// depending on the UI.
type ParsedInferenceRequest struct {
	URL                 string   `json:"url"`
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
	// Model is the model that actually served the request, as reported by the
	// provider. This may differ from the requested model (e.g. a dated snapshot).
	Model      string            `json:"model,omitempty"`
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
func ParseInput(req Request) (ParsedInferenceRequest, error) {
	switch req.Format {
	case FormatAnthropic:
		return parseAnthropicInput(req)
	case FormatOpenAIChat:
		// OpenAI Chat is the default format, so fall through to the default.
		fallthrough
	default:
		// Parse everything else as an OpenAI Chat request.
		return parseOpenAIInput(req)
	}
}

func parseOpenAIInput(req Request) (ParsedInferenceRequest, error) {
	rf := OpenAIChatCompletionRequest{}
	if err := json.Unmarshal(req.Body, &rf); err != nil {
		return ParsedInferenceRequest{}, err
	}

	// Parse the tool choice.  ToolChoice is an `any`, so JSON decodes it as
	// either a string ("auto", "none", "required") or a map when a specific
	// function is forced: {"type":"function","function":{"name":"..."}}.
	var toolChoice string
	switch t := rf.ToolChoice.(type) {
	case string:
		toolChoice = t
	case map[string]any:
		if fn, ok := t["function"].(map[string]any); ok {
			toolChoice, _ = fn["name"].(string)
		}
	}

	// Parse the tools.
	tools := make([]ToolUseRequest, 0, len(rf.Tools))
	for _, t := range rf.Tools {
		if t.Function == nil {
			continue
		}

		tools = append(tools, ToolUseRequest{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			Parameters:  rawJSON(t.Function.Parameters),
		})
	}

	return ParsedInferenceRequest{
		URL:                 req.URL,
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

// anthropicRequest is the subset of an Anthropic Messages request that we
// record.  It differs from the OpenAI format: tools are flat with an
// input_schema, stop sequences are named stop_sequences, and a forced tool is
// {"type":"tool","name":"..."}.
type anthropicRequest struct {
	Model         string               `json:"model"`
	MaxTokens     int                  `json:"max_tokens"`
	Temperature   float32              `json:"temperature"`
	TopP          float32              `json:"top_p"`
	StopSequences []string             `json:"stop_sequences"`
	Tools         []anthropicTool      `json:"tools"`
	ToolChoice    *anthropicToolChoice `json:"tool_choice"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func parseAnthropicInput(req Request) (ParsedInferenceRequest, error) {
	ar := anthropicRequest{}
	if err := json.Unmarshal(req.Body, &ar); err != nil {
		return ParsedInferenceRequest{}, err
	}

	// Mirror the OpenAI behaviour: record the forced tool's name, otherwise
	// the choice mode ("auto" or "any").
	var toolChoice string
	if ar.ToolChoice != nil {
		toolChoice = ar.ToolChoice.Type
		if ar.ToolChoice.Type == "tool" {
			toolChoice = ar.ToolChoice.Name
		}
	}

	tools := make([]ToolUseRequest, 0, len(ar.Tools))
	for _, t := range ar.Tools {
		tools = append(tools, ToolUseRequest{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		})
	}

	return ParsedInferenceRequest{
		URL:           req.URL,
		Model:         ar.Model,
		Temprature:    ar.Temperature,
		TopP:          ar.TopP,
		MaxTokens:     ar.MaxTokens,
		StopSequences: ar.StopSequences,
		ToolChoice:    toolChoice,
		Tools:         tools,
	}, nil
}

// rawJSON converts a tool's parameter schema to raw JSON.  The schema is an
// `any` in the OpenAI types, so decoded requests hold a map rather than raw
// bytes.
func rawJSON(v any) json.RawMessage {
	switch p := v.(type) {
	case nil:
		return nil
	case json.RawMessage:
		return p
	case []byte:
		return p
	case string:
		return json.RawMessage(p)
	default:
		b, err := json.Marshal(p)
		if err != nil {
			return nil
		}
		return b
	}
}

func ParseOutput(format string, response []byte) (ParsedInferenceResponse, error) {
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
			Model:      string(r.Model),
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
				Model:     r.Model,
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
			Model:      r.Model,
			TokensIn:   int32(r.Usage.PromptTokens),
			TokensOut:  int32(r.Usage.CompletionTokens),
			StopReason: string(choice.FinishReason),
			Tools:      tools,
		}, nil
	}
}
