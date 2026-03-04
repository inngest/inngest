package apiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestStreamIngestAndOutput(t *testing.T) {
	r := require.New(t)
	secret := []byte("test-secret-key-for-jwt-signing")
	envID := uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil)

	token, err := apiv1auth.CreateRunJWT(secret, envID, runID)
	r.NoError(err)

	streams := newStreamRegistry()
	api := &checkpointAPI{
		Router:          chi.NewRouter(),
		Opts:            Opts{AuthFinder: apiv1auth.NilAuthFinder},
		runClaimsSecret: secret,
		streams:         streams,
	}

	// Build a payload: header frame + body chunks.
	headerFrame := StreamHeaderFrame{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "text/plain"},
	}
	headerBytes, err := json.Marshal(headerFrame)
	r.NoError(err)

	bodyChunks := []string{"hello ", "world", "!"}

	// Combine into one request body: header frame on first line, then body.
	var ingestBody bytes.Buffer
	ingestBody.Write(headerBytes)
	ingestBody.WriteByte('\n')
	for _, chunk := range bodyChunks {
		ingestBody.WriteString(chunk)
	}

	// Start ingesting in the background.
	ingestDone := make(chan struct{})
	go func() {
		defer close(ingestDone)

		// We need to set up a chi context with the runID param.
		req := httptest.NewRequest(http.MethodPost, "/checkpoint/"+runID.String()+"/stream", &ingestBody)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("runID", runID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rec := httptest.NewRecorder()
		api.StreamIngest(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	}()

	// Give the ingest handler a moment to start buffering.
	time.Sleep(50 * time.Millisecond)

	// Now connect as the client to read the stream output.
	req := httptest.NewRequest(http.MethodGet, "/checkpoint/"+runID.String()+"/stream?token="+token, nil)
	rec := httptest.NewRecorder()

	api.StreamOutput(rec, req)

	// Wait for ingest to complete.
	<-ingestDone

	result := rec.Result()
	defer result.Body.Close()

	require.Equal(t, 200, result.StatusCode)
	require.Equal(t, "text/plain", result.Header.Get("Content-Type"))

	body, err := io.ReadAll(result.Body)
	require.NoError(t, err)
	require.Equal(t, "hello world!", string(body))
}
