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

	"github.com/inngest/inngest/inngest/version"
)

// DoGQL makes a gql request and returns the response
func (c httpClient) DoGQL(ctx context.Context, input Params) (*Response, error) {
	buf := jsonBuffer(ctx, input)

	req, err := c.NewRequest(http.MethodPost, "/gql", buf)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("content-type", "application/json")
	req.Header.Set("user-agent", fmt.Sprintf("inngestctl-%s-%s", version.Version, version.Hash))
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

	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	r := &Response{}
	if err = json.Unmarshal(byt, r); err != nil {
		return nil, fmt.Errorf("invalid json response: %w", err)
	}

	if len(r.Errors) > 0 {
		return nil, r.Errors
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

func (e ErrorList) Error() string {
	str := make([]string, len(e))
	for i := 0; i < len(e); i++ {
		str[i] = e[i].Message
	}
	return strings.Join(str, ", ")
}

type Error struct {
	Message string   `json:"message"`
	Path    []string `json:"path,omitempty"`
}

func (e Error) Error() string {
	return e.Message
}

func (c *httpClient) NewRequest(method string, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", c.api, path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "inngestctl")
	return req, nil
}

func jsonBuffer(ctx context.Context, input interface{}) io.Reader {
	byt, err := json.Marshal(input)
	if err != nil {
		panic(err.Error())
	}
	return bytes.NewBuffer(byt)
}
