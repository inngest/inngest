package worker

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMinimalChildEnv(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\test\AppData\Local`)
	t.Setenv("LocalAppData", `C:\Users\test\AppData\LocalMixed`)
	t.Setenv("SUPER_SECRET_TOKEN", "do-not-leak")

	env := minimalChildEnv([]string{"EXTRA_VAR=1"})

	if !slices.Contains(env, `LOCALAPPDATA=C:\Users\test\AppData\Local`) &&
		!slices.Contains(env, `LocalAppData=C:\Users\test\AppData\LocalMixed`) {
		t.Fatalf("expected LOCALAPPDATA/LocalAppData to be preserved, got %v", env)
	}
	if slices.Contains(env, "SUPER_SECRET_TOKEN=do-not-leak") {
		t.Fatalf("expected secret env var to be dropped, got %v", env)
	}
	if !slices.Contains(env, "EXTRA_VAR=1") {
		t.Fatalf("expected extra env var to be appended, got %v", env)
	}
}

func TestQueryWithStreamsCancelsSiblingFeedsOnSourceError(t *testing.T) {
	base := ""
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "dw-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "m.sock")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Mock server accepts OpQueryStream, reads OpStreamError, and responds with KindError.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var req Request
		if err := ReadFrame(conn, &req); err != nil {
			return
		}
		for {
			var frame Request
			if err := ReadFrame(conn, &frame); err != nil {
				return
			}
			if frame.Op == OpStreamError {
				_ = WriteFrame(conn, Response{Kind: KindError, ErrorCode: ErrCodeStreamSource, Error: frame.Error})
				return
			}
		}
	}()

	client := &Client{socketPath: sockPath}
	sess := &Session{client: client, id: "s-1"}

	siblingCanceled := make(chan struct{})
	srcErr := errors.New("boom")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = sess.QueryWithStreams(ctx, "SELECT 1", []SendStream{
		{
			ID: "a",
			Next: func(context.Context) ([][]any, error) {
				return nil, srcErr
			},
		},
		{
			ID: "b",
			Next: func(feedCtx context.Context) ([][]any, error) {
				<-feedCtx.Done()
				close(siblingCanceled)
				return nil, feedCtx.Err()
			},
		},
	})
	if !errors.Is(err, ErrStreamSource) || !errors.Is(err, srcErr) {
		t.Fatalf("QueryWithStreams err = %v, want wrapped %v", err, srcErr)
	}
	select {
	case <-siblingCanceled:
	case <-time.After(time.Second):
		t.Fatal("sibling stream feedCtx was not cancelled on source error")
	}
}

func TestQueryWithStreamsPreservesSQLErrorOverCleanupCancel(t *testing.T) {
	base := ""
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "dw-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "sqlerr.sock")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// Mock server accepts OpQueryStream and immediately returns a SQL KindError
	// while the client's feed goroutine is blocked waiting on feedCtx.Done().
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var req Request
		if err := ReadFrame(conn, &req); err != nil {
			return
		}
		_ = WriteFrame(conn, Response{
			Kind:  KindError,
			Error: "Binder Error: Referenced column \"missing_col\" not found",
		})
	}()

	client := &Client{socketPath: sockPath}
	sess := &Session{client: client, id: "s-1"}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = sess.QueryWithStreams(ctx, "SELECT missing_col", []SendStream{
		{
			ID: "a",
			Next: func(feedCtx context.Context) ([][]any, error) {
				<-feedCtx.Done()
				return nil, feedCtx.Err()
			},
		},
	})
	if err == nil {
		t.Fatal("expected SQL error, got nil")
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, ErrStreamSource) {
		t.Fatalf("QueryWithStreams hid SQL error behind cleanup cancellation: %v", err)
	}
	if got := err.Error(); got != `Binder Error: Referenced column "missing_col" not found` {
		t.Fatalf("QueryWithStreams err = %q, want SQL error", got)
	}
}

func TestStartCancelWatcherClosesSessionOnBothPaths(t *testing.T) {
	base := ""
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "dw-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "cancel.sock")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var closeCount atomic.Int32
	closedCh := make(chan struct{}, 2)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				var req Request
				if err := ReadFrame(c, &req); err != nil {
					return
				}
				if req.Op == OpCloseSession {
					closeCount.Add(1)
					select {
					case closedCh <- struct{}{}:
					default:
					}
				}
				_ = WriteFrame(c, Response{Kind: KindOK})
			}(conn)
		}
	}()

	var abandoned atomic.Int64
	client := &Client{socketPath: sockPath}
	sess := &Session{client: client, id: "s-cancel", abandonedCounter: &abandoned}

	// Simulate cancellation that races with stopWatch().
	ctx, cancel := context.WithCancel(context.Background())
	stopWatch := sess.startCancelWatcher(ctx, nil, 1)
	cancel()
	stopWatch()

	if !sess.Abandoned() {
		t.Fatal("expected session to be marked abandoned")
	}
	select {
	case <-closedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("expected abandoned session to send OpCloseSession to worker")
	}
	if got := closeCount.Load(); got != 1 {
		t.Fatalf("expected exactly 1 OpCloseSession, got %d", got)
	}
	deadline := time.Now().Add(2 * time.Second)
	for abandoned.Load() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := abandoned.Load(); got != 0 {
		t.Fatalf("abandonedCounter = %d, want 0", got)
	}
}

func TestQueryWithStreamsFlowControlThrottlesFastStreamWithoutBlockingSibling(t *testing.T) {
	origWindow := defaultStreamWindowBytes
	defaultStreamWindowBytes = 4 << 10 // 4 KiB window
	t.Cleanup(func() { defaultStreamWindowBytes = origWindow })

	base := ""
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "dw-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "fc.sock")
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	const fastBatches = 10
	payload := strings.Repeat("x", 3<<10) // ~3 KiB per batch; 2 batches exhaust 4 KiB window

	var fastReceived atomic.Int32
	slowArrived := make(chan struct{})
	var serverErr error
	serverDone := make(chan struct{})

	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			serverErr = err
			return
		}
		defer conn.Close()

		var req Request
		if err := ReadFrame(conn, &req); err != nil {
			serverErr = err
			return
		}

		var writeMu sync.Mutex
		sendResp := func(r Response) error {
			writeMu.Lock()
			defer writeMu.Unlock()
			return WriteFrame(conn, r)
		}

		// Hold fast credits until the slow stream delivers its batch so we
		// prove fast throttles instead of starving or deadlocking slow.
		var heldFastCredits []int64
		slowSeen := false
		remaining := len(req.StreamIDs)
		for remaining > 0 {
			var frame Request
			if err := ReadFrame(conn, &frame); err != nil {
				serverErr = err
				return
			}
			switch frame.Op {
			case OpStreamData:
				if frame.StreamID == "fast" {
					fastReceived.Add(1)
					if !slowSeen {
						heldFastCredits = append(heldFastCredits, int64(len(frame.Blob)))
					} else {
						if err := sendResp(Response{Kind: KindStreamCredit, StreamID: "fast", Bytes: int64(len(frame.Blob))}); err != nil {
							serverErr = err
							return
						}
					}
				} else if frame.StreamID == "slow" {
					if !slowSeen {
						slowSeen = true
						close(slowArrived)
						for _, b := range heldFastCredits {
							if err := sendResp(Response{Kind: KindStreamCredit, StreamID: "fast", Bytes: b}); err != nil {
								serverErr = err
								return
							}
						}
						heldFastCredits = nil
					}
					if err := sendResp(Response{Kind: KindStreamCredit, StreamID: "slow", Bytes: int64(len(frame.Blob))}); err != nil {
						serverErr = err
						return
					}
				}
			case OpStreamEOF:
				remaining--
			}
		}

		_ = sendResp(Response{Kind: KindMeta, Columns: []ColumnInfo{{Name: "ok", Type: TypeBigInt}}})
		_ = sendResp(Response{Kind: KindChunk, Rows: EncodeRows([]ColumnInfo{{Name: "ok", Type: TypeBigInt}}, [][]any{{int64(1)}}) })
		_ = sendResp(Response{Kind: KindDone})
	}()

	client := &Client{socketPath: sockPath}
	sess := &Session{client: client, id: "s-fc"}

	var fastSent atomic.Int32
	slowGate := make(chan struct{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wait briefly after fast exhausts its window, verify it's throttled
	// (< fastBatches), then release slow to verify slow isn't blocked.
	go func() {
		deadline := time.Now().Add(2 * time.Second)
		for fastSent.Load() < 2 && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		time.Sleep(50 * time.Millisecond)
		close(slowGate)
	}()

	var slowSent atomic.Bool
	qs, err := sess.QueryWithStreams(ctx, "SELECT 1", []SendStream{
		{
			ID:      "fast",
			Columns: []StreamColumnKind{StreamColumnVarchar},
			Next: func(ctx context.Context) ([][]any, error) {
				i := fastSent.Add(1)
				if i > fastBatches {
					return nil, io.EOF
				}
				return [][]any{{payload}}, nil
			},
		},
		{
			ID:      "slow",
			Columns: []StreamColumnKind{StreamColumnVarchar},
			Next: func(ctx context.Context) ([][]any, error) {
				if slowSent.Swap(true) {
					return nil, io.EOF
				}
				select {
				case <-slowGate:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
				// Before slow sends, fast must have been throttled well below all 10 batches.
				if sent := fastSent.Load(); sent >= fastBatches {
					return nil, errors.New("fast stream was not throttled by credit window")
				}
				return [][]any{{"slow-row"}}, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("QueryWithStreams: %v", err)
	}
	defer qs.Close()

	rows, err := qs.NextChunk()
	if err != nil {
		t.Fatalf("NextChunk: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != int64(1) {
		t.Fatalf("rows = %#v, want [[1]]", rows)
	}
	if more, err := qs.NextChunk(); err != nil || more != nil {
		t.Fatalf("expected EOF, got (%v, %v)", more, err)
	}
	<-serverDone
	if serverErr != nil {
		t.Fatalf("mock server error: %v", serverErr)
	}
	if got := fastReceived.Load(); got != fastBatches {
		t.Fatalf("fastReceived = %d, want %d", got, fastBatches)
	}
}
