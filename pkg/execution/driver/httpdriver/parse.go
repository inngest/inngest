package httpdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/dateutil"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
)

// ParseGenerator parses generator responses from a JSON response.
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

func ParseStream(resp []byte) (*StreamResponse, error) {
	body := &StreamResponse{}
	if err := json.Unmarshal(resp, &body); err != nil {
		return nil, fmt.Errorf("error reading response body to check for status code: %w", err)
	}
	// Check to see if the body is double-encoded.
	if len(body.Body) > 0 && body.Body[0] == '"' && body.Body[len(body.Body)-1] == '"' {
		var str string
		if err := json.Unmarshal(body.Body, &str); err == nil {
			body.Body = []byte(str)
		}
	}
	return body, nil
}

type StreamResponse struct {
	StatusCode int               `json:"status"`
	Body       json.RawMessage   `json:"body"`
	RetryAt    *string           `json:"retryAt"`
	NoRetry    bool              `json:"noRetry"`
	Headers    map[string]string `json:"headers"`
}

// ParseRetry attempts to parse the retry-after header value.  It first checks to see
// if we have a reasonably sized second value (<= weeks), then parses the value as unix
// seconds.
//
// It falls back to parsing value in multiple formats: RFC3339, RFC1123, etc.
//
// This clips time within the minimums and maximums specified within consts.
func ParseRetry(retry string) (time.Time, error) {
	at, err := parseRetry(retry)
	if err != nil {
		return at, err
	}

	now := time.Now().UTC().Truncate(time.Second)

	dur := time.Until(at)
	if dur > consts.MaxRetryDuration {
		return now.Add(consts.MaxRetryDuration), nil
	}
	if dur < consts.MinRetryDuration {
		return now.Add(consts.MinRetryDuration), nil
	}
	return at, nil
}

func parseRetry(retry string) (time.Time, error) {
	if retry == "" {
		return time.Time{}, ErrNoRetryAfter
	}
	if len(retry) <= 7 {
		// Assume this is an int;  no dates can be <= 7 characters.
		secs, _ := strconv.Atoi(retry)
		if secs > 0 {
			return time.Now().UTC().Truncate(time.Second).Add(time.Second * time.Duration(secs)), nil
		}
	}
	return dateutil.ParseString(retry)
}

func parseResponse(byt []byte) any {
	if len(byt) == 0 {
		return nil
	}

	if byt[0] == '{' {
		// Is the response valid JSON?  If so, ensure that we don't re-marshal the
		// JSON string.
		respjson := map[string]interface{}{}
		if err := json.Unmarshal(byt, &respjson); err == nil {
			return respjson
		}
	}

	// This isn't a map, so check the first character for json encoding.  If this isn't
	// a string or array, then the body must be treated as text.
	//
	// This is a stop-gap safety check to see if SDKs respond with text that's not JSON,
	// in the case of an internal issue or a host processing error we can't control.
	if byt[0] != '[' && byt[0] != '"' {
		return string(byt)
	}

	// This may have been a string-encoded object, because encoding generally
	// sucks.  Sometimes this has happened (by who?  how?).
	//
	// Check to see if the unmarshalled string starts with a '{' and can be
	// decoded into a map.
	//
	// EG: `"{}"` as a literal object with quotes.
	if len(byt) >= 4 && byt[0] == '"' && byt[1] == '{' && byt[len(byt)-1] == '"' && byt[len(byt)-2] == '}' {
		// Parse this as a string.  In this case, an SDK may have
		// double-encoded the string.
		var respstr string
		if err := json.Unmarshal(byt, &respstr); err != nil {
			// Treat this as raw text.
			return string(byt)
		}
		respjson := map[string]interface{}{}
		if err := json.Unmarshal([]byte(respstr), &respjson); err == nil {
			return respjson
		}
	}

	return string(byt)
}
