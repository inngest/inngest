package worker

import (
	"context"
	"errors"
	"net"
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

func TestRoundTripCancelsAgainstStalledServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	received := make(chan Request, 4)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

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
				received <- req
				<-release
			}(conn)
		}
	}()

	client := &Client{socketPath: ln.Addr().String()}

	cases := []struct {
		name string
		call func(context.Context) error
	}{
		{
			name: "Ping",
			call: func(ctx context.Context) error {
				_, err := client.Ping(ctx)
				return err
			},
		},
		{
			name: "OpenSession",
			call: func(ctx context.Context) error {
				_, err := client.OpenSession(ctx)
				return err
			},
		},
		{
			name: "SessionClose",
			call: func(ctx context.Context) error {
				sess := &Session{client: client, id: "s-test-1"}
				return sess.Close(ctx)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			errCh := make(chan error, 1)
			go func() {
				errCh <- tc.call(ctx)
			}()

			select {
			case <-received:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for stalled server to receive request")
			}

			cancel()

			select {
			case err := <-errCh:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("got err=%v, want context.Canceled", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("call did not unblock after context cancellation")
			}
		})
	}
}
