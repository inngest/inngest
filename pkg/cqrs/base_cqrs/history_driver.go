package base_cqrs

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
)

func NewHistoryDriver(adapter dbpkg.Adapter) history.Driver {
	return historyDriver{
		q: adapter.Q(),
	}
}

type historyDriver struct {
	q dbpkg.Querier
}

func (d historyDriver) Write(ctx context.Context, h history.History) (err error) {
	params, err := convertHistoryToWriter(h)
	if err != nil {
		return err
	}

	if err := d.q.InsertHistory(context.Background(), *params); err != nil {
		return err
	}

	switch h.Type {
	case enums.HistoryTypeFunctionCancelled.String(),
		enums.HistoryTypeFunctionCompleted.String(),
		enums.HistoryTypeFunctionFailed.String():

		// We must convert the history type into a proper enums.RunStatus field.
		status, err := enums.RunStatusString(strings.ReplaceAll(h.Type, "Function", ""))
		if err != nil {
			return err
		}

		// Add a function ends row.
		end := dbpkg.InsertFunctionFinishParams{
			RunID:     h.RunID,
			Status:    sql.NullString{String: status.String(), Valid: true},
			CreatedAt: sql.NullTime{Time: h.CreatedAt, Valid: true},
			// TODO: Completed step count.
			CompletedStepCount: sql.NullInt64{Int64: 0, Valid: true},
			Output: sql.NullString{
				String: "{}",
				Valid:  true,
			},
		}
		if h.Result != nil {
			marshalled, _ := marshalJSONAsString(h.Result.Output)
			end.Output = sql.NullString{String: marshalled, Valid: true}
		}
		return d.q.InsertFunctionFinish(context.Background(), end)
	default:
		return nil
	}
}

func (historyDriver) Close(ctx context.Context) error { return nil }

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
