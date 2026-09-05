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

	_, _, err = sess.exec(context.Background(), "CREATE TABLE t (id INTEGER, name VARCHAR);")
	require.NoError(t, err)

	_, _, err = sess.exec(context.Background(), "INSERT INTO t VALUES (1, 'a'), (2, 'b');")
	require.NoError(t, err)

	_, rows, err := sess.exec(context.Background(), "SELECT COUNT(*) AS n FROM t;")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(2), rows[0]["n"])
}

// TestQuackSessionExecPreservesColumnOrder pins quack's column order to the
// query's own left-to-right order. "zebra" before "apple" sorts the wrong
// way alphabetically, so this fails if cols is ever derived by any means
// other than the PrepareResponse's own result_names field order.
func TestQuackSessionExecPreservesColumnOrder(t *testing.T) {
	listenURL, cleanup := startQuackServerForTest(t, "test-token")
	defer cleanup()

	sess, err := newQuackSession(context.Background(), listenURL, "test-token")
	require.NoError(t, err)

	cols, rows, err := sess.exec(context.Background(), "SELECT 1 AS zebra, 2 AS apple;")
	require.NoError(t, err)
	require.Equal(t, []string{"zebra", "apple"}, cols)
	require.Len(t, rows, 1)
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

	_, _, err = sess.exec(context.Background(), "INSERT INTO nonexistent_table VALUES (1);")
	require.Error(t, err)
	require.True(t, errors.Is(err, errStatementFailed), "got: %v", err)
}

// TestQuackSessionFetchesAllRowsWhenResultExceedsOneInlineResponse pins the
// fix for a real failure observed in practice: a result too large for one
// inline quack response (needsMoreFetch=true) used to come back as
// errStatementFailed because this client didn't implement the FetchRequest
// continuation protocol. exec now pages through FetchRequest/FetchResponse
// internally and returns every row. quack_fetch_batch_chunks is forced down
// to 1 DataChunk per response (default 12) so a modest 5,000-row query — one
// DuckDB vector is 2048 rows — still needs several Fetch round trips instead
// of requiring an unwieldy row count to reproduce.
func TestQuackSessionFetchesAllRowsWhenResultExceedsOneInlineResponse(t *testing.T) {
	listenURL, cleanup := startQuackServerForTest(t, "test-token")
	defer cleanup()

	sess, err := newQuackSession(context.Background(), listenURL, "test-token")
	require.NoError(t, err)

	_, _, err = sess.exec(context.Background(), "SET quack_fetch_batch_chunks = 1;")
	require.NoError(t, err)

	const wantRows = 5000
	cols, rows, err := sess.exec(context.Background(), fmt.Sprintf("SELECT range AS n FROM range(%d) ORDER BY n;", wantRows))
	require.NoError(t, err)
	require.Equal(t, []string{"n"}, cols)
	require.Len(t, rows, wantRows)
	require.Equal(t, int64(0), rows[0]["n"])
	require.Equal(t, int64(wantRows-1), rows[wantRows-1]["n"])
}

// TestQuackSessionFetchAfterResultClosedMapsToErrStatementFailed exercises
// the FETCH_REQUEST error path directly: fetching against a result_uuid the
// server no longer recognizes (a fresh PREPARE on the same connection
// discards the previous one — see quack_server.cpp's PREPARE_REQUEST
// handler) is a rejected request, not a dead subprocess, exactly like
// TestQuackSessionStatementErrorMapsToErrStatementFailed's PREPARE_REQUEST
// case.
func TestQuackSessionFetchAfterResultClosedMapsToErrStatementFailed(t *testing.T) {
	listenURL, cleanup := startQuackServerForTest(t, "test-token")
	defer cleanup()

	sess, err := newQuackSession(context.Background(), listenURL, "test-token")
	require.NoError(t, err)

	_, _, err = sess.exec(context.Background(), "SET quack_fetch_batch_chunks = 1;")
	require.NoError(t, err)

	staleUUID := quackHugeint{}
	hdr, r, err := sess.send(context.Background(), encodeQuackFetchRequest(sess.connectionID, staleUUID))
	require.NoError(t, err)
	require.Equal(t, byte(quackMsgErrorResponse), hdr.Type)

	err = decodeQuackStatementError(r)
	require.True(t, errors.Is(err, errStatementFailed), "got: %v", err)
}

// TestQuackSessionExecDecodesStructAndListOfStruct is a real-subprocess
// round-trip for STRUCT and LIST(STRUCT(...)) columns — the same queries used
// to reverse-engineer decodeQuackStructVector's/decodeQuackStructFieldTypes's
// wire shape in the first place (see their doc comments). Kept alongside the
// hand-constructed-bytes unit tests in quack_protocol_test.go so a future
// DuckDB point release that changes either transport's wire details is caught
// here even if the hand-built bytes still (now stale-ly) decode.
func TestQuackSessionExecDecodesStructAndListOfStruct(t *testing.T) {
	listenURL, cleanup := startQuackServerForTest(t, "test-token")
	defer cleanup()

	sess, err := newQuackSession(context.Background(), listenURL, "test-token")
	require.NoError(t, err)

	cols, rows, err := sess.exec(context.Background(), "SELECT {'a': 1, 'b': 'x'} AS s")
	require.NoError(t, err)
	require.Equal(t, []string{"s"}, cols)
	require.Len(t, rows, 1)
	require.Equal(t, map[string]any{"a": int64(1), "b": "x"}, rows[0]["s"])

	cols, rows, err = sess.exec(context.Background(), "SELECT {'a': 1, 'b': NULL} AS s")
	require.NoError(t, err)
	require.Equal(t, []string{"s"}, cols)
	require.Len(t, rows, 1)
	require.Equal(t, map[string]any{"a": int64(1), "b": nil}, rows[0]["s"])

	cols, rows, err = sess.exec(context.Background(), "SELECT [{'a': 1}, {'a': 2}] AS s")
	require.NoError(t, err)
	require.Equal(t, []string{"s"}, cols)
	require.Len(t, rows, 1)
	require.Equal(t, []any{
		map[string]any{"a": int64(1)},
		map[string]any{"a": int64(2)},
	}, rows[0]["s"])
}

func TestQuackSessionExecAgainstUnreachableServerErrors(t *testing.T) {
	requireDuckDBBinary(t) // consistent skip behavior with the rest of this file

	sess := &quackSession{endpoint: "http://127.0.0.1:1/quack", httpClient: newQuackHTTPClient()}
	_, _, err := sess.exec(context.Background(), "SELECT 1;")
	require.Error(t, err)
	require.False(t, errors.Is(err, errStatementFailed), "transport failure must not be classified as a statement failure: %v", err)
}
