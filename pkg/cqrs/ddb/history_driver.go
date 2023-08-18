package ddb

import (
	"context"
	"database/sql"

	"github.com/inngest/inngest/pkg/cqrs/ddb/sqlc"
	"github.com/inngest/inngest/pkg/execution/history"
)

func NewHistoryDriver(db *sql.DB) history.Driver {
	return historyDriver{
		q: sqlc.New(db),
	}
}

type historyDriver struct {
	q *sqlc.Queries
}

func (d historyDriver) Write(ctx context.Context, h history.History) error {
	params := sqlc.InsertHistoryParams{
		ID: h.ID,
	}
	return d.q.InsertHistory(ctx, params)
}

func (historyDriver) Close() error { return nil }
