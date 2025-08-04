package event

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// GenericEvent represents a single event generated from your system to be sent to
// Inngest.
type GenericEvent[DATA any] struct {
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

	// Deprecated
	User map[string]any `json:"user,omitempty"`

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
func (ge GenericEvent[D]) Event() Event {
	byt, _ := json.Marshal(ge)
	val := Event{}
	_ = json.Unmarshal(byt, &val)
	return val
}

func (ge GenericEvent[D]) Validate() error {
	if ge.Name == "" {
		return fmt.Errorf("event name must be present")
	}

	err := ValidateEventDataType(ge.Data)
	if err != nil {
		return err
	}

	return nil
}

func (ge GenericEvent[D]) Map() map[string]any {
	var data any = ge.Data
	if reflect.TypeOf(data).Kind() == reflect.Ptr && reflect.ValueOf(data).IsNil() {
		data = make(map[string]any)
	}

	out := map[string]any{
		"name": ge.Name,
		"data": data,
		"user": ge.User,
		// We cast to float64 because marshalling and unmarshalling from
		// JSON automatically uses float64 as its type;  JS has no notion
		// of ints.
		"ts": float64(ge.Timestamp),
	}

	if ge.Version != "" {
		out["v"] = ge.Version
	}
	if ge.ID != nil {
		out["id"] = *ge.ID
	}

	return out
}

type Event = GenericEvent[map[string]any]

// ValidateEventDataType checks if the given type is one of: nil, map, or
// struct.
func ValidateEventDataType(v any) error {
	if v == nil {
		return nil
	}

	isDataMap := reflect.TypeOf(v).Kind() == reflect.Map
	if isDataMap {
		return nil
	}

	isDataStruct := reflect.TypeOf(v).Kind() == reflect.Struct
	if isDataStruct {
		return nil
	}

	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		if reflect.ValueOf(v).IsNil() {
			return fmt.Errorf("event data must be a map or struct")
		}

		val := reflect.ValueOf(v).Elem()
		if val.Kind() == reflect.Map {
			return nil
		}
		if val.Kind() == reflect.Struct {
			return nil
		}
	}

	return fmt.Errorf("event data must be a map or struct")
}
