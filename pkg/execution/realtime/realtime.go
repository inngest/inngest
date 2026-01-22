package realtime

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution/realtime/streamingtypes"
)

type (
	Message = streamingtypes.Message
	Topic   = streamingtypes.Topic
	Chunk   = streamingtypes.Chunk
)

// Publisher accepts messages from other services (eg. the executor, or the event API) and
// publishes messages to any subscribers.
type Publisher interface {
	// Write publishes arbitrary data to a channel.  Note that this does
	// not have any Message wrapping, and is raw data to be read by an
	// end user.
	//
	// Because of this, there is no topic, as theres no way to indicate
	// which topic we're writing to without the Message wrapper.
	Write(ctx context.Context, envID uuid.UUID, channel string, data []byte)

	// Publish publishes a message to any realtime subscribers.
	//
	// Note that this returns no error;  we expect that the publisher retries
	// internally and/or handles durability of the message.  Once Publish is called
	// the caller is no longer responsible for the lifetime of the message.  This
	// simplifies all caller code.
	Publish(ctx context.Context, m Message)

	// PublishChunk publishes streams of data to subscribers.
	//
	// A stream of data starts with a standard Publish() call using
	// the kind "datastream", with a stream ID in the data channel.
	//
	// Data for this stream is then published via this method, which
	// gets sent to subscribers with a "${streamID}:" prefix in plaintext
	//
	// Note that this requires the 'datastream-start' message to grab topics
	// from when publishing.
	PublishChunk(ctx context.Context, m Message, c Chunk)
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

	// WriteChunk publishes a chunk in a stream - data for a given stream ID to the subscription.
	WriteChunk(c Chunk) error

	// Write forwards bytes directly to the subscription.  It is different to WriteMessage in
	// that no encapsulation is written;  the bytes are written directly as-is.
	//
	// This is useful when forwarding raw bytes from eg. durable endpoints to its redirected
	// endpoint.
	Write(b []byte) error

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
