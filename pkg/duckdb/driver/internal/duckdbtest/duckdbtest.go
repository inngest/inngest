// Package duckdbtest holds test helpers shared by the duckdb driver and its
// internal transport packages. They skip, rather than fail, when the real
// duckdb binary or quack extension isn't available, so CI environments
// without them degrade gracefully (scripts/duckdb-smoke.sh is what fails
// when a required test skips).
package duckdbtest

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// RequireBinary skips the calling test if no "duckdb" binary is on PATH,
// returning its path otherwise.
func RequireBinary(t testing.TB) string {
	t.Helper()
	path, err := exec.LookPath("duckdb")
	if err != nil {
		t.Skip("duckdb binary not found on PATH; skipping subprocess test")
	}
	return path
}

// FreeLocalAddr resolves an ephemeral local port, closing the listener
// immediately. Only for callers that must know a port in advance (e.g. a
// DuckLake quack catalog address another process attaches to); anything
// that can read the bound port back should bind port 0 instead.
func FreeLocalAddr(t testing.TB) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return fmt.Sprintf("127.0.0.1:%d", port)
}

// RequireQuackExtension skips the test if `INSTALL quack; LOAD quack;`
// doesn't succeed against binPath — no network access, or a duckdb build
// that predates the quack extension.
func RequireQuackExtension(t testing.TB, binPath string) {
	t.Helper()
	cmd := exec.Command(binPath, ":memory:", "-jsonlines")
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	cmd.Stdout = outW
	cmd.Stderr = outW
	require.NoError(t, cmd.Start())
	require.NoError(t, outW.Close())

	var lines []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(outR)
		for sc.Scan() {
			lines = append(lines, sc.Text())
		}
	}()

	fmt.Fprintln(stdin, "INSTALL quack; LOAD quack; SELECT 'ok' AS ok;")
	_ = stdin.Close()
	waitErr := cmd.Wait()
	<-done

	ok := false
	for _, l := range lines {
		if l == `{"ok":"ok"}` {
			ok = true
		}
	}
	if waitErr != nil || !ok {
		t.Skipf("quack extension unavailable (no network access or unsupported duckdb version); output: %v, err: %v", lines, waitErr)
	}
}

// SpawnQuackServer starts a bare duckdb subprocess and bootstraps a quack
// listener on addr (host:port, no "quack:" prefix), returning the
// server-reported listen URL and a cleanup func. Independent of process.go's
// process type — used by tests that only need a live quack endpoint, not the
// full supervised-subprocess lifecycle.
func SpawnQuackServer(t testing.TB, binPath, addr, token string) (listenURL string, cleanup func()) {
	t.Helper()

	cmd := exec.Command(binPath, ":memory:")
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	cmd.Stdout = outW
	cmd.Stderr = outW
	require.NoError(t, cmd.Start())
	require.NoError(t, outW.Close())

	var lines []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(outR)
		for sc.Scan() {
			lines = append(lines, sc.Text())
		}
	}()

	kill := func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		<-done
	}

	fmt.Fprintf(stdin, "INSTALL quack;\nLOAD quack;\nCALL quack_serve('quack:%s', token = '%s');\n", addr, token)
	time.Sleep(800 * time.Millisecond)

	if cmd.ProcessState != nil {
		kill()
		t.Skipf("duckdb subprocess exited early; quack extension likely unavailable: %v", lines)
	}

	return "http://" + addr, kill
}
