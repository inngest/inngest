package realtime

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
