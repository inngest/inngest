package aigateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	// NOTE: We don't use the default `openai` package because Stainless SDKs don't
	// support Unmarshal() on Param structs, due to their Field handling.
	// See https://play.golang.com/p/UX60snrf3gp for more info.
	openai "github.com/sashabaranov/go-openai"
)

const (
	// FormatOpenAIChat represents the default OpenAI chat completion request.
	FormatOpenAIChat = "openai-chat"
	FormatAnthropic  = "anthropic"
	FormatGemini     = "gemini"
	FormatBedrock    = "bedrock"
)

type Request struct {
	// URL is the full endpoint that we're sending the request to.  This must
	// always be provided by our SDKs.
	URL string `json:"url,omitempty"`
	// Headers represent additional headers to send in the request.
	Headers map[string]string `json:"headers,omitempty"`
	// AuthKey is an API key to be sent with the request.  This contains
	// API tokens which are never logged.
	AuthKey string `json:"auth_key,omitempty"`
	// AutoToolCall indicates whether the request should automatically invoke functions
	// when using inngest functions as tools.  This allows us to immediately execute without
	// round trips.
	AutoToolCall bool `json:"auto_tool_call"`
	// Format represents the request format type, eg. an OpenAI compatible endpoint
	// request, or a Groq request.
	Format string `json:"format"`
	// Body indicates the raw content of the request, as a slice of JSON bytes.
	// It's expected that this comes from our SDKs directly.
	Body json.RawMessage `json:"body"`
}

func (r Request) MarshalJSON() ([]byte, error) {
	// Do not allow this to be marshalled.  We do not want the auth creds to
	// be logged.
	return nil, nil
}

func (r Request) HTTPRequest() (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, r.URL, bytes.NewReader(r.Body))
	if err != nil {
		return nil, err
	}

	// Always sending JSON.
	req.Header.Add("content-type", "application/json")

	// Add auth, depending on the format.
	switch r.Format {
	case FormatGemini:
		// Gemini adds the auth key as a query param
		values := req.URL.Query()
		values.Add("key", r.AuthKey)
		req.URL.RawQuery = values.Encode()
	case FormatBedrock:
		// Bedrock's auth should be the fully-generated AWS key derived from the
		// secret and signing key.
		req.Header.Add("Authorization", r.AuthKey)
	case FormatAnthropic:
		// Anthropic uses a non-standard header.
		req.Header.Add("x-api-key", r.AuthKey)
	default:
		// By default, use standards.
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.AuthKey))
	}

	// Overwrite any headers if custom headers are added to opts.
	for header, val := range r.Headers {
		req.Header.Add(header, val)
	}

	return req, nil
}

type (
	// OpenAIChatCompletionRequest represents an OpenAI compatible format.
	OpenAIChatCompletionRequest openai.ChatCompletionRequest
)
