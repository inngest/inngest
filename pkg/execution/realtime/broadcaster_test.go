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
	appender := func(m Message) error {
		l.Lock()
		messages = append(messages, m)
		l.Unlock()
		return nil
	}

	sub := NewInmemorySubscription(id, appender)

	t.Run("broadcasting", func(t *testing.T) {
		msg := Message{
			Kind:  MessageKindRun,
			Data:  json.RawMessage(`"output"`),
			RunID: ulid.MustNew(ulid.Now(), rand.Reader),
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
			sub := NewInmemorySubscription(id, func(m Message) error {
				if failed {
					failed = false
					err := appender(m)
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

// TestBroadcasterConds ensures that the sync.Cond mechanisms work for subscribing and
// unsubscribing
func TestBroadcasterConds(t *testing.T) {
	t.Run("single subscriber", func(t *testing.T) {
		var (
			ctx = context.Background()
			b   = NewInProcessBroadcaster().(*broadcaster)
			sub = NewInmemorySubscription(uuid.New(), nil)
			msg = Message{
				Kind:  MessageKindRun,
				Data:  json.RawMessage(`"output"`),
				RunID: ulid.MustNew(ulid.Now(), rand.Reader),
			}
			unsubCalled int32
			wg          sync.WaitGroup
		)

		wg.Add(1)
		err := b.subscribe(
			ctx,
			sub,
			msg.Topics(),
			func(ctx context.Context, topic Topic) {
				require.Nil(t, ctx.Err())
				require.Equal(t, msg.Topics()[0], topic)

				// We should have a closed ctx
				<-time.After(20 * time.Millisecond)

				require.NotNil(t, ctx.Err(), "expected ctx to be cancelled")
				wg.Done()
			},
			func(ctx context.Context, t Topic) {
				atomic.AddInt32(&unsubCalled, 1)
			},
		)

		<-time.After(10 * time.Millisecond)

		require.NoError(t, err)
		err = b.Unsubscribe(ctx, sub.ID(), msg.Topics())
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&unsubCalled) >= int32(1)
		}, time.Second, time.Millisecond, "unsubscribe should be called")
		wg.Wait()
	})

	t.Run("single subscriber with same topic subscriptions", func(t *testing.T) {
		var (
			ctx = context.Background()
			b   = NewInProcessBroadcaster().(*broadcaster)
			sub = NewInmemorySubscription(uuid.New(), nil)
			msg = Message{
				Kind:  MessageKindRun,
				Data:  json.RawMessage(`"output"`),
				RunID: ulid.MustNew(ulid.Now(), rand.Reader),
			}
			unsubCalled int32
			wg          sync.WaitGroup
		)

		// This asserts that the subscribe and unsubscribe callbacks work, even if
		// the actual underlying subscription<>topic pair has been deduplicated.
		for i := 0; i < 10; i++ {
			wg.Add(1)
			err := b.subscribe(
				ctx,
				sub,
				msg.Topics(),
				func(ctx context.Context, topic Topic) {
					require.Nil(t, ctx.Err())
					require.Equal(t, msg.Topics()[0], topic)

					// We should have a closed ctx
					<-time.After(20 * time.Millisecond)

					require.NotNil(t, ctx.Err(), "expected ctx to be cancelled")
					wg.Done()
				},
				func(ctx context.Context, t Topic) {
					atomic.AddInt32(&unsubCalled, 1)
				},
			)
			require.NoError(t, err)
		}

		<-time.After(10 * time.Millisecond)

		err := b.Unsubscribe(ctx, sub.ID(), msg.Topics())
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&unsubCalled) == int32(10)
		}, time.Second, time.Millisecond, "unsubscribe should be called")
		wg.Wait()
	})

}
