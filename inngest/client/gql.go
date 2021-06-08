package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/inngest/inngestctl/inngest/log"
)

// DoGQL makes a gql request and returns the response
func (c httpClient) DoGQL(ctx context.Context, input Params) (*Response, error) {
	buf := jsonBuffer(ctx, input)

	req, err := c.NewRequest(http.MethodPost, "/gql", buf)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", "inngestctl")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", c.creds))

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("invalid status code %d: %s", resp.StatusCode, string(body))
	}

	r := &Response{}
	if err = json.NewDecoder(resp.Body).Decode(r); err != nil {
		return nil, fmt.Errorf("invalid json response: %w", err)
	}

	if len(r.Errors) > 0 {
		str := make([]string, len(r.Errors))
		for i := 0; i < len(r.Errors); i++ {
			str[i] = r.Errors[i].Message
		}
		return nil, fmt.Errorf("%s", strings.Join(str, ", "))
	}

	return r, nil
}

type Params struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

type Response struct {
	Data   json.RawMessage `json:"data"`
	Errors ErrorList       `json:"errors,omitempty"`
}

type ErrorList []*Error

type Error struct {
	Message string   `json:"message"`
	Path    []string `json:"path,omitempty"`
}

func (c *httpClient) NewRequest(method string, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, fmt.Sprintf("%s%s", c.api, path), body)
}

func jsonBuffer(ctx context.Context, input interface{}) io.Reader {
	byt, err := json.Marshal(input)
	if err != nil {
		log.From(ctx).Fatal().Err(err).Msg("unable to marshal input")
	}
	return bytes.NewBuffer(byt)
}
