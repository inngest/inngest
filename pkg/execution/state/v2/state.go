package state

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/enums"
)

var (
	ErrMetadataNotFound = fmt.Errorf("metadata not found")

	// ErrDeferLimitExceeded is returned by SaveDefer when adding a new defer
	// would exceed consts.MaxDefersPerRun for the run. Updates to an existing
	// hashedID never trip this.
	ErrDeferLimitExceeded = fmt.Errorf("defer limit per run exceeded")

	// ErrDeferInputAggregateExceeded means the Input would push the run
	// past MaxDeferInputAggregateSize. SaveDefer writes a Rejected sentinel.
	ErrDeferInputAggregateExceeded = fmt.Errorf("defer input aggregate size exceeded")
)

type State struct {
	Metadata Metadata
	// Events stores json-encoded events directly within state.
	Events []json.RawMessage
	// Steps stores all step inputs/outputs.
	Steps map[string]json.RawMessage
}

// DeferMeta is the metadata-only view of a Defer (no Input). Keep fields
// in sync with Defer below.
type DeferMeta struct {
	FnSlug         string
	HashedID       string
	UserlandID     string
	ScheduleStatus enums.DeferStatus
}

type Defer struct {
	// Fully-qualified function slug (`{app-slug}-{fn-slug}`) of the
	// `onDefer` Inngest function that will handle this deferred run.
	FnSlug string

	// Hashed defer ID
	HashedID string

	// UserlandID is the SDK-caller-supplied defer id (the first argument to
	// `defer("foo", ...)`). HashedID is `sha1(UserlandID)`; this field
	// preserves the original string so the dev-server UI can show the id the
	// user typed instead of the hash.
	UserlandID string

	// Status for scheduling the deferred run:
	// - Already scheduled?
	// - Schedule after the parent run ends?
	// - Never schedule (i.e. aborted)?
	//
	// Aborted is terminal within a run: once a defer transitions to
	// DeferStatusAborted, it stays there. The Lua-level SaveDefer silently
	// no-ops any subsequent write for the same hashedID. There is no "unabort"
	// path: same hashedID + abort is final.
	ScheduleStatus enums.DeferStatus

	// Data passed to the defer
	Input json.RawMessage
}

func (d Defer) Validate() error {
	if d.FnSlug == "" {
		return fmt.Errorf("FnSlug is required")
	}
	if d.HashedID == "" {
		return fmt.Errorf("HashedID is required")
	}
	if d.ScheduleStatus == enums.DeferStatusUnknown {
		return fmt.Errorf("ScheduleStatus is required")
	}
	return nil
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
