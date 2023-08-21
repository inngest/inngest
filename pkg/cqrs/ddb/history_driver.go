package ddb

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/inngest/inngest/pkg/cqrs/ddb/sqlc"
	"github.com/inngest/inngest/pkg/enums"
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
	}
	if h.BatchID != nil {
		params.BatchID = *h.BatchID
	}
	if h.GroupID != nil {
		params.GroupID = sql.NullString{
			String: h.GroupID.String(),
			Valid:  true,
		}
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
	params.CancelRequest, err = marshalJSONAsNullString(h.Cancel)
	if err != nil {
		return err
	}
	params.WaitResult, err = marshalJSONAsNullString(h.WaitResult)
	if err != nil {
		return err
	}

	if err := d.q.InsertHistory(context.Background(), params); err != nil {
		return err
	}

	switch h.Type {
	case enums.HistoryTypeFunctionCancelled.String(),
		enums.HistoryTypeFunctionCompleted.String(),
		enums.HistoryTypeFunctionFailed.String():
		// Add a function ends row.
		end := sqlc.InsertFunctionFinishParams{
			RunID:     h.RunID,
			Status:    h.Type,
			CreatedAt: h.CreatedAt,
		}
		if h.Result != nil {
			end.Output, _ = marshalJSONAsString(h.Result.Output)
		}
		return d.q.InsertFunctionFinish(context.Background(), end)
	default:
		return nil
	}
}

func (historyDriver) Close() error { return nil }

func marshalJSONAsString(input any) (string, error) {
	switch v := input.(type) {
	case []byte:
		return string(v), nil
	case json.RawMessage:
		return string(v), nil
	case string:
		return v, nil
	}

	byt, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(byt), nil
}

func marshalJSONAsNullString(input any) (sql.NullString, error) {
	str, err := marshalJSONAsString(input)
	if err != nil || str == "" {
		return sql.NullString{}, nil
	}
	return sql.NullString{
		Valid:  true,
		String: str,
	}, nil
}
