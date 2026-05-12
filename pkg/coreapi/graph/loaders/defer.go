package loader

import (
	"context"
	"sync"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type deferReader struct {
	reader cqrs.HistoryReader
}

// loadByRunID provides request-scoped memoization and per-key concurrency,
// not batched SQL: the underlying CQRS reads are per-run.
func (dr *deferReader) loadByRunID(
	ctx context.Context,
	keys dataloader.Keys,
	fetch func(context.Context, ulid.ULID) (any, error),
) []*dataloader.Result {
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
			data, err := fetch(ctx, runID)
			if err != nil {
				res.Error = err
				return
			}
			res.Data = data
		}(ctx, results[i], key)
	}

	wg.Wait()
	return results
}

func (dr *deferReader) GetRunDefers(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	return dr.loadByRunID(ctx, keys, func(ctx context.Context, runID ulid.ULID) (any, error) {
		return dr.reader.GetRunDefers(ctx, runID)
	})
}

func (dr *deferReader) GetRunDeferredFrom(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	return dr.loadByRunID(ctx, keys, func(ctx context.Context, runID ulid.ULID) (any, error) {
		return dr.reader.GetRunDeferredFrom(ctx, runID)
	})
}
