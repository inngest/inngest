package extractors

import (
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/oklog/ulid/v2"
)

const KindInngestSkip metadata.Kind = "inngest.skip"

// SkipMetadata contains information about why a function run was skipped.
type SkipMetadata struct {
	Reason        string `json:"reason"`
	ExistingRunID string `json:"existing_run_id,omitempty"`
}

func (m SkipMetadata) Kind() metadata.Kind {
	return KindInngestSkip
}

func (m SkipMetadata) Op() metadata.Opcode {
	return enums.MetadataOpcodeSet
}

func (m SkipMetadata) Serialize() (metadata.Values, error) {
	var v metadata.Values
	err := v.FromStruct(m)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// NewSkipMetadata creates skip metadata for the given reason.
// For singleton skips, pass the existing run ID that caused the skip.
func NewSkipMetadata(reason enums.SkipReason, singletonRunID *ulid.ULID) SkipMetadata {
	md := SkipMetadata{Reason: reason.String()}
	if singletonRunID != nil {
		md.ExistingRunID = singletonRunID.String()
	}
	return md
}
