package anthropic

import (
	"context"
	"net/http"
)

type CompleteRequest struct {
	Model             Model  `json:"model"`
	Prompt            string `json:"prompt"`
	MaxTokensToSample int    `json:"max_tokens_to_sample"`

	StopSequences []string       `json:"stop_sequences,omitempty"`
	Temperature   *float32       `json:"temperature,omitempty"`
	TopP          *float32       `json:"top_p,omitempty"`
	TopK          *int           `json:"top_k,omitempty"`
	MetaData      map[string]any `json:"meta_data,omitempty"`
	Stream        bool           `json:"stream,omitempty"`
}

func (c *CompleteRequest) SetTemperature(t float32) {
	c.Temperature = &t
}

func (c *CompleteRequest) SetTopP(p float32) {
	c.TopP = &p
}

func (c *CompleteRequest) SetTopK(k int) {
	c.TopK = &k
}

type CompleteResponse struct {
	httpHeader

	Type       string `json:"type"`
	ID         string `json:"id"`
	Completion string `json:"completion"`
	// possible values are: stop_sequence、max_tokens、null
	StopReason string `json:"stop_reason"`
	Model      Model  `json:"model"`
}

func (c *Client) CreateComplete(
	ctx context.Context,
	request CompleteRequest,
) (response CompleteResponse, err error) {
	request.Stream = false

	urlSuffix := "/complete"
	req, err := c.newRequest(ctx, http.MethodPost, urlSuffix, &request)
	if err != nil {
		return
	}

	err = c.sendRequest(req, &response)
	return
}
