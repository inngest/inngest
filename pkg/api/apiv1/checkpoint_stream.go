package apiv1

import (
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime"
)

const (
	// streamReadChunkSize is the size of each read from the app's request body.
	streamReadChunkSize = 4 * 1024 // 4KB
	// streamOutputTimeout is how long the client endpoint waits for the stream to start.
	streamOutputTimeout = 5 * time.Minute
	// streamTopicName is the topic name used for durable endpoint streaming.
	streamTopicName = "$stream"
)

// StreamHeaderFrame is the first frame sent by the SDK, containing the
// HTTP response metadata. All subsequent data is raw body bytes.
type StreamHeaderFrame struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
}

// rawSubscription implements realtime.Subscription to forward raw bytes from
// the broadcaster to an http.ResponseWriter. The body data is expected to be
// SSE-formatted (from the SDK), so we set text/event-stream headers up front.
type rawSubscription struct {
	id uuid.UUID
	w  http.ResponseWriter
	rc *http.ResponseController
	mu sync.Mutex
}

func newRawSubscription(w http.ResponseWriter) *rawSubscription {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	_ = rc.Flush()

	return &rawSubscription{
		id: uuid.New(),
		w:  w,
		rc: rc,
	}
}

func (s *rawSubscription) ID() uuid.UUID {
	return s.id
}

func (s *rawSubscription) Protocol() string {
	return "raw"
}

func (s *rawSubscription) SendKeepalive(_ realtime.Message) error {
	// Raw streaming has no keepalive framing; just flush to detect broken conns.
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rc.Flush()
}

func (s *rawSubscription) WriteMessage(_ realtime.Message) error {
	// Not used — StreamIngest publishes via Broadcaster.Write, not Publish.
	return nil
}

func (s *rawSubscription) WriteChunk(_ realtime.Chunk) error {
	// Not used — StreamIngest publishes via Broadcaster.Write, not PublishChunk.
	return nil
}

func (s *rawSubscription) Write(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.w.Write(b); err != nil {
		return err
	}
	return s.rc.Flush()
}

func (s *rawSubscription) Close() error {
	return nil
}
