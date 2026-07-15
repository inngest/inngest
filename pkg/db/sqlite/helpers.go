package sqlite

import (
	"encoding/json"
	"strings"
	"time"

	sq "github.com/doug-martin/goqu/v9"
	sqexp "github.com/doug-martin/goqu/v9/exp"
	"github.com/inngest/inngest/pkg/dateutil"
	"github.com/inngest/inngest/pkg/db/driverhelp"
	"github.com/inngest/inngest/pkg/run"
)

var _ driverhelp.DialectHelpers = (*helpers)(nil)

type helpers struct{}

func (h *helpers) GoquDialect() string { return "sqlite3" }

func (h *helpers) CELConverter() run.ExprSQLConverter { return run.SpanEventSQLiteConverter }

func (h *helpers) EventIDsExpr() sqexp.Expression {
	return sq.L("MAX(spans.event_ids)").As("event_ids")
}

func (h *helpers) RootEventIDsExpr() sqexp.Expression {
	return sq.L("spans.event_ids").As("event_ids")
}

func (h *helpers) EventIDsContain(ids []string) sqexp.Expression {
	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	return sq.L(
		"EXISTS (SELECT 1 FROM json_each(NULLIF(spans.event_ids, '')) WHERE value IN ("+placeholders+"))",
		values...,
	)
}

func (h *helpers) RunOutputExpr() sqexp.Expression {
	return sq.L(`COALESCE((
		SELECT CAST(output_lookup.output AS TEXT)
		FROM spans output_lookup
		WHERE output_lookup.run_id = spans.run_id
			AND output_lookup.output IS NOT NULL
			AND output_lookup.debug_run_id IS NULL
		ORDER BY
			CASE WHEN COALESCE(CAST(output_lookup.attributes->>'$."_inngest.is.function.output"' AS TEXT), '') IN ('true', '1') THEN 0 ELSE 1 END,
			output_lookup.end_time DESC
		LIMIT 1
	), '')`).As("output")
}

func (h *helpers) BuildEventJoin(q *sq.SelectDataset) *sq.SelectDataset {
	// SQLite: json_each for unnesting.
	// json_each('') errors with "malformed JSON", so we use NULLIF to convert empty strings
	// to NULL. json_each(NULL) safely returns no rows.
	return q.InnerJoin(sq.L("json_each(NULLIF(spans.event_ids, '')) AS je"), sq.On(sq.L("1=1"))).
		InnerJoin(sq.L("events"), sq.On(sq.L("je.value = events.event_id")))
}

func (h *helpers) ParseEventIDs(raw *string) []string {
	// SQLite: plain JSON array
	var ids []string
	if raw != nil && *raw != "" {
		_ = json.Unmarshal([]byte(*raw), &ids)
	}
	return ids
}

func (h *helpers) ParseTime(s string) (time.Time, error) {
	// SQLite: strip monotonic clock suffix if present
	if idx := strings.Index(s, " m="); idx != -1 {
		s = s[:idx]
	}
	// Aggregates return stored Go time strings; direct DATETIME reads return RFC3339.
	return dateutil.ParseString(s)
}
