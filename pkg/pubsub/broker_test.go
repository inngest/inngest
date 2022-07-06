package pubsub

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/stretchr/testify/require"
)

const testTopic = "test"

func testbroker(t *testing.T) *broker {
	t.Helper()
	ctx := context.Background()

	// Configure the broker to use an in-memory implementation.
	c := &config.MessagingService{}
	c.Set(config.InMemoryMessaging{Topic: testTopic})
	ps, err := NewPublishSubscriber(ctx, c)
	require.NoError(t, err)

	// Create the test topic.
	err = ps.Publish(ctx, Message{}, testTopic)
	return ps.(*broker)
}

func TestBasicPublishSubscribe(t *testing.T) {
	var err error
	b := testbroker(t)
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
		err = b.Subscribe(ctx, testTopic, run)
		require.NoError(t, err)
	}()

	// Wait for the subscription.
	// XXX: Let's create a way to assert that this is done without timing.
	<-time.After(time.Second)

	err = b.Publish(ctx, sent, testTopic)
	require.NoError(t, err)

	select {
	case <-time.After(1 * time.Second):
		t.Fail()
	case received := <-ok:
		require.EqualValues(t, sent, received)
	}
}
