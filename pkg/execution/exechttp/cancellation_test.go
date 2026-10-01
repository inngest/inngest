package exechttp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDoRequestCancellationClosesSDK(t *testing.T) {
	for _, publishing := range []bool{false, true} {
		for _, headers := range []bool{false, true} {
			name := "ordinary"
			if publishing {
				name = "publishing"
			}
			if headers {
				name += "/during_body"
			} else {
				name += "/before_headers"
			}
			t.Run(name, func(t *testing.T) {
				entered, closed, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
				sdk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if headers {
						_, _ = io.WriteString(w, "first chunk\n")
						w.(http.Flusher).Flush()
					}
					close(entered)
					select {
					case <-r.Context().Done():
						close(closed)
					case <-release:
					}
				}))
				publisherEntered := make(chan struct{})
				publisher := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					close(publisherEntered)
					_, _ = io.Copy(io.Discard, r.Body)
				}))
				defer func() { close(release); sdk.Close(); publisher.Close() }()
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				client := ExtendedClient{Client: sdk.Client(), publish: publishing}
				done := make(chan error, 1)
				go func() {
					_, err := client.DoRequest(ctx, SerializableRequest{Method: "GET", URL: sdk.URL, Header: http.Header{}, Publish: RequestPublishOpts{Channel: "test", Topic: "stream", Token: "test", PublishURL: publisher.URL}})
					done <- err
				}()
				select {
				case <-entered:
				case <-time.After(3 * time.Second):
					t.Fatal("SDK not reached")
				}
				if publishing && headers {
					select {
					case <-publisherEntered:
					case <-time.After(3 * time.Second):
						t.Fatal("publisher not reached")
					}
				}
				cancel()
				select {
				case err := <-done:
					require.ErrorIs(t, err, context.Canceled)
				case <-time.After(3 * time.Second):
					t.Fatal("request did not cancel")
				}
				select {
				case <-closed:
				case <-time.After(3 * time.Second):
					t.Fatal("SDK connection remained open")
				}
			})
		}
	}
}
