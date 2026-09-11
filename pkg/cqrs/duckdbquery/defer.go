package duckdbquery

import (
	"cmp"
	"context"
	"database/sql"
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

// GetRunDefers returns the deferred child runs each of runIDs scheduled,
// reconstructed from run_trace_spans' executor.defer rows. Multiple
// physical rows can back one logical defer (an Add row, then later an
// Abort row), so rows are grouped by the defer.hashed_id attribute, which
// is already unique per (run_id, defer).
//
// The merge happens in SQL: LAST_VALUE(... IGNORE NULLS) OVER a
// (run_id, hashed_id)-partitioned, start_time-ordered window forward-fills
// each field from the most recent row that set it, so an Abort row's
// absent fn_slug/userland_id doesn't blank out values an earlier Add row
// set, while its status still overwrites Add's. QUALIFY ROW_NUMBER() = 1
// then keeps only the final row per group.
//
// The child run ID is resolved separately by resolveDeferChildRunIDs, from
// each defer's own defer.event_id, rather than a value some later write
// stamps back onto this span once the child schedules.
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

	hashedIDExpr := fmt.Sprintf("attributes->>'%s'", meta.Attrs.DeferHashedID.Key())
	userlandIDExpr := fmt.Sprintf("attributes->>'%s'", meta.Attrs.DeferUserlandID.Key())
	fnSlugExpr := fmt.Sprintf("attributes->>'%s'", meta.Attrs.DeferFnSlug.Key())
	statusExpr := fmt.Sprintf("attributes->>'%s'", meta.Attrs.DeferStatus.Key())
	eventIDExpr := fmt.Sprintf("attributes->>'%s'", meta.Attrs.DeferEventID.Key())

	// hashed_id IS NOT NULL excludes any row that isn't ours; every defer
	// span sets it unconditionally.
	query := fmt.Sprintf(`
SELECT
  run_id,
  %s AS hashed_id,
  LAST_VALUE(%s IGNORE NULLS) OVER w AS userland_id,
  LAST_VALUE(%s IGNORE NULLS) OVER w AS fn_slug,
  LAST_VALUE(%s IGNORE NULLS) OVER w AS status,
  LAST_VALUE(%s IGNORE NULLS) OVER w AS event_id
FROM %s.run_trace_spans
WHERE name = ? AND run_id IN (%s) AND %s IS NOT NULL
WINDOW w AS (PARTITION BY run_id, %s ORDER BY start_time ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW)
QUALIFY ROW_NUMBER() OVER (PARTITION BY run_id, %s ORDER BY start_time DESC) = 1;`,
		hashedIDExpr, userlandIDExpr, fnSlugExpr, statusExpr, eventIDExpr,
		duckdb.DuckLakeAlias, strings.Join(placeholders, ", "), hashedIDExpr,
		hashedIDExpr, hashedIDExpr,
	)
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying defer spans: %w", err)
	}
	defer rows.Close()

	// pendingChildLink batches resolveDeferChildRunIDs into one lookup after
	// all rows are read, then patches RunID back onto its out[runID][idx] slot.
	type pendingChildLink struct {
		runID   ulid.ULID
		idx     int
		eventID ulid.ULID
	}
	var pending []pendingChildLink
	seenEventIDs := make(map[ulid.ULID]struct{})

	out := make(map[ulid.ULID][]cqrs.RunDefer)
	for rows.Next() {
		var rawRunID any
		var hashedID string
		var userlandID, fnSlug, status, eventID sql.NullString
		if err := rows.Scan(&rawRunID, &hashedID, &userlandID, &fnSlug, &status, &eventID); err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning defer span row: %w", err)
		}
		runID, err := ulidColumn(rawRunID, "run_id")
		if err != nil {
			return nil, err
		}
		// No row for this hashed_id set a status — shouldn't happen, but
		// skip rather than surface a half-formed defer.
		if !status.Valid {
			continue
		}
		deferStatus, err := enums.DeferStatusString(status.String)
		if err != nil {
			return nil, fmt.Errorf("duckdbquery: parsing defer.status %q: %w", status.String, err)
		}

		out[runID] = append(out[runID], cqrs.RunDefer{
			HashedDeferID:   hashedID,
			UserlandDeferID: userlandID.String,
			FnSlug:          fnSlug.String,
			Status:          deferStatus,
		})

		if eventID.Valid {
			eid, err := ulid.Parse(eventID.String)
			if err == nil {
				seenEventIDs[eid] = struct{}{}
				pending = append(pending, pendingChildLink{runID: runID, idx: len(out[runID]) - 1, eventID: eid})
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbquery: reading defer span rows: %w", err)
	}

	if len(pending) > 0 {
		childRunIDs, err := m.resolveDeferChildRunIDs(ctx, seenEventIDs)
		if err != nil {
			return nil, err
		}
		for _, p := range pending {
			if id, ok := childRunIDs[p.eventID]; ok {
				out[p.runID][p.idx].RunID = &id
			}
		}
	}

	// Row order within a run_id partition isn't guaranteed across calls;
	// sort by HashedDeferID for a stable result.
	for runID := range out {
		slices.SortFunc(out[runID], func(a, b cqrs.RunDefer) int {
			return cmp.Compare(a.HashedDeferID, b.HashedDeferID)
		})
	}
	return out, nil
}

// resolveDeferChildRunIDs resolves every event ID in eventIDs to the run
// inngest.runs recorded as triggered by that event. A deferred child is
// scheduled off the deterministic event.DeferEventID(parentRunID, hashedID),
// so checking inngest.runs.event_ids for that ID is exactly the
// parent->child link. A defer whose event was never published or scheduled
// yet simply has no matching run.
func (m *Manager) resolveDeferChildRunIDs(ctx context.Context, eventIDs map[ulid.ULID]struct{}) (map[ulid.ULID]ulid.ULID, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}

	// list_has_any(event_ids, ?) checks the whole search list in one call;
	// encodeLiteral encodes the bound []string as a DuckDB array literal.
	ids := make([]string, 0, len(eventIDs))
	for id := range eventIDs {
		ids = append(ids, id.String())
	}
	query := fmt.Sprintf(
		"SELECT run_id, event_ids FROM %s.runs WHERE list_has_any(event_ids, ?);",
		duckdb.DuckLakeAlias,
	)
	rows, err := m.db.QueryContext(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("duckdbquery: querying runs for defer child linkage: %w", err)
	}
	defer rows.Close()

	out := make(map[ulid.ULID]ulid.ULID, len(eventIDs))
	for rows.Next() {
		var rawRunID, rawEventIDs any
		if err := rows.Scan(&rawRunID, &rawEventIDs); err != nil {
			return nil, fmt.Errorf("duckdbquery: scanning run row for defer child linkage: %w", err)
		}
		runID, err := ulidColumn(rawRunID, "run_id")
		if err != nil {
			return nil, err
		}
		// event_ids is a VARCHAR[] (NULL for cron-only runs), decoded as []any.
		items, ok := rawEventIDs.([]any)
		if !ok {
			continue
		}
		for _, v := range items {
			s, ok := v.(string)
			if !ok {
				continue
			}
			eid, err := ulid.Parse(s)
			if err != nil {
				continue
			}
			if _, ok := eventIDs[eid]; !ok {
				continue
			}
			out[eid] = runID
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("duckdbquery: reading run rows for defer child linkage: %w", err)
	}
	return out, nil
}

// GetRunDeferredFrom returns the parent run(s) that scheduled each of
// runIDs via defer() — read from the child's own executor.run.queued
// marker span, which OnFunctionScheduled stamps with
// DeferParentRunIDs/DeferParentFnSlug once at schedule time. Unlike
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
