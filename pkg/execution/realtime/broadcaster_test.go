package realtime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestBroadcaster(t *testing.T) {
	ctx := context.Background()
	b := NewInProcessBroadcaster()

	var (
		id       = uuid.New()
		messages = []Message{}
		l        sync.Mutex
	)
	appender := func(b []byte) error {
		var m Message
		if err := json.Unmarshal(b, &m); err != nil {
			return err
		}
		l.Lock()
		messages = append(messages, m)
		l.Unlock()
		return nil
	}

	sub := NewInmemorySubscription(id, appender)

	t.Run("broadcasting", func(t *testing.T) {
		msg := Message{
			Kind:    streamingtypes.MessageKindData,
			Data:    json.RawMessage(`"output"`),
			Channel: ulid.MustNew(ulid.Now(), rand.Reader).String(),
			Topic:   "sometopic",
		}

		t.Run("no subscriptions", func(t *testing.T) {
			b.Publish(ctx, msg)
			require.Empty(t, messages)
		})

		t.Run("matching subscription", func(t *testing.T) {
			require.Equal(t, 1, len(msg.Topics()))

			err := b.Subscribe(ctx, sub, msg.Topics())
			require.NoError(t, err)

			b.Publish(ctx, msg)
			require.Equal(t, 1, len(messages))
			require.Equal(t, msg, messages[0])

			t.Run("subscribing twice on the same sub ID only sends one message", func(t *testing.T) {
				err := b.Subscribe(ctx, sub, msg.Topics())
				require.NoError(t, err)

				b.Publish(ctx, msg)
				require.Equal(t, 2, len(messages))
			})
		})

		t.Run("removed subscription", func(t *testing.T) {
			err := b.CloseSubscription(ctx, id)
			require.NoError(t, err)

			b.Publish(ctx, msg)
			require.Equal(t, 2, len(messages))
			require.Equal(t, msg, messages[0])
		})

		t.Run("unsubscribing", func(t *testing.T) {
			b = NewInProcessBroadcaster()
			messages = []Message{}

			err := b.Subscribe(ctx, sub, msg.Topics())
			require.NoError(t, err)

			b.Publish(ctx, msg)
			require.Equal(t, 1, len(messages))
			require.Equal(t, msg, messages[0])

			err = b.Unsubscribe(ctx, sub.ID(), msg.Topics())
			require.NoError(t, err)
			b.Publish(ctx, msg)

			// No change in count
			require.Equal(t, 1, len(messages))
			require.Equal(t, msg, messages[0])
		})

		t.Run("many subscriptions", func(t *testing.T) {
			messages = []Message{}

			count := 10

			for i := 0; i < count; i++ {
				sub := NewInmemorySubscription(uuid.New(), appender)
				err := b.Subscribe(ctx, sub, msg.Topics())
				require.NoError(t, err)
			}

			b.Publish(ctx, msg)
			require.Equal(t, count, len(messages))
			require.Equal(t, msg, messages[0])
		})

		t.Run("With failing writer", func(t *testing.T) {
			// This fails on the first write attempt, then retries.  We should
			// always get a retry.
			b = NewInProcessBroadcaster()
			messages = []Message{}

			failed := false
			sub := NewInmemorySubscription(id, func(b []byte) error {
				if failed {
					failed = false
					err := appender(b)
					require.NoError(t, err)
					return nil
				}
				failed = true
				return fmt.Errorf("intermittent failure")
			})

			err := b.Subscribe(ctx, sub, msg.Topics())
			require.NoError(t, err)

			b.Publish(ctx, msg)

			l.Lock()
			require.Equal(t, 0, len(messages))
			l.Unlock()

			<-time.After(WriteRetryInterval + (5 * time.Millisecond))

			l.Lock()
			require.Equal(t, 1, len(messages))
			l.Unlock()
			require.Equal(t, msg, messages[0])
		})
	})
}

// TestBroadcasterHooks ensures that the TopicStart and TopicStop hooks work correctly
func TestBroadcasterHooks(t *testing.T) {
	t.Run("Lifecycle hooks", func(t *testing.T) {
		b := NewInProcessBroadcaster()

		var startCalled, stopCalled int32

		b.TopicStart = func(ctx context.Context, t Topic) error {
			atomic.AddInt32(&startCalled, 1)
			return nil
		}
		b.TopicStop = func(ctx context.Context, t Topic) error {
			atomic.AddInt32(&stopCalled, 1)
			return nil
		}

		sub1 := NewInmemorySubscription(uuid.New(), nil)
		sub2 := NewInmemorySubscription(uuid.New(), nil)
		msg := Message{
			Kind:    streamingtypes.MessageKindData,
			Channel: "test",
			Topic:   "topic1",
		}

		// 1. First subscribe -> Start called
		err := b.Subscribe(context.Background(), sub1, msg.Topics())
		require.NoError(t, err)
		require.Equal(t, int32(1), atomic.LoadInt32(&startCalled))
		require.Equal(t, int32(0), atomic.LoadInt32(&stopCalled))

		// 2. Second subscribe -> Start NOT called
		err = b.Subscribe(context.Background(), sub2, msg.Topics())
		require.NoError(t, err)
		require.Equal(t, int32(1), atomic.LoadInt32(&startCalled))

		// 3. First unsubscribe -> Stop NOT called
		err = b.Unsubscribe(context.Background(), sub1.ID(), msg.Topics())
		require.NoError(t, err)
		require.Equal(t, int32(0), atomic.LoadInt32(&stopCalled))

		// 4. Second unsubscribe -> Stop called
		err = b.Unsubscribe(context.Background(), sub2.ID(), msg.Topics())
		require.NoError(t, err)
		require.Equal(t, int32(1), atomic.LoadInt32(&stopCalled))
	})
}

func TestBroadcasterStream(t *testing.T) {
	ctx := context.Background()
	b := NewInProcessBroadcaster()

	var (
		id       = uuid.New()
		messages = []Message{}
		streams  = []Chunk{}
		l        sync.Mutex
	)
	appender := func(b []byte) error {
		// First, check the "kind" field to determine the type
		var kindCheck struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(b, &kindCheck); err != nil {
			return err
		}

		switch kindCheck.Kind {
		case "chunk":
			var c Chunk
			if err := json.Unmarshal(b, &c); err != nil {
				return err
			}
			l.Lock()
			streams = append(streams, c)
			l.Unlock()
		default:
			var m Message
			if err := json.Unmarshal(b, &m); err != nil {
				return err
			}
			l.Lock()
			messages = append(messages, m)
			l.Unlock()
		}
		return nil
	}

	sub := NewInmemorySubscription(id, appender)

	// This is the message we'll publish.
	msg := Message{
		Kind:    streamingtypes.MessageKindDataStreamStart,
		Channel: "user:123",
		Topic:   "openai",
		Data:    json.RawMessage(`"streamid123"`),
	}

	err := b.Subscribe(ctx, sub, msg.Topics())
	require.NoError(t, err)

	// Publish a stream start.
	t.Run("stream starts publish", func(t *testing.T) {
		b.Publish(ctx, msg)
		require.EqualValues(t, 1, len(messages), messages)
		require.EqualValues(t, 0, len(streams))
	})

	t.Run("streaming data works", func(t *testing.T) {
		b.PublishChunk(ctx, msg, streamingtypes.ChunkFromMessage(msg, "a"))
		require.EqualValues(t, 1, len(messages), messages)
		require.EqualValues(t, 1, len(streams), streams)
		require.Equal(t, Chunk{
			Kind:     string(streamingtypes.MessageKindDataStreamChunk),
			StreamID: `"streamid123"`,
			Data:     "a",
		}, streams[0])

		b.PublishChunk(ctx, msg, streamingtypes.ChunkFromMessage(msg, "b"))
		require.EqualValues(t, 1, len(messages), messages)
		require.EqualValues(t, 2, len(streams), streams)
		require.Equal(t, Chunk{
			Kind:     string(streamingtypes.MessageKindDataStreamChunk),
			StreamID: `"streamid123"`,
			Data:     "b",
		}, streams[1])
	})

	// Publish a stream start.
	t.Run("stream starts publish", func(t *testing.T) {
		msg.Kind = streamingtypes.MessageKindDataStreamEnd
		b.Publish(ctx, msg)
		require.EqualValues(t, 2, len(messages), messages)
	})
}
