package event

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SerializedEvents is an immutable JSON snapshot of one or more events.
//
// The snapshot owns one string containing the complete JSON array. Event
// boundaries are stored as ranges into that string, so Input and Event return
// zero-copy views backed by the same memory. Copying a SerializedEvents value
// does not copy the JSON and is safe for concurrent fan-out consumers.
//
// For example:
//
//	input: [{"name":"a"},{"name":"b"}]
//	        └──────────┘ └──────────┘
//	          Event(0)     Event(1)
//
// Input returns the complete array for trace input. Event(0) returns
// {"name":"a"}, and Event(1) returns {"name":"b"}; state creation sends these
// as individual event values together in one CreateState call. Both events are
// substring views into input.
//
// Construction copies the source json.RawMessages into the owned string so
// later source mutations cannot affect the snapshot. RawMessages creates new,
// independently owned byte slices for APIs that require mutable []byte values;
// callers should otherwise keep the data in its string-backed form.
type SerializedEvents struct {
	events []serializedEventRange
	input  string
}

type serializedEventRange struct {
	start int
	end   int
}

func NewSerializedEvents(events []json.RawMessage) (SerializedEvents, error) {
	serialized := SerializedEvents{events: make([]serializedEventRange, len(events))}
	var input strings.Builder
	input.WriteByte('[')
	for i, event := range events {
		if !json.Valid(event) {
			return SerializedEvents{}, fmt.Errorf("serialized event %d is invalid JSON", i)
		}
		if i > 0 {
			input.WriteByte(',')
		}
		start := input.Len()
		input.Write(event)
		serialized.events[i] = serializedEventRange{start: start, end: input.Len()}
	}
	input.WriteByte(']')
	serialized.input = input.String()
	return serialized, nil
}

func (s SerializedEvents) Len() int {
	return len(s.events)
}

func (s SerializedEvents) Event(index int) string {
	event := s.events[index]
	return s.input[event.start:event.end]
}

func (s SerializedEvents) Input() string {
	return s.input
}

func (s SerializedEvents) Equal(other SerializedEvents) bool {
	return len(s.events) == len(other.events) && s.input == other.input
}

func (s SerializedEvents) RawMessages() []json.RawMessage {
	events := make([]json.RawMessage, len(s.events))
	for i := range s.events {
		events[i] = json.RawMessage(s.Event(i))
	}
	return events
}
