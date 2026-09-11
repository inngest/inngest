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
// physical rows can back one logical defer (an Add row, and later an Abort
// row from OnDeferAbort) — run_trace_spans requires span_id stay unique per
// row (see OnDeferAdd/OnDeferAbort's own doc comments), so those rows are
// grouped by the defer.hashed_id attribute instead, which is already unique
// per (run_id, defer) with no extra dynamic-span-identity attribute needed
// the way SQLite/Postgres's own dynamic_span_id column requires (see
// pkg/cqrs/manager's own GetRunDefers, which merges dynamic_span_id
// fragments the same conceptual way via mapSpanFromRow).
//
// The merge itself happens entirely in SQL: LAST_VALUE(... IGNORE NULLS)
// OVER a (run_id, hashed_id)-partitioned, start_time-ordered window
// forward-fills each field from the most recent row that actually set it —
// so an Abort row's absent fn_slug/userland_id never blanks out the values
// an earlier Add row set, while its status=Aborted still overwrites Add's
// status=AfterRun — and QUALIFY's ROW_NUMBER() = 1 then keeps only the
// final, fully-forward-filled row per group. defer.* attributes live inside
// the attributes JSON blob and a run has at most consts.MaxDefersPerRun (20)
// distinct defers, so this costs nothing an equivalent Go-side fold
// wouldn't, without needing to buffer or reconstruct rows in Go at all.
//
// The child run ID isn't one of the merged fields: it's resolved separately,
// by resolveDeferChildRunIDs, from each defer's own defer.event_id (present
// from the defer's very first Add row onward, since it's a pure function of
// (parent run ID, hashedID) — see event.DeferEventID) rather than a value
// some later write has to stamp back onto this span once/if the child
// actually gets scheduled.
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

	// hashed_id IS NOT NULL pre-window excludes any row that isn't one of
	// ours: every span OnDeferAdd/OnDeferAbort create sets defer.hashed_id
	// unconditionally, so its absence means this row belongs to some other
	// (impossible, given name = ?) writer.
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

	// pendingChildLink defers resolveDeferChildRunIDs's one batched lookup
	// until every defer row has been read, then patches RunID back onto the
	// exact out[runID][idx] slot it came from.
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
		// No row for this hashed_id ever set a status — shouldn't happen
		// (OnDeferAdd always sets it), but skip rather than surface a
		// half-formed defer.
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

	// Row order within a (run_id) partition isn't guaranteed across calls;
	// sort by HashedDeferID so repeated queries return identical orderings —
	// matches pkg/cqrs/manager's own GetRunDefers.
	for runID := range out {
		slices.SortFunc(out[runID], func(a, b cqrs.RunDefer) int {
			return cmp.Compare(a.HashedDeferID, b.HashedDeferID)
		})
	}
	return out, nil
}

// resolveDeferChildRunIDs resolves every event ID in eventIDs to the run
// inngest.runs recorded as triggered by that event. A deferred child is
// scheduled directly off the inngest/deferred.schedule event whose
// deterministic ID event.DeferEventID(parentRunID, hashedID) computes, so
// checking inngest.runs.event_ids for that same ID (the list_contains
// pattern latestRunsCTE already uses for GetTraceRunFilter.EventID) is
// exactly the parent->child link — nothing needs to be written back onto
// the parent's own span once the child schedules. A defer whose event was
// never published (e.g. Aborted before scheduling) or whose child hasn't
// been scheduled yet simply has no matching run, the same "nil RunID"
// result GetRunDefers's caller already expects.
func (m *Manager) resolveDeferChildRunIDs(ctx context.Context, eventIDs map[ulid.ULID]struct{}) (map[ulid.ULID]ulid.ULID, error) {
	if len(eventIDs) == 0 {
		return nil, nil
	}

	// list_has_any(event_ids, ?) checks the whole search list in one call —
	// pkg/db/duckdb/literal.go's encodeLiteral has a []string case that
	// encodes a bound Go slice as a real DuckDB array literal ('['a', 'b']'),
	// so a single []string arg works fine here; no per-ID list_contains OR
	// chain is needed.
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
		// event_ids is a real VARCHAR[] (NULL for cron-only runs) — the
		// driver hands it back as []any regardless of transport (see
		// runs.go's own event_ids decode for the same shape).
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
