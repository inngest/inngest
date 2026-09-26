package event

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SerializedEvents is an immutable JSON snapshot of one or more events. It
// keeps both individual events and their JSON-array representation so fan-out
// consumers can share event and trace input bytes without rebuilding either.
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
