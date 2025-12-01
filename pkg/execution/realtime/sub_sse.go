package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// NewSSESubscription creates a new server-sent-event based subscription that fulfils the
// Subscriber interface.
func NewSSESubscription(
	ctx context.Context,
	w http.ResponseWriter,
) subSSE {
	// Ensure SSE headers are sent.
	sseHeaders(w)

	return subSSE{
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
	id uuid.UUID
	w  http.ResponseWriter
}

// ID returns a unique ID for the given subscription
func (s subSSE) ID() uuid.UUID {
	return s.id
}

// Protocol is the name of the protocol/implementation
func (s subSSE) Protocol() string {
	return "sse"
}

func (s subSSE) Write(b []byte) error {
	return s.write(b)
}

// SendKeepalive is called by the broadcaster to keep the current connection alive.  This
// may be a noop, depending on the implementation.  Note that keepalives are sent every
// 30 seconds - this is not implementation specific.
//
// If SendKeepalive fails consecutively, the subscription will be closed.
func (s subSSE) SendKeepalive(m Message) error {
	// Send the minimum empty message to ensure the conn is active.
	return s.write([]byte(":\n\n"))
}

// WriteMessage allows the writing of messages to the particular subscription.  This is
// implementation agnostic;  messages may be written via websockets or HTTP connections
// for server-sent-events.
//
// Note that each subscription implementation may write different formats of a Message,
// so this cannot fulfil io.Writer.
func (s subSSE) WriteMessage(m Message) error {
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
func (s subSSE) WriteChunk(c Chunk) error {
	byt, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.writeSSE(byt)
}

// Closer closes the current subscription immediately, terminating any active connections.
func (s subSSE) Close() error {
	// Writer is a reguler http.ResponseWriter.  This is usually handled and closed
	// by the http server.  However, when Close is called we can attempt to hijack this
	// conn and call Close directly.
	hj, ok := s.w.(http.Hijacker)
	if !ok {
		return nil
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return fmt.Errorf("error hijacking sse sub: %w", err)
	}
	if err := bufrw.Flush(); err != nil {
		return fmt.Errorf("error flushing hijacked sse sub: %w", err)
	}
	return conn.Close()
}

// writeSSE formats the data as a proper SSE event and writes it
func (s subSSE) writeSSE(data []byte) error {
	// Format as SSE event: "data: {json}\n\n"
	sseData := fmt.Sprintf("data: %s\n\n", data)
	return s.write([]byte(sseData))
}

func (s subSSE) write(b []byte) error {
	if _, err := s.w.Write(b); err != nil {
		return err
	}
	if f, ok := s.w.(http.Flusher); ok {
		f.Flush()
	}
	return nil
}
