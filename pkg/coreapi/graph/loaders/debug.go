package loader

import (
	"context"
	"fmt"
	"sort"
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

			debugRuns, err := tr.reader.GetSpansByDebugRunID(ctx, req.DebugRunID)
			if err != nil {
				res.Error = fmt.Errorf("error retrieving debug run trace: %w", err)
				return
			}

			gqlRoots := make([]*models.RunTraceSpan, 0, len(debugRuns))
			for _, rootSpan := range debugRuns {
				gqlRoot, err := tr.convertRunSpanToGQL(ctx, rootSpan)
				if err != nil {
					res.Error = fmt.Errorf("error converting debug run span to GQL, skipping: %w", err)
					continue
				}

				if gqlRoot != nil {
					gqlRoots = append(gqlRoots, gqlRoot)
				}
			}

			sort.Slice(gqlRoots, func(i, j int) bool {
				a, b := gqlRoots[i].StartedAt, gqlRoots[j].StartedAt
				if a == nil || b == nil {
					return a == nil && b != nil
				}
				return a.Before(*b)
			})

			res.Data = gqlRoots
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

			debugRuns, err := tr.reader.GetSpansByDebugSessionID(ctx, req.DebugSessionID)
			if err != nil {
				res.Error = fmt.Errorf("error retrieving debug traces by session id: %w", err)
				return
			}

			var debugSessionRuns []*models.DebugSessionRun
			for _, runSpans := range debugRuns {
				converted := make([]*models.RunTraceSpan, 0, len(runSpans))
				for _, span := range runSpans {
					gqlSpan, err := tr.convertRunSpanToGQL(ctx, span)
					if err != nil {
						res.Error = fmt.Errorf("error converting debug run span to GQL for debug session, skipping: %w", err)
						continue
					}
					if gqlSpan != nil {
						converted = append(converted, gqlSpan)
					}
				}

				if len(converted) == 0 {
					continue
				}

				sort.Slice(converted, func(i, j int) bool {
					a, b := converted[i].StartedAt, converted[j].StartedAt
					if a == nil || b == nil {
						return a == nil && b != nil
					}
					return a.Before(*b)
				})

				last := converted[len(converted)-1]
				debugSessionRuns = append(debugSessionRuns, &models.DebugSessionRun{
					Status:     last.Status,
					QueuedAt:   last.QueuedAt,
					StartedAt:  last.StartedAt,
					EndedAt:    last.EndedAt,
					DebugRunID: last.DebugRunID,
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
