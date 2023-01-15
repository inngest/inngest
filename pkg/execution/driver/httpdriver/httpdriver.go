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
	"time"

	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
)

var (
	DefaultExecutor = executor{
		client: &http.Client{
			Timeout: 15 * time.Minute,
		},
	}
)

func Execute(ctx context.Context, s state.State, action inngest.ActionVersion, step inngest.Step) (*state.DriverResponse, error) {
	return DefaultExecutor.Execute(ctx, s, action, step)
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

func (e executor) Execute(ctx context.Context, s state.State, action inngest.ActionVersion, step inngest.Step) (*state.DriverResponse, error) {
	rt, ok := action.Runtime.Runtime.(inngest.RuntimeHTTP)
	if !ok {
		return nil, fmt.Errorf("Unable to use HTTP executor for non-HTTP runtime")
	}

	input, err := driver.MarshalV1(ctx, s, step)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, rt.URL, bytes.NewBuffer(input))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	if len(e.signingKey) > 0 {
		req.Header.Add("X-Inngest-Signature", Sign(ctx, e.signingKey, input))
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}

	byt, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 206 {
		// This is a generator-based function returning opcodes.
		gen := &state.GeneratorOpcode{}
		if err := json.Unmarshal(byt, gen); err != nil {
			// When we return a 206, we always expect that this is
			// a generator function.  Users SHOULD NOT return a 206
			// in any other circumstance.
			return nil, fmt.Errorf("error reading generator opcode response: %w", err)
		}
		if gen.Op == enums.OpcodeNone {
			return nil, fmt.Errorf("invalid opcode returned in response")
		}

		return &state.DriverResponse{
			Generator:     gen,
			ActionVersion: action.Version,
		}, nil
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
	}

	// Add an error to driver.Response if the status code isn't 2XX.
	err = nil
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		err = fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	return &state.DriverResponse{
		Output: map[string]interface{}{
			"status": resp.StatusCode,
			"body":   body,
		},
		Err:           err,
		ActionVersion: action.Version,
	}, nil
}
