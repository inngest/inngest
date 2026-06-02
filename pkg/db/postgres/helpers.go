package postgres

import (
	"encoding/json"
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
	// PostgreSQL: double-encoded JSON (a JSON string containing a JSON array)
	var ids []string
	if raw != nil && *raw != "" {
		var innerStr string
		if err := json.Unmarshal([]byte(*raw), &innerStr); err == nil {
			_ = json.Unmarshal([]byte(innerStr), &ids)
		}
	}
	return ids
}

func (h *helpers) ParseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, s)
}
