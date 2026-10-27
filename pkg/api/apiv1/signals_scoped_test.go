package apiv1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type resumeCall struct {
	workspaceID uuid.UUID
	signalID    string
	data        string
}

type fakeSignalExecutor struct {
	execution.Executor
	matched bool
	calls   []resumeCall
}

func (f *fakeSignalExecutor) ResumeSignal(_ context.Context, workspaceID uuid.UUID, signalID string, data json.RawMessage) (*execution.ResumeSignalResult, error) {
	f.calls = append(f.calls, resumeCall{workspaceID, signalID, string(data)})
	if !f.matched {
		return &execution.ResumeSignalResult{}, nil
	}
	runID := ulid.Make()
	return &execution.ResumeSignalResult{MatchedSignal: true, RunID: &runID}, nil
}

func scopedSignalRequest(t *testing.T, handler http.Handler, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/signals/scoped", strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func TestReceiveScopedSignal(t *testing.T) {
	secret := []byte("test-secret")
	env := uuid.New()
	valid := func(t *testing.T, signalID string, expiresAt time.Time) string {
		token, err := apiv1auth.CreateSignalJWT(secret, env, signalID, expiresAt)
		require.NoError(t, err)
		return token
	}

	setup := func(matched bool) (http.Handler, *fakeSignalExecutor) {
		exec := &fakeSignalExecutor{matched: matched}
		r := chi.NewRouter()
		AddRoutes(r, Opts{Executor: exec, SignalJWTSecret: secret})
		return r, exec
	}

	t.Run("sends the token's signal into the token's environment", func(t *testing.T) {
		h, exec := setup(true)
		res := scopedSignalRequest(t, h, valid(t, "build:1", time.Now().Add(time.Hour)), `{"signal":"build:1","data":{"exitCode":3}}`)
		require.Equal(t, http.StatusOK, res.Code)
		require.Equal(t, []resumeCall{{env, "build:1", `{"exitCode":3}`}}, exec.calls)
	})

	t.Run("rejects a token used for another signal", func(t *testing.T) {
		h, exec := setup(true)
		res := scopedSignalRequest(t, h, valid(t, "build:1", time.Now().Add(time.Hour)), `{"signal":"build:2","data":{}}`)
		require.Equal(t, http.StatusForbidden, res.Code)
		require.Empty(t, exec.calls)
	})

	t.Run("rejects an expired token", func(t *testing.T) {
		h, exec := setup(true)
		res := scopedSignalRequest(t, h, valid(t, "build:1", time.Now().Add(-time.Minute)), `{"signal":"build:1","data":{}}`)
		require.Equal(t, http.StatusUnauthorized, res.Code)
		require.Empty(t, exec.calls)
	})

	t.Run("rejects a missing or malformed token", func(t *testing.T) {
		h, exec := setup(true)
		require.Equal(t, http.StatusUnauthorized, scopedSignalRequest(t, h, "", `{"signal":"build:1"}`).Code)
		require.Equal(t, http.StatusUnauthorized, scopedSignalRequest(t, h, "not-a-jwt", `{"signal":"build:1"}`).Code)
		require.Empty(t, exec.calls)
	})

	t.Run("rejects a run token", func(t *testing.T) {
		h, exec := setup(true)
		runToken, err := apiv1auth.CreateRunJWT(secret, env, ulid.Make())
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, scopedSignalRequest(t, h, runToken, `{"signal":"build:1"}`).Code)
		require.Empty(t, exec.calls)
	})

	t.Run("returns 404 when no wait matches", func(t *testing.T) {
		h, _ := setup(false)
		res := scopedSignalRequest(t, h, valid(t, "build:1", time.Now().Add(time.Hour)), `{"signal":"build:1","data":{}}`)
		require.Equal(t, http.StatusNotFound, res.Code)
	})

	t.Run("rejects an oversized body", func(t *testing.T) {
		h, exec := setup(true)
		big := `{"signal":"build:1","data":"` + strings.Repeat("a", maxScopedSignalBodyBytes) + `"}`
		res := scopedSignalRequest(t, h, valid(t, "build:1", time.Now().Add(time.Hour)), big)
		require.Equal(t, http.StatusRequestEntityTooLarge, res.Code)
		require.Empty(t, exec.calls)
	})

	t.Run("rejects data after the body", func(t *testing.T) {
		h, exec := setup(true)
		res := scopedSignalRequest(t, h, valid(t, "build:1", time.Now().Add(time.Hour)), `{"signal":"build:1","data":{}} garbage`)
		require.Equal(t, http.StatusBadRequest, res.Code)
		require.Empty(t, exec.calls)
	})

	t.Run("route is not mounted without a secret", func(t *testing.T) {
		r := chi.NewRouter()
		AddRoutes(r, Opts{Executor: &fakeSignalExecutor{}})
		res := scopedSignalRequest(t, r, "anything", `{"signal":"x"}`)
		require.Equal(t, http.StatusNotFound, res.Code)
	})
}

func TestReceiveScopedSignalRejectsForgedTokens(t *testing.T) {
	secret := []byte("test-secret")
	exec := &fakeSignalExecutor{matched: true}
	r := chi.NewRouter()
	AddRoutes(r, Opts{Executor: exec, SignalJWTSecret: secret})
	body := `{"signal":"build:1","data":{}}`

	good, err := apiv1auth.CreateSignalJWT(secret, uuid.New(), "build:1", time.Now().Add(time.Hour))
	require.NoError(t, err)

	t.Run("tampered signature", func(t *testing.T) {
		require.Equal(t, http.StatusUnauthorized, scopedSignalRequest(t, r, good[:len(good)-2]+"xx", body).Code)
	})

	t.Run("alg none", func(t *testing.T) {
		unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, apiv1auth.SignalClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    apiv1auth.SignalClaimsIssuer,
				Subject:   apiv1auth.SignalClaimsSubject,
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			Env:      uuid.New(),
			SignalID: "build:1",
		})
		token, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, scopedSignalRequest(t, r, token, body).Code)
	})

	t.Run("nil environment", func(t *testing.T) {
		token, err := apiv1auth.CreateSignalJWT(secret, uuid.Nil, "build:1", time.Now().Add(time.Hour))
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, scopedSignalRequest(t, r, token, body).Code)
	})

	require.Empty(t, exec.calls)
}
