package loader

import (
	"context"
	"fmt"
	"sync"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
)

func (tr *traceReader) GetDebugRunTrace(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	var wg sync.WaitGroup

	for i, key := range keys {
		results[i] = &dataloader.Result{}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result, key dataloader.Key) {
			defer wg.Done()

			req, ok := key.Raw().(*DebugRunRequestKey)
			if !ok {
				res.Error = fmt.Errorf("unexpected type %T", key.Raw())
				return
			}

			rootSpan, err := tr.reader.GetSpansByDebugRunID(ctx, req.DebugRunID)
			if err != nil {
				res.Error = fmt.Errorf("error retrieving debug run trace: %w", err)
				return
			}

			if rootSpan == nil {
				res.Data = nil
				return
			}

			gqlRoot, err := tr.convertRunSpanToGQL(ctx, rootSpan)
			if err != nil {
				res.Error = fmt.Errorf("error converting debug run root to GQL: %w", err)
				return
			}

			res.Data = gqlRoot
		}(ctx, results[i], key)
	}

	wg.Wait()
	return results
}

func (tr *traceReader) GetDebugSessionTrace(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	var wg sync.WaitGroup

	for i, key := range keys {
		results[i] = &dataloader.Result{}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result, key dataloader.Key) {
			defer wg.Done()

			req, ok := key.Raw().(*DebugSessionRequestKey)
			if !ok {
				res.Error = fmt.Errorf("unexpected type %T", key.Raw())
				return
			}

			rootSpans, err := tr.reader.GetSpansByDebugSessionID(ctx, req.DebugSessionID)
			if err != nil {
				res.Error = fmt.Errorf("error retrieving debug session traces: %w", err)
				return
			}

			var debugSessionRuns []*models.DebugSessionRun
			for _, rootSpan := range rootSpans {
				runTraceSpan, err := tr.convertRunSpanToGQL(ctx, rootSpan)
				if err != nil {
					res.Error = fmt.Errorf("error converting debug session span to GQL: %w", err)
					return
				}
				debugSessionRuns = append(debugSessionRuns, &models.DebugSessionRun{
					Status:     runTraceSpan.Status,
					QueuedAt:   rootSpan.GetQueuedAtTime(),
					StartedAt:  runTraceSpan.StartedAt,
					EndedAt:    runTraceSpan.EndedAt,
					DebugRunID: runTraceSpan.DebugRunID,
					// TODO: add tags and versions
					Tags:     []string{},
					Versions: []string{},
				})
			}

			res.Data = &models.DebugSession{
				DebugRuns: debugSessionRuns,
			}
		}(ctx, results[i], key)
	}

	wg.Wait()
	return results
}
