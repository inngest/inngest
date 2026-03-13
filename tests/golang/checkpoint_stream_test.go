package golang

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// checkpointStreamHarness establishes both sides of a streaming checkpoint
// connection against the real Dev Server.
//
// Tests use `pushChunk` and `expectChunk` to verify data flows incrementally.
type checkpointStreamHarness struct {
	t *testing.T
	r *require.Assertions

	// SDK writes here (goes to `POST /v1/checkpoint/{runID}/stream`)
	Writer *io.PipeWriter // app side: write chunks here

	// Client reads here (goes to `GET /v1/checkpoint/{runID}/stream`)
	Reader *bufio.Reader

	done chan struct{}
}

// newCheckpointStreamHarness opens a GET (output) and POST (ingest)
// connection to the dev server's stream endpoint, sends the initial
// header frame, and returns a ready-to-use harness.
//
// The client (GET) connects BEFORE the ingest (POST) starts, since
// there is no buffering — the broadcaster forwards data only to
// subscribers that are already connected.
func newCheckpointStreamHarness(t *testing.T) *checkpointStreamHarness {
	t.Helper()
	r := require.New(t)

	runID := ulid.MustNew(ulid.Now(), nil)

	// The GET (output) endpoint authenticates via a signed JWT in the query
	// string
	token, err := apiv1auth.CreateRunJWT(
		consts.DevServerRunJWTSecret,
		consts.DevServerEnvID,
		runID,
	)
	r.NoError(err)

	streamURL := DEV_URL + "/v1/checkpoint/" + runID.String() + "/stream"

	// Open the output (GET) connection FIRST. The client must be subscribed
	// before the app starts streaming, since there is no buffering.
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	t.Cleanup(cancel)

	outReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		streamURL+"?token="+token,
		nil,
	)
	r.NoError(err)

	outResp, err := http.DefaultClient.Do(outReq)
	r.NoError(err)
	t.Cleanup(func() { outResp.Body.Close() })

	// Give the subscription time to register with the broadcaster
	time.Sleep(100 * time.Millisecond)

	// `io.Pipe` gives the test precise control over when each chunk is
	// delivered to the ingest endpoint. Writes block until `Read` is called
	ingestReader, ingestWriter := io.Pipe()

	// Open the ingest (POST) connection in the background. Errors are reported
	// via channel rather than `t.Fatal` to avoid panics after the test function
	// returns
	ingestErr := make(chan error, 1)
	ingestDone := make(chan struct{})
	go func() {
		defer close(ingestDone)
		resp, err := http.Post(
			streamURL,
			"application/octet-stream",
			ingestReader,
		)
		if err != nil {
			ingestErr <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			ingestErr <- fmt.Errorf(
				"ingest returned status %d",
				resp.StatusCode,
			)
		}
	}()

	// The first line must be a JSON header frame (consumed by StreamIngest
	// but NOT forwarded to the broadcaster). We still send it because the
	// SDK protocol requires it.
	headerFrame := apiv1.StreamHeaderFrame{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "text/plain"},
	}
	byt, err := json.Marshal(headerFrame)
	r.NoError(err)
	_, err = ingestWriter.Write(append(byt, '\n'))
	r.NoError(err)

	// Wait for the ingest handler to read and forward the header frame
	time.Sleep(100 * time.Millisecond)

	select {
	case err := <-ingestErr:
		t.Fatalf("ingest request failed (is the dev server running?): %v", err)
	default:
	}

	// The client should have received SSE headers (the StreamHeaderFrame is
	// consumed by StreamIngest and NOT forwarded to the subscriber).
	r.Equal(200, outResp.StatusCode)
	r.Equal("text/event-stream", outResp.Header.Get("Content-Type"))

	return &checkpointStreamHarness{
		t:      t,
		r:      r,
		Writer: ingestWriter,
		Reader: bufio.NewReader(outResp.Body),
		done:   ingestDone,
	}
}

// pushChunk writes data to the Dev Server
func (h *checkpointStreamHarness) pushChunk(data string) {
	h.t.Helper()
	_, err := h.Writer.Write([]byte(data))
	h.r.NoError(err)
}

// expectChunk asserts that exactly the given data arrives on the client side
// within 2 seconds. Fails the test if the read times out, which indicates the
// response is being buffered instead of streamed
func (h *checkpointStreamHarness) expectChunk(expected string) {
	h.t.Helper()

	buf := make([]byte, len(expected))
	done := make(chan error, 1)
	go func() {
		_, err := io.ReadFull(h.Reader, buf)
		done <- err
	}()

	select {
	case err := <-done:
		h.r.NoError(err)
		h.r.Equal(expected, string(buf))
	case <-time.After(2 * time.Second):
		h.t.Fatalf(
			"timed out waiting for chunk %q — response is being buffered",
			expected,
		)
	}
}

// close signals EOF on the Dev Server and waits for it to finish
func (h *checkpointStreamHarness) close() {
	h.t.Helper()
	h.r.NoError(h.Writer.Close())
	<-h.done
}

// TestCheckpointStream_NotBuffered hits the real dev server endpoints to verify
// that chunks streamed into `POST /v1/checkpoint/{runID}/stream` are delivered
// incrementally (not buffered) via `GET /v1/checkpoint/{runID}/stream`
func TestCheckpointStream_NotBuffered(t *testing.T) {
	h := newCheckpointStreamHarness(t)

	// Push a chunk and assert it arrives immediately
	h.pushChunk("first chunk")
	h.expectChunk("first chunk")

	// Do it again to confirm continuous streaming, not a one-shot flush
	h.pushChunk("second chunk")
	h.expectChunk("second chunk")

	h.close()
}
