package postgres

import (
	"encoding/json"
	"strings"
	"time"

	sq "github.com/doug-martin/goqu/v9"
	sqexp "github.com/doug-martin/goqu/v9/exp"
	"github.com/inngest/inngest/pkg/db/driverhelp"
	"github.com/inngest/inngest/pkg/run"
)

var _ driverhelp.DialectHelpers = (*helpers)(nil)

type helpers struct{}

func (h *helpers) GoquDialect() string { return "postgres" }

func (h *helpers) CELConverter() run.ExprSQLConverter { return run.SpanEventPostgresConverter }

func (h *helpers) EventIDsExpr() sqexp.Expression {
	// PostgreSQL: cast JSONB to text first (no MAX for JSONB)
	return sq.L("MAX(spans.event_ids::text)").As("event_ids")
}

func (h *helpers) RootEventIDsExpr() sqexp.Expression {
	// PostgreSQL: cast JSONB to text to match ParseEventIDs' double-decode.
	return sq.L("spans.event_ids::text").As("event_ids")
}

func (h *helpers) EventIDsContain(ids []string) sqexp.Expression {
	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	return sq.L(
		"EXISTS (SELECT 1 FROM jsonb_array_elements_text(NULLIF(spans.event_ids#>>'{}', '')::jsonb) AS eid(event_id) WHERE eid.event_id IN ("+placeholders+"))",
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
			CASE WHEN COALESCE((output_lookup.attributes#>>'{}')::json->>'_inngest.is.function.output', '') IN ('true', '1') THEN 0 ELSE 1 END,
			output_lookup.end_time DESC NULLS LAST
		LIMIT 1
	), '')`).As("output")
}

func (h *helpers) RunFunctionSlugExpr() sqexp.LiteralExpression {
	return sq.L(`CASE
		WHEN COALESCE((run_functions.config::json)->>'slug', '') <> ''
			AND (run_functions.config::json)->>'slug' <> run_functions.slug
			THEN (run_functions.config::json)->>'slug'
		WHEN COALESCE(run_apps.name, '') <> ''
			AND substr(COALESCE(NULLIF((run_functions.config::json)->>'slug', ''), run_functions.slug), 1, length(run_apps.name) + 1) = run_apps.name || '-'
			THEN substr(COALESCE(NULLIF((run_functions.config::json)->>'slug', ''), run_functions.slug), length(run_apps.name) + 2)
		ELSE COALESCE(NULLIF((run_functions.config::json)->>'slug', ''), run_functions.slug)
	END`)
}

func (h *helpers) RunFunctionNameExpr() sqexp.LiteralExpression {
	return sq.L("COALESCE(NULLIF((run_functions.config::json)->>'name', ''), run_functions.name)")
}

func (h *helpers) BuildEventJoin(q *sq.SelectDataset) *sq.SelectDataset {
	// PostgreSQL: jsonb_array_elements_text for unnesting.
	// event_ids is JSONB containing a JSON string (double-encoded), e.g. "[\"uuid\"]" or ""
	// Extract string with #>>'{}', use NULLIF to handle empty strings, then parse as JSON
	return q.InnerJoin(
		sq.L("jsonb_array_elements_text(NULLIF(spans.event_ids#>>'{}', '')::jsonb) AS eid(event_id)"),
		sq.On(sq.L("true")),
	).InnerJoin(sq.T("events"), sq.On(sq.L("eid.event_id = events.event_id")))
}

func (h *helpers) ParseEventIDs(raw *string) []string {
	var ids []string
	if raw == nil || *raw == "" {
		return ids
	}
	if err := json.Unmarshal([]byte(*raw), &ids); err == nil {
		return ids
	}

	var encoded string
	if err := json.Unmarshal([]byte(*raw), &encoded); err == nil {
		_ = json.Unmarshal([]byte(encoded), &ids)
	}
	return ids
}

func (h *helpers) ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
