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

func parseRunIDKeys(keys dataloader.Keys) []ulid.ULID {
	runIDs := make([]ulid.ULID, 0, len(keys))
	for _, key := range keys.Keys() {
		runID, err := ulid.Parse(key)
		if err != nil {
			continue
		}
		runIDs = append(runIDs, runID)
	}
	return runIDs
}

func (dr *deferReader) GetRunDefers(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))

	defersByRun, err := dr.reader.GetRunDefers(ctx, parseRunIDKeys(keys))
	if err != nil {
		for i := range results {
			results[i] = &dataloader.Result{Error: err}
		}
		return results
	}

	for i, key := range keys.Keys() {
		runID, err := ulid.Parse(key)
		if err != nil {
			results[i] = &dataloader.Result{Error: err}
			continue
		}
		results[i] = &dataloader.Result{Data: defersByRun[runID]}
	}
	return results
}

func (dr *deferReader) GetRunDeferredFrom(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))

	linksByChild, err := dr.reader.GetRunDeferredFrom(ctx, parseRunIDKeys(keys))
	if err != nil {
		for i := range results {
			results[i] = &dataloader.Result{Error: err}
		}
		return results
	}

	for i, key := range keys.Keys() {
		runID, err := ulid.Parse(key)
		if err != nil {
			results[i] = &dataloader.Result{Error: err}
			continue
		}
		results[i] = &dataloader.Result{Data: linksByChild[runID]}
	}
	return results
}
