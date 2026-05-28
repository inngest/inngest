package manager

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
)

// Run-to-run linkage (defers, deferred-from, invoked-from) is reconstructed
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
	// child-schedule time. A defer with no such span yet (still pending, or
	// aborted before scheduling) is simply absent.
	childRunIDByParentHashed := make(map[ulid.ULID]map[string]ulid.ULID)
	var allChildRunIDs []ulid.ULID
	for parentRunID, spans := range spansByParent {
		// A single defer can have several executor.defer spans: the original
		// schedule span (defer.status = after_run), a second abort span
		// (defer.status = aborted) if it was aborted, and a child-run-id span
		// (defer.child_run_id, no status) once its child run is scheduled.
		// Collapse the status-bearing spans by hashed ID — preferring the
		// terminal status so an aborted defer surfaces as Aborted not Scheduled
		// — and capture the child run ID separately. See defers/span.go.
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
				allChildRunIDs = append(allChildRunIDs, *s.Attributes.DeferChildRunID)
				continue
			}

			if s.Attributes.DeferStatus == nil {
				continue
			}
			status := *s.Attributes.DeferStatus
			// Defensively skip any status the GraphQL converter
			// (models.ToRunDeferStatus) can't surface. Today that's everything
			// except AfterRun and Aborted; an unrecognized status here would
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
			if s.Attributes.DeferUserID != nil {
				rd.UserDeferID = *s.Attributes.DeferUserID
			}
			if s.Attributes.DeferFnSlug != nil {
				rd.FnSlug = *s.Attributes.DeferFnSlug
			}

			existing, ok := byHashedID[hashedID]
			if !ok {
				byHashedID[hashedID] = rd
				continue
			}
			// Prefer the terminal (Aborted) status. The abort span only
			// carries hashed ID + status, so preserve the richer fields
			// (user/fn slug) from whichever span has them.
			merged := existing
			if deferStatusRank(status) > deferStatusRank(existing.Status) {
				merged.Status = status
			}
			if merged.UserDeferID == "" {
				merged.UserDeferID = rd.UserDeferID
			}
			if merged.FnSlug == "" {
				merged.FnSlug = rd.FnSlug
			}
			byHashedID[hashedID] = merged
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

	if len(allChildRunIDs) > 0 {
		childRuns, err := w.GetTraceRunsByRunIDs(ctx, allChildRunIDs)
		if err != nil {
			return nil, fmt.Errorf("error loading deferred child runs: %w", err)
		}
		for parentRunID, defers := range out {
			for i := range defers {
				childRunID, ok := childRunIDByParentHashed[parentRunID][defers[i].HashedDeferID]
				if !ok {
					continue
				}
				if run, ok := childRuns[childRunID]; ok {
					defers[i].Run = run
				}
			}
		}
	}

	return out, nil
}

// isSurfaceableDeferStatus reports whether the GraphQL converter
// (models.ToRunDeferStatus) can map this status. That converter errors on
// anything other than AfterRun and Aborted, and the Defers resolver propagates
// that error — failing the entire run-linkage query. Keeping this in lockstep
// with the converter lets a single odd span be skipped instead of poisoning
// the whole query.
func isSurfaceableDeferStatus(s enums.DeferStatus) bool {
	switch s {
	case enums.DeferStatusAfterRun, enums.DeferStatusAborted:
		return true
	default:
		return false
	}
}

// deferStatusRank orders the surfaceable defer statuses so collapsing by hashed
// ID prefers the terminal one. Aborted is terminal and wins over the
// still-scheduled AfterRun.
func deferStatusRank(s enums.DeferStatus) int {
	switch s {
	case enums.DeferStatusAborted:
		return 1
	default:
		return 0
	}
}

func (w wrapper) GetRunDeferredFrom(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]*cqrs.RunDeferredFrom, error) {
	if len(runIDs) == 0 {
		return map[ulid.ULID][]*cqrs.RunDeferredFrom{}, nil
	}

	// Parents are now recorded on the CHILD's own executor.run span via the
	// defer.parent_run_ids attribute, so an indexed lookup by run_id finds the
	// breadcrumb in one shot rather than scanning every parent's defer spans.
	spansByChild, err := w.GetSpansByRunIDsAndName(ctx, runIDs, meta.SpanNameRun)
	if err != nil {
		return nil, fmt.Errorf("error loading executor.run spans: %w", err)
	}

	out := make(map[ulid.ULID][]*cqrs.RunDeferredFrom)
	// Collapse duplicate (child, parent) pairs in case the attribute lists the
	// same parent more than once (e.g. batched runs).
	seen := make(map[ulid.ULID]map[ulid.ULID]struct{})
	parentRunIDSet := make(map[ulid.ULID]struct{})
	for childRunID, spans := range spansByChild {
		for _, span := range spans {
			if span.Attributes.DeferParentRunIDs == nil {
				continue
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

				out[childRunID] = append(out[childRunID], &cqrs.RunDeferredFrom{ParentRunID: parentRunID})
				parentRunIDSet[parentRunID] = struct{}{}
			}
		}
	}

	if len(parentRunIDSet) > 0 {
		parentRuns, err := w.GetTraceRunsByRunIDs(ctx, slices.Collect(maps.Keys(parentRunIDSet)))
		if err != nil {
			return nil, fmt.Errorf("error loading deferred-from parent runs: %w", err)
		}
		for _, rdfs := range out {
			for _, rdf := range rdfs {
				if run, ok := parentRuns[rdf.ParentRunID]; ok {
					rdf.ParentRun = run
				}
			}
		}
	}

	// Stable ordering so repeated queries return identical results.
	for _, rdfs := range out {
		slices.SortFunc(rdfs, func(a, b *cqrs.RunDeferredFrom) int {
			return a.ParentRunID.Compare(b.ParentRunID)
		})
	}

	return out, nil
}

func (w wrapper) GetRunInvokedFrom(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID]*cqrs.RunInvokedFrom, error) {
	if len(runIDs) == 0 {
		return map[ulid.ULID]*cqrs.RunInvokedFrom{}, nil
	}

	strIDs := make([]string, len(runIDs))
	for i, id := range runIDs {
		strIDs[i] = id.String()
	}

	rows, err := w.queryInvokedFromSpans(ctx, strIDs)
	if err != nil {
		return nil, fmt.Errorf("error loading invoked-from spans: %w", err)
	}

	out := make(map[ulid.ULID]*cqrs.RunInvokedFrom)
	parentRunIDSet := make(map[ulid.ULID]struct{})
	for _, row := range rows {
		span, err := mapSpanFromRow(ctx, row, nil)
		if err != nil {
			return nil, err
		}
		if span.Attributes.StepInvokeRunID == nil {
			continue
		}
		childRunID := *span.Attributes.StepInvokeRunID
		// The query orders by run_id then start_time; the first matching
		// fragment group wins so a child appears at most once.
		if _, exists := out[childRunID]; exists {
			continue
		}

		parentRunID := span.RunID
		out[childRunID] = &cqrs.RunInvokedFrom{
			ParentRunID: parentRunID,
			StepName:    span.Attributes.StepName,
		}
		parentRunIDSet[parentRunID] = struct{}{}
	}

	if len(parentRunIDSet) > 0 {
		parentRuns, err := w.GetTraceRunsByRunIDs(ctx, slices.Collect(maps.Keys(parentRunIDSet)))
		if err != nil {
			return nil, fmt.Errorf("error loading invoked-from parent runs: %w", err)
		}
		for _, rif := range out {
			if run, ok := parentRuns[rif.ParentRunID]; ok {
				rif.ParentRun = run
			}
		}
	}

	return out, nil
}

// Time and fragment columns stay as `any` so mapSpanFromRow's existing
// decoders own the parsing.
type linkageSpanRow struct {
	runID         string
	traceID       string
	dynamicSpanID sql.NullString
	startTime     any
	endTime       any
	parentSpanID  sql.NullString
	spanFragments any
}

func (r *linkageSpanRow) GetTraceID() string               { return r.traceID }
func (r *linkageSpanRow) GetRunID() string                 { return r.runID }
func (r *linkageSpanRow) GetDynamicSpanID() sql.NullString { return r.dynamicSpanID }
func (r *linkageSpanRow) GetParentSpanID() sql.NullString  { return r.parentSpanID }
func (r *linkageSpanRow) GetStartTime() any                { return r.startTime }
func (r *linkageSpanRow) GetEndTime() any                  { return r.endTime }
func (r *linkageSpanRow) GetSpanFragments() any            { return r.spanFragments }

var _ normalizedSpan = (*linkageSpanRow)(nil)

// queryInvokedFromSpans finds, for each child run ID, the parent's invoke
// linkage: the invoked run ID lives on an EXTEND fragment (written via
// UpdateSpan), so it matches step.invoke.run.id on name = 'EXTEND'.
func (w wrapper) queryInvokedFromSpans(ctx context.Context, childRunIDs []string) ([]*linkageSpanRow, error) {
	return w.queryLinkageSpansByAttr(ctx, meta.SpanNameDynamicExtension, meta.Attrs.StepInvokeRunID.Key(), childRunIDs)
}

// queryLinkageSpansByAttr returns the spans whose dynamic_span_id matches a span
// named spanName carrying attrKey ∈ runIDs, with all fragments aggregated for
// mapSpanFromRow's read-time merge. Both reverse-linkage queries (invoked-from,
// deferred-from) share this shape and differ only in the span name and the
// attribute matched.
//
// Hand-rolled SQL: sqlc 1.30's SQLite parser can't bind a slice when the LHS is
// a json_extract(...) call. Narrowing the inner subquery on name keeps the JSON
// predicate off the full spans table.
func (w wrapper) queryLinkageSpansByAttr(ctx context.Context, spanName, attrKey string, runIDs []string) ([]*linkageSpanRow, error) {
	var (
		nameArg      string
		placeholders = make([]string, len(runIDs))
		jsonPath     string
		jsonObject   string
		jsonAgg      string
	)
	switch dialect := w.dialect(); dialect {
	case "postgres":
		nameArg = "$1"
		for i := range runIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
		}
		jsonPath = fmt.Sprintf(`attributes->>'%s'`, attrKey)
		jsonObject = "json_build_object"
		jsonAgg = "json_agg"
	case "sqlite3":
		nameArg = "?"
		for i := range runIDs {
			placeholders[i] = "?"
		}
		jsonPath = fmt.Sprintf(`json_extract(attributes, '$."%s"')`, attrKey)
		jsonObject = "json_object"
		jsonAgg = "json_group_array"
	default:
		return nil, fmt.Errorf("queryLinkageSpansByAttr: unsupported dialect %q", dialect)
	}

	args := make([]any, 0, len(runIDs)+1)
	args = append(args, spanName)
	for _, id := range runIDs {
		args = append(args, id)
	}

	inList := strings.Join(placeholders, ",")
	query := `
SELECT
  run_id,
  trace_id,
  dynamic_span_id,
  MIN(start_time) AS start_time,
  MAX(end_time) AS end_time,
  parent_span_id,
  ` + jsonAgg + `(` + jsonObject + `(
    'span_id', span_id,
    'name', name,
    'attributes', attributes,
    'links', links,
    'output_span_id', CASE WHEN output IS NOT NULL THEN span_id ELSE NULL END,
    'input_span_id', CASE WHEN input IS NOT NULL THEN span_id ELSE NULL END
  )) AS span_fragments
FROM spans
WHERE dynamic_span_id IN (
  SELECT dynamic_span_id
  FROM spans
  WHERE name = ` + nameArg + ` AND ` + jsonPath + ` IN (` + inList + `)
)
GROUP BY run_id, trace_id, dynamic_span_id, parent_span_id
ORDER BY run_id, start_time
`

	sqlRows, err := w.adapter.Conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer sqlRows.Close()

	var out []*linkageSpanRow
	for sqlRows.Next() {
		r := &linkageSpanRow{}
		if err := sqlRows.Scan(
			&r.runID,
			&r.traceID,
			&r.dynamicSpanID,
			&r.startTime,
			&r.endTime,
			&r.parentSpanID,
			&r.spanFragments,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := sqlRows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
