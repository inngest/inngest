package worker

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
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
