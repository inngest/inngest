package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
)

type Client struct {
	*http.Client
	*testing.T

	APIHost string
}

func New(t *testing.T) *Client {
	return &Client{
		Client:  &http.Client{},
		T:       t,
		APIHost: "http://127.0.0.1:8288",
	}
}

func (c *Client) NewRequest(method string, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(
		method,
		fmt.Sprintf("%s%s", c.APIHost, path),
		body,
	)
}

func (c *Client) MustDoGQL(ctx context.Context, input graphql.RawParams) *graphql.Response {
	resp, err := c.DoGQL(ctx, input)
	if err != nil {
		c.Fatal(err.Error())
	}

	return resp
}

func (c *Client) DoGQL(ctx context.Context, input graphql.RawParams) (*graphql.Response, error) {
	c.Helper()

	resp := c.doGQL(ctx, input)
	if len(resp.Errors) > 0 {
		str := make([]string, len(resp.Errors))
		for i := 0; i < len(resp.Errors); i++ {
			str[i] = resp.Errors[i].Message
		}
		return nil, fmt.Errorf("err with gql: %#v", strings.Join(str, ", "))
	}

	return resp, nil
}

func (c *Client) doGQL(ctx context.Context, input graphql.RawParams) *graphql.Response {
	c.Helper()

	buf := jsonBuffer(input)
	req, err := c.NewRequest(http.MethodPost, "/v0/gql", buf)
	if err != nil {
		c.Fatal(err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	for headerKey, headerValues := range input.Headers {
		for _, headerValue := range headerValues {
			req.Header.Set(headerKey, headerValue)
		}
	}

	resp, err := c.Do(req)
	if err != nil {
		c.Fatal(err.Error())
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		c.Fatalf("invalid gql status code: %d\n\t%s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)

	r := &graphql.Response{}
	if err = json.Unmarshal(body, &r); err != nil {
		c.Fatal(err.Error())
	}

	return r
}

func jsonBuffer(input any) io.Reader {
	byt, err := json.Marshal(input)
	if err != nil {
		panic(fmt.Errorf("unable to marshal input: %w", err))
	}
	return bytes.NewBuffer(byt)
}
