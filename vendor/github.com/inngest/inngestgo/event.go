package inngestgo

import (
	"time"

	"github.com/inngest/inngestgo/internal/event"
)

const (
	// ExternalID is the field name used to reference the user's ID within your
	// systems.  This is _your_ UUID or ID for referencing the user, and allows
	// Inngest to match contacts to your users.
	ExternalID = "external_id"

	// Email is the field name used to reference the user's email.
	Email = "email"
)

type Event = event.Event
type GenericEvent[DATA any] = event.GenericEvent[DATA]

// NowMillis returns a timestamp with millisecond precision used for the Event.Timestamp
// field.
func NowMillis() int64 {
	return time.Now().UnixMilli()
}

// Timestamp converts a go time.Time into a timestamp with millisecond precision
// used for the Event.Timestamp field.
func Timestamp(t time.Time) int64 {
	return t.UnixNano() / 1_000_000
}
