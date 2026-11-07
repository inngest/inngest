package loader

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
)

type appFunctionBatchReader interface {
	GetAppsByIDs(ctx context.Context, ids []uuid.UUID) ([]*cqrs.App, error)
	GetFunctionsByInternalUUIDs(ctx context.Context, ids []uuid.UUID) ([]*cqrs.Function, error)
}

type appFunctionReader struct {
	reader appFunctionBatchReader
}

func (r appFunctionReader) GetApps(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	ids, indexes, results := parseUUIDKeys(keys)
	if len(ids) == 0 {
		return results
	}

	apps, err := r.reader.GetAppsByIDs(ctx, ids)
	if err != nil {
		return setErrors(results, indexes, err)
	}

	byID := make(map[uuid.UUID]*cqrs.App, len(apps))
	for _, app := range apps {
		byID[app.ID] = app
	}
	for i, id := range ids {
		app, ok := byID[id]
		if !ok {
			results[indexes[i]] = &dataloader.Result{Error: sql.ErrNoRows}
			continue
		}
		results[indexes[i]] = &dataloader.Result{Data: app}
	}
	return results
}

func (r appFunctionReader) GetFunctions(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	ids, indexes, results := parseUUIDKeys(keys)
	if len(ids) == 0 {
		return results
	}

	functions, err := r.reader.GetFunctionsByInternalUUIDs(ctx, ids)
	if err != nil {
		return setErrors(results, indexes, err)
	}

	byID := make(map[uuid.UUID]*cqrs.Function, len(functions))
	for _, fn := range functions {
		byID[fn.ID] = fn
	}
	for i, id := range ids {
		fn, ok := byID[id]
		if !ok {
			results[indexes[i]] = &dataloader.Result{Error: sql.ErrNoRows}
			continue
		}
		results[indexes[i]] = &dataloader.Result{Data: fn}
	}
	return results
}

func parseUUIDKeys(keys dataloader.Keys) ([]uuid.UUID, []int, []*dataloader.Result) {
	ids := make([]uuid.UUID, 0, len(keys))
	indexes := make([]int, 0, len(keys))
	results := make([]*dataloader.Result, len(keys))
	for i, key := range keys {
		id, err := uuid.Parse(key.String())
		if err != nil {
			results[i] = &dataloader.Result{Error: err}
			continue
		}
		ids = append(ids, id)
		indexes = append(indexes, i)
	}
	return ids, indexes, results
}

func setErrors(results []*dataloader.Result, indexes []int, err error) []*dataloader.Result {
	for _, index := range indexes {
		results[index] = &dataloader.Result{Error: err}
	}
	return results
}
