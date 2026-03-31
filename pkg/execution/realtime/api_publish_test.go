package realtime

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_PostPublishTee(t *testing.T) {
	t.Run("successfully forwards raw data to subscribers", func(t *testing.T) {
		broadcaster := NewInProcessBroadcaster()
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: broadcaster,
		}))
		defer server.Close()

		// Create a channel to capture received data
		var mu sync.Mutex
		receivedData := [][]byte{}

		// Create a test subscription that collects raw bytes
		testSub := NewInmemorySubscription(uuid.New(), func(data []byte) error {
			mu.Lock()
			receivedData = append(receivedData, append([]byte(nil), data...)) // Copy data
			mu.Unlock()
			return nil
		})

		// Subscribe to a test channel
		channel := "test-channel-123"
		topic := Topic{
			Kind:    streamingtypes.TopicKindRun,
			Channel: channel,
			Name:    "test-topic",
			EnvID:   uuid.New(),
		}

		err := broadcaster.Subscribe(context.Background(), testSub, []Topic{topic})
		require.NoError(t, err)

		// Give time for subscription to be established
		time.Sleep(20 * time.Millisecond)

		// Create JWT with publish permissions
		accountID := uuid.New()
		envID := topic.EnvID
		publishJWT, err := NewPublishJWT(context.Background(), []byte("test-secret"), accountID, envID)
		require.NoError(t, err)

		// Test data to send
		testData := []byte("Hello from PostPublishTee endpoint!")

		// Create request
		req, err := http.NewRequest("POST", server.URL+"/realtime/publish/tee?channel="+channel, bytes.NewReader(testData))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+publishJWT)

		// Make the request
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Verify successful response
		require.Equal(t, 200, resp.StatusCode)

		// Wait for data to be processed
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(receivedData) == 1
		}, 2*time.Second, 50*time.Millisecond)

		// Verify the subscription received the raw data
		mu.Lock()
		require.Len(t, receivedData, 1, "Should receive exactly one message")
		assert.Equal(t, testData, receivedData[0], "Should receive the exact raw data")
		mu.Unlock()
	})

	t.Run("handles large data correctly", func(t *testing.T) {
		broadcaster := NewInProcessBroadcaster()
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: broadcaster,
		}))
		defer server.Close()

		var mu sync.Mutex
		receivedData := [][]byte{}

		testSub := NewInmemorySubscription(uuid.New(), func(data []byte) error {
			mu.Lock()
			receivedData = append(receivedData, append([]byte(nil), data...))
			mu.Unlock()
			return nil
		})

		channel := "large-data-channel"
		topic := Topic{
			Kind:    streamingtypes.TopicKindRun,
			Channel: channel,
			Name:    "test-topic",
			EnvID:   uuid.New(),
		}

		err := broadcaster.Subscribe(context.Background(), testSub, []Topic{topic})
		require.NoError(t, err)

		time.Sleep(20 * time.Millisecond)

		// Create JWT with publish permissions
		accountID := uuid.New()
		envID := topic.EnvID
		jwt, err := NewPublishJWT(context.Background(), []byte("test-secret"), accountID, envID)
		require.NoError(t, err)

		// Create large test data (10KB)
		largeData := bytes.Repeat([]byte("A"), 10*1024)

		req, err := http.NewRequest("POST", server.URL+"/realtime/publish/tee?channel="+channel, bytes.NewReader(largeData))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+jwt)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 200, resp.StatusCode)

		// Wait for data to be processed - io.Copy may send data in multiple chunks
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			totalBytes := 0
			for _, chunk := range receivedData {
				totalBytes += len(chunk)
			}
			return totalBytes == len(largeData)
		}, 2*time.Second, 50*time.Millisecond)

		// Verify the large data was received correctly (may be in multiple chunks)
		mu.Lock()
		var reconstructedData []byte
		for _, chunk := range receivedData {
			reconstructedData = append(reconstructedData, chunk...)
		}
		assert.Equal(t, largeData, reconstructedData, "Reconstructed data should match original")
		mu.Unlock()
	})

	t.Run("handles multiple subscribers correctly", func(t *testing.T) {
		broadcaster := NewInProcessBroadcaster()
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: broadcaster,
		}))
		defer server.Close()

		var mu1, mu2 sync.Mutex
		receivedData1 := [][]byte{}
		receivedData2 := [][]byte{}

		// Create two subscribers
		sub1 := NewInmemorySubscription(uuid.New(), func(data []byte) error {
			mu1.Lock()
			receivedData1 = append(receivedData1, append([]byte(nil), data...))
			mu1.Unlock()
			return nil
		})

		sub2 := NewInmemorySubscription(uuid.New(), func(data []byte) error {
			mu2.Lock()
			receivedData2 = append(receivedData2, append([]byte(nil), data...))
			mu2.Unlock()
			return nil
		})

		channel := "multi-sub-channel"
		topic := Topic{
			Kind:    streamingtypes.TopicKindRun,
			Channel: channel,
			Name:    "test-topic",
			EnvID:   uuid.New(),
		}

		// Subscribe both to the same channel
		err := broadcaster.Subscribe(context.Background(), sub1, []Topic{topic})
		require.NoError(t, err)
		err = broadcaster.Subscribe(context.Background(), sub2, []Topic{topic})
		require.NoError(t, err)

		time.Sleep(20 * time.Millisecond)

		// Create JWT
		accountID := uuid.New()
		envID := topic.EnvID
		jwt, err := NewPublishJWT(context.Background(), []byte("test-secret"), accountID, envID)
		require.NoError(t, err)

		testData := []byte("Broadcast to multiple subscribers")

		req, err := http.NewRequest("POST", server.URL+"/realtime/publish/tee?channel="+channel, bytes.NewReader(testData))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+jwt)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 200, resp.StatusCode)

		// Wait for both subscribers to receive data
		assert.Eventually(t, func() bool {
			mu1.Lock()
			mu2.Lock()
			defer mu1.Unlock()
			defer mu2.Unlock()
			return len(receivedData1) == 1 && len(receivedData2) == 1
		}, 2*time.Second, 50*time.Millisecond)

		// Verify both subscribers received the same data
		mu1.Lock()
		mu2.Lock()
		assert.Equal(t, testData, receivedData1[0])
		assert.Equal(t, testData, receivedData2[0])
		mu1.Unlock()
		mu2.Unlock()
	})

	t.Run("unauthorized request fails", func(t *testing.T) {
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: NewInProcessBroadcaster(),
		}))
		defer server.Close()

		testData := []byte("Unauthorized data")
		req, err := http.NewRequest("POST", server.URL+"/realtime/publish/tee?channel=test", bytes.NewReader(testData))
		require.NoError(t, err)
		// No authorization header

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 401, resp.StatusCode)
	})

	t.Run("missing channel parameter fails", func(t *testing.T) {
		broadcaster := NewInProcessBroadcaster()
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: broadcaster,
		}))
		defer server.Close()

		// Create JWT with publish permissions
		accountID := uuid.New()
		envID := uuid.New()
		jwt, err := NewPublishJWT(context.Background(), []byte("test-secret"), accountID, envID)
		require.NoError(t, err)

		testData := []byte("Test data")
		req, err := http.NewRequest("POST", server.URL+"/realtime/publish/tee", bytes.NewReader(testData)) // No channel parameter
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+jwt)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 400, resp.StatusCode)
	})

	t.Run("invalid JWT fails", func(t *testing.T) {
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: NewInProcessBroadcaster(),
		}))
		defer server.Close()

		testData := []byte("Test data")
		req, err := http.NewRequest("POST", server.URL+"/realtime/publish/tee?channel=test", bytes.NewReader(testData))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid-jwt")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 401, resp.StatusCode)
	})

	t.Run("streaming data works correctly", func(t *testing.T) {
		broadcaster := NewInProcessBroadcaster()
		server := httptest.NewServer(NewAPI(APIOpts{
			JWTSecret:   []byte("test-secret"),
			Broadcaster: broadcaster,
		}))
		defer server.Close()

		var mu sync.Mutex
		receivedData := [][]byte{}

		testSub := NewInmemorySubscription(uuid.New(), func(data []byte) error {
			mu.Lock()
			receivedData = append(receivedData, append([]byte(nil), data...))
			mu.Unlock()
			return nil
		})

		channel := "streaming-channel"
		topic := Topic{
			Kind:    streamingtypes.TopicKindRun,
			Channel: channel,
			Name:    "test-topic",
			EnvID:   uuid.New(),
		}

		err := broadcaster.Subscribe(context.Background(), testSub, []Topic{topic})
		require.NoError(t, err)

		time.Sleep(20 * time.Millisecond)

		// Create JWT
		accountID := uuid.New()
		envID := topic.EnvID
		jwt, err := NewPublishJWT(context.Background(), []byte("test-secret"), accountID, envID)
		require.NoError(t, err)

		// Create a streaming reader that sends data in chunks
		streamingData := "data: First chunk\n\ndata: Second chunk\n\ndata: Third chunk\n\n"
		reader := strings.NewReader(streamingData)

		req, err := http.NewRequest("POST", server.URL+"/realtime/publish/tee?channel="+channel, reader)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+jwt)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, 200, resp.StatusCode)

		// Wait for data to be processed
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(receivedData) > 0
		}, 2*time.Second, 50*time.Millisecond)

		// Verify the streaming data was received as expected
		mu.Lock()
		require.Len(t, receivedData, 1)
		assert.Equal(t, []byte(streamingData), receivedData[0])
		mu.Unlock()
	})
}
