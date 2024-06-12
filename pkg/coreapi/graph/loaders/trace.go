package loader

import (
	"context"
	"fmt"
	"sync"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
)

type TraceRequestKey struct {
	*cqrs.TraceRunIdentifier
}

func (k *TraceRequestKey) Raw() any {
	return k
}

func (k *TraceRequestKey) String() string {
	return fmt.Sprintf("%s-%s", k.TraceID, k.RunID)
}

type traceReader struct {
	loaders *loaders
	reader  cqrs.TraceReader
}

func (tr *traceReader) GetRunTrace(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))

	var wg sync.WaitGroup
	for i, key := range keys {
		results[i] = &dataloader.Result{}

		req, ok := key.Raw().(*TraceRequestKey)
		if !ok {
			results[i].Error = fmt.Errorf("unexpected type %T", key.Raw())
			continue
		}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result) {
			defer wg.Done()

			spans, err := tr.reader.GetTraceSpansByRun(ctx, *req.TraceRunIdentifier)
			if err != nil {
				res.Error = fmt.Errorf("error retrieving span: %w", err)
				return
			}
			if len(spans) < 1 {
				return
			}

			// TODO: build tree from spans

			// TODO: prime tree
		}(ctx, results[i])
	}

	wg.Wait()

	return results
}
