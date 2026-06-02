package loader

import (
	"context"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type deferReader struct {
	reader cqrs.HistoryReader
}

func (dr *deferReader) GetRunDefers(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	return loadByRunID(ctx, keys, dr.reader.GetRunDefers)
}

func (dr *deferReader) GetRunDeferredFrom(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	return loadByRunID(ctx, keys, dr.reader.GetRunDeferredFrom)
}

func loadByRunID[V any](
	ctx context.Context,
	keys dataloader.Keys,
	fetch func(context.Context, []ulid.ULID) (map[ulid.ULID]V, error),
) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	type parsed struct {
		index int
		runID ulid.ULID
	}
	parsedKeys := make([]parsed, 0, len(keys))
	runIDs := make([]ulid.ULID, 0, len(keys))

	for i, key := range keys.Keys() {
		runID, err := ulid.Parse(key)
		if err != nil {
			results[i] = &dataloader.Result{Error: err}
			continue
		}
		parsedKeys = append(parsedKeys, parsed{index: i, runID: runID})
		runIDs = append(runIDs, runID)
	}

	byRunID, err := fetch(ctx, runIDs)
	if err != nil {
		for i, r := range results {
			if r != nil {
				continue
			}
			results[i] = &dataloader.Result{Error: err}
		}
		return results
	}

	for _, p := range parsedKeys {
		results[p.index] = &dataloader.Result{Data: byRunID[p.runID]}
	}
	return results
}
