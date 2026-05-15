package rueidisconn

import (
	"context"
	"crypto/tls"
	"net"
	"sync/atomic"
	"time"

	"github.com/redis/rueidis"
)

// TTFBEvent is emitted on the first byte read after one or more writes.
type TTFBEvent struct {
	// TTFB is the duration from the first write to the first response byte.
	TTFB time.Duration
	// Command is the Redis command name extracted from the first write in the
	// batch.
	Command string
	// ScriptName is the name of the script, as SHA1
	ScriptName string
}

// TTFBHandler is called on each TTFB measurement. Implementations must be
// safe for concurrent use; a single handler may be invoked from many
// connection goroutines simultaneously.
type TTFBHandler func(TTFBEvent)

// NewClient returns an instrumented client that provides TTFB metrics.
//
// This replaces DialCtxFn, using the given option.Dialer and option.TLSConfig
// to make this work.  DialCtxFn takes precedence over any option.Dialer.
func NewClient(option rueidis.ClientOption, handler TTFBHandler) (rueidis.Client, error) {
	option.DialCtxFn = func(ctx context.Context, dst string, dialer *net.Dialer, tlsc *tls.Config) (net.Conn, error) {
		var d net.Dialer
		if dialer != nil {
			d = *dialer
		}

		var (
			conn net.Conn
			err  error
		)

		if tlsc != nil {
			td := tls.Dialer{
				NetDialer: &d,
				Config:    tlsc,
			}
			conn, err = td.DialContext(ctx, "tcp", dst)
		} else {
			conn, err = d.DialContext(ctx, "tcp", dst)
		}
		if err != nil {
			return nil, err
		}

		return &ttfbConn{Conn: conn, onTTFB: handler}, nil
	}

	c, err := rueidis.NewClient(option)
	return c, err
}

type ttfbConn struct {
	net.Conn
	onTTFB TTFBHandler

	// wroteAt encodes the state machine:
	//   0  = idle (no write pending, or after Read consumed)
	//  -1  = first write was non-Lua; waiting for Read to reset
	//  >0  = unix nano of Lua write completion; waiting for Read
	wroteAt atomic.Int64

	// Written by the Write goroutine BEFORE wroteAt.Store(>0).
	// Read by the Read goroutine AFTER wroteAt.Swap(0).
	// The atomic store→swap provides the happens-before guarantee.
	command    string
	scriptName string
}

// Write inspects the first write of each write/read batch and starts
// tracking when the command is a Lua script execution.
func (c *ttfbConn) Write(b []byte) (n int, err error) {
	// First write in batch wins. wroteAt != 0 means a prior write
	// in this batch already claimed it. Skip parsing entirely.
	firstInBatch := c.wroteAt.Load() == 0

	n, err = c.Conn.Write(b)
	if err != nil {
		c.wroteAt.Store(0)
		return n, err
	}

	if !firstInBatch {
		return n, nil
	}

	commandKind, scriptName, ok := parseLuaCommandAndSHA(b)
	if ok && commandKind != luaCommandNone {
		c.command = luaCommandString(commandKind)
		c.scriptName = scriptName
		c.wroteAt.Store(time.Now().UnixNano()) // release
	} else {
		c.wroteAt.Store(-1)
	}
	return n, nil
}

// Read emits TTFB on the first response bytes after a tracked write batch.
func (c *ttfbConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if wt := c.wroteAt.Swap(0); wt > 0 && n > 0 && c.onTTFB != nil {
		c.onTTFB(TTFBEvent{
			TTFB:       time.Duration(time.Now().UnixNano() - wt),
			Command:    c.command,
			ScriptName: c.scriptName,
		})
	}
	return n, err
}

type luaCommand uint8

const (
	luaCommandNone luaCommand = iota
	luaCommandEval
	luaCommandEvalSHA
	luaCommandEvalRO
	luaCommandEvalSHARO
)

func parseLuaCommandAndSHA(b []byte) (command luaCommand, sha string, ok bool) {
	if len(b) == 0 || b[0] != '*' {
		return luaCommandNone, "", false
	}

	arrayLen, next, ok := readRESPUIntCRLF(b, 1)
	if !ok || arrayLen < 1 {
		return luaCommandNone, "", false
	}

	commandBulk, next, ok := readRESPBulkString(b, next)
	if !ok {
		return luaCommandNone, "", false
	}

	command = classifyLuaCommand(commandBulk)
	if command == luaCommandNone {
		return luaCommandNone, "", true
	}

	if command != luaCommandEvalSHA && command != luaCommandEvalSHARO {
		return command, "", true
	}
	if arrayLen < 2 {
		return luaCommandNone, "", false
	}

	shaBulk, _, ok := readRESPBulkString(b, next)
	if !ok {
		return luaCommandNone, "", false
	}

	return command, string(shaBulk), true
}

func luaCommandString(command luaCommand) string {
	switch command {
	case luaCommandEval:
		return "EVAL"
	case luaCommandEvalSHA:
		return "EVALSHA"
	case luaCommandEvalRO:
		return "EVAL_RO"
	case luaCommandEvalSHARO:
		return "EVALSHA_RO"
	default:
		return ""
	}
}

func classifyLuaCommand(command []byte) luaCommand {
	switch len(command) {
	case 4:
		if matchASCII(command, "EVAL") {
			return luaCommandEval
		}
	case 7:
		if matchASCII(command, "EVALSHA") {
			return luaCommandEvalSHA
		}
		if matchASCII(command, "EVAL_RO") {
			return luaCommandEvalRO
		}
	case 10:
		if matchASCII(command, "EVALSHA_RO") {
			return luaCommandEvalSHARO
		}
	}

	return luaCommandNone
}

func matchASCII(actual []byte, expected string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range actual {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}

func readRESPBulkString(b []byte, pos int) (value []byte, next int, ok bool) {
	if pos >= len(b) || b[pos] != '$' {
		return nil, 0, false
	}

	size, next, ok := readRESPUIntCRLF(b, pos+1)
	if !ok || next+size+2 > len(b) {
		return nil, 0, false
	}

	value = b[next : next+size]
	next += size
	if b[next] != '\r' || b[next+1] != '\n' {
		return nil, 0, false
	}

	return value, next + 2, true
}

func readRESPUIntCRLF(b []byte, start int) (val int, next int, ok bool) {
	if start >= len(b) || b[start] < '0' || b[start] > '9' {
		return 0, 0, false
	}

	n := 0
	for i := start; i < len(b); i++ {
		ch := b[i]
		if ch >= '0' && ch <= '9' {
			n = (n * 10) + int(ch-'0')
			continue
		}
		if ch == '\r' && i+1 < len(b) && b[i+1] == '\n' {
			return n, i + 2, true
		}
		return 0, 0, false
	}

	return 0, 0, false
}
