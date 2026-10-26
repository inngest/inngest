package fakesandboxes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/stretchr/testify/require"
)

type received struct {
	signal string
	data   map[string]any
}

// stubAPI stands in for POST /v1/signals/scoped.
func stubAPI(t *testing.T, secret []byte, env uuid.UUID) (*httptest.Server, func() []received) {
	t.Helper()
	var mu sync.Mutex
	var got []received
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := apiv1auth.VerifySignalJWT(secret, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if err != nil || claims.Env != env {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body struct {
			Signal string         `json:"signal"`
			Data   map[string]any `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Signal != claims.SignalID {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		mu.Lock()
		got = append(got, received{body.Signal, body.Data})
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []received {
		mu.Lock()
		defer mu.Unlock()
		return append([]received(nil), got...)
	}
}

func call(t *testing.T, h http.Handler, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = "127.0.0.1:1234"
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	var out map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &out)
	return res.Code, out
}

func start(t *testing.T, h http.Handler, command string, signalID *string) (string, string) {
	t.Helper()
	_, sb := call(t, h, "POST", "/v2/sandboxes", map[string]any{"name": "demo"})
	sandboxID := sb["data"].(map[string]any)["id"].(string)
	body := map[string]any{"command": []string{"/bin/sh", "-c", command}}
	if signalID != nil {
		body["signalId"] = *signalID
	}
	code, p := call(t, h, "POST", "/v2/sandboxes/"+sandboxID+"/processes", body)
	require.Equal(t, http.StatusCreated, code)
	return sandboxID, p["data"].(map[string]any)["id"].(string)
}

func eventually(t *testing.T, f func() bool) {
	t.Helper()
	require.Eventually(t, f, 5*time.Second, 10*time.Millisecond)
}

func TestExitSignal(t *testing.T) {
	secret := []byte("s")
	env := uuid.New()
	api, got := stubAPI(t, secret, env)
	h := New(api.URL, secret, env)

	t.Run("failing command signals its exit code and tails", func(t *testing.T) {
		id := "build:fail"
		sb, pid := start(t, h, "echo compiling; err boom; exit 3", &id)

		code, res := call(t, h, "POST", "/v2/sandboxes/"+sb+"/processes/"+pid+"/wait?timeout=5s", nil)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, "EXITED", res["data"].(map[string]any)["state"])

		eventually(t, func() bool {
			for _, r := range got() {
				if r.signal == id {
					return r.data["exitCode"] == float64(3) && r.data["stdout"] == "compiling\n" && r.data["stderr"] == "boom\n" && r.data["truncated"] == false
				}
			}
			return false
		})
	})

	t.Run("killed process signals KILLED", func(t *testing.T) {
		id := "build:kill"
		start(t, h, "sleep 0.05; kill 9", &id)
		eventually(t, func() bool {
			for _, r := range got() {
				if r.signal == id {
					_, hasExit := r.data["exitCode"]
					return r.data["state"] == "KILLED" && r.data["terminationSignal"] == float64(9) && !hasExit
				}
			}
			return false
		})
	})

	t.Run("output beyond 8 KiB is cut to the tail and flagged", func(t *testing.T) {
		id := "build:flood"
		start(t, h, "flood 20; echo done", &id)
		eventually(t, func() bool {
			for _, r := range got() {
				if r.signal == id {
					out := r.data["stdout"].(string)
					return len(out) <= tailBytes && strings.HasSuffix(out, "done\n") && r.data["truncated"] == true
				}
			}
			return false
		})
	})

	t.Run("a token cannot signal another id", func(t *testing.T) {
		id := "build:probe"
		start(t, h, "probe; exit 0", &id)
		eventually(t, func() bool {
			_, stats := call(t, h, "GET", "/v2/sandboxes/_fake/requests", nil)
			probes, _ := stats["data"].(map[string]any)["probes"].([]any)
			return len(probes) == 1 && strings.HasSuffix(probes[0].(string), "HTTP 403")
		})
		for _, r := range got() {
			require.NotEqual(t, id+":other", r.signal)
		}
	})

	t.Run("no signalId sends nothing", func(t *testing.T) {
		before := len(got())
		sb, pid := start(t, h, "exit 0", nil)
		code, _ := call(t, h, "POST", "/v2/sandboxes/"+sb+"/processes/"+pid+"/wait?timeout=5s", nil)
		require.Equal(t, http.StatusOK, code)
		time.Sleep(50 * time.Millisecond)
		require.Len(t, got(), before)
	})
}

func TestWaitTimesOutWhileRunning(t *testing.T) {
	h := New("http://127.0.0.1:1", []byte("s"), uuid.New())
	sb, pid := start(t, h, "sleep 5", nil)
	code, res := call(t, h, "POST", "/v2/sandboxes/"+sb+"/processes/"+pid+"/wait?timeout=50ms", nil)
	require.Equal(t, http.StatusGatewayTimeout, code)
	require.Equal(t, "sandbox_process_wait_timed_out", res["errors"].([]any)[0].(map[string]any)["code"])
}

func TestRejectsNonLoopback(t *testing.T) {
	h := New("http://127.0.0.1:1", []byte("s"), uuid.New())
	req := httptest.NewRequest("POST", "/v2/sandboxes", strings.NewReader(`{"name":"x"}`))
	req.RemoteAddr = "203.0.113.5:1234"
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	require.Equal(t, http.StatusForbidden, res.Code)
}

func TestCountsRequestsByRoute(t *testing.T) {
	h := New("http://127.0.0.1:1", []byte("s"), uuid.New())
	start(t, h, "exit 0", nil)
	_, stats := call(t, h, "GET", "/v2/sandboxes/_fake/requests", nil)
	data := stats["data"].(map[string]any)
	require.Equal(t, float64(2), data["total"])
	require.Equal(t, float64(1), data["byRoute"].(map[string]any)["POST /v2/sandboxes/{id}/processes"])
}
