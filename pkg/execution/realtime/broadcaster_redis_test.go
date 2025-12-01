package realtime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
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
	s1 := NewInmemorySubscription(uuid.New(), func(b []byte) error {
		var m Message
		if err := json.Unmarshal(b, &m); err != nil {
			return err
		}
		l.Lock()
		m1 = append(m1, m)
		l.Unlock()
		return nil
	})
	s2 := NewInmemorySubscription(uuid.New(), func(b []byte) error {
		var m Message
		if err := json.Unmarshal(b, &m); err != nil {
			return err
		}
		l.Lock()
		m2 = append(m2, m)
		l.Unlock()
		return nil
	})

	// Create two messages with two separate topics.
	msg1 := streamingtypes.NewMessage(streamingtypes.MessageKindRun, "output")
	msg1.Channel = ulid.MustNew(ulid.Now(), rand.Reader).String()
	msg2 := streamingtypes.NewMessage(streamingtypes.MessageKindRun, "output")
	msg2.Channel = ulid.MustNew(ulid.Now(), rand.Reader).String()

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

func TestRedisBroadcasterWrite(t *testing.T) {
	ctx := context.Background()

	// Set up Redis server and clients
	r := miniredis.RunT(t)
	pubc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer pubc.Close()

	subc1, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer subc1.Close()

	subc2, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer subc2.Close()

	// Create two broadcasters
	b1 := NewRedisBroadcaster(pubc, subc1)
	b2 := NewRedisBroadcaster(pubc, subc2)

	// Track received raw data for each subscription
	var l sync.Mutex
	channel1Data := [][]byte{}
	channel2Data := [][]byte{}

	// Create subscriptions that collect raw bytes (for Write method)
	s1 := NewInmemorySubscription(uuid.New(), func(data []byte) error {
		l.Lock()
		channel1Data = append(channel1Data, append([]byte(nil), data...)) // Copy data
		l.Unlock()
		return nil
	})

	s2 := NewInmemorySubscription(uuid.New(), func(data []byte) error {
		l.Lock()
		channel2Data = append(channel2Data, append([]byte(nil), data...)) // Copy data
		l.Unlock()
		return nil
	})

	// Create topics for two different channels
	channel1 := "test-channel-1"
	channel2 := "test-channel-2"

	topic1 := Topic{
		Kind:    streamingtypes.TopicKindRun,
		Channel: channel1,
		Name:    "test-topic",
		EnvID:   uuid.New(),
	}

	topic2 := Topic{
		Kind:    streamingtypes.TopicKindRun,
		Channel: channel2,
		Name:    "test-topic",
		EnvID:   uuid.New(),
	}

	t.Run("Write method isolates channels correctly", func(t *testing.T) {
		// Subscribe s1 to channel1 via b1, s2 to channel2 via b2
		err := b1.Subscribe(ctx, s1, []Topic{topic1})
		require.NoError(t, err)

		err = b2.Subscribe(ctx, s2, []Topic{topic2})
		require.NoError(t, err)

		// Wait for Redis subscriptions to be established
		time.Sleep(100 * time.Millisecond)

		// Write data to channel1 - only s1 should receive it
		testData1 := []byte("Hello from channel 1")
		b1.Write(ctx, channel1, testData1)

		// Write data to channel2 - only s2 should receive it
		testData2 := []byte("Hello from channel 2")
		b2.Write(ctx, channel2, testData2)

		// Wait for data propagation via Redis
		assert.Eventually(t, func() bool {
			l.Lock()
			defer l.Unlock()
			return len(channel1Data) == 1 && len(channel2Data) == 1
		}, 5*time.Second, 50*time.Millisecond)

		l.Lock()
		// Assert s1 received only channel1 data
		require.Len(t, channel1Data, 1, "s1 should receive exactly one message")
		assert.Equal(t, testData1, channel1Data[0], "s1 should receive data from channel1")

		// Assert s2 received only channel2 data
		require.Len(t, channel2Data, 1, "s2 should receive exactly one message")
		assert.Equal(t, testData2, channel2Data[0], "s2 should receive data from channel2")
		l.Unlock()

		// Write more data to verify continued isolation
		testData3 := []byte("Second message to channel 1")
		b1.Write(ctx, channel1, testData3)

		// Wait for additional data
		assert.Eventually(t, func() bool {
			l.Lock()
			defer l.Unlock()
			return len(channel1Data) == 2 && len(channel2Data) == 1
		}, 5*time.Second, 50*time.Millisecond)

		l.Lock()
		// Verify isolation is maintained
		require.Len(t, channel1Data, 2, "s1 should have received 2 messages")
		require.Len(t, channel2Data, 1, "s2 should still have only 1 message")
		assert.Equal(t, testData3, channel1Data[1], "s1 should receive second message")
		l.Unlock()
	})
}
