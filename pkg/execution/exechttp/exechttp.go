// Package exechttp defines HTTP-related utilities for execution.
package exechttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/inngest/go-httpstat"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/realtime"
	"github.com/inngest/inngest/pkg/logger"
)

const (
	// MaxRedirects is the maximum number of redirects to allow before failing the outgoing
	// request.  This is a high number to account for the way modal.com redirects when there
	// are long running jobs.
	MaxRedirects = 60
)

var (
	ErrUnableToReach       = fmt.Errorf("Unable to reach SDK URL")
	ErrDenied              = fmt.Errorf("Your server blocked the connection") // "connection timed out"
	ErrServerClosed        = fmt.Errorf("Your server closed the request before finishing.")
	ErrConnectionReset     = fmt.Errorf("Your server reset the connection while we were sending the request.")
	ErrUnexpectedEnd       = fmt.Errorf("Your server reset the connection while we were reading the reply: Unexpected ending response")
	ErrInvalidResponse     = fmt.Errorf("Error performing request to SDK URL")
	ErrBodyTooLarge        = fmt.Errorf("http response size is greater than the limit")
	ErrTLSHandshakeTimeout = fmt.Errorf("Your server didn't complete the TLS handshake in time")
)

// HTTPClient is an interface for a standard http.Client
type HTTPClient interface {
	Do(r *http.Request) (*http.Response, error)
}

// RequestExecutor executes requests defined as a SerializableRequest.  This is used instead
// of an *http.Request in order to add execution-specific attributes to each request,
// such as streaming definitions.
//
// Note that it would be possible to do this using standard requests and HTTP headers,
// but this leads to potentially leaking internal specifics into the request, which is
// unnecessary.
type RequestExecutor interface {
	DoRequest(ctx context.Context, r SerializableRequest) (*Response, error)
}

// DialFunc represents a dialer which returns a network connection given
// a network type and address.
type DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// WithRealtimePublishing sets the tee publish URL on the client.  Without this being set,
// the HTTP client will not tee and publish any responses directly.
func WithRealtimePublishing() ClientOpt {
	return func(e *ExtendedClient) {
		e.publish = true
	}
}

type ClientOpt func(e *ExtendedClient)

// Client returns a new HTTP client which fulfils the ClientExecutor interface, allowing us to
// execute SerializableRequest types.
func Client(dialopts SecureDialerOpts, opts ...ClientOpt) ExtendedClient {
	c := ExtendedClient{
		Client: &http.Client{
			Timeout:       consts.MaxFunctionTimeout,
			CheckRedirect: CheckRedirect,
			Transport:     Transport(dialopts),
		},
	}
	for _, o := range opts {
		o(&c)
	}
	return c
}

// ExtendedClient wraps an *http.Client to fulfil the RequestExecutor method, using the embedded
// http.Client to make the given request.
type ExtendedClient struct {
	*http.Client

	// publish is used to publish the request in real-time using Inngest's realtime APIs.  Note that
	// this is false by default;  without this set, any SerializableRequest structs executed will not be
	// streamed to the realtime publishing endpoint specified in each request.
	publish bool
}

// DoRequest performs the SerializableRequest with tracking, reading the response and handling
// common HTTP errors.
//
// This will ALWAYS return either an error or a non-nil response.
func (e ExtendedClient) DoRequest(ctx context.Context, r SerializableRequest) (*Response, error) {
	req, err := r.HTTPRequest()
	if err != nil {
		return nil, err
	}

	tracking := &httpstat.Result{}
	req = req.WithContext(httpstat.WithHTTPStat(req.Context(), tracking))
	resp, err := e.Do(req)
	tracking.End(time.Now())

	if resp != nil {
		// If we have a response, always close the body to free up the underlying net conn.
		defer func() {
			_ = resp.Body.Close()
		}()
	}

	if isUnreachable(resp, err) {
		return nil, ErrUnableToReach
	}

	if err != nil {
		return nil, fmt.Errorf("error performing request: %w", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("nil response received from URL: %s", r.URL)
	}

	// We're going to reassign this to use a LimitReader, ensuring we don't read an infinite amount
	// of data from the request.  We need to do this so that we can continue to close resp.Body to
	// release the underlying conn in the above dever.
	body := io.LimitReader(resp.Body, consts.MaxSDKResponseBodySize+1)

	if e.publish && r.Publish.ShouldPublish() {
		rdr, err := realtime.TeeStreamReaderToAPI(body, r.Publish.PublishURL, realtime.TeeStreamOptions{
			Channel: r.Publish.Channel,
			Topic:   r.Publish.Topic,
			Token:   r.Publish.Token,
			Metadata: map[string]any{
				"url":          req.URL,
				"content-type": resp.Header.Get("content-type"),
				"request_id":   r.Publish.RequestID,
			},
		})
		if err == nil {
			body = rdr
		}
		if err != nil {
			logger.StdlibLogger(ctx).Warn(
				"error teeing request to publish endpoint",
				"error", err,
				"url", r.URL,
				"channel", r.Publish.Channel,
				"topic", r.Publish.Topic,
				"response_status", resp.StatusCode,
			)
		}
	}

	// Read 1 extra byte above the max so that we can check if the response is too large
	byt, err := io.ReadAll(body)
	if err != nil {
		return nil, ErrUnexpectedEnd
	}

	out := &Response{
		Body:       byt,
		Header:     resp.Header,
		StatResult: tracking,
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Proto:      resp.Proto,
		ProtoMajor: resp.ProtoMajor,
		ProtoMinor: resp.ProtoMinor,
	}

	if len(byt) > consts.MaxSDKResponseBodySize {
		return out, ErrBodyTooLarge
	}

	// parse errors into common responses
	err = CommonHTTPErrors(err)
	return out, err
}

type Response struct {
	Body       []byte
	Header     http.Header
	StatResult *httpstat.Result

	StatusCode int    // e.g. 200
	Status     string // e.g. "200 OK"
	Proto      string // e.g. "HTTP/1.0"
	ProtoMajor int    // e.g. 1
	ProtoMinor int    // e.g. 0

	// Hostname represents the hostname of the machine executing the request.
	// This is optional and may be unset.
	Hostname string

	// Attempts returns the number of attempts taken to execute the request.
	// This is optional and may be unset.
	Attempts int
}

// Client returns a new HTTP transport.
func Transport(opts SecureDialerOpts) *http.Transport {
	t := &http.Transport{
		DialContext:           SecureDialer(opts),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          5,
		IdleConnTimeout:       2 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     true,
		// New, ensuring that services can take their time before
		// responding with headers as they process long running
		// jobs.
		ResponseHeaderTimeout: consts.MaxFunctionTimeout,
	}

	return t
}

// CheckRedirect is an http client utility to follow redirects in outgoing requests.
func CheckRedirect(req *http.Request, via []*http.Request) (err error) {
	if len(via) == 0 {
		return nil
	}

	if len(via) > MaxRedirects {
		return fmt.Errorf("stopped after %d redirects", MaxRedirects)
	}

	if via[0].Body != nil && via[0].GetBody != nil {
		req.Body, err = via[0].GetBody()
		if err != nil {
			return err
		}
	}

	req.ContentLength = via[0].ContentLength

	// Combine headers from the original request and the redirect request
	for k, v := range via[0].Header {
		if len(v) > 0 {
			req.Header.Set(k, v[0])
		}
	}

	// Retain the original query params
	qp := req.URL.Query()
	for k, v := range via[0].URL.Query() {
		qp.Set(k, v[0])
	}
	req.URL.RawQuery = qp.Encode()

	return nil
}

func CommonHTTPErrors(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, io.EOF) {
		return err
	}

	{
		// timeouts
		if urlErr, ok := err.(*url.Error); ok && urlErr.Err == context.DeadlineExceeded {
			// This timed out.
			return context.DeadlineExceeded
		}
		if errors.Is(err, context.DeadlineExceeded) {
			// timed out
			return context.DeadlineExceeded
		}
	}

	if errors.Is(err, syscall.EPIPE) {
		return ErrServerClosed
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return ErrConnectionReset
	}
	// If we get an unexpected EOF and the response is nil, error immediately.
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return ErrUnexpectedEnd
	}

	// use the error as-is, wrapped with a prefix for users.
	return fmt.Errorf("%s: %w", ErrInvalidResponse, err)
}

func isUnreachable(resp *http.Response, err error) bool {
	if resp != nil {
		return false
	}
	if err == nil {
		return false
	}

	str := err.Error()

	// see net/dial_test.go, line 789 of go 1.24.1.
	return errors.Is(err, io.EOF) ||
		strings.Contains(str, "connection refused") ||
		strings.Contains(str, "unreachable") ||
		strings.Contains(str, "no route to host")
}
