package base_cqrs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sq "github.com/doug-martin/goqu/v9"
	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/oklog/ulid/v2"
)

// historyBulkChunkSize bounds the number of rows per INSERT so we stay under
// Postgres' 65535 bound parameters per query (24 cols * 500 rows = 12000 args).
const historyBulkChunkSize = 500

var historyCols = []any{
	"id", "created_at", "run_started_at", "function_id", "function_version",
	"run_id", "event_id", "batch_id", "group_id", "idempotency_key",
	"type", "attempt", "latency_ms", "step_name", "step_id",
	"step_type", "url", "cancel_request", "sleep", "wait_for_event",
	"wait_result", "invoke_function", "invoke_function_result", "result",
}

func NewHistoryDriver(adapter dbpkg.Adapter) history.Driver {
	return &historyDriver{
		q:       adapter.Q(),
		adapter: adapter,
	}
}

type historyDriver struct {
	q       dbpkg.Querier
	adapter dbpkg.Adapter
}

func (d *historyDriver) Write(ctx context.Context, h history.History) error {
	params, err := historyToParams(h)
	if err != nil {
		return err
	}

	if err := d.q.InsertHistory(context.Background(), params); err != nil {
		return err
	}

	return d.maybeInsertFunctionFinish(h)
}

// WriteBatch performs a bulk INSERT for non-terminal history items. Terminal
// items (cancelled/completed/failed) must be written via Write so that
// InsertFunctionFinish is called; any terminal items passed here will be
// inserted but their function_runs status update will be skipped.
func (d *historyDriver) WriteBatch(ctx context.Context, items []history.History) error {
	if len(items) == 0 {
		return nil
	}

	dialect := goquDialect(d.adapter.Dialect())
	rows := make([][]any, 0, len(items))
	var buildErrs []error
	for i, h := range items {
		params, err := historyToParams(h)
		if err != nil {
			buildErrs = append(buildErrs, fmt.Errorf("history[%d]: %w", i, err))
			continue
		}
		rows = append(rows, historyParamsToRow(params))
	}

	var insertErr error
	if len(rows) > 0 {
		insertErr = d.bulkInsertHistory(ctx, dialect, rows)
	}

	return errors.Join(append(buildErrs, insertErr)...)
}

func (d *historyDriver) Close(ctx context.Context) error { return nil }

// maybeInsertFunctionFinish writes an InsertFunctionFinish row for terminal
// history types (cancelled, completed, failed).
func (d *historyDriver) maybeInsertFunctionFinish(h history.History) error {
	switch h.Type {
	case enums.HistoryTypeFunctionCancelled.String(),
		enums.HistoryTypeFunctionCompleted.String(),
		enums.HistoryTypeFunctionFailed.String():

		status, err := enums.RunStatusString(strings.ReplaceAll(h.Type, "Function", ""))
		if err != nil {
			return err
		}

		end := dbpkg.InsertFunctionFinishParams{
			RunID:              h.RunID,
			Status:             sql.NullString{String: status.String(), Valid: true},
			CreatedAt:          sql.NullTime{Time: h.CreatedAt, Valid: true},
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

// historyToParams converts a History domain object into InsertHistoryParams.
func historyToParams(h history.History) (dbpkg.InsertHistoryParams, error) {
	params := dbpkg.InsertHistoryParams{
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
	if h.LatencyMS != nil {
		params.LatencyMs = sql.NullInt64{
			Valid: true,
			Int64: *h.LatencyMS,
		}
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
	if h.StepType != nil {
		params.StepType = sql.NullString{
			Valid:  true,
			String: h.StepType.String(),
		}
	}
	if h.URL != nil {
		params.Url = sql.NullString{
			Valid:  true,
			String: *h.URL,
		}
	}

	var err error
	params.Sleep, err = marshalJSONAsNullString(h.Sleep)
	if err != nil {
		return params, err
	}
	params.WaitForEvent, err = marshalJSONAsNullString(h.WaitForEvent)
	if err != nil {
		return params, err
	}
	params.Result, err = marshalJSONAsNullString(h.Result)
	if err != nil {
		return params, err
	}
	params.CancelRequest, err = marshalJSONAsNullString(h.Cancel)
	if err != nil {
		return params, err
	}
	params.WaitResult, err = marshalJSONAsNullString(h.WaitResult)
	if err != nil {
		return params, err
	}
	params.InvokeFunction, err = marshalJSONAsNullString(h.InvokeFunction)
	if err != nil {
		return params, err
	}
	params.InvokeFunctionResult, err = marshalJSONAsNullString(h.InvokeFunctionResult)
	if err != nil {
		return params, err
	}

	return params, nil
}

// historyParamsToRow flattens InsertHistoryParams into a row slice matching
// historyCols order for goqu bulk inserts.
func historyParamsToRow(p dbpkg.InsertHistoryParams) []any {
	return []any{
		p.ID, p.CreatedAt, p.RunStartedAt, p.FunctionID, p.FunctionVersion,
		p.RunID, p.EventID, p.BatchID, p.GroupID, p.IdempotencyKey,
		p.Type, p.Attempt, p.LatencyMs, p.StepName, p.StepID,
		p.StepType, p.Url, p.CancelRequest, p.Sleep, p.WaitForEvent,
		p.WaitResult, p.InvokeFunction, p.InvokeFunctionResult, p.Result,
	}
}

func (d *historyDriver) bulkInsertHistory(ctx context.Context, dialect string, rows [][]any) error {
	var chunkErrs []error
	for i := 0; i < len(rows); i += historyBulkChunkSize {
		end := i + historyBulkChunkSize
		if end > len(rows) {
			end = len(rows)
		}
		ds := sq.Dialect(dialect).
			Insert("history").
			Cols(historyCols...).
			Vals(rows[i:end]...)
		sqlStr, args, err := ds.ToSQL()
		if err != nil {
			chunkErrs = append(chunkErrs, fmt.Errorf("error building bulk history insert (chunk %d-%d): %w", i, end, err))
			continue
		}
		if _, err := d.adapter.ExecContext(ctx, sqlStr, args...); err != nil {
			chunkErrs = append(chunkErrs, fmt.Errorf("error executing bulk history insert (chunk %d-%d): %w", i, end, err))
			continue
		}
	}
	return errors.Join(chunkErrs...)
}

// goquDialect maps db.Dialect to the goqu dialect name.
func goquDialect(d dbpkg.Dialect) string {
	switch d {
	case dbpkg.DialectPostgres:
		return "postgres"
	case dbpkg.DialectSQLite:
		return "sqlite3"
	default:
		return "postgres"
	}
}

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
