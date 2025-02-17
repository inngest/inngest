package anthropic

import (
	"context"
	"net/http"
)

type CountTokensResponse struct {
	httpHeader

	InputTokens int `json:"input_tokens"`
}

func (c *Client) CountTokens(
	ctx context.Context,
	request MessagesRequest,
) (response CountTokensResponse, err error) {
	var setters []requestSetter
	if len(c.config.BetaVersion) > 0 {
		setters = append(setters, withBetaVersion(c.config.BetaVersion...))
	}

	urlSuffix := "/messages/count_tokens"
	req, err := c.newRequest(ctx, http.MethodPost, urlSuffix, &request, setters...)
	if err != nil {
		return
	}

	err = c.sendRequest(req, &response)
	return
}
