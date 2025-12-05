package realtime

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/stretchr/testify/require"
)

func TestNewSSESubscription(t *testing.T) {
	ctx := context.Background()
	w := httptest.NewRecorder()

	sub := NewSSESubscription(ctx, w)

	require.NotEqual(t, uuid.Nil, sub.ID())
	require.Equal(t, "sse", sub.Protocol())

	// Verify SSE headers are set
	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "Content-Type", w.Header().Get("Access-Control-Expose-Headers"))
	require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	require.Equal(t, "keep-alive", w.Header().Get("Connection"))
}

func TestSSESubscription_ID(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	id1 := sub.ID()
	id2 := sub.ID()

	require.Equal(t, id1, id2) // ID should be consistent
	require.NotEqual(t, uuid.Nil, id1)
}

func TestSSESubscription_Protocol(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	require.Equal(t, "sse", sub.Protocol())
}

func TestSSESubscription_Write(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	testData := []byte("test data")
	err := sub.Write(testData)

	require.NoError(t, err)
	require.Equal(t, testData, w.Body.Bytes())
	require.True(t, w.Flushed)
}

func TestSSESubscription_WriteMessage(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	msg := Message{
		Kind:      streamingtypes.MessageKindData,
		Data:      json.RawMessage(`"test output"`),
		CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
		Channel:   "user:123",
		Topic:     "ai",
	}

	err := sub.WriteMessage(msg)
	require.NoError(t, err)

	// Verify the message was written in SSE format
	output := w.Body.String()
	require.True(t, strings.HasPrefix(output, "data: "))
	require.True(t, strings.HasSuffix(output, "\n\n"))
	
	// Extract and verify the JSON content
	jsonPart := strings.TrimPrefix(output, "data: ")
	jsonPart = strings.TrimSuffix(jsonPart, "\n\n")
	
	var writtenMsg Message
	err = json.Unmarshal([]byte(jsonPart), &writtenMsg)
	require.NoError(t, err)
	require.Equal(t, msg, writtenMsg)
	require.True(t, w.Flushed)
}

func TestSSESubscription_WriteMessage_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	// Create a message with invalid JSON data
	msg := Message{
		Kind:      streamingtypes.MessageKindData,
		Data:      json.RawMessage(`invalid json`), // Invalid JSON
		CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
		Channel:   "user:123",
		Topic:     "ai",
	}

	err := sub.WriteMessage(msg)
	require.NoError(t, err)

	// Verify the message was written in SSE format
	output := w.Body.String()
	require.True(t, strings.HasPrefix(output, "data: "))
	require.True(t, strings.HasSuffix(output, "\n\n"))
	
	// Extract and verify the JSON content
	jsonPart := strings.TrimPrefix(output, "data: ")
	jsonPart = strings.TrimSuffix(jsonPart, "\n\n")
	
	// Verify the invalid JSON was converted to a valid JSON string
	var writtenMsg Message
	err = json.Unmarshal([]byte(jsonPart), &writtenMsg)
	require.NoError(t, err)
	require.Equal(t, json.RawMessage(`"invalid json"`), writtenMsg.Data)
	require.True(t, w.Flushed)
}

func TestSSESubscription_WriteChunk(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	chunk := Chunk{
		Kind:     "chunk",
		StreamID: "stream123",
		Data:     "chunk data",
	}

	err := sub.WriteChunk(chunk)
	require.NoError(t, err)

	// Verify the chunk was written in SSE format
	output := w.Body.String()
	require.True(t, strings.HasPrefix(output, "data: "))
	require.True(t, strings.HasSuffix(output, "\n\n"))
	
	// Extract and verify the JSON content
	jsonPart := strings.TrimPrefix(output, "data: ")
	jsonPart = strings.TrimSuffix(jsonPart, "\n\n")
	
	var writtenChunk Chunk
	err = json.Unmarshal([]byte(jsonPart), &writtenChunk)
	require.NoError(t, err)
	require.Equal(t, chunk, writtenChunk)
	require.True(t, w.Flushed)
}

func TestSSESubscription_SendKeepalive(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	msg := Message{
		Kind: streamingtypes.MessageKindPing,
	}

	err := sub.SendKeepalive(msg)
	require.NoError(t, err)

	// Verify the keepalive format (SSE comment)
	require.Equal(t, ":\n\n", w.Body.String())
	require.True(t, w.Flushed)
}

func TestSSESubscription_Close_WithoutHijacker(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	// Should not error when ResponseWriter doesn't implement Hijacker
	err := sub.Close()
	require.NoError(t, err)
}

func TestSSEHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	
	// Call sseHeaders directly
	sseHeaders(w)

	// Verify all the required SSE headers are set
	headers := w.Header()
	require.Equal(t, "*", headers.Get("Access-Control-Allow-Origin"))
	require.Equal(t, "Content-Type", headers.Get("Access-Control-Expose-Headers"))
	require.Equal(t, "text/event-stream", headers.Get("Content-Type"))
	require.Equal(t, "no-cache", headers.Get("Cache-Control"))
	require.Equal(t, "keep-alive", headers.Get("Connection"))
}

func TestSSESubscription_MultipleMessages(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	// Send multiple messages
	msg1 := Message{
		Kind:      streamingtypes.MessageKindData,
		Data:      json.RawMessage(`"first message"`),
		CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
		Channel:   "user:123",
		Topic:     "ai",
	}
	
	msg2 := Message{
		Kind:      streamingtypes.MessageKindData,
		Data:      json.RawMessage(`"second message"`),
		CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
		Channel:   "user:123", 
		Topic:     "ai",
	}

	err := sub.WriteMessage(msg1)
	require.NoError(t, err)
	
	err = sub.WriteMessage(msg2)
	require.NoError(t, err)

	// Check the raw output - this will show us what format we're actually using
	output := w.Body.String()
	t.Logf("Actual output: %q", output)
	
	// For now, let's see what the current behavior is
	require.NotEmpty(t, output)
}

func TestSSESubscription_SSEMessageFormat(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	msg := Message{
		Kind:      streamingtypes.MessageKindData,
		Data:      json.RawMessage(`"test message"`),
		CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
		Channel:   "user:123",
		Topic:     "ai",
	}

	err := sub.WriteMessage(msg)
	require.NoError(t, err)

	output := w.Body.String()
	
	// Verify proper SSE format: "data: {json}\n\n"
	require.True(t, strings.HasPrefix(output, "data: "), "SSE message should start with 'data: '")
	require.True(t, strings.HasSuffix(output, "\n\n"), "SSE message should end with double newline")
	
	// Verify the JSON content is between the SSE formatting
	jsonPart := strings.TrimPrefix(output, "data: ")
	jsonPart = strings.TrimSuffix(jsonPart, "\n\n")
	
	var parsedMsg Message
	err = json.Unmarshal([]byte(jsonPart), &parsedMsg)
	require.NoError(t, err, "JSON content should be valid")
	require.Equal(t, msg, parsedMsg, "Parsed message should match original")
}

func TestSSESubscription_MultipleMessagesFormat(t *testing.T) {
	w := httptest.NewRecorder()
	sub := NewSSESubscription(context.Background(), w)

	// Send multiple messages
	msg1 := Message{
		Kind:      streamingtypes.MessageKindData,
		Data:      json.RawMessage(`"first message"`),
		CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
		Channel:   "user:123",
		Topic:     "ai",
	}
	
	msg2 := Message{
		Kind:      streamingtypes.MessageKindData,
		Data:      json.RawMessage(`"second message"`),
		CreatedAt: time.Now().Truncate(time.Millisecond).UTC(),
		Channel:   "user:123", 
		Topic:     "ai",
	}

	err := sub.WriteMessage(msg1)
	require.NoError(t, err)
	
	err = sub.WriteMessage(msg2)
	require.NoError(t, err)

	output := w.Body.String()
	
	// Split on double newlines to get individual SSE events
	events := strings.Split(output, "\n\n")
	require.Len(t, events, 3, "Should have 2 events plus empty string after final \\n\\n") // 2 events + 1 empty string after final \n\n
	
	// Verify first event
	require.True(t, strings.HasPrefix(events[0], "data: "), "First event should start with 'data: '")
	jsonPart1 := strings.TrimPrefix(events[0], "data: ")
	var parsedMsg1 Message
	err = json.Unmarshal([]byte(jsonPart1), &parsedMsg1)
	require.NoError(t, err)
	require.Equal(t, msg1, parsedMsg1)
	
	// Verify second event  
	require.True(t, strings.HasPrefix(events[1], "data: "), "Second event should start with 'data: '")
	jsonPart2 := strings.TrimPrefix(events[1], "data: ")
	var parsedMsg2 Message
	err = json.Unmarshal([]byte(jsonPart2), &parsedMsg2)
	require.NoError(t, err)
	require.Equal(t, msg2, parsedMsg2)
	
	// Last element should be empty (after final \n\n)
	require.Equal(t, "", events[2])
}