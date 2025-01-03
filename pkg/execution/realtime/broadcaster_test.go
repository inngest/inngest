package realtime

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
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
			Data:  "output",
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
		})

		t.Run("removed subscription", func(t *testing.T) {
			err := b.CloseSubscription(ctx, id)
			require.NoError(t, err)

			b.Publish(ctx, msg)
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
					appender(m)
					return nil
				}
				failed = true
				return fmt.Errorf("intermittent failure")
			})

			err := b.Subscribe(ctx, sub, msg.Topics())
			require.NoError(t, err)

			b.Publish(ctx, msg)

			require.Equal(t, 0, len(messages))

			<-time.After(WriteRetryInterval + (5 * time.Millisecond))

			require.Equal(t, 1, len(messages))
			require.Equal(t, msg, messages[0])
		})
	})
}
