package realtime

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

type (
	// MessageKind represents the type of data in the message, eg. whether
	// this is a step output, custom data, a run result, etc.
	MessageKind string
	// TopicKind indicates whether the subscribed topic is for an event or
	// run.  This allows us to move to pusher-style event forwarding in
	// the future.  Note that this is not the same as topic names.
	TopicKind string
)

const (
	// MessageKindStep represents step output
	MessageKindStep = MessageKind("step")
	// MessageKindRun represents a run's return value
	MessageKindRun = MessageKind("run")
	// MessageKindData represents misc data published on a custom run channel
	MessageKindData = MessageKind("data")
	// MessageKindEvent represents event data
	MessageKindEvent = MessageKind("event")
	// MessageKindClosing is a message kind sent when the server is closing the
	// realtime connection.  The subscriber should attempt to reconnect immediately,
	// as the broadcaster will stop broadcasting on the current connection within 5
	// minutes.
	MessageKindClosing = MessageKind("closing")

	// TopicKindRun indicates a topic for run data, eg. step output, run results,
	// or arbitrary data published within a run.
	TopicKindRun = TopicKind("run")
	// TopicKindEvent indicates a topic subscribed to all events, eg. pusher-style
	// pub-sub broadcasting.
	TopicKindEvent = TopicKind("event")

	// TopicStep represents a channel for step output data.
	// When subscribing to this channel across a run ID, the user can retrieve
	// all step output.  Note that this may be a security concern;  some step outputs
	// could be considered private.  To this effect, we allow people to subscribe to
	// specific step outputs or custom channels.
	TopicNameStep = "$step"
	// TopicNameRun represents a topic for the run's result.
	TopicNameRun = "$run"
)

// Publisher accepts messages from other services (eg. the executor, or the event API) and
// publishes messages to any subscribers.
type Publisher interface {
	// Publish publishes a message to any realtime subscribers.
	//
	// Note that this returns no error;  we expect that the publisher retries
	// internally and/or handles durability of the message.  Once Publish is called
	// the caller is no longer responsible for the lifetime of the message.  This
	// simplifies all caller code.
	Publish(ctx context.Context, m Message)
}

// Broadcaster manages all subscriptions to channels, and handles the forwarding of
// messages to each subscription
type Broadcaster interface {
	// Publish writes a given message to all subscriptions for the given Message
	Publisher

	// Subscribe adds a new authenticated Subscription.  Note that if the subscription
	// currently exists, the current channels will be *added to* the subscribed set.
	Subscribe(ctx context.Context, s Subscription, topics []Topic) error

	// CloseSubscription closes a subscription, removing it from the broadcaster
	// and stopping any messages from being published.  This terminates the subscription,
	// unsubscribing it from all topics.
	CloseSubscription(ctx context.Context, subscriptionID uuid.UUID) error

	// Close terminates the Broadcaster, which prevents any new Subscribe calls from
	// succeeding.  Note that this will terminate all Subscriptions after a grace period.
	//
	// Any acrtive subscribers receive "closing" notifications to resubscribe to another
	// broadcaster service.
	Close(context.Context) error
}

// Subscription represents a subscription to a specific set of channels.
type Subscription interface {
	// ID returns a unique ID for the given subscription
	ID() uuid.UUID

	// SendKeepalive is called by the broadcaster to keep the current connection alive.  This
	// may be a noop, depending on the implementation.  Note that keepalives are sent every
	// 30 seconds - this is not implementation specific.
	//
	// If SendKeepalive fails consecutively, the subscription will be closed.
	SendKeepalive() error

	// Writer allows the writing of messages to the particular subscription.  This is
	// implementation agnostic;  messages may be written via websockets or HTTP connections
	// for server-sent-events.
	//
	// Note that each subscription implementation may write different formats of a Message,
	// so this cannot fulfil io.Writer.
	WriteMessage(m Message) error

	// Closer closes the current subscription immediately, terminating any active connections.
	io.Closer
}

// Topic represents a topic for a message.  This is used for publishing and subscribing.
// Each message is published to one or more topics.
type Topic struct {
	// Kind represents the topic kind, ie. whether this topic is for events or run data.
	Kind TopicKind
	// RunID represents the run that this topic represents, if this is a
	// topic for a run.
	RunID ulid.ULID
	// EnvID represents the environment ID that this topic is subscribed to.  This
	// must always be present for both run and event topics.
	EnvID uuid.UUID
	// Name represents a topic name, such as "$step", "$result", "step-name",
	// or eg. "api/event.name".
	Name string

	// TODO: Implement event pub/sub and realtime message filtering.
	// Expression is used to filter messages such as events, eg "event.data.value > 500".
	// Expression *string
}

func (t Topic) String() string {
	switch t.Kind {
	case TopicKindRun:
		return fmt.Sprintf("%s.%s.%s", t.EnvID, t.RunID, t.Name)
	case TopicKindEvent:
		return fmt.Sprintf("%s.%s", t.EnvID, t.Name)
	}

	return fmt.Sprintf("%s.%s", t.EnvID, t.Name)
}

// Message represents a single message sent on realtime topics.
type Message struct {
	// Kind represents the message kind.
	Kind MessageKind `json:"kind"`
	// Data represents the data in the message.
	Data any `json:"data"`

	// FnID is the function ID that this message is related to.
	FnID uuid.UUID `json:"fn_id,omitempty,omitzero"`
	// FnSlug is the function slug that this message is related to.
	FnSlug string `json:"fn_slug,omitempty,omitzero"`
	// RunID is the run ID that this message is related to.
	RunID ulid.ULID `json:"run_id,omitempty,omitzero"`
	// CreatedAt is the time that this message was created.
	CreatedAt time.Time `json:"created_at"`
	// TopicNames represents the custom channels that this message should be broadcast
	// on.  For steps, this must include the unhashed step ID.  For custom broadcasts,
	// this is the chosen channel name in the SDK.
	TopicNames []string
}

// Topics returns all topics for the given message.
func (m Message) Topics() []Topic {
	switch m.Kind {
	case MessageKindStep:
		// This message is a step output.
		topics := make([]Topic, len(m.TopicNames)+1)

		// Always publish step outputs to the "$step" topic, alongside
		// the topic names within the message (which includes the step name)
		topics[0] = Topic{
			Kind:  TopicKindRun,
			Name:  TopicNameStep,
			RunID: m.RunID,
		}

		for n, v := range m.TopicNames {
			topics[n+1] = Topic{
				Kind:  TopicKindRun,
				RunID: m.RunID,
				Name:  v,
			}
		}

		return topics
	case MessageKindRun:
		// This message is a run output.
		topics := make([]Topic, len(m.TopicNames)+1)

		// Always publish step outputs to the "$step" topic, alongside
		// the topic names within the message (which includes the step name)
		topics[0] = Topic{
			Kind:  TopicKindRun,
			Name:  TopicNameRun,
			RunID: m.RunID,
		}

		for n, v := range m.TopicNames {
			topics[n+1] = Topic{
				Kind:  TopicKindRun,
				RunID: m.RunID,
				Name:  v,
			}
		}

		return topics
	}

	// Default to topic kinds of Run
	topics := make([]Topic, len(m.TopicNames))
	for n, v := range m.TopicNames {
		topics[n+1] = Topic{
			Kind:  TopicKindRun,
			RunID: m.RunID,
			Name:  v,
		}
	}
	return topics
}
