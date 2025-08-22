package base_cqrs

import (
	"encoding/json"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/oklog/ulid/v2"
)

// SQLiteToCQRS accepts an input and converter, and convert the type to another
func SQLiteToCQRS[T, R any](input *T, converter func(*T) *R) *R {
	if input == nil {
		return nil
	}
	return converter(input)
}

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

// convertSQLiteFunctionToCQRS converts sqlc function to cqrs function
func convertSQLiteFunctionToCQRS(fn *sqlc.Function) *cqrs.Function {
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

func convertSQLiteEventToCQRS(obj *sqlc.Event) *cqrs.Event {
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

func convertSQLiteEventBatchToCQRS(obj *sqlc.EventBatch) *cqrs.EventBatch {
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
