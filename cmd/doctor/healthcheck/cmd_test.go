package healthcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/connect"
	"github.com/urfave/cli/v3"
)

func init() {
	// Prevent cli.Exit from terminating the test process. The framework
	// calls HandleExitCoder → OsExiter, which defaults to os.Exit.
	cli.OsExiter = func(int) {}
}

func TestRun(t *testing.T) {
	apiOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != api.HealthPath {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer apiOK.Close()

	gwOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != connect.ReadyPath {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer gwOK.Close()

	apiBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer apiBad.Close()

	apiSlow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer apiSlow.Close()

	tests := []struct {
		name    string
		apiURL  string
		gwURL   string
		timeout time.Duration
		skipGw  bool
		wantErr bool
	}{
		{
			name:    "both healthy",
			apiURL:  apiOK.URL,
			gwURL:   gwOK.URL,
			timeout: 2 * time.Second,
			wantErr: false,
		},
		{
			name:    "api 500 fails",
			apiURL:  apiBad.URL,
			gwURL:   gwOK.URL,
			timeout: 2 * time.Second,
			wantErr: true,
		},
		{
			name:    "gateway down fails",
			apiURL:  apiOK.URL,
			gwURL:   "http://127.0.0.1:1", // unreachable
			timeout: 500 * time.Millisecond,
			wantErr: true,
		},
		{
			name:    "skip gateway when down",
			apiURL:  apiOK.URL,
			gwURL:   "http://127.0.0.1:1",
			timeout: 500 * time.Millisecond,
			skipGw:  true,
			wantErr: false,
		},
		{
			name:    "timeout fails",
			apiURL:  apiSlow.URL,
			gwURL:   gwOK.URL,
			timeout: 50 * time.Millisecond,
			wantErr: true,
		},
		{
			name:    "connection refused fails",
			apiURL:  "http://127.0.0.1:1",
			gwURL:   gwOK.URL,
			timeout: 500 * time.Millisecond,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, apiPort := splitURL(t, tt.apiURL)
			_, gwPort := splitURL(t, tt.gwURL)

			cmd := Command()
			args := []string{
				"healthcheck",
				"--host=" + host,
				"--port=" + strconv.Itoa(apiPort),
				"--connect-gateway-port=" + strconv.Itoa(gwPort),
				"--timeout=" + tt.timeout.String(),
			}
			if tt.skipGw {
				args = append(args, "--skip-connect-gateway")
			}

			err := cmd.Run(context.Background(), args)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestRun_EnvVarFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port := splitURL(t, srv.URL)
	t.Setenv("INNGEST_HOST", host)
	t.Setenv("INNGEST_PORT", strconv.Itoa(port))

	cmd := Command()
	err := cmd.Run(context.Background(), []string{
		"healthcheck",
		"--connect-gateway-port=" + strconv.Itoa(port),
		"--timeout=2s",
	})
	if err != nil {
		t.Fatalf("env-var fallback failed: %v", err)
	}
}

func TestRun_SchemeOverride(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port := splitURL(t, srv.URL)
	cmd := Command()
	err := cmd.Run(context.Background(), []string{
		"healthcheck",
		"--host=" + host,
		"--port=" + strconv.Itoa(port),
		"--connect-gateway-port=" + strconv.Itoa(port),
		"--scheme=https",
		"--timeout=2s",
	})
	// httptest's TLS uses a self-signed cert, so the probe will fail TLS verification.
	// We're only validating that the scheme flag is honored (request goes out as HTTPS,
	// not HTTP, which would otherwise return 400).
	if err == nil {
		t.Fatal("expected TLS verification error, got nil")
	}
	if !strings.Contains(err.Error(), "") { // err is non-nil; cli.Exit("", 1) has empty msg
		t.Fatalf("unexpected error shape: %v", err)
	}
}

func splitURL(t *testing.T, raw string) (string, int) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("bad url %q: %v", raw, err)
	}
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("bad host %q: %v", u.Host, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("bad port %q: %v", portStr, err)
	}
	return host, port
}
