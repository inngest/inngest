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
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	cliauth "github.com/inngest/inngest/cmd/internal/auth"
	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/stretchr/testify/require"
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
	r.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	b.ServeHTTP(w, r)
	return w
}

func TestLocalAuthorizationAndRouteBoundary(t *testing.T) {
	var calls atomic.Int32
	b, _ := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeData(w, map[string]any{})
	})
	for _, test := range []struct {
		name, method, path, host, origin, remote string
		status                                   int
	}{
		{"local SDK needs no token", "GET", "/dev/cloud/status", "localhost:8288", "", "127.0.0.1:1234", 200},
		{"local browser", "GET", "/dev/cloud/status", "localhost:8288", "http://localhost:8288", "[::1]:1234", 200},
		{"cross origin", "POST", "/v2/sandboxes", "localhost:8288", "https://evil.example", "127.0.0.1:1234", 403},
		{"opaque origin", "GET", "/dev/cloud/status", "localhost:8288", "null", "127.0.0.1:1234", 403},
		{"DNS rebinding", "GET", "/dev/cloud/status", "evil.example:8288", "", "127.0.0.1:1234", 403},
		{"other port", "GET", "/dev/cloud/status", "localhost:9999", "", "127.0.0.1:1234", 403},
		{"remote caller with local Host", "POST", "/v2/sandboxes", "localhost:8288", "", "192.0.2.1:1234", 403},
		{"missing remote address", "POST", "/v2/sandboxes", "localhost:8288", "", "", 403},
		{"unrelated API", "POST", "/v2/events", "localhost:8288", "", "127.0.0.1:1234", 404},
		{"credential API", "GET", "/v2/signing-keys", "localhost:8288", "", "127.0.0.1:1234", 404},
		{"unknown sandbox method", "PATCH", "/v2/sandboxes/" + sandboxID, "localhost:8288", "", "127.0.0.1:1234", 405},
		{"escaped path", "GET", "/v2/sandboxes/a%2Fb", "localhost:8288", "", "127.0.0.1:1234", 400},
		{"dot path", "GET", "/v2/sandboxes/../account", "localhost:8288", "", "127.0.0.1:1234", 400},
		{"removed browser login", "POST", "/dev/cloud/login", "localhost:8288", "", "127.0.0.1:1234", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest(test.method, "http://localhost:8288"+test.path, nil)
			r.Host = test.host
			r.RemoteAddr = test.remote
			r.Header.Set("Origin", test.origin)
			r.Header.Set("X-Forwarded-For", "127.0.0.1")
			w := httptest.NewRecorder()
			b.ServeHTTP(w, r)
			require.Equal(t, test.status, w.Code, w.Body.String())
		})
	}
	require.Zero(t, calls.Load())
	w := request(b, "GET", "/dev/cloud/status")
	require.Equal(t, 200, w.Code)
	for _, secret := range []string{"cloud-access", "cloud-refresh"} {
		require.NotContains(t, w.Body.String(), secret)
	}
}

func TestAllSandboxRoutesAndHeaderIsolation(t *testing.T) {
	var path, method string
	b, _ := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, path, r.URL.Path)
		require.Equal(t, method, r.Method)
		require.Equal(t, "Bearer cloud-access", r.Header.Get("Authorization"))
		for _, header := range []string{"Cookie", "X-Inngest-Env", "Baggage", "Forwarded", "X-Forwarded-Host"} {
			require.Empty(t, r.Header.Get(header), header)
		}
		writeData(w, map[string]any{})
	})
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
			r.RemoteAddr = "127.0.0.1:12345"
			r.Header.Set("Origin", "http://localhost:8288")
			r.Header.Set("Authorization", "Bearer caller-key")
			for _, header := range []string{"Cookie", "X-Inngest-Env", "Baggage", "Forwarded", "X-Forwarded-Host"} {
				r.Header.Set(header, "do-not-forward")
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			require.Equal(t, 200, w.Code, "%s %s: %s", method, prefix+p, w.Body.String())
		}
	}
}

func TestSavedLoginAndRecovery(t *testing.T) {
	var mutations atomic.Int32
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		mutations.Add(1)
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(201)
		fmt.Fprintf(w, `{"data":{"id":%q,"name":"local-test","status":"RUNNING"}}`, sandboxID)
	})
	require.NoError(t, b.auth.Store().Delete(nil))
	missingLogin := request(b, "POST", "/v2/sandboxes")
	require.Equal(t, 401, missingLogin.Code)
	require.Contains(t, missingLogin.Body.String(), "inngest login")
	require.Zero(t, mutations.Load())

	// A login made after startup is picked up without connecting or restarting.
	saveSession(t, b, metadata)
	create := request(b, "POST", "/v2/sandboxes")
	require.Equal(t, 201, create.Code, create.Body.String())
	require.JSONEq(t, fmt.Sprintf(`{"data":{"id":%q,"name":"local-test","status":"RUNNING"}}`, sandboxID), create.Body.String())

	restarted, err := New(t.Context(), 8288)
	require.NoError(t, err)
	require.Contains(t, request(restarted, "GET", "/dev/cloud/status").Body.String(), sandboxID)

	// Switching the CLI login switches the environment and its project journal.
	other := *metadata
	workspace := "other-env"
	other.WorkspaceID = &workspace
	saveSession(t, restarted, &other)
	otherStatus := request(restarted, "GET", "/dev/cloud/status")
	require.Equal(t, 200, otherStatus.Code)
	require.Contains(t, otherStatus.Body.String(), workspace)
	require.NotContains(t, otherStatus.Body.String(), sandboxID)
	require.Equal(t, 201, request(restarted, "POST", "/v2/sandboxes").Code)
	require.Equal(t, []string{sandboxID}, restarted.sandboxes[restarted.identity(&other)])

	saveSession(t, restarted, metadata)
	require.Equal(t, 204, request(restarted, "DELETE", "/v2/sandboxes/"+sandboxID).Code)
	require.NotContains(t, request(restarted, "GET", "/dev/cloud/status").Body.String(), sandboxID)
	require.Equal(t, int32(3), mutations.Load())

	require.NoError(t, restarted.auth.Store().Delete(nil))
	require.Equal(t, 401, request(restarted, "GET", "/v2/sandboxes").Code)
	require.Equal(t, 401, request(restarted, "GET", "/dev/cloud/status").Code)
	require.Equal(t, int32(3), mutations.Load())
}

func TestAcceptedDeleteForgetsSandbox(t *testing.T) {
	b, _ := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		// Cloud answers 202 with the TERMINATING resource while termination is in progress.
		if r.Method == "DELETE" {
			w.WriteHeader(202)
			fmt.Fprintf(w, `{"data":{"id":%q,"status":"TERMINATING"}}`, sandboxID)
			return
		}
		w.WriteHeader(201)
		fmt.Fprintf(w, `{"data":{"id":%q,"status":"RUNNING"}}`, sandboxID)
	})
	require.Equal(t, 201, request(b, "POST", "/v2/sandboxes").Code)
	require.Contains(t, request(b, "GET", "/dev/cloud/status").Body.String(), sandboxID)

	require.Equal(t, 202, request(b, "DELETE", "/v2/sandboxes/"+sandboxID).Code)
	require.NotContains(t, request(b, "GET", "/dev/cloud/status").Body.String(), sandboxID)
}

func TestInvalidSavedLogin(t *testing.T) {
	var calls atomic.Int32
	b, metadata := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
	})
	for _, test := range []struct {
		name   string
		change func(*cliauth.Metadata)
		status int
	}{
		{"all environments", func(m *cliauth.Metadata) { m.WorkspaceID = nil }, 400},
		{"empty environment", func(m *cliauth.Metadata) { m.WorkspaceID = new(string) }, 400},
		{"expired session", func(m *cliauth.Metadata) { m.SessionExpiresAt = time.Now().Add(-time.Minute) }, 401},
		{"different API", func(m *cliauth.Metadata) { m.Resource = "https://other.example/v2" }, 401},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := *metadata
			test.change(&invalid)
			saveSession(t, b, &invalid)
			for _, path := range []string{"/dev/cloud/status", "/v2/sandboxes"} {
				w := request(b, "GET", path)
				require.Equal(t, test.status, w.Code, w.Body.String())
				require.Contains(t, w.Body.String(), "inngest login")
			}
		})
	}
	require.Zero(t, calls.Load())
}

func TestCloudAuthorizationErrors(t *testing.T) {
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls atomic.Int32
			b, _ := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				writeError(w, code, "auth_error", "Cloud access denied")
			})
			w := request(b, "POST", "/v2/sandboxes")
			require.Equal(t, code, w.Code)
			if code == http.StatusUnauthorized {
				require.Contains(t, w.Body.String(), "inngest login")
			} else {
				require.Contains(t, w.Body.String(), "Cloud access denied")
			}
			require.Equal(t, int32(1), calls.Load())
		})
	}
}

func TestCommandTransportFailureIsNotRetried(t *testing.T) {
	var calls atomic.Int32
	b, _ := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		conn, _, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		_ = conn.Close() // Command may have executed but response is lost.
	})
	w := request(b, "POST", "/v2/sandboxes/"+sandboxID+"/exec")
	require.Equal(t, 502, w.Code)
	require.Contains(t, w.Body.String(), "may have executed")
	require.Equal(t, int32(1), calls.Load())
}

func TestStreamingAndCancellation(t *testing.T) {
	canceled := make(chan struct{})
	b, _ := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"type":"log","data":"aGVsbG8="}`)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(canceled)
	})
	server := httptest.NewServer(b)
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	r, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/v2/sandboxes/"+sandboxID+"/logs", nil)
	require.NoError(t, err)
	r.Host = "localhost:8288"
	r.RemoteAddr = "127.0.0.1:12345"
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
	b, _ := testBridge(t, func(w http.ResponseWriter, r *http.Request) {
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
	for _, method := range []string{"GET", "PUT"} {
		r := httptest.NewRequest(method, "http://localhost:8288/v2/sandboxes/"+sandboxID+"/files?path=%2Ftmp%2Fa%20b.bin", strings.NewReader(string(payload)))
		r.RemoteAddr = "127.0.0.1:12345"
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
	b.statePath = t.TempDir() // Cannot rename a file over a directory.
	w := request(b, "POST", "/v2/sandboxes")
	require.Equal(t, 201, w.Code)
	require.Contains(t, request(b, "GET", "/dev/cloud/status").Body.String(), "Could not save")
	require.Equal(t, []string{sandboxID}, b.sandboxes[b.identity(metadata)])
}

func TestInvalidJournalDoesNotPreventStartup(t *testing.T) {
	b, _ := testBridge(t, func(w http.ResponseWriter, r *http.Request) {})
	require.NoError(t, os.MkdirAll(filepath.Dir(b.statePath), 0o700))
	require.NoError(t, os.WriteFile(b.statePath, []byte("invalid JSON"), 0o600))
	restarted, err := New(t.Context(), 8288)
	require.NoError(t, err)
	w := request(restarted, "GET", "/dev/cloud/status")
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "Could not load saved sandbox IDs")
	data, err := os.ReadFile(b.statePath)
	require.NoError(t, err)
	require.Equal(t, "invalid JSON", string(data))
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
