package rueidisconn

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type readResult struct {
	data []byte
	err  error
}

type stubConn struct {
	reads   []readResult
	readIdx int
}

func (s *stubConn) Read(b []byte) (int, error) {
	if s.readIdx >= len(s.reads) {
		return 0, io.EOF
	}
	r := s.reads[s.readIdx]
	s.readIdx++
	n := copy(b, r.data)
	return n, r.err
}

func (s *stubConn) Write(b []byte) (int, error) {
	return len(b), nil
}

func (s *stubConn) Close() error                       { return nil }
func (s *stubConn) LocalAddr() net.Addr                { return stubAddr("local") }
func (s *stubConn) RemoteAddr() net.Addr               { return stubAddr("remote") }
func (s *stubConn) SetDeadline(t time.Time) error      { return nil }
func (s *stubConn) SetReadDeadline(t time.Time) error  { return nil }
func (s *stubConn) SetWriteDeadline(t time.Time) error { return nil }

type stubAddr string

func (s stubAddr) Network() string { return "tcp" }
func (s stubAddr) String() string  { return string(s) }

// fixedReadConn always returns the same data on every Read call, avoiding
// per-iteration allocations in benchmarks.
type fixedReadConn struct{ stubConn; data []byte }

func (f *fixedReadConn) Read(b []byte) (int, error) { return copy(b, f.data), nil }

func TestTTFBConn_ReadEmitsForLuaCommands(t *testing.T) {
	conn := &stubConn{
		reads: []readResult{{data: []byte("+OK\r\n")}},
	}

	var events []TTFBEvent
	c := &ttfbConn{
		Conn: conn,
		onTTFB: func(evt TTFBEvent) {
			events = append(events, evt)
		},
	}

	_, err := c.Write(respCommand("EVALSHA", "abcdef", "0"))
	require.NoError(t, err)

	time.Sleep(2 * time.Millisecond)

	_, err = c.Read(make([]byte, 16))
	require.NoError(t, err)

	require.Len(t, events, 1)
	require.Equal(t, "EVALSHA", events[0].Command)
	require.Equal(t, "abcdef", events[0].ScriptName)
	require.Greater(t, events[0].TTFB, time.Duration(0))
}

func TestTTFBConn_ReadSkipsNonLuaCommands(t *testing.T) {
	conn := &stubConn{
		reads: []readResult{{data: []byte("+OK\r\n")}},
	}

	var events []TTFBEvent
	c := &ttfbConn{
		Conn: conn,
		onTTFB: func(evt TTFBEvent) {
			events = append(events, evt)
		},
	}

	_, err := c.Write(respCommand("GET", "key"))
	require.NoError(t, err)

	_, err = c.Read(make([]byte, 16))
	require.NoError(t, err)

	require.Empty(t, events)
}

func TestTTFBConn_TracksConnectionReuseAndFirstReadOnly(t *testing.T) {
	conn := &stubConn{
		reads: []readResult{
			{data: []byte("+OK\r\n")},
			{data: []byte("+QUEUED\r\n")},
			{data: []byte("+DONE\r\n")},
		},
	}

	var events []TTFBEvent
	c := &ttfbConn{
		Conn: conn,
		onTTFB: func(evt TTFBEvent) {
			events = append(events, evt)
		},
	}

	_, err := c.Write(respCommand("EVALSHA", "abcdef", "0"))
	require.NoError(t, err)

	_, err = c.Read(make([]byte, 16))
	require.NoError(t, err)

	// Second read without a new write should not emit again.
	_, err = c.Read(make([]byte, 16))
	require.NoError(t, err)
	require.Len(t, events, 1)

	_, err = c.Write(respCommand("EVAL_RO", "return 1", "0"))
	require.NoError(t, err)

	_, err = c.Read(make([]byte, 16))
	require.NoError(t, err)

	require.Len(t, events, 2)
	require.Equal(t, "EVALSHA", events[0].Command)
	require.Equal(t, "abcdef", events[0].ScriptName)
	require.Equal(t, "EVAL_RO", events[1].Command)
	require.Empty(t, events[1].ScriptName)
}

func TestTTFBConn_ReadEmitsSHAForEvalshaRo(t *testing.T) {
	conn := &stubConn{
		reads: []readResult{{data: []byte("+OK\r\n")}},
	}

	var events []TTFBEvent
	c := &ttfbConn{
		Conn: conn,
		onTTFB: func(evt TTFBEvent) {
			events = append(events, evt)
		},
	}

	_, err := c.Write(respCommand("EVALSHA_RO", "1234abcd", "0"))
	require.NoError(t, err)

	_, err = c.Read(make([]byte, 16))
	require.NoError(t, err)

	require.Len(t, events, 1)
	require.Equal(t, "EVALSHA_RO", events[0].Command)
	require.Equal(t, "1234abcd", events[0].ScriptName)
}

func TestTTFBConn_FirstWriteInBatchMustBeLua(t *testing.T) {
	conn := &stubConn{
		reads: []readResult{{data: []byte("+OK\r\n")}},
	}

	var events []TTFBEvent
	c := &ttfbConn{
		Conn: conn,
		onTTFB: func(evt TTFBEvent) {
			events = append(events, evt)
		},
	}

	// First write in the batch is non-Lua.
	_, err := c.Write(respCommand("GET", "key"))
	require.NoError(t, err)

	// Later Lua write in the same write/read batch should not start tracking.
	_, err = c.Write(respCommand("EVALSHA", "abcdef", "0"))
	require.NoError(t, err)

	_, err = c.Read(make([]byte, 16))
	require.NoError(t, err)

	require.Empty(t, events)
}

func respCommand(cmd string, args ...string) []byte {
	parts := append([]string{cmd}, args...)
	s := fmt.Sprintf("*%d\r\n", len(parts))
	for _, part := range parts {
		s += fmt.Sprintf("$%d\r\n%s\r\n", len(part), part)
	}
	return []byte(s)
}

func BenchmarkWriteNonLua(b *testing.B) {
	conn := &stubConn{reads: []readResult{{data: []byte("+OK\r\n")}}}
	c := &ttfbConn{Conn: conn, onTTFB: func(TTFBEvent) {}}
	payload := respCommand("GET", "key")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.wroteAt.Store(0) // reset between iterations
		c.Write(payload)
	}
}

func BenchmarkWriteLua(b *testing.B) {
	conn := &stubConn{reads: []readResult{{data: []byte("+OK\r\n")}}}
	c := &ttfbConn{Conn: conn, onTTFB: func(TTFBEvent) {}}
	payload := respCommand("EVALSHA", "abcdef0123456789abcdef0123456789abcdef01", "0")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.wroteAt.Store(0) // reset between iterations
		c.Write(payload)
	}
}

func BenchmarkReadNoEvent(b *testing.B) {
	conn := &fixedReadConn{data: []byte("+OK\r\n")}
	c := &ttfbConn{Conn: conn, onTTFB: func(TTFBEvent) {}}
	buf := make([]byte, 64)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.wroteAt.Store(-1) // non-Lua write pending
		c.Read(buf)
	}
}

func BenchmarkReadWithEvent(b *testing.B) {
	conn := &fixedReadConn{data: []byte("+OK\r\n")}
	c := &ttfbConn{Conn: conn, onTTFB: func(TTFBEvent) {}}
	c.command = "EVALSHA"
	c.scriptName = "abcdef"
	buf := make([]byte, 64)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.wroteAt.Store(time.Now().UnixNano()) // Lua write pending
		c.Read(buf)
	}
}

func TestParseLuaCommandAndSHA(t *testing.T) {
	t.Run("EVALSHA returns command and sha", func(t *testing.T) {
		cmd, sha, ok := parseLuaCommandAndSHA(respCommand("EVALSHA", "abcdef", "0"))
		require.True(t, ok)
		require.Equal(t, luaCommandEvalSHA, cmd)
		require.Equal(t, "abcdef", sha)
	})

	t.Run("EVAL_RO returns command without sha", func(t *testing.T) {
		cmd, sha, ok := parseLuaCommandAndSHA(respCommand("EVAL_RO", "return 1", "0"))
		require.True(t, ok)
		require.Equal(t, luaCommandEvalRO, cmd)
		require.Empty(t, sha)
	})

	t.Run("non-lua command parses but does not classify", func(t *testing.T) {
		cmd, sha, ok := parseLuaCommandAndSHA(respCommand("GET", "key"))
		require.True(t, ok)
		require.Equal(t, luaCommandNone, cmd)
		require.Empty(t, sha)
	})

	t.Run("lowercase command is not classified", func(t *testing.T) {
		cmd, sha, ok := parseLuaCommandAndSHA(respCommand("evalsha", "abcdef", "0"))
		require.True(t, ok)
		require.Equal(t, luaCommandNone, cmd)
		require.Empty(t, sha)
	})

	t.Run("malformed payload fails closed", func(t *testing.T) {
		cmd, sha, ok := parseLuaCommandAndSHA([]byte("*3\r\n$7\r\nEVALSHA\r\n$6\r\nabc"))
		require.False(t, ok)
		require.Equal(t, luaCommandNone, cmd)
		require.Empty(t, sha)
	})
}
