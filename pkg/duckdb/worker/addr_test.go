package worker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		in      string
		wantNet string
		wantAdr string
	}{
		{"/tmp/duckdb.sock", "unix", "/tmp/duckdb.sock"},
		{"unix:///tmp/duckdb.sock", "unix", "/tmp/duckdb.sock"},
		{`C:\Users\me\duckdb.sock`, "unix", `C:\Users\me\duckdb.sock`},
		{"tcp://127.0.0.1:9090", "tcp", "127.0.0.1:9090"},
		{"tcp://0.0.0.0:9090", "tcp", "0.0.0.0:9090"},
		{"127.0.0.1:9090", "tcp", "127.0.0.1:9090"},
		{":9090", "tcp", ":9090"},
		{"[::1]:9090", "tcp", "[::1]:9090"},
		{"duckdb-worker.sysop.svc:9090", "tcp", "duckdb-worker.sysop.svc:9090"},
	}
	for _, tt := range tests {
		netw, addr := ParseAddress(tt.in)
		require.Equal(t, tt.wantNet, netw, "input=%q", tt.in)
		require.Equal(t, tt.wantAdr, addr, "input=%q", tt.in)
	}
}
