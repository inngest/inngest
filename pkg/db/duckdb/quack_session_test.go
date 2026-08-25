package duckdb

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// startQuackServerForTest resolves a free local port and bootstraps a quack
// listener on it via spawnQuackServer, skipping the test if the duckdb
// binary or the quack extension isn't available (see requireQuackExtension).
func startQuackServerForTest(t *testing.T, token string) (listenURL string, cleanup func()) {
	t.Helper()
	binPath := requireDuckDBBinary(t)
	requireQuackExtension(t, binPath)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return spawnQuackServer(t, binPath, addr, token)
}

func TestQuackSessionHandshakeAndExec(t *testing.T) {
	listenURL, cleanup := startQuackServerForTest(t, "test-token")
	defer cleanup()

	sess, err := newQuackSession(context.Background(), listenURL, "test-token")
	require.NoError(t, err)

	_, err = sess.exec(context.Background(), "CREATE TABLE t (id INTEGER, name VARCHAR);")
	require.NoError(t, err)

	_, err = sess.exec(context.Background(), "INSERT INTO t VALUES (1, 'a'), (2, 'b');")
	require.NoError(t, err)

	rows, err := sess.exec(context.Background(), "SELECT COUNT(*) AS n FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(2), rows[0]["n"])
}

func TestQuackSessionHandshakeWrongTokenErrors(t *testing.T) {
	listenURL, cleanup := startQuackServerForTest(t, "right-token")
	defer cleanup()

	_, err := newQuackSession(context.Background(), listenURL, "wrong-token")
	require.Error(t, err)
}

func TestQuackSessionStatementErrorMapsToErrStatementFailed(t *testing.T) {
	listenURL, cleanup := startQuackServerForTest(t, "test-token")
	defer cleanup()

	sess, err := newQuackSession(context.Background(), listenURL, "test-token")
	require.NoError(t, err)

	_, err = sess.exec(context.Background(), "INSERT INTO nonexistent_table VALUES (1);")
	require.Error(t, err)
	require.True(t, errors.Is(err, errStatementFailed), "got: %v", err)
}

func TestQuackSessionExecAgainstUnreachableServerErrors(t *testing.T) {
	requireDuckDBBinary(t) // consistent skip behavior with the rest of this file

	sess := &quackSession{endpoint: "http://127.0.0.1:1/quack", httpClient: newQuackHTTPClient()}
	_, err := sess.exec(context.Background(), "SELECT 1;")
	require.Error(t, err)
	require.False(t, errors.Is(err, errStatementFailed), "transport failure must not be classified as a statement failure: %v", err)
}
