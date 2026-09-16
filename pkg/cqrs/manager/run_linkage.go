package manager

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

func (w wrapper) GetRunDefers(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]cqrs.RunDefer, error) {
	if len(runIDs) == 0 {
		return map[ulid.ULID][]cqrs.RunDefer{}, nil
	}

	// Merged read picks up post-emit updates: status flips on abort and the
	// child run ID stamped on at schedule time.
	spansByParent, err := w.GetSpansByRunIDsAndName(ctx, runIDs, meta.SpanNameDefer)
	if err != nil {
		return nil, fmt.Errorf("error loading executor.defer spans: %w", err)
	}

	out := make(map[ulid.ULID][]cqrs.RunDefer, len(spansByParent))
	for parentRunID, spans := range spansByParent {
		for _, s := range spans {
			if s.Attributes.DeferHashedID == nil || s.Attributes.DeferStatus == nil {
				continue
			}
			if *s.Attributes.DeferStatus == enums.DeferStatusUnknown {
				logger.StdlibLogger(ctx).Warn(
					"skipping defer span with unknown status",
					"run_id", parentRunID.String(),
					"hashed_id", *s.Attributes.DeferHashedID,
				)
				continue
			}
			rd := cqrs.RunDefer{
				HashedDeferID: *s.Attributes.DeferHashedID,
				Status:        *s.Attributes.DeferStatus,
				// DeferChildRunID is stamped onto the defer span when the
				// child is scheduled; nil means the child hasn't been
				// scheduled yet (parent still running, or schedule event
				// not yet processed).
				RunID: s.Attributes.DeferChildRunID,
			}
			if s.Attributes.DeferUserlandID != nil {
				rd.UserlandDeferID = *s.Attributes.DeferUserlandID
			}
			if s.Attributes.DeferFnSlug != nil {
				rd.FnSlug = *s.Attributes.DeferFnSlug
			}
			out[parentRunID] = append(out[parentRunID], rd)
		}
	}

	// Map iteration above is non-deterministic; sort by HashedDeferID so
	// repeated queries return identical orderings.
	for _, defers := range out {
		slices.SortFunc(defers, func(a, b cqrs.RunDefer) int {
			return cmp.Compare(a.HashedDeferID, b.HashedDeferID)
		})
	}

	return out, nil
}
