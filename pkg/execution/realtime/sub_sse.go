package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
)

// NewSSESubscription creates a new server-sent-event based subscription that fulfils the
// Subscriber interface.
func NewSSESubscription(
	ctx context.Context,
	w http.ResponseWriter,
) *subSSE {
	// Ensure SSE headers are sent and flushed immediately so that
	// clients (e.g. browser fetch) receive the response headers
	// without waiting for the first data write or keepalive.
	sseHeaders(w)
	w.WriteHeader(http.StatusOK)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return &subSSE{
		id: uuid.New(),
		w:  w,
	}
}

func sseHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

type subSSE struct {
	id     uuid.UUID
	w      http.ResponseWriter
	mu     sync.Mutex // Protects concurrent writes to http.ResponseWriter
	closed bool       // Set when the handler is returning; prevents writes after the response is finalized
}

// ID returns a unique ID for the given subscription
func (s *subSSE) ID() uuid.UUID {
	return s.id
}

// Protocol is the name of the protocol/implementation
func (s *subSSE) Protocol() string {
	return "sse"
}

func (s *subSSE) Write(b []byte) error {
	return s.write(b)
}

// SendKeepalive is called by the broadcaster to keep the current connection alive.  This
// may be a noop, depending on the implementation.  Note that keepalives are sent every
// 30 seconds - this is not implementation specific.
//
// If SendKeepalive fails consecutively, the subscription will be closed.
func (s *subSSE) SendKeepalive(m Message) error {
	// Send the minimum empty message to ensure the conn is active.
	return s.write([]byte(":\n\n"))
}

// WriteMessage allows the writing of messages to the particular subscription.  This is
// implementation agnostic;  messages may be written via websockets or HTTP connections
// for server-sent-events.
//
// Note that each subscription implementation may write different formats of a Message,
// so this cannot fulfil io.Writer.
func (s *subSSE) WriteMessage(m Message) error {
	// Ensure that m.Data - a RawMessage - is valid JSON.
	if !json.Valid(m.Data) {
		enc, err := json.Marshal(string(m.Data))
		if err != nil {
			return err
		}
		m.Data = enc
	}

	byt, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return s.writeSSE(byt)
}

// WriteChunk publishes a chunk in a stream - data for a given stream ID to the subscription.
func (s *subSSE) WriteChunk(c Chunk) error {
	byt, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.writeSSE(byt)
}

// CloseWriter marks the subscription as closed so that no further writes will
// be attempted on the underlying `http.ResponseWriter`. This MUST be called
// before the HTTP handler returns to avoid a data race between the keepalive
// goroutine and the HTTP server finalizing the response.
//
// Unlike Close, this does not hijack the connection. The HTTP server retains
// ownership of the connection lifecycle.
func (s *subSSE) CloseWriter() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

// Close marks the subscription as closed and forcefully terminates the
// underlying connection by hijacking it from the HTTP server.  This is
// needed during broadcaster shutdown to unblock handler goroutines that
// are waiting in a select.
func (s *subSSE) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true

	hj, ok := s.w.(http.Hijacker)
	if !ok {
		return nil
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		// Already closed or finalized by the HTTP server — not an error.
		return nil
	}
	_ = bufrw.Flush()
	return conn.Close()
}

// writeSSE formats the data as a proper SSE event and writes it
func (s *subSSE) writeSSE(data []byte) error {
	// Format as SSE event: "data: {json}\n\n"
	sseData := fmt.Sprintf("data: %s\n\n", data)
	return s.write([]byte(sseData))
}

func (s *subSSE) write(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	if _, err := s.w.Write(b); err != nil {
		return err
	}
	if f, ok := s.w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}
