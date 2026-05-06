package base_cqrs

import (
	"database/sql"

	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/oklog/ulid/v2"
)

func convertHistoryToWriter(h history.History) (*dbpkg.InsertHistoryParams, error) {
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
		params.LatencyMs = sql.NullInt64{Valid: true, Int64: *h.LatencyMS}
	}
	if h.BatchID != nil {
		params.BatchID = *h.BatchID
	}
	if h.GroupID != nil {
		params.GroupID = sql.NullString{Valid: true, String: h.GroupID.String()}
	}
	if h.StepName != nil {
		params.StepName = sql.NullString{Valid: true, String: *h.StepName}
	}
	if h.StepID != nil {
		params.StepID = sql.NullString{Valid: true, String: *h.StepID}
	}
	if h.StepType != nil {
		params.StepType = sql.NullString{Valid: true, String: h.StepType.String()}
	}
	if h.URL != nil {
		params.Url = sql.NullString{Valid: true, String: *h.URL}
	}

	var err error
	params.Sleep, err = marshalJSONAsNullString(h.Sleep)
	if err != nil {
		return nil, err
	}
	params.WaitForEvent, err = marshalJSONAsNullString(h.WaitForEvent)
	if err != nil {
		return nil, err
	}
	params.Result, err = marshalJSONAsNullString(h.Result)
	if err != nil {
		return nil, err
	}
	params.CancelRequest, err = marshalJSONAsNullString(h.Cancel)
	if err != nil {
		return nil, err
	}
	params.WaitResult, err = marshalJSONAsNullString(h.WaitResult)
	if err != nil {
		return nil, err
	}
	params.InvokeFunction, err = marshalJSONAsNullString(h.InvokeFunction)
	if err != nil {
		return nil, err
	}
	params.InvokeFunctionResult, err = marshalJSONAsNullString(h.InvokeFunctionResult)
	if err != nil {
		return nil, err
	}

	return &params, nil
}

