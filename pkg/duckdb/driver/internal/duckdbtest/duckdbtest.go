// Package duckdbtest holds test helpers shared by the duckdb driver tests.
package duckdbtest

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// FreeLocalAddr resolves an ephemeral local port, closing the listener immediately.
func FreeLocalAddr(t testing.TB) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return fmt.Sprintf("127.0.0.1:%d", port)
}

// RequireQuackExtension is a no-op now that the DuckDB UDS worker natively
// provides concurrent sessions, transactions, appenders, and streams without
// requiring an external quack extension.
func RequireQuackExtension(t testing.TB, _ string) {
	t.Helper()
}
