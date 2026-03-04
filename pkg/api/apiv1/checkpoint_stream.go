package apiv1

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
)

const (
	// streamBufferMaxBytes is the max total size of buffered chunks per run.
	streamBufferMaxBytes = 4 * 1024 * 1024 // 4MB
	// streamBufferTTL is how long a stream buffer lives after completion.
	streamBufferTTL = 10 * time.Minute
	// streamReadChunkSize is the size of each read from the app's request body.
	streamReadChunkSize = 4 * 1024 // 4KB
	// streamOutputTimeout is how long the client endpoint waits for the stream to start.
	streamOutputTimeout = 5 * time.Minute
)

// StreamHeaderFrame is the first frame sent by the SDK, containing the
// HTTP response metadata. All subsequent data is raw body bytes.
type StreamHeaderFrame struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
}

// streamBuffer holds chunks for a single run's stream, allowing late-joining
// clients to catch up by replaying buffered data.
type streamBuffer struct {
	mu     sync.RWMutex
	chunks [][]byte
	size   int
	done   bool
	err    error
	// notify is closed when the stream completes (done or error).
	notify chan struct{}
	// chunkCh receives a signal each time a new chunk is appended,
	// allowing the output handler to wake up without polling.
	chunkCh chan struct{}
}

func newStreamBuffer() *streamBuffer {
	return &streamBuffer{
		notify:  make(chan struct{}),
		chunkCh: make(chan struct{}, 1),
	}
}

// append adds a chunk to the buffer. Returns false if the buffer is full.
func (b *streamBuffer) append(data []byte) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.size+len(data) > streamBufferMaxBytes {
		return false
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	b.chunks = append(b.chunks, cp)
	b.size += len(data)

	// Non-blocking signal to wake any waiting reader.
	select {
	case b.chunkCh <- struct{}{}:
	default:
	}

	return true
}

// finish marks the stream as complete.
func (b *streamBuffer) finish(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.done = true
	b.err = err
	close(b.notify)

	// Wake any waiting reader.
	select {
	case b.chunkCh <- struct{}{}:
	default:
	}
}

// snapshot returns all chunks from the given offset and whether the stream is done.
func (b *streamBuffer) snapshot(offset int) (chunks [][]byte, done bool, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if offset < len(b.chunks) {
		chunks = b.chunks[offset:]
	}
	return chunks, b.done, b.err
}

// streamRegistry manages per-run stream buffers with TTL-based cleanup.
type streamRegistry struct {
	mu      sync.RWMutex
	buffers map[ulid.ULID]*streamBuffer
}

func newStreamRegistry() *streamRegistry {
	return &streamRegistry{
		buffers: make(map[ulid.ULID]*streamBuffer),
	}
}

// getOrCreate returns the buffer for the given run, creating one if needed.
func (r *streamRegistry) getOrCreate(runID ulid.ULID) *streamBuffer {
	r.mu.Lock()
	defer r.mu.Unlock()

	if buf, ok := r.buffers[runID]; ok {
		return buf
	}

	buf := newStreamBuffer()
	r.buffers[runID] = buf
	return buf
}

// get returns the buffer for the given run, or nil.
func (r *streamRegistry) get(runID ulid.ULID) *streamBuffer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.buffers[runID]
}

// remove deletes the buffer for the given run.
func (r *streamRegistry) remove(runID ulid.ULID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.buffers, runID)
}

// StreamIngest handles POST /checkpoint/{runID}/stream.
// The SDK streams the HTTP response body through this endpoint.
//
// The first line of the request body (up to \n) MUST be a JSON-encoded
// StreamHeaderFrame. Everything after the first newline is raw body data
// streamed in chunks.
func (a checkpointAPI) StreamIngest(w http.ResponseWriter, r *http.Request) {
	_, err := a.AuthFinder(r.Context())
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "Unauthorized"))
		return
	}

	runIDStr := chi.URLParam(r, "runID")
	runID, err := ulid.Parse(runIDStr)
	if err != nil {
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Invalid run ID"))
		return
	}

	buf := a.streams.getOrCreate(runID)

	defer r.Body.Close()

	// Schedule cleanup after TTL.
	defer func() {
		go func() {
			time.Sleep(streamBufferTTL)
			a.streams.remove(runID)
		}()
	}()

	reader := bufio.NewReader(r.Body)

	// Read the header frame (first line).
	headerLine, err := reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		logger.StdlibLogger(r.Context()).Warn("error reading stream header", "error", err, "run_id", runID)
		buf.finish(err)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Error reading stream header"))
		return
	}
	headerLine = bytes.TrimRight(headerLine, "\n")
	if len(headerLine) > 0 {
		if !buf.append(headerLine) {
			buf.finish(errors.New("stream buffer full"))
			_ = publicerr.WriteHTTP(w, publicerr.Errorf(413, "Stream buffer full"))
			return
		}
	}

	// Read the remaining body in chunks.
	for {
		chunk := make([]byte, streamReadChunkSize)
		n, readErr := reader.Read(chunk)

		if n > 0 {
			data := chunk[:n]
			if !buf.append(data) {
				buf.finish(errors.New("stream buffer full"))
				_ = publicerr.WriteHTTP(w, publicerr.Errorf(413, "Stream buffer full"))
				return
			}
		}

		if errors.Is(readErr, io.EOF) {
			buf.finish(nil)
			w.WriteHeader(http.StatusOK)
			return
		}
		if readErr != nil {
			logger.StdlibLogger(r.Context()).Warn("error reading stream body",
				"error", readErr, "run_id", runID)
			buf.finish(readErr)
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(readErr, 500, "Error reading stream"))
			return
		}
	}
}

// StreamOutput handles GET /checkpoint/{runID}/stream?token=<jwt>.
// The client consumes the streamed response through this endpoint.
func (a checkpointAPI) StreamOutput(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	claims, err := apiv1auth.VerifyRunJWT(r.Context(), a.runClaimsSecret, token)
	if err != nil || claims == nil {
		w.WriteHeader(401)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "Unable to authenticate"))
		return
	}

	runID := claims.RunID

	// Wait for the buffer to appear (the app may not have started streaming yet).
	deadline := time.After(streamOutputTimeout)
	var buf *streamBuffer
	for buf == nil {
		buf = a.streams.get(runID)
		if buf != nil {
			break
		}
		select {
		case <-deadline:
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusGatewayTimeout)
			_, _ = w.Write([]byte(`{"status":"timeout","message":"stream did not start within timeout"}`))
			return
		case <-r.Context().Done():
			return
		case <-time.After(200 * time.Millisecond):
			// Poll for stream buffer creation.
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "Streaming not supported"))
		return
	}

	headersSent := false
	offset := 0

	for {
		chunks, done, streamErr := buf.snapshot(offset)

		for _, chunk := range chunks {
			if !headersSent {
				// First chunk is the header frame.
				var hdr StreamHeaderFrame
				if jsonErr := json.Unmarshal(chunk, &hdr); jsonErr == nil && hdr.StatusCode > 0 {
					for k, v := range hdr.Headers {
						w.Header().Set(k, v)
					}
					w.WriteHeader(hdr.StatusCode)
					flusher.Flush()
					headersSent = true
					offset++
					continue
				}
				// If we can't parse the header frame, fall back to raw streaming.
				w.Header().Set("content-type", "application/octet-stream")
				w.WriteHeader(http.StatusOK)
				headersSent = true
			}

			if _, writeErr := w.Write(chunk); writeErr != nil {
				return
			}
			flusher.Flush()
			offset++
		}

		if done {
			if streamErr != nil && !headersSent {
				w.Header().Set("content-type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(`{"status":"error","message":"upstream stream failed"}`))
			}
			return
		}

		// Wait for new chunks or completion.
		select {
		case <-buf.notify:
			// Stream finished — do one final drain.
			continue
		case <-buf.chunkCh:
			continue
		case <-r.Context().Done():
			return
		case <-time.After(streamOutputTimeout):
			return
		}
	}
}
