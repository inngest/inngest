package streamingtypes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	// MessageKindDataStreamStart represents the start of a streaming chunk block,
	// streamed to subscribers via multiple messages with the same datastream ID.
	MessageKindDataStreamStart = MessageKind("datastream-start")
	// MessageKindDataStreamEnd acknowledges the end of a datastream.
	MessageKindDataStreamEnd   = MessageKind("datastream-end")
	MessageKindDataStreamChunk = MessageKind("chunk")

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

// Message represents a single message sent on realtime topics.
type Message struct {
	// Kind represents the message kind.
	Kind MessageKind `json:"kind"`
	// Data represents the data in the message.
	Data json.RawMessage `json:"data"`
	// Metadata is optional data regarding the message contents.  This is used
	// specifically to store the content-type and other data around streamed
	// HTTP gateway responses.
	Metadata json.RawMessage `json:"metadata,omitempty"`
	// CreatedAt is the time that this message was created.
	CreatedAt time.Time `json:"created_at"`

	//
	// Required tenant and grouping fields
	//

	// Channel is the channel (or run ID) that this message is related to.
	Channel string `json:"channel,omitempty,omitzero"`
	// EnvID is the environment ID that the message belongs to.
	EnvID uuid.UUID `json:"env_id,omitempty,omitzero"`
	// Topic represents the custom topic that this message should be broadcast
	// on.  For steps, this must include the unhashed step ID.  For custom broadcasts,
	// this is the chosen topic name in the SDK.
	Topic string `json:"topic"`

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
		topics := make([]Topic, 2)

		// Always publish step outputs to the "$step" topic, alongside
		// the topic names within the message (which includes the step name)
		topics[0] = Topic{
			Kind:    TopicKindRun,
			Name:    TopicNameStep,
			Channel: m.Channel,
			EnvID:   m.EnvID,
		}

		topics[1] = Topic{
			Kind:    TopicKindRun,
			Name:    m.Topic,
			Channel: m.Channel,
			EnvID:   m.EnvID,
		}

		return topics
	case MessageKindRun:
		// This message is a run output.
		// Always publish step outputs to the "$run" topic.
		builtin := Topic{
			Kind:    TopicKindRun,
			Name:    TopicNameRun,
			Channel: m.Channel,
			EnvID:   m.EnvID,
		}

		if m.Topic == "" {
			// No topic name for run ends;  use the builtin only.
			return []Topic{builtin}
		}

		topics := make([]Topic, 2)
		topics[0] = builtin
		topics[1] = Topic{
			Kind:    TopicKindRun,
			Name:    m.Topic,
			Channel: m.Channel,
			EnvID:   m.EnvID,
		}
		return topics
	}

	// Default to topic kinds of Run
	return []Topic{{
		Kind:    TopicKindRun,
		Name:    m.Topic,
		Channel: m.Channel,
		EnvID:   m.EnvID,
	}}
}

func ChunkFromMessage(m Message, data string) Chunk {
	return Chunk{
		Kind:     string(MessageKindDataStreamChunk),
		StreamID: string(m.Data),
		Data:     data,
		FnID:     m.FnID,
		FnSlug:   m.FnSlug,
		RunID:    m.RunID,
	}
}

// Chunk represents a chunk of a stream.
type Chunk struct {
	// Kind represents the message kind.  This must always
	// be "chunk" and is present to help clients differentiate
	// between chunks and regular messages.
	Kind string `json:"kind"`
	// StreamID is the stream ID for the chunk
	StreamID string `json:"stream_id"`
	// Data is the data in the chunk
	Data string `json:"data"`

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
