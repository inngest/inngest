package extractors

import (
	"encoding/json"

	"github.com/inngest/inngest/pkg/tracing/metadata"
)

// experimentOpts mirrors the subset of fields the SDK spreads onto a variant
// sub-step's opcode opts when it is dispatched inside a group.experiment()
// callback. Fields are JSON-tagged to match the SDK's camelCase wire format.
//
// Only ExperimentName is required for the extractor to emit metadata. The
// other fields may be missing on older SDK versions that predate the
// shrunken metadata PR (see inngest-js#1458 successor).
type experimentOpts struct {
	ExperimentName    string `json:"experimentName"`
	Variant           string `json:"variant"`
	SelectionStrategy string `json:"selectionStrategy"`
}

// ExtractExperimentOptsMetadata inspects GeneratorOpcode.Opts for the flat
// experiment context fields the SDK attaches to variant sub-steps inside
// group.experiment() and returns an ExperimentMetadata ready to be written
// as a step-scoped metadata span.
//
// The executor owns this emission so that:
//   - SDKs do not need to issue an explicit addMetadata() call per variant
//     step (reducing cross-SDK surface area and avoiding duplicate rows).
//   - Clients on older SDK versions automatically get experiment metadata
//     in ClickHouse without a client upgrade.
//
// Returns (nil, nil) when opts carry no experiment name, which is the
// expected case for every non-variant step.
func ExtractExperimentOptsMetadata(opts any) (metadata.Structured, error) {
	if opts == nil {
		return nil, nil
	}

	// Opts arrive as either a decoded map (common path) or raw JSON bytes
	// (driver-dependent). Normalize to JSON then decode into the typed
	// struct so the extractor is agnostic to which shape the executor
	// hands us.
	var raw []byte
	switch v := opts.(type) {
	case []byte:
		raw = v
	case json.RawMessage:
		raw = v
	default:
		b, err := json.Marshal(opts)
		if err != nil {
			return nil, err
		}
		raw = b
	}

	var parsed experimentOpts
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Opts that don't deserialize into our shape are not an error at
		// this layer — they just don't carry experiment fields.
		return nil, nil
	}

	if parsed.ExperimentName == "" {
		return nil, nil
	}

	return ExperimentMetadata{
		ExperimentName:    parsed.ExperimentName,
		Variant:           parsed.Variant,
		SelectionStrategy: parsed.SelectionStrategy,
	}, nil
}
