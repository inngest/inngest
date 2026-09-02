package duckdbquery

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db/duckdb"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	tracingv3 "github.com/inngest/inngest/pkg/tracing/v3"
	"github.com/oklog/ulid/v2"
)

// deferSpanRow is the set of defer.* fields present on one physical
// inngest.run_trace_spans row, tracking which fields were actually present
// in that row's attributes JSON (vs. absent) so mergeDeferSpanRow can
// distinguish "this row didn't set this field" from "this row cleared it" —
// the latter never happens in practice (defer.* attrs are only ever added,
// never blanked), but the distinction is what makes the merge correct.
type deferSpanRow struct {
	hashedID      string
	hasHashedID   bool
	userlandID    string
	hasUserlandID bool
	fnSlug        string
	hasFnSlug     bool
	status        enums.DeferStatus
	hasStatus     bool
	childRunID    *ulid.ULID
}

// mergeDeferSpanRow folds next's present fields onto acc, leaving any field
// next doesn't set untouched — the same semantics
// pkg/cqrs/manager/cqrs.go's mapSpanFromRow uses to reconstruct a logical
// span from its physical dynamic_span_id fragments (maps.Copy(dst, src)
// only overwrites keys src actually has). Callers must fold rows in
// write-chronological order (oldest first) so a later row's fields
// correctly take precedence — e.g. an Abort row's status=Aborted must win
// over an earlier Add row's status=AfterRun, while the Abort row's absent
// fn_slug/userland_id must NOT blank out the Add row's values.
func mergeDeferSpanRow(acc, next deferSpanRow) deferSpanRow {
	if next.hasHashedID {
		acc.hashedID, acc.hasHashedID = next.hashedID, true
	}
	if next.hasUserlandID {
		acc.userlandID, acc.hasUserlandID = next.userlandID, true
	}
	if next.hasFnSlug {
		acc.fnSlug, acc.hasFnSlug = next.fnSlug, true
	}
	if next.hasStatus {
		acc.status, acc.hasStatus = next.status, true
	}
	if next.childRunID != nil {
		acc.childRunID = next.childRunID
	}
	return acc
}

func scanDeferSpanRow(rawAttrs any) (deferSpanRow, error) {
	attrs, err := asMap(rawAttrs, "attributes")
	if err != nil {
		return deferSpanRow{}, err
	}

	var row deferSpanRow
	if v, ok := attrs[meta.Attrs.DeferHashedID.Key()]; ok {
		s, ok := v.(string)
		if !ok {
			return deferSpanRow{}, fmt.Errorf("duckdbquery: expected string for defer.hashed_id, got %T", v)
		}
		row.hashedID, row.hasHashedID = s, true
	}
	if v, ok := attrs[meta.Attrs.DeferUserlandID.Key()]; ok {
		s, ok := v.(string)
		if !ok {
			return deferSpanRow{}, fmt.Errorf("duckdbquery: expected string for defer.userland_id, got %T", v)
		}
		row.userlandID, row.hasUserlandID = s, true
	}
	if v, ok := attrs[meta.Attrs.DeferFnSlug.Key()]; ok {
		s, ok := v.(string)
		if !ok {
			return deferSpanRow{}, fmt.Errorf("duckdbquery: expected string for defer.fn_slug, got %T", v)
		}
		row.fnSlug, row.hasFnSlug = s, true
	}
	if v, ok := attrs[meta.Attrs.DeferStatus.Key()]; ok {
		s, ok := v.(string)
		if !ok {
			return deferSpanRow{}, fmt.Errorf("duckdbquery: expected string for defer.status, got %T", v)
		}
		status, err := enums.DeferStatusString(s)
		if err != nil {
			return deferSpanRow{}, fmt.Errorf("duckdbquery: parsing defer.status %q: %w", s, err)
		}
		row.status, row.hasStatus = status, true
	}
	if v, ok := attrs[meta.Attrs.DeferChildRunID.Key()]; ok {
		s, ok := v.(string)
		if !ok {
			return deferSpanRow{}, fmt.Errorf("duckdbquery: expected string for defer.child_run_id, got %T", v)
		}
		id, err := ulid.Parse(s)
		if err != nil {
			return deferSpanRow{}, fmt.Errorf("duckdbquery: parsing defer.child_run_id %q: %w", s, err)
		}
		row.childRunID = &id
	}
	return row, nil
}

// GetRunDefers returns the deferred child runs each of runIDs scheduled,
// reconstructed from run_trace_spans' executor.defer rows by merging every
// physical row sharing a (run_id, span_id) group field-by-field — the
// DuckDB-flat-model counterpart to pkg/cqrs/manager's own GetRunDefers,
// which reconstructs a logical span the same way from SQLite/Postgres's
// dynamic-span fragments (mapSpanFromRow's maps.Copy per fragment). This
// keeps a field an earlier write set (e.g. Add's fn_slug/userland_id) even
// once a later write (Abort, or the schedule-link update) only sets a
// different field (status, child_run_id) — see mergeDeferSpanRow. Merge
// happens in Go, not SQL: defer.* attributes live inside the attributes
// JSON blob, and a run has at most consts.MaxDefersPerRun (20) distinct
// defers, so the per-call result set is small enough that a SQL JSON-path
// merge isn't worth the complexity.
func (m *Manager) GetRunDefers(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]cqrs.RunDefer, error) {
	if len(runIDs) == 0 {
		return map[ulid.ULID][]cqrs.RunDefer{}, nil
	}

	placeholders := make([]string, len(runIDs))
	args := make([]any, len(runIDs)+1)
	args[0] = meta.SpanNameDefer
	for i, id := range runIDs {
		placeholders[i] = "?"
		args[i+1] = id.String()
	}

	// ORDER BY start_time ASC is load-bearing: mergeDeferSpanRow must fold
	// rows oldest-first so a later write's fields correctly take precedence
	// over an earlier one's. Add always physically precedes Abort/the
	// schedule-link update for the same defer, so this ordering is reliable.
	query := fmt.Sprintf(
		"SELECT run_id, span_id, attributes FROM %s.run_trace_spans WHERE name = ? AND run_id IN (%s) ORDER BY start_time ASC;",
		duckdb.DuckLakeAlias, strings.Join(placeholders, ", "),
	)
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying defer spans: %w", err)
	}
	defer rows.Close()

	// merged[runID][spanID] accumulates the field-level merge of every row
	// seen so far for each logical defer.
	merged := make(map[ulid.ULID]map[string]deferSpanRow)
	for rows.Next() {
		var rawRunID, rawSpanID, rawAttrs any
		if err := rows.Scan(&rawRunID, &rawSpanID, &rawAttrs); err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning defer span row: %w", err)
		}
		runID, err := ulidColumn(rawRunID, "run_id")
		if err != nil {
			return nil, err
		}
		spanID, err := asString(rawSpanID, "span_id")
		if err != nil {
			return nil, err
		}
		row, err := scanDeferSpanRow(rawAttrs)
		if err != nil {
			return nil, err
		}

		if merged[runID] == nil {
			merged[runID] = make(map[string]deferSpanRow)
		}
		merged[runID][spanID] = mergeDeferSpanRow(merged[runID][spanID], row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbquery: reading defer span rows: %w", err)
	}

	out := make(map[ulid.ULID][]cqrs.RunDefer, len(merged))
	for runID, bySpan := range merged {
		defers := make([]cqrs.RunDefer, 0, len(bySpan))
		for _, row := range bySpan {
			if !row.hasHashedID || !row.hasStatus {
				continue
			}
			defers = append(defers, cqrs.RunDefer{
				HashedDeferID:   row.hashedID,
				UserlandDeferID: row.userlandID,
				FnSlug:          row.fnSlug,
				Status:          row.status,
				RunID:           row.childRunID,
			})
		}
		// Map iteration above is non-deterministic; sort by HashedDeferID so
		// repeated queries return identical orderings — matches
		// pkg/cqrs/manager's own GetRunDefers.
		slices.SortFunc(defers, func(a, b cqrs.RunDefer) int {
			return cmp.Compare(a.HashedDeferID, b.HashedDeferID)
		})
		out[runID] = defers
	}
	return out, nil
}

// GetRunDeferredFrom returns the parent run(s) that scheduled each of
// runIDs via defer() — read from the child's own executor.run.queued
// marker span, which pkg/execution/dualwrite's OnFunctionScheduled stamps
// with DeferParentRunIDs/DeferParentFnSlug exactly once, at schedule time,
// for every deferred child (see that hook's addDeferParentAttrs). Unlike
// GetRunDefers, this needs no collapse: those attrs are written once and
// never revised.
func (m *Manager) GetRunDeferredFrom(ctx context.Context, runIDs []ulid.ULID) (map[ulid.ULID][]cqrs.RunDeferredFrom, error) {
	if len(runIDs) == 0 {
		return map[ulid.ULID][]cqrs.RunDeferredFrom{}, nil
	}

	placeholders := make([]string, len(runIDs))
	args := make([]any, len(runIDs)+1)
	args[0] = tracingv3.SpanNameRunQueued
	for i, id := range runIDs {
		placeholders[i] = "?"
		args[i+1] = id.String()
	}

	query := fmt.Sprintf(
		"SELECT run_id, attributes FROM %s.run_trace_spans WHERE name = ? AND run_id IN (%s);",
		duckdb.DuckLakeAlias, strings.Join(placeholders, ", "),
	)
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying run-queued spans for defer linkage: %w", err)
	}
	defer rows.Close()

	out := make(map[ulid.ULID][]cqrs.RunDeferredFrom)
	for rows.Next() {
		var rawRunID, rawAttrs any
		if err := rows.Scan(&rawRunID, &rawAttrs); err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning run-queued span row: %w", err)
		}
		runID, err := ulidColumn(rawRunID, "run_id")
		if err != nil {
			return nil, err
		}
		attrs, err := asMap(rawAttrs, "attributes")
		if err != nil {
			return nil, err
		}

		fnSlug, _ := attrs[meta.Attrs.DeferParentFnSlug.Key()].(string)
		rawParentIDs, ok := attrs[meta.Attrs.DeferParentRunIDs.Key()].([]any)
		if fnSlug == "" || !ok || len(rawParentIDs) == 0 {
			continue
		}

		parents := make([]cqrs.RunDeferredFrom, 0, len(rawParentIDs))
		for _, v := range rawParentIDs {
			s, ok := v.(string)
			if !ok {
				continue
			}
			parentRunID, err := ulid.Parse(s)
			if err != nil {
				continue
			}
			parents = append(parents, cqrs.RunDeferredFrom{RunID: parentRunID, FnSlug: fnSlug})
		}
		if len(parents) > 0 {
			out[runID] = parents
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbquery: reading run-queued span rows: %w", err)
	}
	return out, nil
}
