package base_cqrs

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

// domainToCQRS accepts an input and converter, and converts the type to another.
func domainToCQRS[T, R any](input *T, converter func(*T) *R) *R {
	if input == nil {
		return nil
	}
	return converter(input)
}

// domainToCQRSList converts a slice of inputs using the provided converter function.
func domainToCQRSList[T, R any](inputs []*T, converter func(*T) *R) []*R {
	if len(inputs) == 0 {
		return []*R{}
	}

	results := make([]*R, len(inputs))
	for i, input := range inputs {
		results[i] = domainToCQRS(input, converter)
	}
	return results
}

// Deprecated: SQLiteToCQRS is an alias kept during migration.
func SQLiteToCQRS[T, R any](input *T, converter func(*T) *R) *R {
	return domainToCQRS(input, converter)
}

// Deprecated: SQLiteToCQRSList is an alias kept during migration.
func SQLiteToCQRSList[T, R any](inputs []*T, converter func(*T) *R) []*R {
	return domainToCQRSList(inputs, converter)
}

//
// Converters from domain types (pkg/db) to CQRS types (pkg/cqrs)
//

func domainFunction(fn *dbpkg.Function) *cqrs.Function {
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

func domainEvent(obj *dbpkg.Event) *cqrs.Event {
	if obj == nil {
		return nil
	}

	var accountID uuid.UUID
	if obj.AccountID.Valid {
		accountID, _ = uuid.Parse(obj.AccountID.String)
	}
	var workspaceID uuid.UUID
	if obj.WorkspaceID.Valid {
		workspaceID, _ = uuid.Parse(obj.WorkspaceID.String)
	}

	evt := &cqrs.Event{
		ID:           obj.InternalID,
		AccountID:    accountID,
		WorkspaceID:  workspaceID,
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

func domainEventBatch(obj *dbpkg.EventBatch) *cqrs.EventBatch {
	if obj == nil {
		return nil
	}

	var evtIDs []ulid.ULID
	strids := strings.Split(string(obj.EventIds), ",")
	for _, sid := range strids {
		if id, err := ulid.Parse(sid); err == nil {
			evtIDs = append(evtIDs, id)
		}
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

func domainApp(obj *dbpkg.App) *cqrs.App {
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

func domainFunctionFinish(obj *dbpkg.FunctionFinish) *cqrs.FunctionRunFinish {
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
