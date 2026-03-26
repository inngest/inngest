package sqlite

import (
	"encoding/json"
	"strings"
	"time"

	sq "github.com/doug-martin/goqu/v9"
	sqexp "github.com/doug-martin/goqu/v9/exp"
	"github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/run"
)

var _ db.DialectHelpers = (*helpers)(nil)

type helpers struct{}

func (h *helpers) GoquDialect() string { return "sqlite3" }

func (h *helpers) CELConverter() run.ExprSQLConverter { return run.SpanEventSQLiteConverter }

func (h *helpers) EventIDsExpr() sqexp.Expression {
	return sq.L("MAX(spans.event_ids)").As("event_ids")
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
	return time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", s)
}
