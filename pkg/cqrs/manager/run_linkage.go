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
	"github.com/inngest/inngest/pkg/util"
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
	var allChildRunIDs []ulid.ULID
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
			}
			if s.Attributes.DeferUserID != nil {
				rd.UserDeferID = *s.Attributes.DeferUserID
			}
			if s.Attributes.DeferFnSlug != nil {
				rd.FnSlug = *s.Attributes.DeferFnSlug
			}
			out[parentRunID] = append(out[parentRunID], rd)
			allChildRunIDs = append(allChildRunIDs, util.DeterministicChildRunID(parentRunID, rd.HashedDeferID))
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
		// Hydrate by recomputing the deterministic child run ID from each
		// defer's HashedDeferID, so this loop doesn't depend on the order
		// defers were appended above.
		for parentRunID, defers := range out {
			for i := range defers {
				childRunID := util.DeterministicChildRunID(parentRunID, defers[i].HashedDeferID)
				if run, ok := childRuns[childRunID]; ok {
					defers[i].Run = run
				}
			}
		}
	}

	return out, nil
}

func (w wrapper) GetRunDeferredFrom(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID]*cqrs.RunDeferredFrom, error) {
	if len(runIDs) == 0 {
		return map[ulid.ULID]*cqrs.RunDeferredFrom{}, nil
	}

	spansByChild, err := w.GetSpansByRunIDsAndName(ctx, runIDs, meta.SpanNameRun)
	if err != nil {
		return nil, fmt.Errorf("error loading child executor.run spans: %w", err)
	}

	out := make(map[ulid.ULID]*cqrs.RunDeferredFrom, len(spansByChild))
	parentRunIDSet := make(map[ulid.ULID]struct{})
	for childRunID, spans := range spansByChild {
		// Defer linkage attrs live on the root executor.run span only;
		// extension spans share the dynamic_span_id but lack them.
		for _, s := range spans {
			if !s.GetIsRoot() || s.Attributes.DeferParentRunID == nil {
				continue
			}
			out[childRunID] = &cqrs.RunDeferredFrom{ParentRunID: *s.Attributes.DeferParentRunID}
			parentRunIDSet[*s.Attributes.DeferParentRunID] = struct{}{}
			break
		}
	}

	if len(parentRunIDSet) > 0 {
		parentRuns, err := w.GetTraceRunsByRunIDs(ctx, slices.Collect(maps.Keys(parentRunIDSet)))
		if err != nil {
			return nil, fmt.Errorf("error loading deferred-from parent runs: %w", err)
		}
		for _, rdf := range out {
			if run, ok := parentRuns[rdf.ParentRunID]; ok {
				rdf.ParentRun = run
			}
		}
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
type invokedFromSpanRow struct {
	runID         string
	traceID       string
	dynamicSpanID sql.NullString
	startTime     any
	endTime       any
	parentSpanID  sql.NullString
	spanFragments any
}

func (r *invokedFromSpanRow) GetTraceID() string               { return r.traceID }
func (r *invokedFromSpanRow) GetRunID() string                 { return r.runID }
func (r *invokedFromSpanRow) GetDynamicSpanID() sql.NullString { return r.dynamicSpanID }
func (r *invokedFromSpanRow) GetParentSpanID() sql.NullString  { return r.parentSpanID }
func (r *invokedFromSpanRow) GetStartTime() any                { return r.startTime }
func (r *invokedFromSpanRow) GetEndTime() any                  { return r.endTime }
func (r *invokedFromSpanRow) GetSpanFragments() any            { return r.spanFragments }

var _ normalizedSpan = (*invokedFromSpanRow)(nil)

// Hand-rolled SQL: sqlc 1.30's SQLite parser can't bind a slice when the
// LHS is a json_extract(...) call. Narrowing on `name = 'EXTEND'` in the
// inner subquery keeps the JSON predicate off the full spans table.
func (w wrapper) queryInvokedFromSpans(ctx context.Context, childRunIDs []string) ([]*invokedFromSpanRow, error) {
	stepInvokeRunIDKey := meta.Attrs.StepInvokeRunID.Key()

	var (
		nameArg      string
		placeholders = make([]string, len(childRunIDs))
		jsonPath     string
		jsonObject   string
		jsonAgg      string
	)
	switch dialect := w.dialect(); dialect {
	case "postgres":
		nameArg = "$1"
		for i := range childRunIDs {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
		}
		jsonPath = fmt.Sprintf(`attributes->>'%s'`, stepInvokeRunIDKey)
		jsonObject = "json_build_object"
		jsonAgg = "json_agg"
	case "sqlite3":
		nameArg = "?"
		for i := range childRunIDs {
			placeholders[i] = "?"
		}
		jsonPath = fmt.Sprintf(`json_extract(attributes, '$."%s"')`, stepInvokeRunIDKey)
		jsonObject = "json_object"
		jsonAgg = "json_group_array"
	default:
		return nil, fmt.Errorf("queryInvokedFromSpans: unsupported dialect %q", dialect)
	}

	args := make([]any, 0, len(childRunIDs)+1)
	args = append(args, meta.SpanNameDynamicExtension)
	for _, id := range childRunIDs {
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

	var out []*invokedFromSpanRow
	for sqlRows.Next() {
		r := &invokedFromSpanRow{}
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
