package realtime

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublishingDeadlineAndCancellation(t *testing.T) {
	for _, cancelCaller := range []bool{false, true} {
		name := "timeout"
		if cancelCaller {
			name = "cancellation"
		}
		t.Run(name, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				close(started)
				<-release
			}))
			defer server.Close()
			defer close(release)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			timeout := 100 * time.Millisecond
			if cancelCaller {
				timeout = time.Minute
			}
			done := make(chan error, 1)
			go func() {
				body, err := teeStreamReaderToAPI(ctx, strings.NewReader("complete response"), server.URL, TeeStreamOptions{Channel: "c", Topic: "t", Token: "secret"}, timeout)
				got, readErr := io.ReadAll(body)
				if readErr != nil || string(got) != "complete response" {
					done <- errors.New("response lost")
					return
				}
				done <- err
			}()
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("publisher not reached")
			}
			if cancelCaller {
				cancel()
			}
			select {
			case err := <-done:
				want := context.DeadlineExceeded
				if cancelCaller {
					want = context.Canceled
				}
				if !errors.Is(err, want) {
					t.Fatalf("got %v; want %v", err, want)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("publishing did not stop")
			}
		})
	}
}

func TestPublishingRejectsInsecureURL(t *testing.T) {
	for _, u := range []string{"http://example.com/publish", "http://localhost/publish", "ftp://example.com", "https://user:password@example.com"} {
		t.Run(u, func(t *testing.T) {
			body, err := TeeStreamReaderToAPIWithContext(context.Background(), strings.NewReader("response"), u, TeeStreamOptions{Channel: "c", Topic: "t", Token: "secret"})
			if err == nil {
				t.Fatal("accepted unsafe URL")
			}
			got, _ := io.ReadAll(body)
			if string(got) != "response" {
				t.Fatal("response lost")
			}
		})
	}
}

func TestPublishingDoesNotFollowRedirect(t *testing.T) {
	var requests atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	body, err := TeeStreamReaderToAPIWithContext(context.Background(), strings.NewReader("response"), server.URL, TeeStreamOptions{Channel: "c", Topic: "t", Token: "secret"})
	if err == nil {
		t.Fatal("expected redirect rejection")
	}
	got, _ := io.ReadAll(body)
	if string(got) != "response" {
		t.Fatal("response lost")
	}
	if requests.Load() != 0 {
		t.Fatal("followed redirect")
	}
}

func TestPublishingEarlyRejectionPreservesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer server.Close()
	input := strings.Repeat("payload", 100000)
	body, err := TeeStreamReaderToAPIWithContext(context.Background(), strings.NewReader(input), server.URL, TeeStreamOptions{Channel: "c", Topic: "t", Token: "secret"})
	if err == nil {
		t.Fatal("expected rejection")
	}
	got, e := io.ReadAll(body)
	if e != nil || string(got) != input {
		t.Fatal("response lost")
	}
}

func TestDevServerPublishingExceptionIsExact(t *testing.T) {
	const callback = "http://192.168.1.20:8288/v1/realtime/publish"
	for _, tt := range []struct {
		name, target, allowed string
		want                  bool
	}{
		{"LAN callback", callback, callback, true},
		{"production rejects LAN", callback, "", false},
		{"other host", "http://192.168.1.21:8288/v1/realtime/publish", callback, false},
		{"other port", "http://192.168.1.20:80/v1/realtime/publish", callback, false},
		{"other path", "http://192.168.1.20:8288/other", callback, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.target)
			if err != nil {
				t.Fatal(err)
			}
			if got := allowedPublishURL(u, tt.target, tt.allowed); got != tt.want {
				t.Fatalf("allowed=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestDevServerHTTPPublishing(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "dev-token" {
			t.Error("missing token")
		}
		_, _ = io.Copy(io.Discard, r.Body)
	}))
	defer server.Close()
	// A hostname is not covered by the literal-loopback exception.
	target := strings.Replace(server.URL, "127.0.0.1", "localhost", 1) + "/v1/realtime/publish"
	body, err := TeeStreamReaderToAPIWithContext(context.Background(), strings.NewReader("response"), target, TeeStreamOptions{
		Channel: "c", Topic: "t", Token: "dev-token", AllowedHTTPURL: target,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(body)
	if err != nil || string(got) != "response" {
		t.Fatal("response lost")
	}
	if calls.Load() != 1 {
		t.Fatal("publishing did not reach the configured server")
	}
}

// An idle SDK stream must not prevent publishing from returning on rejection,
// deadline, or cancellation. Its next bytes still belong to the response reader.
func TestPublishingIdleSourceHandoff(t *testing.T) {
	for _, outcome := range []string{"rejection", "timeout", "cancellation"} {
		t.Run(outcome, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			source, writer := io.Pipe()
			defer source.Close()
			defer writer.Close()
			blocked := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Do not let the HTTP server drain the idle request before replying.
				if err := http.NewResponseController(w).EnableFullDuplex(); err != nil {
					t.Error(err)
					return
				}
				prefix := make([]byte, len("first"))
				if _, err := io.ReadFull(r.Body, prefix); err != nil {
					t.Error(err)
					return
				}
				<-blocked
				switch outcome {
				case "rejection":
					w.WriteHeader(http.StatusUnauthorized)
					w.(http.Flusher).Flush()
				case "cancellation":
					cancel()
				}
				<-release
			}))
			defer server.Close()
			// Release the server before Close waits for its handler.
			defer close(release)
			type result struct {
				body io.Reader
				err  error
			}
			done := make(chan result, 1)
			timeout := 3 * time.Second
			if outcome == "timeout" {
				timeout = 100 * time.Millisecond
			}
			go func() {
				body, err := teeStreamReaderToAPI(ctx, &notifySecondRead{reader: source, blocked: blocked}, server.URL, TeeStreamOptions{Channel: "c", Topic: "t", Token: "secret"}, timeout)
				done <- result{body, err}
			}()
			go func() { _, _ = writer.Write([]byte("first")) }()
			var got result
			select {
			case got = <-done:
				if got.err == nil {
					t.Error("expected publishing failure")
				}
			case <-time.After(2 * time.Second):
				// Unblock transport cleanup before failing the regression test.
				_ = writer.Close()
				t.Error("publishing handoff waited for the idle SDK source")
				got = <-done
			}
			go func() {
				_, _ = writer.Write([]byte("second"))
				_ = writer.Close()
			}()
			body, err := io.ReadAll(got.body)
			if err != nil || string(body) != "firstsecond" {
				t.Errorf("response = %q, %v; want full response", body, err)
			}
		})
	}
}

type notifySecondRead struct {
	reader  io.Reader
	blocked chan struct{}
	reads   int
}

func (r *notifySecondRead) Read(p []byte) (int, error) {
	r.reads++
	if r.reads == 2 {
		close(r.blocked)
	}
	return r.reader.Read(p)
}
