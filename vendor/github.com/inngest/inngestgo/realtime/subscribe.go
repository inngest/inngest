package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coder/websocket"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
)

type Message = streamingtypes.Message
type Topic = streamingtypes.Topic
type Chunk = streamingtypes.Chunk

var (
	DefaultSubscribeURL = "https://api.inngest.com/v1/realtime/connect"
)

// Subscribe subscribes to a given set of channels and topics as granted by
// the current token.
func Subscribe(ctx context.Context, token string) (chan StreamItem, error) {
	return SubscribeWithURL(ctx, DefaultSubscribeURL, token)
}

func SubscribeWithURL(ctx context.Context, url, token string) (chan StreamItem, error) {
	c, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": []string{"Bearer " + token},
		},
	})
	if err != nil {
		return nil, err
	}

	sender := make(chan StreamItem)

	go func() {
		for {
			if ctx.Err() != nil {
				close(sender)
				return
			}

			_, resp, err := c.Read(ctx)
			if isWebsocketClosed(err) {
				close(sender)
				return
			}
			if err != nil {
				_ = c.CloseNow()
				sender <- StreamItem{err: err}
				close(sender)
				return
			}

			// XXX: Check to see if this is a message or a stream.  The only messages
			// sent via our protocol are either JSON objects or streamed data.
			//
			// Therefore, we check the first character of the message to check the type.
			if len(resp) == 0 {
				continue
			}

			// Check to see if this is of kind "chunk".  If so, we know that the
			// this is a chunk within a streaming set of messages.
			kinder := msgKind{}
			if err := json.Unmarshal(resp, &kinder); err != nil {
				sender <- StreamItem{err: fmt.Errorf("error unmarshalling received data: %w", err)}
				continue
			}

			switch kinder.Kind {
			case string(streamingtypes.MessageKindDataStreamChunk):
				// Check to see if this is of kind "chunk".  If so, we know that the
				// this is a chunk within a streaming set of messages.
				chunk := Chunk{}
				if err := json.Unmarshal(resp, &chunk); err != nil {
					sender <- StreamItem{err: fmt.Errorf("error unmarshalling chunk: %w", err)}
					continue
				}
				sender <- StreamItem{chunk: &chunk}
			default:
				// Check to see if this is of kind "chunk".  If so, we know that the
				// this is a chunk within a streaming set of messages.
				msg := Message{}
				if err := json.Unmarshal(resp, &msg); err != nil {
					sender <- StreamItem{err: fmt.Errorf("error unmarshalling message: %w", err)}
					continue
				}
				sender <- StreamItem{message: &msg}
			}

		}
	}()

	return sender, nil
}

type msgKind struct {
	Kind string `json:"kind"`
}

type StreamKind string

const (
	StreamMessage = StreamKind("message")
	StreamChunk   = StreamKind("chunk")
	StreamError   = StreamKind("error")
)

type StreamItem struct {
	message *Message
	chunk   *Chunk
	err     error
}

func (r StreamItem) Kind() StreamKind {
	if r.IsChunk() {
		return StreamChunk
	}
	if r.IsMessage() {
		return StreamMessage
	}
	return StreamError
}

func (r StreamItem) IsMessage() bool {
	return r.message != nil
}

func (r StreamItem) Message() Message {
	return *r.message
}

func (r StreamItem) IsChunk() bool {
	return r.chunk != nil
}

func (r StreamItem) Chunk() Chunk {
	return *r.chunk
}

func (r StreamItem) IsErr() bool {
	return r.err != nil
}

func (r StreamItem) Err() error {
	return r.err
}

func isWebsocketClosed(err error) bool {
	if err == nil {
		return false
	}
	if websocket.CloseStatus(err) != -1 {
		return true
	}
	if err.Error() == "failed to get reader: use of closed network connection" {
		return true
	}
	return false
}
