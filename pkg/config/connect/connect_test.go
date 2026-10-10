package connect

import "testing"

func TestListenAddr(t *testing.T) {
	tests := []struct {
		host string
		port int
		want string
	}{
		{"", 8289, ":8289"},
		{"127.0.0.1", 50052, "127.0.0.1:50052"},
		{"::1", 50053, "[::1]:50053"},
	}
	for _, tt := range tests {
		got := ConnectGRPCConfig{BindHost: tt.host}.ListenAddr(tt.port)
		if got != tt.want {
			t.Errorf("ListenAddr(%q, %d) = %q, want %q", tt.host, tt.port, got, tt.want)
		}
	}
}
