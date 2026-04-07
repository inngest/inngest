package streamingtypes

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChunkFromMessage_JSONStringStreamID(t *testing.T) {
	r := require.New(t)
	msg := Message{
		Kind: MessageKindDataStreamStart,
		Data: json.RawMessage(`"abc123"`),
	}
	chunk := ChunkFromMessage(msg, "hello")
	r.Equal("abc123", chunk.StreamID, "should unquote the JSON string")
	r.Equal("hello", chunk.Data)
}

func TestChunkFromMessage_RawBytesStreamID(t *testing.T) {
	r := require.New(t)

	// Backward compat: raw bytes that are not valid JSON strings.
	msg := Message{
		Kind: MessageKindDataStreamStart,
		Data: json.RawMessage("raw-stream-id"),
	}
	chunk := ChunkFromMessage(msg, "world")
	r.Equal("raw-stream-id", chunk.StreamID, "should fall back to raw bytes")
	r.Equal("world", chunk.Data)
}
