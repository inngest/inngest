package pubsub

import (
	"context"
	"encoding/json"
	"time"
)

// Message represents an event sent across the pub/sub system.
type Message struct {
	Name      string         `json:"name"`
	Version   int            `json:"v"`
	Data      string         `json:"data"`
	Timestamp time.Time      `json:"ts"`
	Metadata  map[string]any `json:"meta,omitempty"`
}

func (m Message) Encode() ([]byte, error) {
	// TODO: Let's NOT use JSON, please.
	return json.Marshal(m)
}

func (m *Message) Decode(byt []byte) error {
	return json.Unmarshal(byt, m)
}

// PerformFunc is called by a subscription when a new message is received on the given
// subscription topic.
//
// These functions are meant to be short-lived (eg. completed within seconds), and
// do not heartbeat.
type PerformFunc func(context.Context, Message) error

// Publisher publishes an event to be consumed by one or more subscribers.
type Publisher interface {
	Publish(ctx context.Context, topic string, m Message) error
}

// Subscriber subscribes to a topic to consume events published by a Publisher.
type Subscriber interface {
	// Subscribe subscribes to the given topic, handling one message at a time
	Subscribe(ctx context.Context, topic string, handler PerformFunc) error

	// SubscribeN subscribes to the given topic, handling N messages at a time
	SubscribeN(ctx context.Context, topic string, handler PerformFunc, concurrency int64) error
}

type PublishSubscriber interface {
	Publisher
	Subscriber
}
