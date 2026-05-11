package loader

import (
	"context"
	"sync"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type deferReader struct {
	loaders *Loaders
	reader  cqrs.HistoryReader
}

// GetRunDefers loads defer projections per run ID. The underlying CQRS read is
// per-run, so this loader provides request-scoped memoization and per-key
// concurrency, not batched SQL.
func (dr *deferReader) GetRunDefers(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	var wg sync.WaitGroup

	for i, key := range keys {
		results[i] = &dataloader.Result{}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result, key dataloader.Key) {
			defer wg.Done()

			runID, err := ulid.Parse(key.String())
			if err != nil {
				res.Error = err
				return
			}
			defers, err := dr.reader.GetRunDefers(ctx, runID)
			if err != nil {
				res.Error = err
				return
			}
			res.Data = defers
		}(ctx, results[i], key)
	}

	wg.Wait()
	return results
}

func (dr *deferReader) GetRunDeferredFrom(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	var wg sync.WaitGroup

	for i, key := range keys {
		results[i] = &dataloader.Result{}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result, key dataloader.Key) {
			defer wg.Done()

			runID, err := ulid.Parse(key.String())
			if err != nil {
				res.Error = err
				return
			}
			deferredFrom, err := dr.reader.GetRunDeferredFrom(ctx, runID)
			if err != nil {
				res.Error = err
				return
			}
			res.Data = deferredFrom
		}(ctx, results[i], key)
	}

	wg.Wait()
	return results
}
