package ddb

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/inngest/inngest/pkg/cqrs/ddb/sqlc"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/oklog/ulid/v2"
)

func NewHistoryDriver(db *sql.DB) history.Driver {
	return historyDriver{
		q: sqlc.New(db),
	}
}

type historyDriver struct {
	q *sqlc.Queries
}

func (d historyDriver) Write(ctx context.Context, h history.History) (err error) {
	params := sqlc.InsertHistoryParams{
		ID:              h.ID,
		CreatedAt:       ulid.Time(h.ID.Time()),
		RunStartedAt:    ulid.Time(h.RunID.Time()),
		FunctionID:      h.FunctionID,
		FunctionVersion: h.FunctionVersion,
		RunID:           h.RunID,
		EventID:         h.EventID,
		IdempotencyKey:  h.IdempotencyKey,
		Type:            h.Type,
		Attempt:         h.Attempt,
		// TODO: Status
		// TODO: Completed step count
	}
	if h.BatchID != nil {
		params.BatchID = *h.BatchID
	}
	if h.GroupID != nil {
		params.GroupID = h.GroupID.String()
	}
	if h.StepName != nil {
		params.StepName = sql.NullString{
			Valid:  true,
			String: *h.StepName,
		}
	}
	if h.StepID != nil {
		params.StepID = sql.NullString{
			Valid:  true,
			String: *h.StepID,
		}
	}
	if h.URL != nil {
		params.Url = sql.NullString{
			Valid:  true,
			String: *h.URL,
		}
	}

	params.Sleep, err = marshalJSONAsNullString(h.Sleep)
	if err != nil {
		return err
	}
	params.WaitForEvent, err = marshalJSONAsNullString(h.WaitForEvent)
	if err != nil {
		return err
	}
	params.Result, err = marshalJSONAsNullString(h.Result)
	if err != nil {
		return err
	}

	// TODO: Cancellation

	return d.q.InsertHistory(ctx, params)
}

func (historyDriver) Close() error { return nil }

func marshalJSONAsNullString(input any) (sql.NullString, error) {
	if input == nil {
		return sql.NullString{}, nil
	}
	byt, err := json.Marshal(input)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{
		Valid:  true,
		String: string(byt),
	}, nil
}
