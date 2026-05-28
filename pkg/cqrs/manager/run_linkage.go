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

// Run-to-run linkage (defers, deferred-from) is reconstructed
// from span attributes here rather than stored in dedicated tables. See
// docs/defer.md for the upstream write path that emits these spans.

func (w wrapper) GetRunDefers(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]cqrs.RunDefer, error) {
	if len(runIDs) == 0 {
		return map[ulid.ULID][]cqrs.RunDefer{}, nil
	}

	spansByParent, err := w.GetSpansByRunIDsAndName(ctx, runIDs, meta.SpanNameDefer)
	if err != nil {
		return nil, fmt.Errorf("error loading executor.defer spans: %w", err)
	}

	out := make(map[ulid.ULID][]cqrs.RunDefer, len(spansByParent))
	// childRunIDByParentHashed maps each (parent, hashedID) defer to the child
	// run scheduled for it, recorded on the child-run-id executor.defer span at
	// child-schedule time. A defer with no such span yet (still pending) is
	// simply absent.
	childRunIDByParentHashed := make(map[ulid.ULID]map[string]ulid.ULID)
	for parentRunID, spans := range spansByParent {
		// A single defer can have two executor.defer spans: the schedule span
		// (defer.status = after_run) and a child-run-id span
		// (defer.child_run_id, no status) once its child run is scheduled.
		// Index by hashed ID so the schedule span and any later child-run-id
		// span share the same RunDefer entry. See defers/span.go.
		byHashedID := make(map[string]cqrs.RunDefer)
		for _, s := range spans {
			if s.Attributes.DeferHashedID == nil {
				continue
			}
			hashedID := *s.Attributes.DeferHashedID

			// The child-run-id span carries no status; it records the child run
			// scheduled for this defer. Capture it and move on.
			if s.Attributes.DeferChildRunID != nil {
				if childRunIDByParentHashed[parentRunID] == nil {
					childRunIDByParentHashed[parentRunID] = make(map[string]ulid.ULID)
				}
				childRunIDByParentHashed[parentRunID][hashedID] = *s.Attributes.DeferChildRunID
				continue
			}

			if s.Attributes.DeferStatus == nil {
				continue
			}
			status := *s.Attributes.DeferStatus
			// Defensively skip any status the GraphQL converter
			// (models.ToRunDeferStatus) can't surface. Today that's everything
			// except AfterRun and Rejected; an unrecognized status here would
			// otherwise propagate an error and fail the whole linkage query.
			if !isSurfaceableDeferStatus(status) {
				logger.StdlibLogger(ctx).Warn(
					"skipping defer span with unsurfaceable status",
					"run_id", parentRunID.String(),
					"hashed_id", hashedID,
					"status", status.String(),
				)
				continue
			}

			rd := cqrs.RunDefer{
				HashedDeferID: hashedID,
				Status:        status,
			}
			if s.Attributes.DeferUserlandID != nil {
				rd.UserlandDeferID = *s.Attributes.DeferUserlandID
			}
			if s.Attributes.DeferFnSlug != nil {
				rd.FnSlug = *s.Attributes.DeferFnSlug
			}

			byHashedID[hashedID] = rd
		}

		for _, rd := range byHashedID {
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

	// Attach the scheduled child run ID to each defer. The Run resolver
	// looks up the TraceRun lazily so the read path skips the join when
	// the caller doesn't ask for it.
	for parentRunID, defers := range out {
		for i := range defers {
			childRunID, ok := childRunIDByParentHashed[parentRunID][defers[i].HashedDeferID]
			if !ok {
				continue
			}
			id := childRunID
			defers[i].RunID = &id
		}
	}

	return out, nil
}

// isSurfaceableDeferStatus reports whether the GraphQL converter
// (models.ToRunDeferStatus) can map this status. That converter errors on
// anything other than AfterRun and Rejected, and the Defers resolver
// propagates that error — failing the entire run-linkage query. Keeping this
// in lockstep with the converter lets a single odd span be skipped instead of
// poisoning the whole query.
func isSurfaceableDeferStatus(s enums.DeferStatus) bool {
	switch s {
	case enums.DeferStatusAfterRun, enums.DeferStatusRejected:
		return true
	default:
		return false
	}
}

func (w wrapper) GetRunDeferredFrom(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]cqrs.RunDeferredFrom, error) {
	if len(runIDs) == 0 {
		return map[ulid.ULID][]cqrs.RunDeferredFrom{}, nil
	}

	// Parents are now recorded on the CHILD's own executor.run span via the
	// defer.parent_run_ids attribute, so an indexed lookup by run_id finds the
	// breadcrumb in one shot rather than scanning every parent's defer spans.
	spansByChild, err := w.GetSpansByRunIDsAndName(ctx, runIDs, meta.SpanNameRun)
	if err != nil {
		return nil, fmt.Errorf("error loading executor.run spans: %w", err)
	}

	out := make(map[ulid.ULID][]cqrs.RunDeferredFrom)
	// Collapse duplicate (child, parent) pairs in case the attribute lists the
	// same parent more than once (e.g. batched runs).
	seen := make(map[ulid.ULID]map[ulid.ULID]struct{})
	for childRunID, spans := range spansByChild {
		for _, span := range spans {
			if span.Attributes.DeferParentRunIDs == nil {
				continue
			}
			// All parents in a batch share a single fn slug (the implicit
			// batch key). The Function resolver looks it up lazily.
			var fnSlug string
			if span.Attributes.DeferParentFnSlug != nil {
				fnSlug = *span.Attributes.DeferParentFnSlug
			}
			for _, parentStr := range *span.Attributes.DeferParentRunIDs {
				parentRunID, err := ulid.Parse(parentStr)
				if err != nil {
					logger.StdlibLogger(ctx).Warn(
						"skipping unparseable defer parent run ID",
						"child_run_id", childRunID.String(),
						"parent_run_id", parentStr,
						"error", err,
					)
					continue
				}

				if seen[childRunID] == nil {
					seen[childRunID] = make(map[ulid.ULID]struct{})
				}
				if _, dup := seen[childRunID][parentRunID]; dup {
					continue
				}
				seen[childRunID][parentRunID] = struct{}{}

				out[childRunID] = append(out[childRunID], cqrs.RunDeferredFrom{
					RunID:  parentRunID,
					FnSlug: fnSlug,
				})
			}
		}
	}

	// Stable ordering so repeated queries return identical results.
	for _, rdfs := range out {
		slices.SortFunc(rdfs, func(a, b cqrs.RunDeferredFrom) int {
			return a.RunID.Compare(b.RunID)
		})
	}

	return out, nil
}
