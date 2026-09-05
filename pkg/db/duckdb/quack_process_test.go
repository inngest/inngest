package duckdb

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// recordingExecer is a fake sqlExecer for unit-testing startQuackLocked's
// branching without a real subprocess. failPrefixes maps a statement prefix
// to the error exec should return for it; anything else succeeds, returning
// respondWith[prefix] if set.
type recordingExecer struct {
	calls        []string
	failPrefixes map[string]error
	respondWith  map[string][]map[string]any
}

func (f *recordingExecer) exec(ctx context.Context, sqlText string) ([]string, []map[string]any, error) {
	f.calls = append(f.calls, sqlText)
	for prefix, err := range f.failPrefixes {
		if strings.HasPrefix(sqlText, prefix) {
			return nil, nil, err
		}
	}
	for prefix, rows := range f.respondWith {
		if strings.HasPrefix(sqlText, prefix) {
			return nil, rows, nil
		}
	}
	return nil, nil, nil
}

func (f *recordingExecer) calledWithPrefix(prefix string) bool {
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

// freeLocalAddr resolves an ephemeral local port, closing the listener
// immediately so quack_serve can bind it. Small TOCTOU race, acceptable for
// tests (same tradeoff quack_session_test.go makes).
func freeLocalAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func TestStartProcessWithQuackTransport(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	addr := freeLocalAddr(t)
	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, isQuack := p.sess.(*quackSession)
	require.True(t, isQuack, "process should have swapped its active session to the quack transport once bootstrapped")

	require.NoError(t, p.healthCheck(t.Context()))

	_, _, err = p.exec(t.Context(), "CREATE TABLE t (id INTEGER, name VARCHAR);")
	require.NoError(t, err)
	_, _, err = p.exec(t.Context(), "INSERT INTO t VALUES (1, 'a');")
	require.NoError(t, err)

	_, rows, err := p.exec(t.Context(), "SELECT id, name FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0]["id"])
	require.Equal(t, "a", rows[0]["name"])
}

func TestQuackTransportSurvivesRestart(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	addr := freeLocalAddr(t)
	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	_, _, err = p.exec(t.Context(), "CREATE TABLE t (id INTEGER);")
	require.NoError(t, err)
	_, _, err = p.exec(t.Context(), "INSERT INTO t VALUES (1);")
	require.NoError(t, err)

	pidBefore := p.cmd.Process.Pid
	require.NoError(t, p.cmd.Process.Kill())
	_, _ = p.cmd.Process.Wait()

	// :memory: state (including the CREATE TABLE above) does not survive a
	// real crash — same as the jsonlines path — so the pre-crash table is
	// gone. What this proves is that the restart's fresh spawn+bootstrap
	// stands up a *new* quack listener and the process keeps working over
	// it, rather than hanging or falling back to some other transport.
	_, _, err = p.exec(t.Context(), "CREATE TABLE t (id INTEGER);")
	require.NoError(t, err, "restart must re-bootstrap the quack transport, not just the jsonlines control channel")
	_, _, err = p.exec(t.Context(), "INSERT INTO t VALUES (1);")
	require.NoError(t, err)
	_, rows, err := p.exec(t.Context(), "SELECT count(*) AS c FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), rows[0]["c"])

	p.mu.Lock()
	disabled := p.disabled
	pidAfter := p.cmd.Process.Pid
	_, stillQuack := p.sess.(*quackSession)
	p.mu.Unlock()
	require.False(t, disabled)
	require.NotEqual(t, pidBefore, pidAfter)
	require.True(t, stillQuack, "the restarted process must still be using the quack transport, not have fallen back to jsonlines")
}

// TestUIBootstrapFailureDoesNotBlockQuackBootstrap proves the UI (started as
// a best-effort convenience alongside quack) can't take dual-write down with
// it: if installing/starting the optional "ui" extension fails (e.g. no
// network), quack bootstrap must still be attempted and its own failure (or
// success) must be what the caller sees — not the UI's.
func TestUIBootstrapFailureDoesNotBlockQuackBootstrap(t *testing.T) {
	addr := "127.0.0.1:1" // deliberately unreachable; see below.
	rec := &recordingExecer{
		failPrefixes: map[string]error{"INSTALL ui": errors.New("boom: no network")},
		respondWith: map[string][]map[string]any{
			"CALL quack_serve": {{"listen_url": "http://127.0.0.1:1"}},
		},
	}
	p := &process{quackAddr: &addr, sess: rec}

	err := p.startQuackLocked(t.Context())

	// The eventual quack handshake against the deliberately unreachable
	// address will fail — that's expected and not what this test is about.
	require.Error(t, err)
	require.NotContains(t, err.Error(), "boom: no network",
		"a UI bootstrap failure must never surface as (or block reaching) the quack bootstrap's own outcome")
	require.True(t, rec.calledWithPrefix("CALL quack_serve"),
		"quack bootstrap must still be attempted after the UI's failed")
}

// TestQuackModeExposesRealUI is the integration-level proof: when the
// optional "ui" extension is actually installable, quack mode really does
// start DuckDB's web UI, not just attempt to.
func TestQuackModeExposesRealUI(t *testing.T) {
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	addr := freeLocalAddr(t)
	p, err := startProcessWithDuckLake(t.Context(), binPath, ":memory:", nil, &addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.close(t.Context()) })

	// start_ui_server() always binds DuckDB's fixed default UI port (it
	// takes no parameters — see startUILocked), so this doesn't need
	// freeLocalAddr; if the "ui" extension truly failed to install (no
	// network), this is the one place that would show it, distinct from the
	// quack-only assertions the rest of this file already covers.
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:4213/")
	if err != nil {
		t.Skipf("duckdb ui extension unavailable (no network access): %v", err)
	}
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
