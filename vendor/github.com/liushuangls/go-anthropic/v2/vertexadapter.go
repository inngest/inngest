package anthropic

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var _ ClientAdapter = (*VertexAdapter)(nil)

type VertexAdapter struct {
}

func (v *VertexAdapter) TranslateError(resp *http.Response, body []byte) (error, bool) {
	switch resp.StatusCode {
	case http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusTooManyRequests:
		var errRes VertexAIErrorResponse
		err := json.Unmarshal(body, &errRes)
		if err != nil {
			// it could be an array
			var errResArr []VertexAIErrorResponse
			err = json.Unmarshal(body, &errResArr)
			if err == nil && len(errResArr) > 0 {
				errRes = errResArr[0]
			}
		}

		if err != nil || errRes.Error == nil {
			reqErr := RequestError{
				StatusCode: resp.StatusCode,
				Err:        err,
				Body:       body,
			}
			return &reqErr, true
		}
		return fmt.Errorf(
			"error, status code: %d, message: %w",
			resp.StatusCode,
			errRes.Error,
		), true
	}
	return nil, false
}

func (v *VertexAdapter) fullURL(baseUrl string, suffix string, model Model) string {
	// replace the first slash with a colon
	return fmt.Sprintf("%s/%s:%s", baseUrl, model.asVertexModel(), suffix[1:])
}

func (v *VertexAdapter) translateUrlSuffix(suffix string, stream bool) (string, error) {
	switch suffix {
	case "/messages":
		if stream {
			return ":streamRawPredict", nil
		} else {
			return ":rawPredict", nil
		}
	}

	return "", fmt.Errorf("unknown suffix: %s", suffix)
}

func (v *VertexAdapter) PrepareRequest(
	c *Client,
	method string,
	urlSuffix string,
	body any,
) (string, error) {
	// if the body implements the ModelGetter interface, use the model from the body
	model := Model("")
	if body != nil {
		if vertexAISupport, ok := body.(VertexAISupport); ok {
			model = vertexAISupport.GetModel()
			vertexAISupport.SetAnthropicVersion(c.config.APIVersion)

			var err error
			urlSuffix, err = v.translateUrlSuffix(urlSuffix, vertexAISupport.IsStreaming())
			if err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("this call is not supported by the Vertex AI API")
		}
	}

	return v.fullURL(c.config.BaseURL, urlSuffix, model), nil
}

func (v *VertexAdapter) SetRequestHeaders(c *Client, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+c.config.GetApiKey())
	return nil
}
