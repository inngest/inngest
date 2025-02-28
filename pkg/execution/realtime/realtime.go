package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/util"
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
	// MessageKindDataStreamStart represents misc data published on a custom run channel,
	// streamed to subscribers via multiple messages with an arbitrary prefix.
	MessageKindDataStreamStart = MessageKind("datastream-start")
	// MessageKindDataStreamEnd acknowledges the end of a datastream.
	MessageKindDataStreamEnd = MessageKind("datastream-end")
	// MessageKindEvent represents event data
	MessageKindEvent = MessageKind("event")
	// MessageKindPing is a message kind sent as a keepalive.
	MessageKindPing = MessageKind("ping")
	// MessageKindSubscribe is a message kind that subscribes to a new set of topics,
	// given a valid JWT embedding the topics directly.
	MessageKindSubscribe = MessageKind("sub")
	// MessageKindUnsubscribe is a message kind that indicates the subscription should
	// stop listening to the given topics
	MessageKindUnsubscribe = MessageKind("unsub")
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

	// PublishStream publishes streams of data to subscribers.
	//
	// A stream of data starts with a standard Publish() call using
	// the kind "datastream", with a stream ID in the data channel.
	//
	// Data for this stream is then published via this method, which
	// gets sent to subscribers with a "${streamID}:" prefix in plaintext
	//
	PublishStream(ctx context.Context, m Message, data string)
}

// Broadcaster manages all subscriptions to channels, and handles the forwarding of
// messages to each subscription
type Broadcaster interface {
	// Publish writes a given message to all subscriptions for the given Message
	Publisher

	// Subscribe adds a new authenticated Subscription subscribed to the given
	// topics.
	//
	// Note that if the subscription currently exists, the current channels will
	// be *added to* the subscribed set.
	//
	// This is non-blocking, running in another thread until the context is
	// cancelled or Unsubscribe is called on the subscription ID and topic pair.
	Subscribe(ctx context.Context, s Subscription, topics []Topic) error

	// Unsubscribe a subscription from a set of specific topics.
	Unsubscribe(ctx context.Context, subID uuid.UUID, topics []Topic) error

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

// Subscription represents a subscription to a specific set of channels, via a given protocol.
// This may be backed by websockets, server-sent-events, and so on.
type Subscription interface {
	// ID returns a unique ID for the given subscription
	ID() uuid.UUID

	// Protocol is the name of the protocol/implementation
	Protocol() string

	// SendKeepalive is called by the broadcaster to keep the current connection alive.  This
	// may be a noop, depending on the implementation.  Note that keepalives are sent every
	// 30 seconds - this is not implementation specific.
	//
	// If SendKeepalive fails consecutively, the subscription will be closed.
	SendKeepalive(m Message) error

	// WriteMessage allows the writing of messages to the particular subscription.  This is
	// implementation agnostic;  messages may be written via websockets or HTTP connections
	// for server-sent-events.
	//
	// Note that each subscription implementation may write different formats of a Message,
	// so this cannot fulfil io.Writer.
	WriteMessage(m Message) error

	// WriteStream publishes data for a given stream ID to the subscription.
	WriteStream(streamID, data string) error

	// Closer closes the current subscription immediately, terminating any active connections.
	io.Closer
}

// ReadWriteSubscription is a subscription which reads messages via the Poll() method, allowing
// the subscription itself to manage subscribing and unsubscribing from topics.
type ReadWriteSubscription interface {
	Subscription

	// Poll polls for new messages, blocking until the Subscription closes or the context
	// is cancelled.
	Poll(ctx context.Context) error
}

// Topic represents a topic for a message.  This is used for publishing and subscribing.
// Each message is published to one or more topics.
type Topic struct {
	// Kind represents the topic kind, ie. whether this topic is for events or run data.
	// This allows consumers to stream events and data from runs separately.
	Kind TopicKind `json:"kind"`

	// EnvID represents the environment ID that this topic is subscribed to.  This
	// must always be present for both run and event topics.
	//
	// This will be auto-filled, and scopes data to individual environments.
	EnvID uuid.UUID `json:"env_id"`

	// RunID is used for debugging purposes only, and does not constrain topics.
	RunID ulid.ULID `json:"run_id,omitempty,omitzero"`

	// Channel represents the channel - or grouping - for the stream.  Within a
	// channel there can be many topics.
	//
	// Each run gets its own channel (using the Run ID as its channel).  The
	// channel can be customized when streaming from SDKs, allowing subscribers
	// to gather data from multiple runs at a time.
	Channel string `json:"channel"`

	// Name represents a topic name, such as "$step", "$result", "step-name",
	// or eg. "api/event.name".
	Name string `json:"name"`

	// TODO: Implement event pub/sub and realtime message filtering.
	// Expression is used to filter messages such as events, eg "event.data.value > 500".
	// Expression *string
}

func (t Topic) String() string {
	switch t.Kind {
	case TopicKindRun:
		// Hash the channel such that user-generated channels aren't too long.
		return fmt.Sprintf("%s:%s:%s", t.EnvID, util.XXHash(t.Channel), t.Name)
	case TopicKindEvent:
		return fmt.Sprintf("%s:%s", t.EnvID, t.Name)
	}

	return fmt.Sprintf("%s:%s", t.EnvID, t.Name)
}

// NewMessage creates a new message with the given kind and data.  If the data is
// not of type byte or json.RawMessage, the data will be marshalled to JSON before
// being set.
//
// Note that other fields in the message are not set.
func NewMessage(kind MessageKind, data any) Message {
	msg := Message{Kind: kind, CreatedAt: time.Now().Truncate(time.Millisecond).UTC()}
	switch v := data.(type) {
	case json.RawMessage:
		msg.Data = v
	case []byte:
		msg.Data = json.RawMessage(v)
	default:
		var err error
		msg.Data, err = json.Marshal(data)
		if err != nil {
			logger.StdlibLogger(context.Background()).
				Error("error marshalling realtime msg data", "error", err)
		}
	}
	return msg
}

// Message represents a single message sent on realtime topics.
type Message struct {
	// Kind represents the message kind.
	Kind MessageKind `json:"kind"`
	// Data represents the data in the message.
	Data json.RawMessage `json:"data"`
	// CreatedAt is the time that this message was created.
	CreatedAt time.Time `json:"created_at"`

	//
	// Required tenant and grouping fields
	//

	// Channel is the channel (or run ID) that this message is related to.
	Channel string `json:"channel,omitempty,omitzero"`
	// EnvID is the environment ID that the message belongs to.
	EnvID uuid.UUID `json:"env_id,omitempty,omitzero"`
	// TopicNames represents the custom channels that this message should be broadcast
	// on.  For steps, this must include the unhashed step ID.  For custom broadcasts,
	// this is the chosen channel name in the SDK.
	TopicNames []string `json:"topics"`

	//
	// Optional fields, set by the executor.
	//

	// FnID is the function ID that this message is related to.
	FnID uuid.UUID `json:"fn_id,omitempty,omitzero"`
	// FnSlug is the function slug that this message is related to.
	FnSlug string `json:"fn_slug,omitempty,omitzero"`
	// RunID is used for debugging purposes only, and does not constrain topics.
	RunID ulid.ULID `json:"run_id,omitempty,omitzero"`
}

func (m Message) Validate() error {
	// Ensure that the Data is present for streams.
	if m.Kind == MessageKindDataStreamStart || m.Kind == MessageKindDataStreamEnd {
		// and assert that the stream ID exists and contains no colon
		if len(m.Data) == 0 {
			return fmt.Errorf("datastream kinds must have a stream id set")
		}
		if bytes.Contains(m.Data, []byte(":")) {
			return fmt.Errorf("datstream stream id must not contain colons (:)")
		}
	}
	return nil
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
			Kind:    TopicKindRun,
			Name:    TopicNameStep,
			Channel: m.Channel,
			EnvID:   m.EnvID,
		}

		for n, v := range m.TopicNames {
			topics[n+1] = Topic{
				Kind:    TopicKindRun,
				Channel: m.Channel,
				EnvID:   m.EnvID,
				Name:    v,
			}
		}

		return topics
	case MessageKindRun:
		// This message is a run output.
		topics := make([]Topic, len(m.TopicNames)+1)

		// Always publish step outputs to the "$step" topic, alongside
		// the topic names within the message (which includes the step name)
		topics[0] = Topic{
			Kind:    TopicKindRun,
			Name:    TopicNameRun,
			Channel: m.Channel,
			EnvID:   m.EnvID,
		}

		for n, v := range m.TopicNames {
			topics[n+1] = Topic{
				Kind:    TopicKindRun,
				Channel: m.Channel,
				EnvID:   m.EnvID,
				Name:    v,
			}
		}

		return topics
	}

	// Default to topic kinds of Run
	topics := make([]Topic, len(m.TopicNames))
	for n, v := range m.TopicNames {
		topics[n] = Topic{
			Kind:    TopicKindRun,
			Channel: m.Channel,
			EnvID:   m.EnvID,
			Name:    v,
		}
	}
	return topics
}
