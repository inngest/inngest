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
	"github.com/valyala/fastjson"
)

// ParseGenerator parses generator responses from a JSON response.
//
// noRetry is passed via the no retry header, allowing us to indicate that
// an OpcodeStepError was raised with a NonRetriableError.
func ParseGenerator(ctx context.Context, byt []byte, noRetry bool) ([]*state.GeneratorOpcode, error) {
	generators, err := parseGenerator(ctx, byt, noRetry)
	for n, item := range generators {
		// Ensure that we set no retries on the opcode error directly.
		// This is needed for the executor to check how to handle the error.
		if item.Op == enums.OpcodeStepError {
			if item.Error == nil {
				return generators, fmt.Errorf("OpcodeStepError received without Error field set: %+v", item)
			}

			item.Error.NoRetry = noRetry
			generators[n] = item
		}
	}
	return generators, err
}

func parseGenerator(ctx context.Context, byt []byte, noRetry bool) (ops []*state.GeneratorOpcode, err error) {
	// When we return a 206, we always expect that this is a generator
	// function.  Users SHOULD NOT return a 206 in any other circumstance.
	if len(byt) == 0 {
		err = ErrEmptyResponse
		return
	}

	// Is this a slice of opcodes or a single opcode?  The SDK can return both:
	// parallelism was added as an incremental improvement.  It would have been
	// nice to always return an array and we can enfore this as an SDK
	// requirement in V1+
	switch byt[0] {
	// 0.x.x SDKs return a single opcode.
	case '{':
		gen := &state.GeneratorOpcode{}
		if err = json.Unmarshal(byt, gen); err != nil {
			err = fmt.Errorf("error reading generator opcode response: %w", err)
			return
		}
		ops = append(ops, gen)
	// 1.x.x+ SDKs return an array of opcodes.
	case '[':
		gen := []*state.GeneratorOpcode{}
		if err = json.Unmarshal(byt, &gen); err != nil {
			err = fmt.Errorf("error reading generator opcode response: %w", err)
			return
		}
		ops = append(ops, gen...)
	}

	// Finally, if the length of resp.Generator == 0 then this is implicitly an
	// enums.OpcodeNone step.  This is added to reduce bandwidth across many
	// calls and normalize the response, such that a non-generator is
	// represented as an empty array.
	if len(ops) == 0 {
		ops = append(ops, &state.GeneratorOpcode{
			Op: enums.OpcodeNone,
		})
	}

	// Check every op we've parsed, making sure it adheres to any limits we're
	// enforcing
	for _, op := range ops {
		if err = op.Validate(); err != nil {
			err = fmt.Errorf("error validating generator opcode %s: %w", op.ID, err)
			return
		}
	}

	return
}

func ParseStream(resp []byte) (*StreamResponse, error) {
	body := &StreamResponse{}
	if err := json.Unmarshal(resp, &body); err != nil {
		return nil, fmt.Errorf("error reading response body to check for status code: %w", err)
	}
	if body.Error != nil {
		return nil, fmt.Errorf("%s", *body.Error)
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
	Error      *string           `json:"error"`
	StatusCode int               `json:"status"`
	Body       json.RawMessage   `json:"body"`
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

func ParseResponse(byt []byte) any {
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

	err := fastjson.ValidateBytes(byt)
	if err == nil {
		return json.RawMessage(byt)
	}

	return string(byt)
}
