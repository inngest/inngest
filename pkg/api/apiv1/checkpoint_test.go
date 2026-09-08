package apiv1

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/apiresult"
	"github.com/inngest/inngest/pkg/execution/executor"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// mockOutputReader implements RunOutputReader for testing
type mockOutputReader struct {
	output []byte
	err    error
}

func (m *mockOutputReader) RunOutput(ctx context.Context, envID uuid.UUID, runID ulid.ULID) ([]byte, error) {
	return m.output, m.err
}

func TestCheckpointAPI_Output(t *testing.T) {
	secret := []byte("test-secret-key-for-jwt-signing")
	envID := uuid.New()
	runID := ulid.MustNew(ulid.Now(), nil)

	// Create a valid JWT for testing
	token, err := apiv1auth.CreateRunJWT(secret, envID, runID)
	require.NoError(t, err)

	t.Run("extracts status code, headers, and body from wrapped APIResult", func(t *testing.T) {
		// Create a wrapped APIResult with custom status, headers, and body
		apiRes := apiresult.APIResult{
			StatusCode: 201,
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
				"Content-Type":    "text/plain",
			},
			Body: "response body content",
		}
		wrappedOutput, err := json.Marshal(map[string]any{"data": apiRes})
		require.NoError(t, err)

		reader := &mockOutputReader{output: wrappedOutput}
		api := &checkpointAPI{
			Router:          chi.NewRouter(),
			runClaimsSecret: secret,
			outputReader:    reader,
		}

		req := httptest.NewRequest(http.MethodGet, "/test/output?token="+token, nil)
		rec := httptest.NewRecorder()

		api.Output(rec, req)

		require.Equal(t, 201, rec.Code, "should use status code from APIResult")
		require.Equal(t, "custom-value", rec.Header().Get("X-Custom-Header"), "should set custom headers from APIResult")
		require.Equal(t, "text/plain", rec.Header().Get("Content-Type"), "should set content-type from APIResult")
		require.Equal(t, "response body content", rec.Body.String(), "should return body from APIResult")
	})

	t.Run("handles APIResult with error status code", func(t *testing.T) {
		apiRes := apiresult.APIResult{
			StatusCode: 500,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"error":"internal server error"}`,
		}
		wrappedOutput, err := json.Marshal(map[string]any{"data": apiRes})
		require.NoError(t, err)

		reader := &mockOutputReader{output: wrappedOutput}
		api := &checkpointAPI{
			Router:          chi.NewRouter(),
			runClaimsSecret: secret,
			outputReader:    reader,
		}

		req := httptest.NewRequest(http.MethodGet, "/test/output?token="+token, nil)
		rec := httptest.NewRecorder()

		api.Output(rec, req)

		require.Equal(t, 500, rec.Code, "should use 500 status code from APIResult")
		require.Equal(t, `{"error":"internal server error"}`, rec.Body.String())
	})

	t.Run("handles APIResult with empty body", func(t *testing.T) {
		apiRes := apiresult.APIResult{
			StatusCode: 204,
			Headers:    map[string]string{},
			Body:       "",
		}
		wrappedOutput, err := json.Marshal(map[string]any{"data": apiRes})
		require.NoError(t, err)

		reader := &mockOutputReader{output: wrappedOutput}
		api := &checkpointAPI{
			Router:          chi.NewRouter(),
			runClaimsSecret: secret,
			outputReader:    reader,
		}

		req := httptest.NewRequest(http.MethodGet, "/test/output?token="+token, nil)
		rec := httptest.NewRecorder()

		api.Output(rec, req)

		require.Equal(t, 204, rec.Code, "should use 204 status code from APIResult")
		require.Empty(t, rec.Body.String(), "should have empty body")
	})

	t.Run("falls back to raw output when not wrapped APIResult format", func(t *testing.T) {
		// Raw JSON that isn't wrapped in {"data":...}
		rawOutput := []byte(`{"some":"other","format":"here"}`)

		reader := &mockOutputReader{output: rawOutput}
		api := &checkpointAPI{
			Router:          chi.NewRouter(),
			runClaimsSecret: secret,
			outputReader:    reader,
		}

		req := httptest.NewRequest(http.MethodGet, "/test/output?token="+token, nil)
		rec := httptest.NewRecorder()

		api.Output(rec, req)

		require.Equal(t, 200, rec.Code, "should default to 200 when falling back to raw output")
		require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		require.Equal(t, `{"some":"other","format":"here"}`, rec.Body.String())
	})

	t.Run("returns 401 for invalid token", func(t *testing.T) {
		reader := &mockOutputReader{output: []byte(`{}`)}
		api := &checkpointAPI{
			Router:          chi.NewRouter(),
			runClaimsSecret: secret,
			outputReader:    reader,
		}

		req := httptest.NewRequest(http.MethodGet, "/test/output?token=invalid-token", nil)
		rec := httptest.NewRecorder()

		api.Output(rec, req)

		require.Equal(t, 401, rec.Code)
	})

	t.Run("returns 401 for missing token", func(t *testing.T) {
		reader := &mockOutputReader{output: []byte(`{}`)}
		api := &checkpointAPI{
			Router:          chi.NewRouter(),
			runClaimsSecret: secret,
			outputReader:    reader,
		}

		req := httptest.NewRequest(http.MethodGet, "/test/output", nil)
		rec := httptest.NewRecorder()

		api.Output(rec, req)

		require.Equal(t, 401, rec.Code)
	})
}

type scheduleErrExecutor struct {
	execution.Executor
	err error
}

func (e scheduleErrExecutor) Schedule(ctx context.Context, r execution.ScheduleRequest) (*ulid.ULID, *sv2.Metadata, error) {
	return nil, nil, e.err
}

type noopPublisher struct {
	wg *sync.WaitGroup
}

func (p noopPublisher) Publish(ctx context.Context, evt event.TrackedEvent) error {
	p.wg.Done()
	return nil
}

type noopAppCreator struct {
	cqrs.AppCreator
}

func (noopAppCreator) UpsertApp(ctx context.Context, arg cqrs.UpsertAppParams) (*cqrs.App, error) {
	return &cqrs.App{ID: arg.ID}, nil
}

type noopFunctionCreator struct {
	cqrs.FunctionCreator
	wg *sync.WaitGroup
}

func (f noopFunctionCreator) UpsertFunction(ctx context.Context, params cqrs.UpsertFunctionParams) (*cqrs.Function, error) {
	f.wg.Done()
	return &cqrs.Function{ID: params.ID}, nil
}

func TestCheckpointAPI_CheckpointNewRun_ScheduleErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "skipped run returns 403 with reason",
			err:        executor.SkippedError{Reason: enums.SkipReasonAccountExecutionCapHit},
			wantStatus: http.StatusForbidden,
			wantMsg:    "AccountExecutionCapHit",
		},
		{
			name:       "rate limited returns 429",
			err:        executor.ErrFunctionRateLimited,
			wantStatus: http.StatusTooManyRequests,
			wantMsg:    "Rate limits exceeded",
		},
		{
			name:       "unknown error returns 500",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "Failed to schedule run",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			wg.Add(2)

			api := NewCheckpointAPI(Opts{
				AuthFinder:      apiv1auth.NilAuthFinder,
				Executor:        scheduleErrExecutor{err: tc.err},
				EventPublisher:  noopPublisher{wg: wg},
				AppCreator:      noopAppCreator{},
				FunctionCreator: noopFunctionCreator{wg: wg},
			})

			body, err := json.Marshal(CheckpointNewRunRequest{
				RunID: ulid.MustNew(ulid.Now(), rand.Reader),
				Event: inngestgo.GenericEvent[NewAPIRunData]{
					Name: "http/request",
					Data: NewAPIRunData{
						Domain: "https://example.com",
						Method: http.MethodPost,
						Path:   "/api/test",
					},
				},
			})
			require.NoError(t, err)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))

			api.CheckpointNewRun(rec, req)
			wg.Wait()

			require.Equal(t, tc.wantStatus, rec.Code)

			resp := struct {
				Error  string `json:"error"`
				Status int    `json:"status"`
			}{}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.Equal(t, tc.wantStatus, resp.Status)
			require.Contains(t, resp.Error, tc.wantMsg)
		})
	}
}
