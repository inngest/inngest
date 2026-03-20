package extractors

import (
	"encoding/json"
	"strconv"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/metadata"
)

const (
	KindInngestUsage metadata.Kind = "inngest.usage"
)

// UsageMetadata captures run-level size metrics emitted at finalization.
type UsageMetadata struct {
	MetadataBytes int `json:"metadata_bytes"`
}

func (m UsageMetadata) Kind() metadata.Kind {
	return KindInngestUsage
}

func (m UsageMetadata) Op() metadata.Opcode {
	return enums.MetadataOpcodeMerge
}

func (m UsageMetadata) Serialize() (metadata.Values, error) {
	return metadata.Values{
		"metadata_bytes": json.RawMessage(strconv.Itoa(m.MetadataBytes)),
	}, nil
}
