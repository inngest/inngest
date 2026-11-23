package jsonlines

import (
	"bytes"
	"io"
	"testing"
	"time"
)

// TestSessionDrainPreventsPipeDeadlock verifies that after Drain() is called,
// writing well over 64 lines (s.lines's buffer size) to the subprocess pipe
// never blocks and the reader goroutine exits cleanly when the pipe closes.
func TestSessionDrainPreventsPipeDeadlock(t *testing.T) {
	pr, pw := io.Pipe()
	s := NewSession(&bytes.Buffer{}, pr)

	s.Drain()

	done := make(chan error, 1)
	go func() {
		for range 200 {
			if _, err := io.WriteString(pw, "warning: noisy diagnostic line\n"); err != nil {
				done <- err
				return
			}
		}
		done <- pw.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("writing to drained pipe failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("writing to drained pipe deadlocked")
	}

	// readLoop closes s.lines on exit; verify no goroutine leak.
	select {
	case _, ok := <-s.lines:
		if ok {
			t.Fatal("expected s.lines to be closed after EOF")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("readLoop did not exit after pipe closed")
	}
}
