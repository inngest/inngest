package runstate

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/oklog/ulid/v2"
)

var (
	ErrStateNotFound = errors.New("state not found")
)

type State struct {
	Steps      map[StepID]json.RawMessage
	Events     []StateEvent
	Stack      []string
	Config     StateConfig
	RunID      ulid.ULID
	FunctionID uuid.UUID
	AppID      uuid.UUID
	EnvID      uuid.UUID
}

// StateEvent represents an Inngest event paired with an internal identifier
// for the event, used within function state.
type StateEvent struct {
	Event      event.Event
	InternalID ulid.ULID
}

type StateConfig struct {
	// ReplayID stores the ID of the replay, if this identifier belongs to a replay.
	ReplayID *uuid.UUID
	// OriginalID is the original run ID if this is a replay.
	OriginalID *ulid.ULID
	// PriorityFactor is the overall priority factor for this particular function
	// run.  This allows individual runs to take precedence within the same queue.
	// The higher the number (up to consts.PriorityFactorMax), the higher priority
	// this run has.  All next steps will use this as the factor when scheduling
	// future edge jobs (on their first attempt).
	PriorityFactor *int64
	// Idempotency represents an optional idempotency component for the key.
	Idempotency string
	// CustomConcurrencyKeys stores custom concurrency keys for this function run.  This
	// allows us to use custom concurrency keys for each job when processing steps for
	// the function, with cached expression results.
	CustomConcurrencyKeys []CustomConcurrency
	// RequestVersion represents the executor request versioning/hashing style
	// used to manage state.
	//
	// TS v3, Go, Rust, Elixir, and Java all use the same hashing style (1).
	//
	// TS v1 + v2 use a unique hashing style (0) which cannot be transferred
	// to other languages.
	//
	// This lets us send the hashing style to SDKs so that we can execute in
	// the correct format with backcompat guarantees built in.
	//
	// NOTE: We can only know this the first time an SDK is responding to a step.
	RequestVersion int

	// DisableImmediateExecution is used to tell the SDK whether it should
	// disallow immediate execution of steps as they are found.
	//
	// TODO: Have the executor push parallel steps into groups and track itself;
	// only then should we call the SDK.  This ensures that immediate execution
	// is enabled for all steps after parallelism.
	DisableImmediateExecution bool
}

type CustomConcurrency struct {
	// Key represents the actual evaluated concurrency key.
	Key string `json:"k"`
	// Hash represents the hash of the concurrency expression - unevaluated -
	// as defined in the function.  This lets us look up the latest concurrency
	// values as defined in the most recent version of the function and use
	// these concurrency values.  Without this, it's impossible to adjust concurrency
	// for in-progress functions.
	Hash string `json:"h"`
	// Limit represents the limit at the time the function started.  If the concurrency
	// key is removed from the fn definition, this pre-computed value will be used instead.
	//
	// NOTE: If the value is removed from the last deployed function we could also disregard
	// this concurrency key.
	Limit int `json:"l"`
}

// ValidateNewState checks whether the given
func ValidateNewState(s State) error {
	var err error
	if s.RunID.Compare(ulid.ULID{}) == 0 {
		err = errors.Join(err, fmt.Errorf("a run ID must be specified"))
	}
	if s.FunctionID == uuid.Nil {
		err = errors.Join(err, fmt.Errorf("a function ID must be specified"))
	}
	if len(s.Events) == 0 {
		err = errors.Join(err, fmt.Errorf("at least one event must be specified"))
	}
	return err
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
