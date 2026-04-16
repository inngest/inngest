package state

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

var ErrMetadataNotFound = fmt.Errorf("metadata not found")

type State struct {
	Metadata Metadata
	// Events stores json-encoded events directly within state.
	Events []json.RawMessage
	// Steps stores all step inputs/outputs.
	Steps map[string]json.RawMessage

	Defers map[string]Defer
}

type Defer struct {
	// Deferred companion's key in `onDefer` record
	CompanionID string

	// Hashed step ID
	HashedID string

	// Status for scheduling the deferred run:
	// - Already scheduled?
	// - Schedule after the parent run ends?
	// - Never schedule (i.e. cancelled)?
	ScheduleStatus ScheduleStatus

	// Data the user passed to `step.defer`
	Input json.RawMessage
}

type ScheduleStatus int

const (
	// Unused
	ScheduleStatusUnknown ScheduleStatus = iota

	// Already scheduled (when defer is configured to run immediately)
	ScheduleStatusScheduled

	// Schedule after parent run ends
	ScheduleStatusAfterRun

	// Will not schedule
	ScheduleStatusCancelled
)

// OpID is the hashed ID for a single step.  This is currently a SHA1
// hash of the step name and step's index, though may be lowered to an 8
// or 10 byte hash in the future.
type OpID [20]byte

func (s OpID) String() string {
	return hex.EncodeToString(s[:])
}

func (s *OpID) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *OpID) UnmarshalJSON(b []byte) error {
	// The first and last bytes of the data should be quotes.
	if len(b) == 0 {
		return nil
	}
	if b[0] != '"' || b[len(b)-1] != '"' {
		return fmt.Errorf("invalid step quote")
	}
	_, err := hex.Decode(s[:], b[1:len(b)-1])
	return err
}
