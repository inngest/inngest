package state

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/event"
)

var (
	ErrIdempotencyConflict = fmt.Errorf("idempotency conflict")
)

type State interface {
	Metadata() Metadata
	Events() []event.Event
	Stack() []string
	Steps() map[StepID]json.RawMessage
}

// StepID is the hashed ID for a single step.  This is currently a SHA1
// hash of the step name and step's index, though may be lowered to an 8
// or 10 byte hash in the future.
type StepID [20]byte

func (s StepID) String() string {
	return hex.EncodeToString(s[:])
}

func (s *StepID) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *StepID) UnmarshalJSON(b []byte) error {
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
