package inngestgo

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// ExternalID is the field name used to reference the user's ID within your
	// systems.  This is _your_ UUID or ID for referencing the user, and allows
	// Inngest to match contacts to your users.
	ExternalID = "external_id"

	// Email is the field name used to reference the user's email.
	Email = "email"
)

type Event struct {
	// ID is an optional event ID used for deduplication.
	ID *string `json:"id,omitempty"`

	// Name represents the name of the event.  We recommend the following
	// simple format: "noun.action".  For example, this may be "signup.new",
	// "payment.succeeded", "email.sent", "post.viewed".
	//
	// Name is required.
	Name string `json:"name"`

	// Data is a key-value map of data belonging to the event.  This should
	// include all relevant data.  For example, a "signup.new" event may include
	// the user's email, their plan information, the signup method, etc.
	Data map[string]any `json:"data"`

	// User is a key-value map of data belonging to the user that authored the
	// event.  This data will be upserted into the contact store.
	//
	// We match the user via one of two fields: "external_id" and "email", defined
	// as consts within this package.
	//
	// If these fields are present in this map the attributes specified here
	// will be updated within Inngest, and the event will be attributed to
	// this contact.
	User any `json:"user,omitempty"`

	// Timestamp is the time the event occured at *millisecond* (not nanosecond)
	// precision.  This defaults to the time the event is received if left blank.
	//
	// Inngest does not guarantee that events are processed within the
	// order specified by this field.  However, we do guarantee that user data
	// is stored correctly according to this timestamp.  For example,  if there
	// two events set the same user attribute, the event with the latest timestamp
	// is guaranteed to set the user attributes correctly.
	Timestamp int64 `json:"ts,omitempty"`

	// Version represents the event's version.  Versions can be used to denote
	// when the structure of an event changes over time.
	//
	// Versions typically change when the keys in `Data` change, allowing you to
	// keep the same event name (eg. "signup.new") as fields change within data
	// over time.
	//
	// We recommend the versioning scheme "YYYY-MM-DD.XX", where .XX increments:
	// "2021-03-19.01".
	Version string `json:"v,omitempty"`
}

// Validate returns  an error if the event is not well formed
func (e Event) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("event name must be present")
	}
	if len(e.Data) == 0 {
		return fmt.Errorf("event data must be present")
	}
	return nil
}

func (e Event) Map() map[string]any {
	if e.Data == nil {
		e.Data = make(map[string]any)
	}
	if e.User == nil {
		e.User = make(map[string]any)
	}

	data := map[string]any{
		"name": e.Name,
		"data": e.Data,
		"user": e.User,
		// We cast to float64 because marshalling and unmarshalling from
		// JSON automatically uses float64 as its type;  JS has no notion
		// of ints.
		"ts": float64(e.Timestamp),
	}

	if e.Version != "" {
		data["v"] = e.Version
	}
	if e.ID != nil {
		data["id"] = *e.ID
	}

	return data
}

// GenericEvent represents a single event generated from your system to be sent to
// Inngest.
type GenericEvent[DATA any, USER any] struct {
	// ID is an optional event ID used for deduplication.
	ID *string `json:"id,omitempty"`
	// Name represents the name of the event.  We recommend the following
	// simple format: "noun.action".  For example, this may be "signup.new",
	// "payment.succeeded", "email.sent", "post.viewed".
	//
	// Name is required.
	Name string `json:"name"`

	// Data is a struct or key-value map of data belonging to the event.  This should
	// include all relevant data.  For example, a "signup.new" event may include
	// the user's email, their plan information, the signup method, etc.
	Data DATA `json:"data"`

	// User is a struct or key-value map of data belonging to the user that authored the
	// event.  This data will be upserted into the contact store.
	//
	// We match the user via one of two fields: "external_id" and "email", defined
	// as consts within this package.
	//
	// If these fields are present in this map the attributes specified here
	// will be updated within Inngest, and the event will be attributed to
	// this contact.
	User USER `json:"user,omitempty"`

	// Timestamp is the time the event occured at *millisecond* (not nanosecond)
	// precision.  This defaults to the time the event is received if left blank.
	//
	// Inngest does not guarantee that events are processed within the
	// order specified by this field.  However, we do guarantee that user data
	// is stored correctly according to this timestamp.  For example,  if there
	// two events set the same user attribute, the event with the latest timestamp
	// is guaranteed to set the user attributes correctly.
	Timestamp int64 `json:"ts,omitempty"`

	// Version represents the event's version.  Versions can be used to denote
	// when the structure of an event changes over time.
	//
	// Versions typically change when the keys in `Data` change, allowing you to
	// keep the same event name (eg. "signup.new") as fields change within data
	// over time.
	//
	// We recommend the versioning scheme "YYYY-MM-DD.XX", where .XX increments:
	// "2021-03-19.01".
	Version string `json:"v,omitempty"`
}

// Event() turns the GenericEvent into a normal Event.
//
// NOTE: This is a naive inefficient implementation and should not be used in performance
// constrained systems.
func (ge GenericEvent[D, U]) Event() Event {
	byt, _ := json.Marshal(ge)
	val := Event{}
	_ = json.Unmarshal(byt, &val)
	return val
}

func (ge GenericEvent[D, U]) Validate() error {
	if ge.Name == "" {
		return fmt.Errorf("event name must be present")
	}
	return nil
}

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
