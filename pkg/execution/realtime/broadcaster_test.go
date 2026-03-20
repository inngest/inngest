package realtime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	// testEventualTimeout is the max time to wait for async Redis delivery in tests.
	testEventualTimeout = 5 * time.Second
	// testEventualPoll is how often to check for async Redis delivery in tests.
	testEventualPoll = 5 * time.Millisecond
)

func TestBroadcaster(t *testing.T) {
	ctx := context.Background()
	b := newTestBroadcaster(t)

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
	msgCount := func() int {
		l.Lock()
		defer l.Unlock()
		return len(messages)
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
			// No subscribers, so nothing should be delivered even after a wait.
			time.Sleep(50 * time.Millisecond)
			require.Empty(t, messages)
		})

		t.Run("matching subscription", func(t *testing.T) {
			require.Equal(t, 1, len(msg.Topics()))

			err := b.Subscribe(ctx, sub, msg.Topics())
			require.NoError(t, err)

			b.Publish(ctx, msg)
			require.Eventually(t, func() bool { return msgCount() == 1 }, testEventualTimeout, testEventualPoll)

			l.Lock()
			require.Equal(t, msg, messages[0])
			l.Unlock()

			t.Run("subscribing twice on the same sub ID only sends one message", func(t *testing.T) {
				err := b.Subscribe(ctx, sub, msg.Topics())
				require.NoError(t, err)

				b.Publish(ctx, msg)
				require.Eventually(t, func() bool { return msgCount() == 2 }, testEventualTimeout, testEventualPoll)
			})
		})

		t.Run("removed subscription", func(t *testing.T) {
			err := b.CloseSubscription(ctx, id)
			require.NoError(t, err)

			b.Publish(ctx, msg)
			// No subscriber, so count should stay at 2.
			time.Sleep(50 * time.Millisecond)
			require.Equal(t, 2, msgCount())
		})

		t.Run("unsubscribing", func(t *testing.T) {
			b = newTestBroadcaster(t)
			l.Lock()
			messages = []Message{}
			l.Unlock()

			err := b.Subscribe(ctx, sub, msg.Topics())
			require.NoError(t, err)

			b.Publish(ctx, msg)
			require.Eventually(t, func() bool { return msgCount() == 1 }, testEventualTimeout, testEventualPoll)

			l.Lock()
			require.Equal(t, msg, messages[0])
			l.Unlock()

			err = b.Unsubscribe(ctx, sub.ID(), msg.Topics())
			require.NoError(t, err)
			b.Publish(ctx, msg)

			// No change in count
			time.Sleep(50 * time.Millisecond)
			require.Equal(t, 1, msgCount())
		})

		t.Run("many subscriptions", func(t *testing.T) {
			b = newTestBroadcaster(t)
			l.Lock()
			messages = []Message{}
			l.Unlock()

			count := 10

			for i := 0; i < count; i++ {
				sub := NewInmemorySubscription(uuid.New(), appender)
				err := b.Subscribe(ctx, sub, msg.Topics())
				require.NoError(t, err)
			}
			// Wait for the Redis subscriber goroutine to be established.
			// Only one is started (for the first subscriber to this topic).

			b.Publish(ctx, msg)
			// The message fans out once through Redis, then is delivered to
			// all 10 local subscribers. But the "unsubscribing" subtest left
			// `sub` unsubscribed from this broadcaster, so only the 10 new
			// subs receive it.
			require.Eventually(t, func() bool { return msgCount() == count }, testEventualTimeout, testEventualPoll)
		})

		t.Run("With failing writer", func(t *testing.T) {
			// This fails on the first write attempt, then retries. We should
			// always get a retry.
			b = newTestBroadcaster(t)
			l.Lock()
			messages = []Message{}
			l.Unlock()

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

			// Initially zero because the first write fails, then the Redis
			// roundtrip delivers and also fails, then retry succeeds.
			time.Sleep(50 * time.Millisecond)
			require.Equal(t, 0, msgCount())

			require.Eventually(t, func() bool { return msgCount() == 1 }, WriteRetryInterval+testEventualTimeout, testEventualPoll)

			l.Lock()
			require.Equal(t, msg, messages[0])
			l.Unlock()
		})
	})
}

// TestBroadcasterTopicLifecycle ensures that topics are created and removed
// based on subscription refcounts.
func TestBroadcasterTopicLifecycle(t *testing.T) {
	t.Run("topic created on first subscribe, removed on last unsubscribe", func(t *testing.T) {
		b := newTestBroadcaster(t).(*broadcaster)

		sub1 := NewInmemorySubscription(uuid.New(), nil)
		sub2 := NewInmemorySubscription(uuid.New(), nil)
		msg := Message{
			Kind:    streamingtypes.MessageKindData,
			Channel: "test",
			Topic:   "topic1",
		}
		topicHash := msg.Topics()[0].String()

		// 1. First subscribe -> topic created with refcount 1
		err := b.Subscribe(context.Background(), sub1, msg.Topics())
		require.NoError(t, err)
		b.l.RLock()
		ts, ok := b.topics[topicHash]
		require.True(t, ok)
		require.Equal(t, 1, ts.refCount)
		b.l.RUnlock()

		// 2. Second subscribe -> refcount incremented
		err = b.Subscribe(context.Background(), sub2, msg.Topics())
		require.NoError(t, err)
		b.l.RLock()
		ts = b.topics[topicHash]
		require.Equal(t, 2, ts.refCount)
		b.l.RUnlock()

		// 3. First unsubscribe -> refcount decremented, topic still exists
		err = b.Unsubscribe(context.Background(), sub1.ID(), msg.Topics())
		require.NoError(t, err)
		b.l.RLock()
		ts, ok = b.topics[topicHash]
		require.True(t, ok)
		require.Equal(t, 1, ts.refCount)
		b.l.RUnlock()

		// 4. Second unsubscribe -> topic removed
		err = b.Unsubscribe(context.Background(), sub2.ID(), msg.Topics())
		require.NoError(t, err)
		b.l.RLock()
		_, ok = b.topics[topicHash]
		require.False(t, ok, "Topic should be removed after last unsubscribe")
		b.l.RUnlock()
	})
}

func TestBroadcasterStream(t *testing.T) {
	ctx := context.Background()
	b := newTestBroadcaster(t)

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
	msgCount := func() int {
		l.Lock()
		defer l.Unlock()
		return len(messages)
	}
	streamCount := func() int {
		l.Lock()
		defer l.Unlock()
		return len(streams)
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
		require.Eventually(t, func() bool { return msgCount() == 1 }, testEventualTimeout, testEventualPoll)
		require.Equal(t, 0, streamCount())
	})

	t.Run("streaming data works", func(t *testing.T) {
		b.PublishChunk(ctx, msg, streamingtypes.ChunkFromMessage(msg, "a"))
		require.Eventually(t, func() bool { return streamCount() == 1 }, testEventualTimeout, testEventualPoll)
		require.Equal(t, 1, msgCount())

		l.Lock()
		require.Equal(t, Chunk{
			Kind:     string(streamingtypes.MessageKindDataStreamChunk),
			StreamID: "streamid123",
			Data:     "a",
		}, streams[0])
		l.Unlock()

		b.PublishChunk(ctx, msg, streamingtypes.ChunkFromMessage(msg, "b"))
		require.Eventually(t, func() bool { return streamCount() == 2 }, testEventualTimeout, testEventualPoll)
		require.Equal(t, 1, msgCount())

		l.Lock()
		require.Equal(t, Chunk{
			Kind:     string(streamingtypes.MessageKindDataStreamChunk),
			StreamID: "streamid123",
			Data:     "b",
		}, streams[1])
		l.Unlock()
	})

	// Publish a stream end.
	t.Run("stream end publish", func(t *testing.T) {
		msg.Kind = streamingtypes.MessageKindDataStreamEnd
		b.Publish(ctx, msg)
		require.Eventually(t, func() bool { return msgCount() == 2 }, testEventualTimeout, testEventualPoll)
	})
}

func TestBroadcasterClose(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	b := newTestBroadcasterWithOpts(t, BroadcasterOpts{
		ShutdownGracePeriod: 50 * time.Millisecond,
	})

	topic := Topic{
		Kind:    streamingtypes.TopicKindRun,
		EnvID:   uuid.New(),
		Channel: "close-test",
		Name:    "test",
	}

	var received []Message
	var mu sync.Mutex
	sub := NewInmemorySubscription(uuid.New(), func(data []byte) error {
		var m Message
		if err := json.Unmarshal(data, &m); err == nil {
			mu.Lock()
			received = append(received, m)
			mu.Unlock()
		}
		return nil
	})

	r.NoError(b.Subscribe(ctx, sub, []Topic{topic}))
	r.Equal(1, subCount(b))

	// Close the broadcaster.
	r.NoError(b.Close(ctx))

	// Should receive a closing message.
	r.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, m := range received {
			if m.Kind == streamingtypes.MessageKindClosing {
				return true
			}
		}
		return false
	}, testEventualTimeout, testEventualPoll)

	// New subscriptions should be rejected.
	err := b.Subscribe(ctx, NewInmemorySubscription(uuid.New(), nil), []Topic{topic})
	r.ErrorIs(err, ErrBroadcasterClosed)

	// Calling Close again should return ErrBroadcasterClosed.
	err = b.Close(ctx)
	r.ErrorIs(err, ErrBroadcasterClosed)

	// After the grace period, topic goroutines should be cancelled.
	bc := b.(*broadcaster)
	r.Eventually(func() bool {
		bc.redisMu.Lock()
		defer bc.redisMu.Unlock()
		return len(bc.topicCancelFuncs) == 0
	}, testEventualTimeout, testEventualPoll)
}
