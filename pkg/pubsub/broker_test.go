package pubsub

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/stretchr/testify/require"
)

func testbroker(t *testing.T) (*broker, string) {
	t.Helper()
	ctx := context.Background()

	topic := uuid.New().String()

	// Configure the broker to use an in-memory implementation.
	c := &config.MessagingService{}
	c.Set(config.InMemoryMessaging{Topic: topic})
	ps, err := NewPublishSubscriber(ctx, c)
	require.NoError(t, err)

	// Create the test topic.
	err = ps.Publish(ctx, topic, Message{})
	require.NoError(t, err)

	return ps.(*broker), topic
}

func TestBasicPublishSubscribe(t *testing.T) {
	var err error
	b, topic := testbroker(t)
	ctx := context.Background()

	sent := Message{Name: "basic", Data: json.RawMessage("{}")}
	ok := make(chan Message)

	go func() {
		// Subscribe in a blocking fashion.
		run := func(c context.Context, m Message) error {
			require.EqualValues(t, sent, m)
			ok <- m
			return nil
		}
		err = b.Subscribe(ctx, topic, run)
		require.NoError(t, err)
	}()

	// Wait for the subscription.
	// XXX: Let's create a way to assert that this is done without timing.
	<-time.After(time.Second)

	err = b.Publish(ctx, topic, sent)
	require.NoError(t, err)

	select {
	case <-time.After(1 * time.Second):
		t.Fail()
	case received := <-ok:
		require.EqualValues(t, sent, received)
	}
}

func TestSubscribeN(t *testing.T) {
	var err error
	b, topic := testbroker(t)
	ctx := context.Background()

	sent := Message{Name: "basic", Data: json.RawMessage("{}")}
	ok := make(chan Message)

	// i stores how often the run function has been invoked.
	var i int32
	var paused int32
	go func() {
		// Subscribe in a blocking fashion.
		run := func(c context.Context, m Message) error {
			atomic.AddInt32(&i, 1)
			require.EqualValues(t, sent, m)
			ok <- m
			for atomic.LoadInt32(&paused) == 0 {
				// Do not allow these funcs to finish.
				<-time.After(50 * time.Millisecond)
			}
			return nil
		}
		err = b.SubscribeN(ctx, topic, run, 10)
		require.NoError(t, err)
	}()

	// Wait for the subscription.
	// XXX: Let's create a way to assert that this is done without timing.
	<-time.After(time.Second)

	// Send 10 events.
	for i := 0; i < 10; i++ {
		err = b.Publish(ctx, topic, sent)
		require.NoError(t, err)
	}

	// Ensure we receive all 10 events within a second.
	timer := time.After(time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-timer:
			// We should receive 10 events within a second.
			t.Fail()
		case received := <-ok:
			require.EqualValues(t, sent, received)
		}
	}

	require.EqualValues(t, 10, i, "Expected run to be invoked 10 times")

	// Sending 10 more should block.
	for i := 0; i < 10; i++ {
		err = b.Publish(ctx, topic, sent)
		require.NoError(t, err)
	}

	<-time.After(2 * time.Second)
	require.EqualValues(t, 10, atomic.LoadInt32(&i), "Expected run to be invoked 10 times, not running for new events while concurrency is blocked")

	// Unpause the run.
	atomic.AddInt32(&paused, 1)

	<-time.After(time.Second)

	// Now they should work, as the first 10 were completed.
	require.EqualValues(t, 20, atomic.LoadInt32(&i), "Expected run to be invoked 20 times after concurrent capacity was freed")
}

func TestCancellation(t *testing.T) {
	var err error
	b, topic := testbroker(t)
	ctx, cancel := context.WithCancel(context.Background())

	sent := Message{Name: "basic", Data: json.RawMessage("{}")}
	ok := make(chan Message)

	// Changed when Subscribe finishes in the goroutine.
	var complete int32

	// i stores how often the run function has been invoked.
	var i int32

	go func() {
		run := func(c context.Context, m Message) error {
			atomic.AddInt32(&i, 1)
			require.EqualValues(t, sent, m)
			ok <- m
			return nil
		}
		err = b.Subscribe(ctx, topic, run)
		require.NoError(t, err)
		atomic.AddInt32(&complete, 1)
	}()

	// Wait for the subscription.
	// XXX: Let's create a way to assert that this is done without timing.
	<-time.After(time.Second)

	err = b.Publish(ctx, topic, sent)
	require.NoError(t, err)

	select {
	case <-time.After(1 * time.Second):
		t.Fail()
	case received := <-ok:
		require.EqualValues(t, sent, received)
	}

	require.EqualValues(t, 1, i)

	// Cancel the context, which should close the function.
	cancel()

	// The Receive batcher... batches, and we need to wait for this.
	now := time.Now()
	for atomic.LoadInt32(&complete) != 1 && time.Until(now) > (-10*time.Second) {
		<-time.After(50 * time.Millisecond)
	}
	require.EqualValues(t, 1, atomic.LoadInt32(&complete), "Expected subscription to stop within 10 seconds")

	// Send another event, and we should not receive nor increase the counter.
	err = b.Publish(context.Background(), topic, sent)
	require.NoError(t, err)

	<-time.After(time.Second)
	require.EqualValues(t, 1, i)
}
