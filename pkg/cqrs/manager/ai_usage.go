package manager

import (
	"context"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
	"github.com/oklog/ulid/v2"
)

// addCountedAIMetadata folds every counted inngest.ai entry into the builder.
// Which entries count is the shared extractors.AIUsageEntryCounted rule,
// decided per run: whether the run has any step-scoped AI entry determines
// whether extended-trace entries count. A malformed entry is logged and
// skipped rather than aborting the summary.
func addCountedAIMetadata(
	ctx context.Context,
	builder *extractors.AISummaryBuilder,
	metadataByParent map[string][]*cqrs.SpanMetadata,
	runID ulid.ULID,
) {
	hasStepAI := false
	for _, entries := range metadataByParent {
		for _, md := range entries {
			if md.Kind == extractors.KindInngestAI && extractors.AIUsageStepScoped(md.Scope) {
				hasStepAI = true
				break
			}
		}
		if hasStepAI {
			break
		}
	}

	for parentSpanID, entries := range metadataByParent {
		for _, md := range entries {
			if md.Kind != extractors.KindInngestAI ||
				!extractors.AIUsageEntryCounted(md.Scope, hasStepAI) {
				continue
			}
			if err := builder.AddCall(md.Values); err != nil {
				logger.StdlibLogger(ctx).Warn(
					"skipping malformed inngest.ai metadata entry",
					"run_id", runID.String(),
					"parent_span_id", parentSpanID,
					"error", err,
				)
			}
		}
	}
}
