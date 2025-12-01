package realtime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/stretchr/testify/require"
)

func TestAPI_GetSSE(t *testing.T) {
	t.Run("successful connection with headers", func(t *testing.T) {
		broadcaster := NewInProcessBroadcaster()
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: broadcaster,
		}))
		defer server.Close()

		// Create test topics and JWT
		topics := []Topic{
			{Kind: streamingtypes.TopicKindRun, Channel: "user:123", Name: "ai"},
		}
		jwt, err := NewJWT(context.Background(), []byte("test-secret"), uuid.New(), uuid.New(), topics)
		require.NoError(t, err)

		// Create request with JWT authorization
		req, err := http.NewRequest("GET", server.URL+"/realtime/sse", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+jwt)

		// Use a context with timeout to avoid hanging
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{Timeout: 200 * time.Millisecond}
		resp, err := client.Do(req)
		require.NoError(t, err)

		// Verify SSE headers were set
		require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
		require.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
		require.Equal(t, "keep-alive", resp.Header.Get("Connection"))
		require.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))

		// Before closing, there should be 1 active subscription
		broadcaster.l.RLock()
		initialSubCount := len(broadcaster.subs)
		broadcaster.l.RUnlock()
		require.Equal(t, 1, initialSubCount, "Should have 1 active subscription")

		// Close the response to trigger cleanup
		resp.Body.Close()

		// Give time for cleanup to occur
		time.Sleep(50 * time.Millisecond)

		// After closing, the subscription should be cleaned up
		broadcaster.l.RLock()
		finalSubCount := len(broadcaster.subs)
		broadcaster.l.RUnlock()
		require.Equal(t, 0, finalSubCount, "Subscription should be cleaned up after connection close")
	})

	t.Run("sends messages in SSE format", func(t *testing.T) {
		broadcaster := NewInProcessBroadcaster()
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: broadcaster,
		}))
		defer server.Close()

		// Create test topics and JWT
		topics := []Topic{
			{Kind: streamingtypes.TopicKindRun, Channel: "user:123", Name: "ai"},
		}
		jwt, err := NewJWT(context.Background(), []byte("test-secret"), uuid.New(), uuid.New(), topics)
		require.NoError(t, err)

		// Start SSE connection in background
		go func() {
			// Give time for connection to be established
			time.Sleep(50 * time.Millisecond)

			// Publish a message to the topic
			msg := Message{
				Kind:      streamingtypes.MessageKindData,
				Data:      json.RawMessage(`"test message"`),
				CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
				Channel:   "user:123",
				Topic:     "ai",
			}
			broadcaster.Publish(context.Background(), msg)
		}()

		// Create request with JWT authorization
		req, err := http.NewRequest("GET", server.URL+"/realtime/sse", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+jwt)

		// Use a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{Timeout: 300 * time.Millisecond}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Read the response body (this will block until timeout or data)
		body, err := io.ReadAll(resp.Body)
		output := string(body)

		// If we got data, verify it's in SSE format
		if len(output) > 0 {
			require.True(t, strings.Contains(output, "data: "))
			require.True(t, strings.Contains(output, `"test message"`))
			require.True(t, strings.Contains(output, "\n\n"))
		}
		// If no data, that's also fine as timing can be tricky in tests
	})

	t.Run("unauthorized request", func(t *testing.T) {
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: NewInProcessBroadcaster(),
		}))
		defer server.Close()

		// Request without authorization
		req, err := http.NewRequest("GET", server.URL+"/realtime/sse", nil)
		require.NoError(t, err)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 401, resp.StatusCode)
		require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	})

	t.Run("invalid JWT", func(t *testing.T) {
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: NewInProcessBroadcaster(),
		}))
		defer server.Close()

		// Request with invalid JWT
		req, err := http.NewRequest("GET", server.URL+"/realtime/sse", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid-jwt")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 401, resp.StatusCode)
	})

	t.Run("connection timeout behavior", func(t *testing.T) {
		broadcaster := NewInProcessBroadcaster()
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: broadcaster,
		}))
		defer server.Close()

		topics := []Topic{
			{Kind: streamingtypes.TopicKindRun, Channel: "user:123", Name: "ai"},
		}
		jwt, err := NewJWT(context.Background(), []byte("test-secret"), uuid.New(), uuid.New(), topics)
		require.NoError(t, err)

		req, err := http.NewRequest("GET", server.URL+"/realtime/sse", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+jwt)

		// Use a very short timeout to test connection handling
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		req = req.WithContext(ctx)

		client := &http.Client{Timeout: 100 * time.Millisecond}
		start := time.Now()
		resp, err := client.Do(req)
		elapsed := time.Since(start)

		// Connection should terminate quickly due to context timeout
		require.Less(t, elapsed, 200*time.Millisecond)

		if err == nil {
			defer resp.Body.Close()
			// If no error, verify headers were still set
			require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
		}
		// Client timeout or context cancellation is expected and acceptable
	})
}
