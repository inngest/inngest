package realtime

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisBroadcaster(t *testing.T) {
	var l sync.Mutex
	ctx := context.Background()

	// Test that two independent broadcasters can publish independently,
	// but still pick up each other's messages via redis' pub-sub.
	r := miniredis.RunT(t)
	pubc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	subc1, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	// Create a brand new redis client for the second broadcaster, to prevent any redis
	// client in-memory caching from giving false positives in tests.
	subc2, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)

	// Create two broadcasters and two subscribers, each of which store messages
	// in a separate slice.
	b1, b2 := NewRedisBroadcaster(pubc, subc1), NewRedisBroadcaster(pubc, subc2)
	m1, m2 := []Message{}, []Message{} // holds all messages
	s1 := NewInmemorySubscription(uuid.New(), func(m Message) error {
		l.Lock()
		m1 = append(m1, m)
		l.Unlock()
		return nil
	})
	s2 := NewInmemorySubscription(uuid.New(), func(m Message) error {
		l.Lock()
		m2 = append(m2, m)
		l.Unlock()
		return nil
	})

	// Create two messages with two separate topics.
	msg1 := NewMessage(MessageKindRun, "output")
	msg1.RunID = ulid.MustNew(ulid.Now(), rand.Reader)
	msg2 := NewMessage(MessageKindRun, "output")
	msg2.RunID = ulid.MustNew(ulid.Now(), rand.Reader)

	t.Run("publishing on b1 broadcasts on b2 subscriber", func(t *testing.T) {
		// Subscribing on msg1 and msg2 works, which tests multiplexing of many
		// redis pub-sub topics on a single channel.
		err := b1.Subscribe(ctx, s1, msg1.Topics())
		require.NoError(t, err)
		err = b1.Subscribe(ctx, s1, msg2.Topics())
		require.NoError(t, err)
		err = b2.Subscribe(ctx, s2, msg1.Topics())
		require.NoError(t, err)
		err = b2.Subscribe(ctx, s2, msg2.Topics())
		require.NoError(t, err)

		// Wait a short delay to ensure all subscriptions have been set up in Redis
		<-time.After(100 * time.Millisecond)

		// and publishing on b1 should also broadcast a message to the s2
		// subscriber via b2.
		b1.Publish(ctx, msg1)

		assert.Eventually(t, func() bool {
			l.Lock()
			defer l.Unlock()
			return len(m1) == 1 && len(m2) == 1
		}, 10*time.Second, 5*time.Millisecond)

		l.Lock()
		fmt.Printf("m1: %d, m2: %d\n", len(m1), len(m2))

		require.Equal(t, 1, len(m1))
		require.Equal(t, 1, len(m2))
		require.Equal(t, msg1, m1[0])
		require.Equal(t, msg1, m2[0])
		l.Unlock()

		// Publish message 2

		b1.Publish(ctx, msg2)

		require.Eventually(t, func() bool {
			l.Lock()
			defer l.Unlock()
			return len(m1) == 2 && len(m2) == 2
		}, time.Second, 5*time.Millisecond)

		require.Equal(t, msg2, m1[1])
		require.Equal(t, msg2, m2[1])
	})
}
