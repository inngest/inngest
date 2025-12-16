package base_cqrs

import (
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// SQLiteToCQRS accepts an input and converter, and convert the type to another
func SQLiteToCQRS[T, R any](input *T, converter func(*T) *R) *R {
	if input == nil {
		return nil
	}
	return converter(input)
}

// SQLiteToCQRSList converts a slice of inputs using the provided converter function
func SQLiteToCQRSList[T, R any](inputs []*T, converter func(*T) *R) []*R {
	if len(inputs) == 0 {
		return []*R{}
	}

	results := make([]*R, len(inputs))
	for i, input := range inputs {
		results[i] = SQLiteToCQRS(input, converter)
	}
	return results
}

//
// Converters
//

// sqliteFunction converts sqlc function to cqrs function
func sqliteFunction(fn *sqlc.Function) *cqrs.Function {
	if fn == nil {
		return nil
	}

	var archivedAt time.Time
	if fn.ArchivedAt.Valid {
		archivedAt = fn.ArchivedAt.Time
	}

	return &cqrs.Function{
		ID:         fn.ID,
		AppID:      fn.AppID,
		Name:       fn.Name,
		Slug:       fn.Slug,
		Config:     json.RawMessage(fn.Config),
		CreatedAt:  fn.CreatedAt,
		ArchivedAt: archivedAt,
	}
}

// sqliteEvent converts sqlc event to cqrs event
func sqliteEvent(obj *sqlc.Event) *cqrs.Event {
	if obj == nil {
		return nil
	}

	evt := &cqrs.Event{
		ID:           obj.InternalID,
		ReceivedAt:   obj.ReceivedAt,
		EventID:      obj.EventID,
		EventName:    obj.EventName,
		EventVersion: obj.EventV.String,
		EventTS:      obj.EventTs.UnixMilli(),
		EventData:    map[string]any{},
		EventUser:    map[string]any{},
	}
	_ = json.Unmarshal([]byte(obj.EventData), &evt.EventData)
	_ = json.Unmarshal([]byte(obj.EventUser), &evt.EventUser)
	return evt
}

// sqliteEventBatch converts sqlc event batch to cqrs event batch
func sqliteEventBatch(obj *sqlc.EventBatch) *cqrs.EventBatch {
	if obj == nil {
		return nil
	}

	var evtIDs []ulid.ULID
	if ids, err := obj.EventIDs(); err == nil {
		evtIDs = ids
	}

	eb := cqrs.NewEventBatch(
		cqrs.WithEventBatchID(obj.ID),
		cqrs.WithEventBatchAccountID(obj.AccountID),
		cqrs.WithEventBatchWorkspaceID(obj.WorkspaceID),
		cqrs.WithEventBatchAppID(obj.AppID),
		cqrs.WithEventBatchRunID(obj.RunID),
		cqrs.WithEventBatchEventIDs(evtIDs),
		cqrs.WithEventBatchExecutedTime(obj.ExecutedAt),
	)

	return eb
}

// sqliteApp converts sqlc app to cqrs app
func sqliteApp(obj *sqlc.App) *cqrs.App {
	if obj == nil {
		return nil
	}

	var deletedAt time.Time
	if obj.ArchivedAt.Valid {
		deletedAt = obj.ArchivedAt.Time
	}

	var appVersion string
	if obj.AppVersion.Valid {
		appVersion = obj.AppVersion.String
	}

	metadata := map[string]string{}
	_ = json.Unmarshal([]byte(obj.Metadata), &metadata)

	return &cqrs.App{
		ID:          obj.ID,
		Name:        obj.Name,
		SdkLanguage: obj.SdkLanguage,
		SdkVersion:  obj.SdkVersion,
		Framework:   obj.Framework,
		Metadata:    metadata,
		Status:      obj.Status,
		Error:       obj.Error,
		Checksum:    obj.Checksum,
		CreatedAt:   obj.CreatedAt,
		DeletedAt:   deletedAt,
		Url:         obj.Url,
		Method:      obj.Method,
		AppVersion:  appVersion,
	}
}

// sqliteFunctionFinish converts sqlc function finish to cqrs function run finish
func sqliteFunctionFinish(obj *sqlc.FunctionFinish) *cqrs.FunctionRunFinish {
	if obj == nil {
		return nil
	}

	var status enums.RunStatus
	if obj.Status.Valid {
		status, _ = enums.RunStatusString(obj.Status.String)
	}

	var output json.RawMessage
	if obj.Output.Valid {
		output = json.RawMessage(obj.Output.String)
	}

	var createdAt time.Time
	if obj.CreatedAt.Valid {
		createdAt = obj.CreatedAt.Time
	}

	var completedStepCount int64
	if obj.CompletedStepCount.Valid {
		completedStepCount = obj.CompletedStepCount.Int64
	}

	return &cqrs.FunctionRunFinish{
		RunID:              obj.RunID,
		Status:             status,
		Output:             output,
		CreatedAt:          createdAt,
		CompletedStepCount: completedStepCount,
	}
}
