package apiv1

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/execution/realtime"
	rtypes "github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
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

// StreamIngest handles POST /checkpoint/{runID}/stream.
// The SDK streams the HTTP response body through this endpoint.
//
// The first line of the request body (up to \n) MUST be a JSON-encoded
// StreamHeaderFrame. Everything after the first newline is raw body data
// streamed in chunks.
func (a checkpointAPI) StreamIngest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	auth, err := a.AuthFinder(ctx)
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

	broadcaster := a.Broadcaster
	if broadcaster == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "Streaming not available"))
		return
	}

	envID := auth.WorkspaceID()
	channel := runID.String()

	defer r.Body.Close()

	reader := bufio.NewReader(r.Body)

	// Read and consume the header frame (first line). The header frame is
	// SDK-specific metadata (status code, headers) that is NOT forwarded
	// through the broadcaster. Only the raw body bytes (SSE-formatted) are
	// broadcast, so that any subscriber (checkpoint StreamOutput, realtime
	// SSE, etc.) receives clean data.
	headerLine, err := reader.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		logger.StdlibLogger(ctx).Warn("error reading stream header", "error", err, "run_id", runID)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 400, "Error reading stream header"))
		return
	}
	_ = headerLine // consumed but not broadcast

	// Read the remaining body in chunks, forwarding each to the broadcaster.
	for {
		chunk := make([]byte, streamReadChunkSize)
		n, readErr := reader.Read(chunk)

		if n > 0 {
			broadcaster.Write(ctx, envID, channel, chunk[:n])
		}

		if errors.Is(readErr, io.EOF) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if readErr != nil {
			logger.StdlibLogger(ctx).Warn("error reading stream body",
				"error", readErr, "run_id", runID)
			_ = publicerr.WriteHTTP(w, publicerr.Wrap(readErr, 500, "Error reading stream"))
			return
		}
	}
}

// StreamOutput handles GET /checkpoint/{runID}/stream?token=<jwt>.
// The client consumes the streamed response through this endpoint.
//
// The client MUST connect before the app starts streaming (there is no
// buffering). The first write from the broadcaster is expected to be a
// JSON-encoded StreamHeaderFrame.
func (a checkpointAPI) StreamOutput(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := r.URL.Query().Get("token")
	claims, err := apiv1auth.VerifyRunJWT(ctx, a.runClaimsSecret, token)
	if err != nil || claims == nil {
		w.WriteHeader(401)
		_ = publicerr.WriteHTTP(w, publicerr.Wrap(err, 401, "Unable to authenticate"))
		return
	}

	broadcaster := a.Broadcaster
	if broadcaster == nil {
		_ = publicerr.WriteHTTP(w, publicerr.Errorf(500, "Streaming not available"))
		return
	}

	runID := claims.RunID
	envID := claims.Env
	channel := runID.String()

	sub := newRawSubscription(w)

	topic := rtypes.Topic{
		Kind:    rtypes.TopicKindRun,
		EnvID:   envID,
		Channel: channel,
		Name:    streamTopicName,
	}

	if err := broadcaster.Subscribe(ctx, sub, []rtypes.Topic{topic}); err != nil {
		logger.StdlibLogger(ctx).Error("error subscribing to stream", "error", err, "run_id", runID)
		http.Error(w, "error subscribing to stream", http.StatusInternalServerError)
		return
	}

	// Block until the client disconnects or the stream times out.
	timeout := time.NewTimer(streamOutputTimeout)
	defer timeout.Stop()

	select {
	case <-ctx.Done():
	case <-timeout.C:
	}

	if err := broadcaster.CloseSubscription(ctx, sub.ID()); err != nil {
		logger.StdlibLogger(ctx).Warn("error closing stream subscription", "error", err, "sub_id", sub.ID())
	}
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
