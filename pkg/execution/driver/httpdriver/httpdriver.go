package httpdriver

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
)

var (
	DefaultExecutor = executor{
		client: &http.Client{
			Timeout:       15 * time.Minute,
			CheckRedirect: CheckRedirect,
		},
	}

	ErrEmptyResponse = fmt.Errorf("no response data")
)

func CheckRedirect(req *http.Request, via []*http.Request) (err error) {
	if len(via) > 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	// If we're redirected we want to ensure that we retain the HTTP method.
	req.Method = via[0].Method
	req.Body, err = via[0].GetBody()
	if err != nil {
		return err
	}
	req.ContentLength = via[0].ContentLength
	req.Header = via[0].Header
	return nil
}

func Execute(ctx context.Context, s state.State, action inngest.ActionVersion, edge inngest.Edge, step inngest.Step, idx int) (*state.DriverResponse, error) {
	return DefaultExecutor.Execute(ctx, s, action, edge, step, idx)
}

type executor struct {
	client     *http.Client
	signingKey []byte
}

// RuntimeType fulfiils the inngest.Runtime interface.
func (e executor) RuntimeType() string {
	return "http"
}

// Sign signs the body with a private key, ensuring that HTTP handlers can verify
// that the request comes from us.
func Sign(ctx context.Context, key, body []byte) string {
	if key == nil {
		return ""
	}

	now := time.Now().Unix()
	mac := hmac.New(sha256.New, key)

	_, _ = mac.Write(body)
	// Write the timestamp as a unix timestamp to the hmac to prevent
	// timing attacks.
	_, _ = mac.Write([]byte(fmt.Sprintf("%d", now)))

	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d&s=%s", now, sig)
}

func ParseGenerator(ctx context.Context, byt []byte) ([]*state.GeneratorOpcode, error) {
	// When we return a 206, we always expect that this is
	// a generator function.  Users SHOULD NOT return a 206
	// in any other circumstance.

	if len(byt) == 0 {
		return nil, ErrEmptyResponse
	}

	// Is this a slice of opcodes or a single opcode?  The SDK can return both:
	// parallelism was added as an incremental improvement.  It would have been nice
	// to always return an array and we can enfore this as an SDK requirement in V1+
	switch byt[0] {
	// 0.x.x SDKs return a single opcode.
	case '{':
		gen := &state.GeneratorOpcode{}
		if err := json.Unmarshal(byt, gen); err != nil {
			return nil, fmt.Errorf("error reading generator opcode response: %w", err)
		}
		return []*state.GeneratorOpcode{gen}, nil
	// 1.x.x+ SDKs return an array of opcodes.
	case '[':
		gen := []*state.GeneratorOpcode{}
		if err := json.Unmarshal(byt, &gen); err != nil {
			return nil, fmt.Errorf("error reading generator opcode response: %w", err)
		}
		// Normalize the response to always return at least an empty op code in the
		// array. With this, a non-generator is represented as an empty array.
		if len(gen) == 0 {
			return []*state.GeneratorOpcode{
				{Op: enums.OpcodeNone},
			}, nil
		}
		return gen, nil
	}

	// Finally, if the length of resp.Generator == 0 then this is implicitly an enums.OpcodeNone
	// step.  This is added to reduce bandwidth across many calls.
	return []*state.GeneratorOpcode{
		{Op: enums.OpcodeNone},
	}, nil
}

func (e executor) Execute(ctx context.Context, s state.State, action inngest.ActionVersion, edge inngest.Edge, step inngest.Step, idx int) (*state.DriverResponse, error) {
	rt, ok := action.Runtime.Runtime.(inngest.RuntimeHTTP)
	if !ok {
		return nil, fmt.Errorf("Unable to use HTTP executor for non-HTTP runtime")
	}

	input, err := driver.MarshalV1(ctx, s, step, idx)
	if err != nil {
		return nil, err
	}

	// If we have a generator step name, ensure we add the step ID parameter
	parsed, _ := url.Parse(rt.URL)
	values, _ := url.ParseQuery(parsed.RawQuery)
	if edge.IncomingGeneratorStep != "" {
		values.Set("stepId", edge.IncomingGeneratorStep)
		parsed.RawQuery = values.Encode()
	} else {
		values.Set("stepId", edge.Incoming)
	}

	byt, status, duration, err := e.do(ctx, parsed.String(), input)
	if err != nil {
		return nil, err
	}

	if status == 206 {
		// This is a generator-based function returning opcodes.
		resp := &state.DriverResponse{
			ActionVersion: action.Version,
			Duration:      duration,
			OutputSize:    len(byt),
		}
		resp.Generator, err = ParseGenerator(ctx, byt)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	var body interface{}
	body = json.RawMessage(byt)
	if len(byt) > 0 {
		// Is the response valid JSON?  If so, ensure that we don't re-marshal the
		// JSON string.
		respjson := map[string]interface{}{}
		if err := json.Unmarshal(byt, &respjson); err == nil {
			body = respjson
		} else {
			// This isn't a map, so check the first character for json encoding.  If this isn't
			// a string or array, then the body must be treated as text.
			//
			// This is a stop-gap safety check to see if SDKs respond with text that's not JSON,
			// in the case of an internal issue or a host processing error we can't control.
			if byt[0] != '[' && byt[0] != '"' {
				body = string(byt)
			}
		}
	} else {
		body = nil
	}

	// Add an error to driver.Response if the status code isn't 2XX.
	err = nil
	if status < 200 || status > 299 {
		err = fmt.Errorf("invalid status code: %d", status)
	}

	return &state.DriverResponse{
		Output: map[string]interface{}{
			"status": status,
			"body":   body,
		},
		Err:           err,
		ActionVersion: action.Version,
		Duration:      duration,
		OutputSize:    len(byt),
	}, nil
}

func (e executor) do(ctx context.Context, url string, input []byte) ([]byte, int, time.Duration, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(input))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	if len(e.signingKey) > 0 {
		req.Header.Add("X-Inngest-Signature", Sign(ctx, e.signingKey, input))
	}
	pre := time.Now()
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error executing request: %w", err)
	}
	dur := time.Since(pre)
	byt, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("error reading response body: %w", err)
	}

	// If the responding status code is 201 Created, the response has been
	// streamed back to us. In this case, the response body will be namespaced
	// under the "body" key, and the status code will be namespaced under the
	// "status" key.
	//
	// Only SDK versions that include the status in the body are expected to
	// send a 201 status code and namespace in this way, so failing to parse
	// here is an error.
	if resp.StatusCode != 201 {
		return byt, resp.StatusCode, dur, nil
	}

	var body struct {
		StatusCode int    `json:"status"`
		Body       string `json:"body"`
	}
	if err := json.Unmarshal(byt, &body); err != nil {
		return nil, 0, dur, fmt.Errorf("error reading response body to check for status code: %w", err)
	}
	return []byte(body.Body), body.StatusCode, dur, nil

}
