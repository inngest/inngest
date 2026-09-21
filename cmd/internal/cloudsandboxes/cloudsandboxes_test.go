package cloudsandboxes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	cliauth "github.com/inngest/inngest/cmd/internal/auth"
	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

const sandboxID = "d1d71e35-615c-47c6-920f-1b228d10441b"

func testBridge(t *testing.T, handler http.HandlerFunc) (*Bridge, *cliauth.Metadata) {
	t.Helper()
	upstream := httptest.NewServer(handler)
	t.Cleanup(upstream.Close)
	t.Setenv("INNGEST_API_HOST", upstream.URL)
	t.Setenv("INNGEST_CONFIG_DIR", t.TempDir())
	b, err := New(t.Context(), 8288)
	require.NoError(t, err)
	workspace := "c3e3c7f5-936a-4af4-9653-f030625cc877"
	metadata := &cliauth.Metadata{
		Issuer: upstream.URL, Resource: upstream.URL + "/v2", ClientID: cliauth.ClientID,
		AccountID: "account-a", AccountName: "Test organization", SessionID: "session-a",
		WorkspaceID: &workspace, WorkspaceName: "Development", SessionExpiresAt: time.Now().Add(time.Hour),
	}
	saveSession(t, b, metadata)
	return b, metadata
}

func saveSession(t *testing.T, b *Bridge, metadata *cliauth.Metadata) {
	t.Helper()
	require.NoError(t, b.auth.Store().Save(*metadata, cliauth.Credential{
		AccessToken: "cloud-access", RefreshToken: "cloud-refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour),
	}, true))
}

func request(b *Bridge, method, path string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://localhost:8288"+path, nil)
	r.Header.Set("Authorization", "Bearer "+b.Token)
	w := httptest.NewRecorder()
	b.ServeHTTP(w, r)
	return w
}

func TestLocalAuthorizationAndRouteBoundary(t *testing.T) {
	var calls atomic.Int32
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeData(w, map[string]any{})
	})
	b.bound = metadata
	for _, test := range []struct {
		name, method, path, host, origin, token string
		status                                  int
	}{
		{"missing capability", "GET", "/dev/cloud/status", "localhost:8288", "", "", 401},
		{"Cloud key is not local auth", "GET", "/v2/sandboxes", "localhost:8288", "", "cloud-access", 401},
		{"cross origin", "POST", "/v2/sandboxes", "localhost:8288", "https://evil.example", b.Token, 403},
		{"opaque origin", "GET", "/dev/cloud/status", "localhost:8288", "null", b.Token, 403},
		{"DNS rebinding", "GET", "/dev/cloud/status", "evil.example:8288", "", b.Token, 403},
		{"other port", "GET", "/dev/cloud/status", "localhost:9999", "", b.Token, 403},
		{"unrelated API", "POST", "/v2/events", "localhost:8288", "", b.Token, 404},
		{"credential API", "GET", "/v2/signing-keys", "localhost:8288", "", b.Token, 404},
		{"unknown sandbox method", "PATCH", "/v2/sandboxes/" + sandboxID, "localhost:8288", "", b.Token, 405},
		{"escaped path", "GET", "/v2/sandboxes/a%2Fb", "localhost:8288", "", b.Token, 400},
		{"dot path", "GET", "/v2/sandboxes/../account", "localhost:8288", "", b.Token, 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest(test.method, "http://localhost:8288"+test.path, nil)
			r.Host = test.host
			r.Header.Set("Origin", test.origin)
			r.Header.Set("Authorization", "Bearer "+test.token)
			w := httptest.NewRecorder()
			b.ServeHTTP(w, r)
			require.Equal(t, test.status, w.Code, w.Body.String())
		})
	}
	require.Zero(t, calls.Load())
	w := request(b, "GET", "/dev/cloud/status")
	require.Equal(t, 200, w.Code)
	for _, secret := range []string{b.Token, "cloud-access", "cloud-refresh"} {
		require.NotContains(t, w.Body.String(), secret)
	}
}

func TestAllSandboxRoutesAndHeaderIsolation(t *testing.T) {
	var path, method string
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, path, r.URL.Path)
		require.Equal(t, method, r.Method)
		require.Equal(t, "Bearer cloud-access", r.Header.Get("Authorization"))
		for _, header := range []string{"Cookie", "X-Inngest-Env", "Baggage", "Forwarded", "X-Forwarded-Host"} {
			require.Empty(t, r.Header.Get(header), header)
		}
		writeData(w, map[string]any{})
	})
	b.bound = metadata
	// Exercise real chi mounts as well as every sandbox route from the proto.
	mux := chi.NewRouter()
	for _, prefix := range []string{"/v2/sandboxes", "/v2/snapshots", "/api/v2/sandboxes", "/api/v2/snapshots"} {
		mux.Mount(prefix, b)
	}
	for _, endpoint := range apiv2endpoint.Discover() {
		if !strings.HasPrefix(endpoint.AuthzPermission, "sandboxes:") {
			continue
		}
		for _, prefix := range []string{"/v2", "/api/v2"} {
			p := endpoint.Path
			for _, param := range endpoint.PathParams {
				p = strings.ReplaceAll(p, "{"+param+"}", sandboxID)
			}
			path, method = "/v2"+p, endpoint.HTTPMethod
			r := httptest.NewRequest(method, "http://localhost:8288"+prefix+p, strings.NewReader(`{"command":["true"]}`))
			r.Header.Set("Authorization", "Bearer "+b.Token)
			r.Header.Set("Origin", "http://localhost:8288")
			for _, header := range []string{"Cookie", "X-Inngest-Env", "Baggage", "Forwarded", "X-Forwarded-Host"} {
				r.Header.Set(header, "do-not-forward")
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			require.Equal(t, 200, w.Code, "%s %s: %s", method, prefix+p, w.Body.String())
		}
	}
}

func TestBindingAndRecovery(t *testing.T) {
	var mutations atomic.Int32
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/account" {
			w.WriteHeader(403)
			return
		} // sandbox-only grant
		mutations.Add(1)
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(201)
		fmt.Fprintf(w, `{"data":{"id":%q,"name":"local-test","status":"RUNNING"}}`, sandboxID)
	})
	require.Equal(t, 401, request(b, "POST", "/v2/sandboxes").Code)
	require.Equal(t, 200, request(b, "POST", "/dev/cloud/connect").Code)
	create := request(b, "POST", "/v2/sandboxes")
	require.Equal(t, 201, create.Code, create.Body.String())
	require.JSONEq(t, fmt.Sprintf(`{"data":{"id":%q,"name":"local-test","status":"RUNNING"}}`, sandboxID), create.Body.String())

	restarted, err := New(t.Context(), 8288)
	require.NoError(t, err)
	require.NotEqual(t, b.Token, restarted.Token)
	require.Nil(t, restarted.bound) // Restart does not silently reconnect.
	require.Contains(t, request(restarted, "GET", "/dev/cloud/status").Body.String(), sandboxID)
	require.Equal(t, 200, request(restarted, "POST", "/dev/cloud/connect").Code)
	require.Equal(t, 204, request(restarted, "DELETE", "/v2/sandboxes/"+sandboxID).Code)
	require.NotContains(t, request(restarted, "GET", "/dev/cloud/status").Body.String(), sandboxID)

	other := *metadata
	workspace := "other-env"
	other.WorkspaceID = &workspace
	saveSession(t, restarted, &other)
	require.Equal(t, 409, request(restarted, "POST", "/v2/sandboxes").Code)
	require.Equal(t, 409, request(restarted, "POST", "/dev/cloud/connect").Code)
	require.Equal(t, int32(2), mutations.Load())
	require.Equal(t, 200, request(restarted, "POST", "/dev/cloud/disconnect").Code)
	require.Equal(t, 401, request(restarted, "GET", "/v2/sandboxes").Code)
	other.WorkspaceID = nil
	saveSession(t, restarted, &other)
	require.Equal(t, 400, request(restarted, "POST", "/dev/cloud/connect").Code)
	require.NoError(t, restarted.auth.Store().Delete(nil))
	require.Equal(t, 401, request(restarted, "POST", "/dev/cloud/connect").Code)
}

func TestCommandTransportFailureIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		_ = conn.Close() // Command may have executed but response is lost.
	})
	b.bound = metadata
	w := request(b, "POST", "/v2/sandboxes/"+sandboxID+"/exec")
	require.Equal(t, 502, w.Code)
	require.Contains(t, w.Body.String(), "may have executed")
	require.Equal(t, int32(1), calls.Load())
}

func TestStreamingAndCancellation(t *testing.T) {
	canceled := make(chan struct{})
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"type":"log","data":"aGVsbG8="}`)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(canceled)
	})
	b.bound = metadata
	server := httptest.NewServer(b)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	r, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/v2/sandboxes/"+sandboxID+"/logs", nil)
	require.NoError(t, err)
	r.Host = "localhost:8288"
	r.Header.Set("Authorization", "Bearer "+b.Token)
	resp, err := http.DefaultClient.Do(r)
	require.NoError(t, err)
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"log","data":"aGVsbG8="}`, line)
	_ = resp.Body.Close()
	cancel()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("upstream was not canceled")
	}
}

func TestRawFilesAndUpstreamErrors(t *testing.T) {
	payload := []byte{0, 255, 3, 128, 0, 42}
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/tmp/a b.bin", r.URL.Query().Get("path"))
		if r.Method == "PUT" {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, payload, body)
			w.WriteHeader(429)
			_, _ = io.WriteString(w, `{"errors":[{"code":"rate_limited","message":"Wait"}]}`)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(payload)
	})
	b.bound = metadata
	for _, method := range []string{"GET", "PUT"} {
		r := httptest.NewRequest(method, "http://localhost:8288/v2/sandboxes/"+sandboxID+"/files?path=%2Ftmp%2Fa%20b.bin", strings.NewReader(string(payload)))
		r.Header.Set("Authorization", "Bearer "+b.Token)
		w := httptest.NewRecorder()
		b.ServeHTTP(w, r)
		if method == "GET" {
			require.Equal(t, payload, w.Body.Bytes())
		} else {
			require.Equal(t, 429, w.Code)
			require.Contains(t, w.Body.String(), "rate_limited")
		}
	}
}

func TestJournalFailureDoesNotFailSuccessfulCreate(t *testing.T) {
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprintf(w, `{"data":{"id":%q}}`, sandboxID)
	})
	b.bound = metadata
	b.statePath = t.TempDir() // Cannot rename a file over a directory.
	w := request(b, "POST", "/v2/sandboxes")
	require.Equal(t, 201, w.Code)
	require.Contains(t, request(b, "GET", "/dev/cloud/status").Body.String(), "Could not save")
	require.Equal(t, []string{sandboxID}, b.sandboxes[b.identity(metadata)])
}

func TestJournalUpdatesAcrossServers(t *testing.T) {
	first, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {})
	second, err := New(t.Context(), 8289)
	require.NoError(t, err)
	other := *metadata
	workspace := "other-environment"
	other.WorkspaceID = &workspace
	for _, test := range []struct {
		name     string
		bridge   *Bridge
		metadata *cliauth.Metadata
		id       string
		remove   bool
		want     []string
	}{
		{"first create", first, metadata, "first", false, []string{"first"}},
		{"stale server create", second, metadata, "second", false, []string{"first", "second"}},
		{"other environment", first, &other, "other", false, []string{"other"}},
		{"delete from stale server", first, metadata, "second", true, []string{"first"}},
		{"do not resurrect deleted ID", second, metadata, "third", false, []string{"first", "third"}},
		{"duplicate create", first, metadata, "third", false, []string{"first", "third"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.bridge.remember(test.metadata, test.id, test.remove)
			require.Empty(t, test.bridge.warning)
			restarted, err := New(t.Context(), 8288)
			require.NoError(t, err)
			require.ElementsMatch(t, test.want, restarted.sandboxes[first.identity(test.metadata)])
		})
	}
	journal, err := first.loadJournal()
	require.NoError(t, err)
	require.Equal(t, []string{"other"}, journal[first.identity(&other)])
}

func TestConcurrentJournalUpdates(t *testing.T) {
	first, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {})
	second, err := New(t.Context(), 8289)
	require.NoError(t, err)
	var wg sync.WaitGroup
	for _, bridge := range []*Bridge{first, second} {
		wg.Go(func() {
			for i := range 10 {
				bridge.remember(metadata, fmt.Sprintf("%s-%d", bridge.port, i), false)
			}
		})
	}
	wg.Wait()
	require.Empty(t, first.warning)
	require.Empty(t, second.warning)
	restarted, err := New(t.Context(), 8288)
	require.NoError(t, err)
	require.Len(t, restarted.sandboxes[first.identity(metadata)], 20)
}

func TestRefreshUsesExistingAuthManager(t *testing.T) {
	var refreshes atomic.Int32
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			refreshes.Add(1)
			require.NoError(t, r.ParseForm())
			require.Equal(t, "refresh_token", r.Form.Get("grant_type"))
			require.Equal(t, "cloud-refresh", r.Form.Get("refresh_token"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"refreshed-access","token_type":"Bearer","refresh_token":"rotated-refresh","expires_in":3600}`)
			return
		}
		require.Equal(t, "Bearer refreshed-access", r.Header.Get("Authorization"))
		writeData(w, []any{})
	})
	require.NoError(t, b.auth.Store().Save(*metadata, cliauth.Credential{AccessToken: "expired", RefreshToken: "cloud-refresh", Expiry: time.Now().Add(-time.Minute)}, true))
	b.bound = metadata
	require.Equal(t, 200, request(b, "GET", "/v2/sandboxes").Code)
	require.Equal(t, int32(1), refreshes.Load())
	_, credential, err := b.auth.Store().Load()
	require.NoError(t, err)
	require.Equal(t, "rotated-refresh", credential.RefreshToken)
	data, err := os.ReadFile(os.Getenv("INNGEST_CONFIG_DIR") + "/auth.json")
	require.NoError(t, err)
	var stored map[string]any
	require.NoError(t, json.Unmarshal(data, &stored))
	require.NotContains(t, string(data), "refreshed-access")
}

func TestDeviceLoginUsesSandboxScopesAndKeepsCredentialsServerSide(t *testing.T) {
	keyring.MockInit()
	var issuer string
	b, _ := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/device/code":
			require.NoError(t, r.ParseForm())
			require.Equal(t, "sandboxes:read:* sandboxes:write:*", r.Form.Get("scope"))
			require.Equal(t, issuer+"/v2", r.Form.Get("resource"))
			_, _ = io.WriteString(w, `{"device_code":"private-device-code","user_code":"ABCD-EFGH","verification_uri":"https://app.example/oauth/device","expires_in":600,"interval":1}`)
		case "/oauth/token":
			require.NoError(t, r.ParseForm())
			require.Equal(t, "private-device-code", r.Form.Get("device_code"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "new-cloud-access", "refresh_token": "new-cloud-refresh", "token_type": "Bearer", "expires_in": 3600,
				"resource": issuer + "/v2", "session_id": "new-session", "account_id": "account-a", "account_name": "Test organization",
				"workspace_id": "development", "workspace_name": "Development", "session_expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
			})
		case "/oauth/revoke":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	})
	issuer = b.issuer
	w := request(b, "POST", "/dev/cloud/login")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "ABCD-EFGH")
	require.NotContains(t, w.Body.String(), "private-device-code")
	require.Eventually(t, func() bool {
		b.mu.Lock()
		defer b.mu.Unlock()
		return b.login == nil
	}, 5*time.Second, 10*time.Millisecond)
	require.Empty(t, b.loginError)
	require.Nil(t, b.bound) // Consent does not silently enable the proxy.
	metadata, credential, err := b.auth.Store().Load()
	require.NoError(t, err)
	require.Equal(t, "new-cloud-access", credential.AccessToken)
	require.Equal(t, "keyring", metadata.Storage)
	require.Equal(t, 200, request(b, "POST", "/dev/cloud/connect").Code)
	w = request(b, "GET", "/dev/cloud/status")
	require.NotContains(t, w.Body.String(), "new-cloud-access")
	require.NotContains(t, w.Body.String(), "new-cloud-refresh")
}
