package httpdriver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/pkg/telemetry"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/propagation"
)

var (
	dialer = &net.Dialer{KeepAlive: 15 * time.Second}

	DefaultTransport = &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          5,
		IdleConnTimeout:       2 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     true,
		// New, ensuring that services can take their time before
		// responding with headers as they process long running
		// jobs.
		ResponseHeaderTimeout: consts.MaxFunctionTimeout,
	}
	DefaultClient = &http.Client{
		Timeout:       consts.MaxFunctionTimeout,
		CheckRedirect: checkRedirect,
		Transport:     DefaultTransport,
	}

	DefaultExecutor = &executor{Client: DefaultClient}

	ErrEmptyResponse = fmt.Errorf("no response data")
	ErrNoRetryAfter  = fmt.Errorf("no retry after present")
)

type executor struct {
	Client     *http.Client
	signingKey []byte
}

// RuntimeType fulfiils the inngest.Runtime interface.
func (e executor) RuntimeType() string {
	return "http"
}

func (e executor) Execute(ctx context.Context, sl sv2.StateLoader, s sv2.Metadata, item queue.Item, edge inngest.Edge, step inngest.Step, idx, attempt int) (*state.DriverResponse, error) {
	uri, err := url.Parse(step.URI)
	if err != nil {
		return nil, err
	}

	input, err := driver.MarshalV1(ctx, sl, s, step, idx, "", attempt)
	if err != nil {
		return nil, err
	}

	return DoRequest(ctx, e.Client, Request{
		SigningKey: e.signingKey,
		URL:        *uri,
		Input:      input,
		Edge:       edge,
		Step:       step,
	})
}

type Request struct {
	// WorkflowID is used for logging purposes, and is not used in the request
	WorkflowID uuid.UUID
	// RunID is used for logging purposes, and is not used in the request
	RunID ulid.ULID

	// Signature, if set, is the signature to use for the request.  If unset,
	// the SigningKey below will be used to sign the input.
	Signature string
	// SigningKey, if set, signs the input using this key.
	SigningKey []byte
	URL        url.URL
	Input      []byte
	Edge       inngest.Edge
	Step       inngest.Step
}

// DoRequest executes the HTTP request with the given input.
func DoRequest(ctx context.Context, c *http.Client, r Request) (*state.DriverResponse, error) {
	if c == nil {
		c = DefaultClient
	}

	if r.URL.Scheme != "http" && r.URL.Scheme != "https" {
		return nil, fmt.Errorf("Unable to use HTTP executor for non-HTTP runtime")
	}

	// If we have a generator step name, ensure we add the step ID parameter
	values, _ := url.ParseQuery(r.URL.RawQuery)
	if r.Edge.IncomingGeneratorStep != "" {
		values.Set("stepId", r.Edge.IncomingGeneratorStep)
		r.URL.RawQuery = values.Encode()
	} else {
		values.Set("stepId", r.Edge.Incoming)
		r.URL.RawQuery = values.Encode()
	}

	resp, err := do(ctx, c, r)
	if err != nil {
		return nil, err
	}

	if resp.statusCode == 206 {
		// This is a generator-based function returning opcodes.
		dr := &state.DriverResponse{
			Step:           r.Step,
			Duration:       resp.duration,
			Output:         string(resp.body),
			OutputSize:     len(resp.body),
			NoRetry:        resp.noRetry,
			RetryAt:        resp.retryAt,
			RequestVersion: resp.requestVersion,
			StatusCode:     resp.statusCode,
			SDK:            resp.sdk,
			Header:         resp.header,
		}
		dr.Generator, err = ParseGenerator(ctx, resp.body, resp.noRetry)
		if err != nil {
			return nil, err
		}

		// NOTE: Generator responses never set dr.Err, as we assume that the
		// SDK finished processing successfully.  An empty array is OpcodeNone.

		// If this was a generator response with a single op, set some
		// relevant step data so that it's easier to identify this step in
		// history.
		if op := dr.HistoryVisibleStep(); op != nil {
			dr.Step.ID = op.ID
			dr.Step.Name = op.UserDefinedName()
		}

		if resp.sysErr != nil {
			dr.SetError(resp.sysErr)
		}

		return dr, nil
	}

	body := parseResponse(resp.body)
	dr := &state.DriverResponse{
		Step:           r.Step,
		Output:         body,
		Duration:       resp.duration,
		OutputSize:     len(resp.body),
		NoRetry:        resp.noRetry,
		RetryAt:        resp.retryAt,
		RequestVersion: resp.requestVersion,
		StatusCode:     resp.statusCode,
		SDK:            resp.sdk,
		Header:         resp.header,
	}
	if resp.sysErr != nil {
		dr.SetError(resp.sysErr)
	}
	if resp.statusCode < 200 || resp.statusCode > 299 {
		// Add an error to driver.Response if the status code isn't 2XX.
		//
		// This is IMPERATIVE, as dr.Err is used to indicate communication errors,
		// SDK failures without graceful responses - each of which uses r.Err to
		// handle retrying.
		//
		// Non 2xx errors are thrown when:
		// - The SDK isn't invoked (proxy error, etc.)
		// - The SDK has a catastrophic failure and does not respond gracefully.
		// - The function fails or errors (these are not *yet* opcodes, but should be).
		err = fmt.Errorf("invalid status code: %d", resp.statusCode)
		dr.SetError(err)
	}
	if resp.noRetry {
		// Ensure we return a NonRetriableError to indicate that
		// we're not retrying when we store the error message.
		//
		// This ensures that errors are handled appropriately from non-SDK step
		// errors.
		err = errors.New("NonRetriableError")
		dr.SetError(err)
	}
	return dr, err
}

func do(ctx context.Context, c *http.Client, r Request) (*response, error) {
	if c == nil {
		c = DefaultClient
	}

	ctx, cancel := context.WithTimeout(ctx, consts.MaxFunctionTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.URL.String(), bytes.NewBuffer(r.Input))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	// Always close the request after reading the body, ensuring the connection is not recycled.
	req.Close = true

	if len(r.SigningKey) > 0 {
		req.Header.Add("X-Inngest-Signature", Sign(ctx, r.SigningKey, r.Input))
	}
	if len(r.Signature) > 0 {
		// Use this if provided, and override any sig added.
		req.Header.Add("X-Inngest-Signature", r.Signature)
	}

	// Add `traceparent` and `tracestate` headers to the request from `ctx`
	telemetry.UserTracer().Propagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	pre := time.Now()
	resp, err := c.Do(req)
	dur := time.Since(pre)
	defer func() {
		if resp != nil {
			_ = resp.Body.Close()
		}
	}()

	if errors.Is(err, io.EOF) && resp == nil {
		log.From(ctx).
			Warn().
			Str("url", r.URL.String()).
			Interface("step", r.Step).
			Interface("edge	", r.Edge).
			Int64("req_dur_ms", dur.Milliseconds()).
			Msg("EOF writing request to SDK")
		return nil, fmt.Errorf("Unable to reach SDK URL: %w", io.EOF)
	}

	if err != nil && !errors.Is(err, io.EOF) {
		if urlErr, ok := err.(*url.Error); ok && urlErr.Err == context.DeadlineExceeded {
			// This timed out.
			return nil, context.DeadlineExceeded
		}
		if errors.Is(err, syscall.EPIPE) {
			return nil, fmt.Errorf("Your server closed the request before finishing.")
		}
		if errors.Is(err, syscall.ECONNRESET) {
			return nil, fmt.Errorf("Your server reset the request connection.")
		}
		// Unexpected EOFs are valid and returned from servers when chunked encoding may
		// be invalid.  Handle any other error by returning immediately.
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, fmt.Errorf("Error performing request to SDK URL: %w", err)
		}
		// If we get an unexpected EOF and the response is nil, error immediately.
		if errors.Is(err, io.ErrUnexpectedEOF) && resp == nil {
			return nil, fmt.Errorf("Invalid response from SDK server: Unexpected EOF ending response")
		}
	}

	byt, err := io.ReadAll(io.LimitReader(resp.Body, consts.MaxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	var sysErr *syscode.Error
	if len(byt) > consts.MaxBodySize {
		sysErr = &syscode.Error{Code: syscode.CodeOutputTooLarge}

		// Override the output so the user sees the syserrV in the UI rather
		// than a JSON parsing error
		byt, _ = json.Marshal(sysErr.Code)
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		err = nil
		log.From(ctx).
			Error().
			Err(err).
			Str("url", r.URL.String()).
			Str("response", string(byt)).
			Interface("headers", resp.Header).
			Interface("step", r.Step).
			Interface("edge	", r.Edge).
			Msg("http eof reading response")
	}

	// These variables are extracted from streaming and non-streaming responses separately.
	//
	// They're defined here so that we can normalize code paths after testing for streaming
	// responses and handling them.
	var (
		statusCode int
		body       []byte
		noRetry    bool
		retryAtStr *string
		retryAt    *time.Time
		headers    = map[string]string{}
	)

	body = byt
	statusCode = resp.StatusCode
	for k, v := range resp.Header {
		headers[strings.ToLower(k)] = v[0]
	}

	// Check if this was a streaming response.  If so, extract headers sent
	// from _after_ the response started within the payload.
	//
	// If the responding status code is 201 Created, the response has been
	// streamed back to us. In this case, the response body will be namespaced
	// under the "body" key, and the status code will be namespaced under the
	// "status" key.
	//
	// Only SDK versions that include the status in the body are expected to
	// send a 201 status code and namespace in this way, so failing to parse
	// here is an error.
	if resp.StatusCode == 201 {
		stream, err := ParseStream(byt)
		if err != nil {
			return nil, err
		}
		// These are all contained within a single wrapper.
		body = stream.Body
		statusCode = stream.StatusCode

		// Upsert headers from the stream.
		for k, v := range stream.Headers {
			headers[k] = v
		}
	}

	// Check the retry status from the headers and versions.
	noRetry = !shouldRetry(statusCode, headers[headerNoRetry], headers[headerSDK])

	// Extract the retry at header if it hasn't been set explicitly in streaming.
	if after := headers["retry-after"]; retryAtStr == nil && after != "" {
		retryAtStr = &after
	}
	if retryAtStr != nil {
		if at, err := ParseRetry(*retryAtStr); err == nil {
			retryAt = &at
		}
	}

	// Get the request version
	rv, _ := strconv.Atoi(headers[headerRequestVersion])
	return &response{
		body:           body,
		statusCode:     statusCode,
		duration:       dur,
		retryAt:        retryAt,
		noRetry:        noRetry,
		requestVersion: rv,
		sdk:            headers[headerSDK],
		header:         resp.Header,
		sysErr:         sysErr,
	}, err

}

type response struct {
	body           []byte
	statusCode     int
	duration       time.Duration
	requestVersion int
	// retryAt is the time to retry this step at, on failure, if specified in the
	// Retry-After headers, or X-Retry-After.
	//
	// This adheres to the HTTP spec; we support both seconds and times in this header.
	retryAt *time.Time
	// noRetry indicates whether this is a non-retryable error
	noRetry bool
	// sdk represents the SDK language and version used for these
	// functions, in the format: "js:v0.1.0"
	sdk string

	header http.Header

	sysErr *syscode.Error
}
