package loader

import (
	"context"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type deferReader struct {
	loaders *Loaders
	reader  cqrs.HistoryReader
}

// GetRunDefers loads defer projections per run ID. The underlying CQRS read is
// per-run, so this loader provides request-scoped memoization, not batched SQL.
func (dr *deferReader) GetRunDefers(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	for i, key := range keys.Keys() {
		runID, err := ulid.Parse(key)
		if err != nil {
			results[i] = &dataloader.Result{Error: err}
			continue
		}
		defers, err := dr.reader.GetRunDefers(ctx, runID)
		results[i] = &dataloader.Result{Data: defers, Error: err}
	}
	return results
}

func (dr *deferReader) GetRunDeferredFrom(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	for i, key := range keys.Keys() {
		runID, err := ulid.Parse(key)
		if err != nil {
			results[i] = &dataloader.Result{Error: err}
			continue
		}
		deferredFrom, err := dr.reader.GetRunDeferredFrom(ctx, runID)
		results[i] = &dataloader.Result{Data: deferredFrom, Error: err}
	}
	return results
}
