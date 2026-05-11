package doctor

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/connect"
	"github.com/urfave/cli/v3"
)

func init() {
	cli.OsExiter = func(int) {}
}

// TestParentPath_EnvVarFallback verifies that env-var Sources on a
// subcommand's flags resolve when the doctor parent runs all subchecks,
// not only when the subcommand is invoked directly. Regression test for
// runSubCheck driving PreParse/PostParse — without it, runAllChecks
// invoked sub.Action with unparsed flags and Sources never resolved, so
// healthcheck probed the static defaults (127.0.0.1:8288 / :8289)
// instead of the env-var-pointed target.
func TestParentPath_EnvVarFallback(t *testing.T) {
	var probed atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case api.HealthPath, connect.ReadyPath:
			probed.Add(1)
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	host, port := splitURL(t, srv.URL)
	portStr := strconv.Itoa(port)
	t.Setenv("INNGEST_HOST", host)
	t.Setenv("INNGEST_PORT", portStr)
	t.Setenv("INNGEST_CONNECT_GATEWAY_PORT", portStr)

	cmd := Command()
	if err := cmd.Run(context.Background(), []string{"doctor"}); err != nil {
		t.Fatalf("doctor run returned error: %v", err)
	}
	if got := probed.Load(); got != 2 {
		t.Fatalf("expected 2 probes against env-var target, got %d — env-var Sources not resolved through doctor parent", got)
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
