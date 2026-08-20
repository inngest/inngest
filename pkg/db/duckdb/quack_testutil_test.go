package duckdb

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// requireQuackExtension skips the test if `INSTALL quack; LOAD quack;`
// doesn't succeed against binPath — no network access, or a duckdb build
// that predates the quack extension. Mirrors requireDuckDBBinary's
// skip-don't-fail convention in process_test.go for the same reason: CI
// environments without network access should degrade gracefully rather than
// fail.
func requireQuackExtension(t *testing.T, binPath string) {
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

// spawnQuackServer starts a bare duckdb subprocess and bootstraps a quack
// listener on addr (host:port, no "quack:" prefix), returning the
// server-reported listen URL and a cleanup func. Independent of process.go's
// process type — used by tests that only need a live quack endpoint, not the
// full supervised-subprocess lifecycle.
func spawnQuackServer(t *testing.T, binPath, addr, token string) (listenURL string, cleanup func()) {
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
