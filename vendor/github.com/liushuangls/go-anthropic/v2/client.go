package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	config ClientConfig
}

type Response interface {
	SetHeader(http.Header)
}

type httpHeader http.Header

func (h *httpHeader) SetHeader(header http.Header) {
	*h = httpHeader(header)
}

func (h *httpHeader) Header() http.Header {
	return http.Header(*h)
}

func (h *httpHeader) GetRateLimitHeaders() (RateLimitHeaders, error) {
	return newRateLimitHeaders(h.Header())
}

// NewClient create new Anthropic API client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	return &Client{
		config: newConfig(apiKey, opts...),
	}
}

func (c *Client) sendRequest(req *http.Request, v Response) error {
	res, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	v.SetHeader(res.Header)

	if err := c.handlerRequestError(res); err != nil {
		return err
	}

	if err = json.NewDecoder(res.Body).Decode(v); err != nil {
		return err
	}

	return nil
}

func (c *Client) handlerRequestError(resp *http.Response) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error, reading response body: %w", err)
		}

		// use the adapter to translate the error, if it can
		if err, handled := c.config.Adapter.TranslateError(resp, body); handled {
			return err
		}

		var errRes ErrorResponse
		err = json.Unmarshal(body, &errRes)
		if err != nil || errRes.Error == nil {
			reqErr := RequestError{
				StatusCode: resp.StatusCode,
				Err:        err,
				Body:       body,
			}
			return &reqErr
		}

		return fmt.Errorf("error, status code: %d, message: %w", resp.StatusCode, errRes.Error)
	}
	return nil
}

type requestSetter func(req *http.Request)

func withBetaVersion(betaVersion ...BetaVersion) requestSetter {
	version := ""
	for i, v := range betaVersion {
		version += string(v)
		if i < len(betaVersion)-1 {
			version += ","
		}
	}

	return func(req *http.Request) {
		req.Header.Set("anthropic-beta", version)
	}
}

func (c *Client) newRequest(
	ctx context.Context,
	method, urlSuffix string,
	body any,
	requestSetters ...requestSetter,
) (req *http.Request, err error) {

	// prepare the request
	var fullURL string
	fullURL, err = c.config.Adapter.PrepareRequest(c, method, urlSuffix, body)
	if err != nil {
		return nil, err
	}

	var reqBody []byte
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err = http.NewRequestWithContext(
		ctx,
		method,
		fullURL,
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	// set any provider-specific headers (including Authorization)
	c.config.Adapter.SetRequestHeaders(c, req)

	for _, setter := range requestSetters {
		setter(req)
	}

	return req, nil
}

func (c *Client) newStreamRequest(
	ctx context.Context,
	method, urlSuffix string,
	body any,
	requestSetters ...requestSetter,
) (req *http.Request,
	err error) {
	req, err = c.newRequest(ctx, method, urlSuffix, body, requestSetters...)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	return req, nil
}
