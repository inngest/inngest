package anthropic

import (
	"fmt"
	"net/http"
)

var _ ClientAdapter = (*DefaultAdapter)(nil)

type DefaultAdapter struct {
}

func (v *DefaultAdapter) TranslateError(resp *http.Response, body []byte) (error, bool) {
	return nil, false
}

func (v *DefaultAdapter) fullURL(baseUrl string, suffix string) string {
	// replace the first slash with a colon
	return fmt.Sprintf("%s%s", baseUrl, suffix)
}

func (v *DefaultAdapter) PrepareRequest(
	c *Client,
	method string,
	urlSuffix string,
	body any,
) (string, error) {
	return v.fullURL(c.config.BaseURL, urlSuffix), nil
}

func (v *DefaultAdapter) SetRequestHeaders(c *Client, req *http.Request) error {
	req.Header.Set("X-Api-Key", c.config.GetApiKey())
	req.Header.Set("Anthropic-Version", string(c.config.APIVersion))

	return nil
}
