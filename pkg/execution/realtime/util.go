package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type TeeStreamOptions struct {
	// AllowedHTTPURL is a trusted, exact dev-server callback URL. Never populate
	// this from request data; production callers leave it empty.
	AllowedHTTPURL string
	Channel        string
	Topic          string
	Token          string
	Metadata       map[string]any
}

// TeeStreamReaderToAPI is a utility function that publishes a reader to the HTTP API,
// streamed.  It returns a reader which contains the data read from the original reader.
//
// The first return value is always valid for reading, even on error.
func TeeStreamReaderToAPI(reader io.Reader, publishURL string, opts TeeStreamOptions) (io.Reader, error) {
	if opts.Channel == "" || opts.Topic == "" || opts.Token == "" {
		// Bypass, don't do anything.
		return reader, nil
	}

	return teeStreamReaderToAPI(context.Background(), reader, publishURL, opts, 5*time.Minute)
}

// TeeStreamReaderToAPIWithContext publishes using the caller's cancellation and
// a five-minute publishing deadline. Failure preserves the remaining response.
func TeeStreamReaderToAPIWithContext(ctx context.Context, reader io.Reader, publishURL string, opts TeeStreamOptions) (io.Reader, error) {
	return teeStreamReaderToAPI(ctx, reader, publishURL, opts, 5*time.Minute)
}

func teeStreamReaderToAPI(ctx context.Context, reader io.Reader, publishURL string, opts TeeStreamOptions, timeout time.Duration) (io.Reader, error) {
	if opts.Channel == "" || opts.Topic == "" || opts.Token == "" {
		return reader, nil
	}
	u, err := url.Parse(publishURL)
	if err != nil {
		return reader, fmt.Errorf("invalid publishing URL")
	}
	if !allowedPublishURL(u, publishURL, opts.AllowedHTTPURL) {
		return reader, fmt.Errorf("publishing requires HTTPS (HTTP is allowed only for literal loopback addresses)")
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tee := &publishingReader{reader: reader}

	qp := u.Query()
	qp.Add("channel", opts.Channel)
	qp.Add("topic", opts.Topic)

	if len(opts.Metadata) > 0 {
		byt, _ := json.Marshal(opts.Metadata)
		qp.Add("metadata", string(byt))
	}

	// Preserve existing endpoint query parameters.
	u.RawQuery = qp.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), tee)
	if err != nil {
		return reader, err
	}
	req.Header.Add("Content-Type", "text/stream")
	req.Header.Add("Authorization", opts.Token)

	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	cancel()
	// Stop transport reads before handing the stream back to the executor.
	remainder := tee.remainder()
	if err != nil {
		return remainder, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return remainder, fmt.Errorf("invalid status code publishing stream: %d", resp.StatusCode)
	}

	return remainder, nil
}

// publishingReader serializes transport reads with returning the response to the
// caller: HTTP may return before its request-body writer has finished.
type publishingReader struct {
	mu      sync.Mutex
	reader  io.Reader
	buf     bytes.Buffer
	stopped bool
}

func (r *publishingReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return 0, io.EOF
	}
	n, err := r.reader.Read(p)
	_, _ = r.buf.Write(p[:n])
	return n, err
}
func (r *publishingReader) remainder() io.Reader {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = true
	return io.MultiReader(&r.buf, r.reader)
}

func allowedPublishURL(u *url.URL, raw, allowedHTTP string) bool {
	if u.User != nil || u.Host == "" {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	return u.Scheme == "https" || (u.Scheme == "http" && ((ip != nil && ip.IsLoopback()) || raw == allowedHTTP))
}
