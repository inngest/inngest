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
	"time"

	"github.com/google/uuid"
	"github.com/inngest/go-httpstat"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	headerspkg "github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/inngest/inngest/pkg/syscode"
	itrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/util"
	"github.com/oklog/ulid/v2"
	"go.opentelemetry.io/otel/propagation"
)

var (
	Dialer = &net.Dialer{KeepAlive: 15 * time.Second}

	ErrEmptyResponse = fmt.Errorf("no response data")
	ErrNoRetryAfter  = fmt.Errorf("no retry after present")
	ErrNotSDK        = syscode.Error{Code: syscode.CodeNotSDK}

	defaultClient = Client(SecureDialerOpts{})
)

const (
	AccountIDHeader = "account-id"
)

// Client returns a new HTTP transport.
func Transport(opts SecureDialerOpts) *http.Transport {
	t := &http.Transport{
		DialContext:           SecureDialer(opts),
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

	return t
}

// Client returns a new HTTP client.
func Client(opts SecureDialerOpts) util.HTTPDoer {
	return util.HTTPDoer(&http.Client{
		Timeout:       consts.MaxFunctionTimeout,
		CheckRedirect: CheckRedirect,
		Transport:     Transport(opts),
	})
}

type executor struct {
	Client                 util.HTTPDoer
	localSigningKey        []byte
	requireLocalSigningKey bool
}

// RuntimeType fulfiils the inngest.Runtime interface.
func (e executor) RuntimeType() string {
	return "http"
}

func (e executor) Execute(ctx context.Context, sl sv2.StateLoader, s sv2.Metadata, item queue.Item, edge inngest.Edge, step inngest.Step, idx, attempt int) (*state.DriverResponse, error) {
	if e.requireLocalSigningKey && len(e.localSigningKey) == 0 {
		return nil, fmt.Errorf("server requires that a signing key is set to run functions")
	}

	uri, err := url.Parse(step.URI)
	if err != nil {
		return nil, err
	}

	input, err := driver.MarshalV1(ctx, sl, s, step, idx, "", attempt)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{}
	itrace.UserTracer().Propagator().Inject(ctx, propagation.MapCarrier(headers))
	if headers["traceparent"] != "" {
		// The span ID will be incorrect here as lifecycles can not affec the
		// ctx. To patch, we manually set the span ID here to what we know it
		// should be based on the item
		parts := strings.Split(headers["traceparent"], "-")
		if len(parts) == 4 {
			spanID, err := item.SpanID()
			if err != nil {
				return nil, fmt.Errorf("error parsing span ID: %w", err)
			}

			parts[2] = spanID.String()
			headers["traceparent"] = strings.Join(parts, "-")
		}
	}

	dr, _, err := DoRequest(ctx, e.Client, Request{
		SigningKey: e.localSigningKey,
		URL:        *uri,
		Input:      input,
		Edge:       edge,
		Step:       step,
		Headers:    headers,
	})
	return dr, err
}

type Request struct {
	// AccountID is a used for feature flag purposes.
	// Meant to be temporary for selectively enabling/disabling grpc requests to sdks
	AccountID uuid.UUID
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

	// Headers are additional headers to add to the request.
	Headers map[string]string
}

// DoRequest executes the HTTP request with the given input.
func DoRequest(ctx context.Context, c util.HTTPDoer, r Request) (*state.DriverResponse, *httpstat.Result, error) {
	if c == nil {
		c = defaultClient
	}

	if r.URL.Scheme != "http" && r.URL.Scheme != "https" {
		return nil, nil, fmt.Errorf("Unable to use HTTP executor for non-HTTP runtime")
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

	resp, tracking, err := do(ctx, c, r)
	if err != nil {
		return nil, tracking, err
	}

	dr, err := HandleHttpResponse(ctx, r, resp)
	return dr, tracking, err
}

func HandleHttpResponse(ctx context.Context, r Request, resp *Response) (*state.DriverResponse, error) {
	var err error
	if resp.StatusCode == 206 {
		// This is a generator-based function returning opcodes.
		dr := &state.DriverResponse{
			Step:           r.Step,
			Duration:       resp.Duration,
			Output:         string(resp.Body),
			OutputSize:     len(resp.Body),
			NoRetry:        resp.NoRetry,
			RetryAt:        resp.RetryAt,
			RequestVersion: resp.RequestVersion,
			StatusCode:     resp.StatusCode,
			SDK:            resp.Sdk,
			Header:         resp.Header,
		}
		dr.Generator, err = ParseGenerator(ctx, resp.Body, resp.NoRetry)
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

		if resp.SysErr != nil {
			dr.SetError(resp.SysErr)
		}

		if !resp.IsSDK {
			dr.SetError(ErrNotSDK)
		}
		return dr, nil
	}

	body := ParseResponse(resp.Body)
	dr := &state.DriverResponse{
		Step:           r.Step,
		Output:         body,
		Duration:       resp.Duration,
		OutputSize:     len(resp.Body),
		NoRetry:        resp.NoRetry,
		RetryAt:        resp.RetryAt,
		RequestVersion: resp.RequestVersion,
		StatusCode:     resp.StatusCode,
		SDK:            resp.Sdk,
		Header:         resp.Header,
	}
	if resp.SysErr != nil {
		dr.SetError(resp.SysErr)
	}

	if dr.Err == nil && resp.StatusCode == 200 && !resp.IsSDK {
		log.From(ctx).Info().
			Interface("headers", resp.Header).
			Str("run_id", r.RunID.String()).
			Str("url", r.URL.String()).
			Msg("response did not come from an Inngest SDK")
		// TODO: Call dr.SetError and set dr.Output. We aren't doing that yet
		// because we want to observe logs first
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
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
		err = fmt.Errorf("invalid status code: %d", resp.StatusCode)
		dr.SetError(err)
	}
	if resp.NoRetry {
		// Ensure we return a NonRetriableError to indicate that
		// we're not retrying when we store the error message.
		//
		// This ensures that errors are handled appropriately from non-SDK step
		// errors.
		err = errors.New("NonRetriableError")
		dr.SetError(err)
	}

	// If there's a RetryAt, ensure we wrap the status code correctly.
	if resp.RetryAt != nil {
		err = queue.RetryAtError(err, resp.RetryAt)
		dr.SetError(err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 && !resp.IsSDK {
		// If we got a successful response but it wasn't from the SDK, then we
		// need to fail the attempt. Otherwise, we may incorrectly mark the
		// function run as "completed".
		dr.SetError(ErrNotSDK)
	}

	return dr, err
}

func do(ctx context.Context, c util.HTTPDoer, r Request) (*Response, *httpstat.Result, error) {
	if c == nil {
		c = defaultClient
	}

	ctx, cancel := context.WithTimeout(ctx, consts.MaxFunctionTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.URL.String(), bytes.NewBuffer(r.Input))
	if err != nil {
		return nil, nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	if r.AccountID != uuid.Nil {
		req.Header.Add(AccountIDHeader, r.AccountID.String())
	}

	// Always close the request after reading the body, ensuring the connection is not recycled.
	req.Close = true

	if len(r.SigningKey) > 0 {
		req.Header.Add("X-Inngest-Signature", Sign(ctx, r.SigningKey, r.Input))
	}
	if len(r.Signature) > 0 {
		// Use this if provided, and override any sig added.
		req.Header.Add("X-Inngest-Signature", r.Signature)
	}

	for k, v := range r.Headers {
		req.Header.Add(k, v)
	}

	// Track HTTP stats, eg. TLS handshake timeouts, DNS lookups, etc.
	tracking := &httpstat.Result{}
	req = req.WithContext(httpstat.WithHTTPStat(req.Context(), tracking))

	// Perform the request.
	resp, byt, dur, err := ExecuteRequest(ctx, c, req)
	tracking.End(time.Now())

	// Handle no response errors.
	if errors.Is(err, ErrUnableToReach) {
		log.From(ctx).
			Warn().
			Str("url", r.URL.String()).
			Interface("step", r.Step).
			Interface("edge	", r.Edge).
			Int64("req_dur_ms", dur.Milliseconds()).
			Msg("EOF writing request to SDK")
		return nil, tracking, err
	}
	if err != nil && len(byt) == 0 {
		return nil, tracking, err
	}

	var sysErr *syscode.Error
	if errors.Is(err, ErrBodyTooLarge) {
		sysErr = &syscode.Error{Code: syscode.CodeOutputTooLarge}
		//
		// downstream executor code expects system error codes here for traces
		// and history to work properly
		err = sysErr

		// Override the output so the user sees the syserrV in the UI rather
		// than a JSON parsing error
		byt, _ = json.Marshal(sysErr.Code)
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
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
	)

	body = byt
	statusCode = resp.StatusCode
	headers := resp.Header

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
	if resp.StatusCode == 201 && sysErr == nil {
		stream, err := ParseStream(byt)
		if err != nil {
			return nil, tracking, fmt.Errorf("error parsing stream: %w", err)
		} else {
			// These are all contained within a single wrapper.
			body = stream.Body
			statusCode = stream.StatusCode

			// Upsert headers from the stream.
			for k, v := range stream.Headers {
				headers.Set(k, v)
			}
		}
	}

	if statusCode == 0 {
		// Unreachable
		log.From(ctx).Error().Err(err).
			Str("body", string(byt)).
			Str("run_id", r.RunID.String()).
			Msg("status code is 0")
	}

	// Check the retry status from the headers and versions.
	noRetry = !ShouldRetry(statusCode, headers.Get(headerNoRetry), headers.Get(headerSDK))

	// Extract the retry at header if it hasn't been set explicitly in streaming.
	if after := headers.Get("retry-after"); after != "" {
		retryAtStr = &after
	}
	if retryAtStr != nil {
		if at, err := ParseRetry(*retryAtStr); err == nil {
			retryAt = &at
		}
	}

	// Get the request version
	rv, _ := strconv.Atoi(headers.Get(headerRequestVersion))
	return &Response{
		Body:           body,
		StatusCode:     statusCode,
		Duration:       dur,
		RetryAt:        retryAt,
		NoRetry:        noRetry,
		RequestVersion: rv,
		IsSDK:          headerspkg.IsSDK(headers),
		Sdk:            headers.Get(headerSDK),
		Header:         headers,
		SysErr:         sysErr,
	}, tracking, err

}

type Response struct {
	Body           []byte
	StatusCode     int
	Duration       time.Duration
	RequestVersion int
	// retryAt is the time to retry this step at, on failure, if specified in the
	// Retry-After headers, or X-Retry-After.
	//
	// This adheres to the HTTP spec; we support both seconds and times in this header.
	RetryAt *time.Time
	// noRetry indicates whether this is a non-retryable error
	NoRetry bool
	// sdk represents the SDK language and version used for these
	// functions, in the format: "js:v0.1.0"
	Sdk string

	Header http.Header

	SysErr *syscode.Error
	IsSDK  bool
}
