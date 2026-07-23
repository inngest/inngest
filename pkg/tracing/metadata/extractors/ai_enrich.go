package extractors

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/metadata"
)

type AIEnrichOpts struct {
	// FallbackLatencyMs is measured latency from the emitting source; only
	// applied when the emitter did not already report latency.
	FallbackLatencyMs int64
}

// Enrich fills derivable gaps in the metadata. It is idempotent and never
// overwrites values the emitter supplied.
func (ms *AIMetadata) Enrich(opts AIEnrichOpts) {
	if ms.TotalTokens == nil && (ms.InputTokens > 0 || ms.OutputTokens > 0) {
		totalTokens := ms.InputTokens + ms.OutputTokens
		ms.TotalTokens = &totalTokens
	}

	if ms.EstimatedCost == nil && (ms.InputTokens > 0 || ms.OutputTokens > 0) {
		// prefer the model that actually served the request for cost estimation
		costModel := ms.ResponseModel
		if costModel == "" {
			costModel = ms.RequestModel
		}
		ms.EstimatedCost = EstimateCost(costModel, ms.InputTokens, ms.OutputTokens)
	}

	if ms.LatencyMs == nil && opts.FallbackLatencyMs > 0 {
		ms.LatencyMs = &opts.FallbackLatencyMs
	}
}

// EnrichAIValues enriches raw SDK-sent inngest.ai values, adding only
// total_tokens and estimated_cost when they are absent and derivable. It
// applies only to inngest.ai non-delete updates: delete values are keys to
// remove, not metrics. Caller keys are never overwritten or removed, and the
// input is returned unchanged when it cannot be parsed as AIMetadata. No
// latency fallback is applied: SDK latency is unknown, and step duration
// would overstate it.
func EnrichAIValues(ctx context.Context, kind metadata.Kind, op metadata.Opcode, v metadata.Values) metadata.Values {
	if kind != KindInngestAI || op == enums.MetadataOpcodeDelete {
		return v
	}

	raw, err := json.Marshal(v)
	if err != nil {
		logger.From(ctx).Warn("failed to serialize inngest.ai metadata for enrichment", "error", err)
		return v
	}

	var md AIMetadata
	if err := json.Unmarshal(raw, &md); err != nil {
		logger.From(ctx).Warn("failed to parse inngest.ai metadata for enrichment", "error", err)
		return v
	}

	md.Enrich(AIEnrichOpts{})

	out := maps.Clone(v)
	if _, ok := v["total_tokens"]; !ok && md.TotalTokens != nil {
		if b, err := json.Marshal(md.TotalTokens); err == nil {
			out["total_tokens"] = b
		}
	}
	if _, ok := v["estimated_cost"]; !ok && md.EstimatedCost != nil {
		if b, err := json.Marshal(md.EstimatedCost); err == nil {
			out["estimated_cost"] = b
		}
	}

	return out
}
