package apiv2

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/appsync"
	"github.com/inngest/inngest/pkg/cqrs/sync"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/pkg/syscode"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const testSigningKey = "signkey-test-deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

// signResponse mirrors inngestgo's signWithoutJCS for response signing.
func signResponse(t *testing.T, body []byte, key string) string {
	t.Helper()
	keyBytes := regexp.MustCompile(`^signkey-\w+-`).ReplaceAll([]byte(key), nil)
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, keyBytes)
	_, _ = mac.Write(body)
	_, _ = fmt.Fprintf(mac, "%d", ts)
	return fmt.Sprintf("t=%d&s=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

type fakeSigningKeyProvider struct {
	key string
}

func (f fakeSigningKeyProvider) GetSigningKeys(ctx context.Context) ([]*apiv2.SigningKey, error) {
	if f.key == "" {
		return nil, nil
	}
	return []*apiv2.SigningKey{{
		Key:       f.key,
		CreatedAt: timestamppb.Now(),
	}}, nil
}

type recordingAppSyncer struct {
	called    bool
	gotReq    sdk.RegisterRequest
	reply     *sync.Reply
	returnErr error
}

func (r *recordingAppSyncer) ProcessSync(ctx context.Context, req sdk.RegisterRequest) (*sync.Reply, error) {
	r.called = true
	r.gotReq = req
	return r.reply, r.returnErr
}

// inBandHandler returns a fake SDK server. customize runs after defaults so
// it can also override (including with empty values).
func inBandHandler(t *testing.T, customize func(w http.ResponseWriter, body *[]byte)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		body := sampleResponseBody(t)
		w.Header().Set(headers.HeaderKeySignature, signResponse(t, body, testSigningKey))
		w.Header().Set(headers.HeaderKeySDK, "go:0.7.0")
		if customize != nil {
			customize(w, &body)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
}

func sampleResponseBody(t *testing.T) []byte {
	t.Helper()
	r := require.New(t)
	resp := appsync.Response{
		AppID:       "my-app",
		SDKAuthor:   "inngest",
		SDKLanguage: "go",
		SDKVersion:  "0.7.0",
		URL:         "http://placeholder/api/inngest",
		Functions:   []sdk.SDKFunction{},
	}
	byt, err := json.Marshal(resp)
	r.NoError(err)
	return byt
}

// requireErrorWithCode asserts the gRPC status and the first embedded ErrorItem code.
func requireErrorWithCode(t *testing.T, err error, wantStatus codes.Code, wantCode string) {
	t.Helper()
	r := require.New(t)
	r.Error(err)
	st, ok := status.FromError(err)
	r.True(ok, "expected gRPC status, got %T", err)
	r.Equal(wantStatus, st.Code(), "grpc code mismatch (msg=%q)", st.Message())

	var resp apiv2base.ErrorResponse
	r.NoError(json.Unmarshal([]byte(st.Message()), &resp), "message=%q", st.Message())
	r.NotEmpty(resp.Errors)
	r.Equal(wantCode, resp.Errors[0].Code)
}

func TestSyncApp(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r := require.New(t)
		srv := inBandHandler(t, nil)
		defer srv.Close()

		syncID := uuid.New()
		appID := uuid.New()
		syncer := &recordingAppSyncer{
			reply: &sync.Reply{OK: true, AppID: &appID, SyncID: &syncID},
		}
		service := NewService(ServiceOptions{
			SigningKeysProvider:      fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:                syncer,
			ServerKind:               headers.ServerKindDev,
			AppSyncAllowInsecureHTTP: true,
		})

		resp, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   srv.URL,
		})
		r.NoError(err)
		r.NotNil(resp)
		r.Equal(syncStatusSuccess, resp.Data.Status)
		r.Equal("my-app", resp.Data.AppId)
		r.Equal(syncID.String(), resp.Data.Id)
		r.Nil(resp.Data.Error)
		r.True(syncer.called)
		r.Equal("my-app", syncer.gotReq.AppName)
		r.Equal("go:0.7.0", syncer.gotReq.SDK)
	})

	t.Run("SDK returns bad signature returns 422", func(t *testing.T) {
		r := require.New(t)
		srv := inBandHandler(t, func(w http.ResponseWriter, _ *[]byte) {
			w.Header().Set(headers.HeaderKeySignature, fmt.Sprintf("t=%d&s=cafebabe", time.Now().Unix()))
		})
		defer srv.Close()

		syncer := &recordingAppSyncer{}
		service := NewService(ServiceOptions{
			SigningKeysProvider:      fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:                syncer,
			ServerKind:               headers.ServerKindDev,
			AppSyncAllowInsecureHTTP: true,
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   srv.URL,
		})
		requireErrorWithCode(t, err, codes.FailedPrecondition, syscode.CodeSigVerificationFailed)
		r.False(syncer.called, "processor should not be invoked on protocol failure")
	})

	t.Run("unreachable returns 422", func(t *testing.T) {
		// Closed server triggers a dial failure.
		srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		url := srv.URL
		srv.Close()

		syncer := &recordingAppSyncer{}
		service := NewService(ServiceOptions{
			SigningKeysProvider:      fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:                syncer,
			ServerKind:               headers.ServerKindDev,
			AppSyncAllowInsecureHTTP: true,
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   url,
		})
		requireErrorWithCode(t, err, codes.FailedPrecondition, syscode.CodeHTTPUnreachable)
	})

	t.Run("app_id mismatch fails sync", func(t *testing.T) {
		r := require.New(t)
		// SDK reports app_id "my-app" (sampleResponseBody default); caller asks
		// to sync "other-app". Sync must fail before ProcessSync runs.
		srv := inBandHandler(t, nil)
		defer srv.Close()

		syncer := &recordingAppSyncer{}
		service := NewService(ServiceOptions{
			SigningKeysProvider:      fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:                syncer,
			ServerKind:               headers.ServerKindDev,
			AppSyncAllowInsecureHTTP: true,
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "other-app",
			Url:   srv.URL,
		})
		requireErrorWithCode(t, err, codes.FailedPrecondition, syscode.CodeAppIDMismatch)
		r.False(syncer.called, "processor should not run on app_id mismatch")
	})

	t.Run("URL scheme rejection returns 400", func(t *testing.T) {
		// Service has AllowInsecureHTTP=false, so http:// is rejected by
		// appsync.checkScheme as CodeURLSchemeDenied. Should map to 400, not 422.
		service := NewService(ServiceOptions{
			SigningKeysProvider: fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:           &recordingAppSyncer{},
			ServerKind:          headers.ServerKindDev,
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   "http://example.com/api/inngest",
		})
		requireErrorWithCode(t, err, codes.InvalidArgument, syscode.CodeURLSchemeDenied)
	})

	t.Run("requires URL", func(t *testing.T) {
		r := require.New(t)
		service := NewService(ServiceOptions{
			SigningKeysProvider: fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:           &recordingAppSyncer{},
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
		})
		r.Error(err)
	})

	t.Run("requires app ID", func(t *testing.T) {
		r := require.New(t)
		service := NewService(ServiceOptions{
			SigningKeysProvider: fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:           &recordingAppSyncer{},
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			Url: "http://example.com",
		})
		r.Error(err)
	})

	t.Run("no app syncer wired", func(t *testing.T) {
		r := require.New(t)
		service := NewService(ServiceOptions{
			SigningKeysProvider: fakeSigningKeyProvider{key: testSigningKey},
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   "http://example.com",
		})
		r.Error(err)
	})

	t.Run("no signing key returns not implemented", func(t *testing.T) {
		service := NewService(ServiceOptions{
			SigningKeysProvider: fakeSigningKeyProvider{},
			AppSyncer:           &recordingAppSyncer{},
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   "http://example.com",
		})
		requireErrorWithCode(t, err, codes.Unimplemented, apiv2base.ErrorNotImplemented)
	})

	t.Run("no signing keys provider returns not implemented", func(t *testing.T) {
		service := NewService(ServiceOptions{
			AppSyncer: &recordingAppSyncer{},
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   "http://example.com",
		})
		requireErrorWithCode(t, err, codes.Unimplemented, apiv2base.ErrorNotImplemented)
	})

	t.Run("processor unknown error sanitized to 500", func(t *testing.T) {
		r := require.New(t)
		srv := inBandHandler(t, nil)
		defer srv.Close()

		syncer := &recordingAppSyncer{returnErr: errors.New("db down: connection refused at 10.0.0.5")}
		service := NewService(ServiceOptions{
			SigningKeysProvider:      fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:                syncer,
			ServerKind:               headers.ServerKindDev,
			AppSyncAllowInsecureHTTP: true,
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   srv.URL,
		})
		requireErrorWithCode(t, err, codes.Internal, apiv2base.ErrorInternalError)

		// The internal error text must NOT leak to the caller.
		st, _ := status.FromError(err)
		r.NotContains(st.Message(), "db down")
		r.NotContains(st.Message(), "10.0.0.5")
		r.Contains(st.Message(), "failed to process sync")
	})

	t.Run("processor publicerr propagates status and message", func(t *testing.T) {
		r := require.New(t)
		srv := inBandHandler(t, nil)
		defer srv.Close()

		syncer := &recordingAppSyncer{returnErr: publicerr.Error{
			Code:    "function_invalid",
			Message: "Function 'foo' is invalid",
			Status:  http.StatusBadRequest,
			Err:     errors.New("internal cause: should not leak"),
		}}
		service := NewService(ServiceOptions{
			SigningKeysProvider:      fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:                syncer,
			ServerKind:               headers.ServerKindDev,
			AppSyncAllowInsecureHTTP: true,
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   srv.URL,
		})
		requireErrorWithCode(t, err, codes.InvalidArgument, "function_invalid")
		st, _ := status.FromError(err)
		r.Contains(st.Message(), "Function 'foo' is invalid")
		r.NotContains(st.Message(), "internal cause")
	})

	t.Run("processor publicerr without code falls back to app_sync_failed", func(t *testing.T) {
		srv := inBandHandler(t, nil)
		defer srv.Close()

		syncer := &recordingAppSyncer{returnErr: publicerr.Wrap(
			errors.New("inner"), http.StatusBadRequest, "Invalid request",
		)}
		service := NewService(ServiceOptions{
			SigningKeysProvider:      fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:                syncer,
			ServerKind:               headers.ServerKindDev,
			AppSyncAllowInsecureHTTP: true,
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   srv.URL,
		})
		requireErrorWithCode(t, err, codes.InvalidArgument, apiv2base.ErrorAppSyncFailed)
	})

	t.Run("processor syscode error maps to 422", func(t *testing.T) {
		srv := inBandHandler(t, nil)
		defer srv.Close()

		syncer := &recordingAppSyncer{returnErr: &syscode.Error{
			Code:    "function_count_invalid",
			Message: "too many functions",
		}}
		service := NewService(ServiceOptions{
			SigningKeysProvider:      fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:                syncer,
			ServerKind:               headers.ServerKindDev,
			AppSyncAllowInsecureHTTP: true,
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   srv.URL,
		})
		requireErrorWithCode(t, err, codes.FailedPrecondition, "function_count_invalid")
	})

	t.Run("rate limited", func(t *testing.T) {
		r := require.New(t)
		syncer := &recordingAppSyncer{}
		service := NewService(ServiceOptions{
			SigningKeysProvider:      fakeSigningKeyProvider{key: testSigningKey},
			AppSyncer:                syncer,
			ServerKind:               headers.ServerKindDev,
			AppSyncAllowInsecureHTTP: true,
			RateLimitProvider:        stubRateLimitProvider{limited: true},
		})

		_, err := service.SyncApp(context.Background(), &apiv2.SyncAppRequest{
			AppId: "my-app",
			Url:   "http://example.com",
		})
		requireErrorWithCode(t, err, codes.ResourceExhausted, apiv2base.ErrorRateLimited)
		r.False(syncer.called, "no outbound work when rate limited")
	})
}

type stubRateLimitProvider struct {
	limited bool
}

func (s stubRateLimitProvider) CheckRateLimit(context.Context, string) RateLimitResult {
	return RateLimitResult{Limited: s.limited}
}
