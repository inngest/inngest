package state

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

var (
	ErrMetadataNotFound = fmt.Errorf("metadata not found")
)

type State struct {
	Metadata Metadata
	// Events stores json-encoded events directly within state.
	Events []json.RawMessage
	// Steps stores all step inputs/outputs.
	Steps map[string]json.RawMessage
}

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
