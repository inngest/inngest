package realtime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"os"
	"os/exec"
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
		require.Equal(t, 1, len(m1))
		require.Equal(t, 1, len(m2))

		// The broadcaster sets the topic name on the outgoing message
		expectedMsg1 := msg1
		expectedMsg1.Topic = "$run"

		require.Equal(t, expectedMsg1, m1[0])
		require.Equal(t, expectedMsg1, m2[0])
		l.Unlock()

		// Publish message 2

		b1.Publish(ctx, msg2)

		require.Eventually(t, func() bool {
			l.Lock()
			defer l.Unlock()
			return len(m1) == 2 && len(m2) == 2
		}, time.Second, 5*time.Millisecond)

		expectedMsg2 := msg2
		expectedMsg2.Topic = "$run"

		require.Equal(t, expectedMsg2, m1[1])
		require.Equal(t, expectedMsg2, m2[1])
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
		Name:    streamingtypes.TopicNameStream,
		EnvID:   uuid.New(),
	}

	topic2 := Topic{
		Kind:    streamingtypes.TopicKindRun,
		Channel: channel2,
		Name:    streamingtypes.TopicNameStream,
		EnvID:   uuid.New(),
	}

	t.Run("Write method isolates channels correctly", func(t *testing.T) {
		// Subscribe s1 to channel1 via b1, s2 to channel2 via b2
		err := b1.Subscribe(ctx, s1, []Topic{topic1})
		require.NoError(t, err)

		err = b2.Subscribe(ctx, s2, []Topic{topic2})
		require.NoError(t, err)

		// Write data to channel1 - only s1 should receive it
		testData1 := []byte("Hello from channel 1")
		b1.Write(ctx, topic1.EnvID, channel1, testData1)

		// Write data to channel2 - only s2 should receive it
		testData2 := []byte("Hello from channel 2")
		b2.Write(ctx, topic2.EnvID, channel2, testData2)

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
		b1.Write(ctx, topic1.EnvID, channel1, testData3)

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

// TestRedisBroadcasterPublishChunk verifies that PublishChunk delivers chunks
// across two independent broadcaster instances sharing the same Redis.
func TestRedisBroadcasterPublishChunk(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	redis := miniredis.RunT(t)
	pubc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{redis.Addr()},
		DisableCache: true,
	})
	r.NoError(err)
	subc1, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{redis.Addr()},
		DisableCache: true,
	})
	r.NoError(err)
	subc2, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{redis.Addr()},
		DisableCache: true,
	})
	r.NoError(err)

	b1 := NewRedisBroadcaster(pubc, subc1)
	b2 := NewRedisBroadcaster(pubc, subc2)

	// Subscribe on b1.
	var mu sync.Mutex
	var chunks []Chunk
	sub := NewInmemorySubscription(uuid.New(), func(data []byte) error {
		var c Chunk
		if err := json.Unmarshal(data, &c); err != nil {
			return err
		}
		mu.Lock()
		chunks = append(chunks, c)
		mu.Unlock()
		return nil
	})

	msg := streamingtypes.NewMessage(streamingtypes.MessageKindDataStreamStart, "output")
	msg.Channel = ulid.MustNew(ulid.Now(), rand.Reader).String()
	msg.Data = json.RawMessage(`"stream-abc"`)

	r.NoError(b1.Subscribe(ctx, sub, msg.Topics()))

	// PublishChunk from b2 — no local subscribers on b2.
	chunk := streamingtypes.ChunkFromMessage(msg, "hello from b2")
	b2.PublishChunk(ctx, msg, chunk)

	r.Eventually(func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(chunks) == 1
	}, 5*time.Second, 5*time.Millisecond)

	mu.Lock()
	r.Equal("hello from b2", chunks[0].Data)
	r.Equal("stream-abc", chunks[0].StreamID)
	mu.Unlock()
}

// TestCrossProcessWrite verifies that `Write()` works across separate processes
// with fully isolated memory. The parent subscribes and spawns a child process
// that calls `Write()` with no local subscribers. The parent asserts directly
// on the data it receives through its subscription.
//
//  1. Parent starts miniredis, creates a broadcaster, subscribes.
//  2. Child creates its own broadcaster, calls `Write()`. No local subscribers.
//  3. Parent receives data through its subscription and asserts.
func TestCrossProcessWrite(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	if os.Getenv("CROSS_PROCESS_ROLE") == "publisher" {
		// Child process: run the publisher role and exit.
		crossProcessPublisher(t,
			os.Getenv("CROSS_PROCESS_REDIS_ADDR"),
			os.Getenv("CROSS_PROCESS_ENV_ID"),
			os.Getenv("CROSS_PROCESS_CHANNEL"),
		)
		return
	}

	redis := miniredis.RunT(t)
	envID := uuid.New()
	channel := "cross-process-channel"

	// Only needed because `NewRedisBroadcaster` requires a publisher client.
	// It won't be used.
	pubc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{redis.Addr()},
		DisableCache: true,
	})
	r.NoError(err)
	subc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{redis.Addr()},
		DisableCache: true,
	})
	r.NoError(err)
	b := NewRedisBroadcaster(pubc, subc)

	received := make(chan []byte, 1)
	sub := NewInmemorySubscription(uuid.New(), func(data []byte) error {
		received <- append([]byte(nil), data...)
		return nil
	})
	topic := Topic{
		Kind:    streamingtypes.TopicKindRun,
		EnvID:   envID,
		Channel: channel,
		Name:    streamingtypes.TopicNameStream,
	}
	r.NoError(b.Subscribe(ctx, sub, []Topic{topic}))

	// Spawn the publisher child, which is a separate process with its own
	// memory.
	pubCmd := exec.Command(os.Args[0], "-test.run=^TestCrossProcessWrite$", "-test.v", "-test.count=1")
	pubCmd.Env = append(os.Environ(),
		"CROSS_PROCESS_ROLE=publisher",
		"CROSS_PROCESS_REDIS_ADDR="+redis.Addr(),
		"CROSS_PROCESS_ENV_ID="+envID.String(),
		"CROSS_PROCESS_CHANNEL="+channel,
	)
	pubCmd.Stdout = os.Stdout
	pubCmd.Stderr = os.Stderr
	r.NoError(pubCmd.Start())

	// Assert directly on what we received.
	select {
	case data := <-received:
		r.Equal([]byte("cross-process payload"), data)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for data from child publisher")
	}

	r.NoError(pubCmd.Wait(), "publisher process failed")
}

// crossProcessPublisher is the child process for `TestCrossProcessWrite`. It
// creates its own broadcaster (with no subscribers) and calls `Write()`.
func crossProcessPublisher(t *testing.T, redisAddr, envIDStr, channel string) {
	r := require.New(t)
	ctx := context.Background()
	envID, err := uuid.Parse(envIDStr)
	r.NoError(err)

	pubc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{redisAddr},
		DisableCache: true,
	})
	r.NoError(err)

	// Only needed because `NewRedisBroadcaster` requires a subscriber client.
	// It won't be used.
	subc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{redisAddr},
		DisableCache: true,
	})
	r.NoError(err)

	b := NewRedisBroadcaster(pubc, subc)

	// Write from this process — no local subscribers exist here.
	b.Write(ctx, envID, channel, []byte("cross-process payload"))
}
