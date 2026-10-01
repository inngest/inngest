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
	tee := &publishingReader{ctx: ctx, reader: reader}

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
	req.Header.Set("Authorization", "Bearer "+opts.Token)

	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	cancel()
	// Stop new transport reads without waiting for an idle source read. The
	// returned reader waits for that read only when the caller requests bytes.
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

// publishingReader allows the transport to stop without closing the SDK stream.
// At most one source read is outstanding; its owned buffer remains available to
// the response reader if publishing ends while the source is idle.
type publishingReader struct {
	mu        sync.Mutex
	ctx       context.Context
	reader    io.Reader
	buf       bytes.Buffer
	pending   chan sourceRead
	sourceErr error
	stopped   bool
}

type sourceRead struct {
	data []byte
	err  error
}

func (r *publishingReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return 0, io.EOF
	}
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	pending := make(chan sourceRead, 1)
	r.pending = pending
	go func() {
		// The transport may reuse p immediately after cancellation, so the
		// source must read into a buffer owned by this outstanding read.
		buf := make([]byte, len(p))
		n, err := r.reader.Read(buf)
		pending <- sourceRead{data: buf[:n], err: err}
	}()
	select {
	case result := <-pending:
		r.pending = nil
		r.sourceErr = result.err
		_, _ = r.buf.Write(result.data)
		return copy(p, result.data), result.err
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	}
}

func (r *publishingReader) remainder() io.Reader {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = true
	return &publishingRemainder{source: r}
}

// publishingRemainder waits for an outstanding source read only when the caller
// requests response bytes, after first returning everything already buffered.
type publishingRemainder struct {
	source *publishingReader
}

func (r *publishingRemainder) Read(p []byte) (int, error) {
	s := r.source
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	if s.buf.Len() > 0 {
		return s.buf.Read(p)
	}
	if s.pending != nil {
		result := <-s.pending
		s.pending = nil
		s.sourceErr = result.err
		_, _ = s.buf.Write(result.data)
		if s.buf.Len() > 0 {
			return s.buf.Read(p)
		}
	}
	if s.sourceErr != nil {
		return 0, s.sourceErr
	}
	return s.reader.Read(p)
}

func allowedPublishURL(u *url.URL, raw, allowedHTTP string) bool {
	if u.User != nil || u.Host == "" {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	return u.Scheme == "https" || (u.Scheme == "http" && ((ip != nil && ip.IsLoopback()) || raw == allowedHTTP))
}
