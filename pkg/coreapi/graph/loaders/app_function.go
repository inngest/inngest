package loader

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
)

// appReader is the dataloader-backed reader for App lookups.
//
// The current implementation calls GetAppByID once per unique key inside the
// batch function — it does NOT yet issue a single batched SQL query. The
// per-request win comes entirely from dataloader's identity cache: when the
// dashboard /runs page (or any list view) renders 100 runs that share, say,
// 5 distinct apps, dataloader collapses them to 5 GetAppByID calls instead
// of 100. See https://github.com/inngest/inngest/issues/4326 for the original
// report. A follow-up that adds GetAppsByIDs to the cqrs interface and a
// `WHERE id = ANY($1)` SQL variant would replace the inner loop here with a
// single call, fully closing the worst-case 100-unique-apps gap.
type appReader struct {
	reader cqrs.Manager
}

func (ar *appReader) GetApps(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	return loadByUUID(ctx, keys, func(ctx context.Context, id uuid.UUID) (*cqrs.App, error) {
		return ar.reader.GetAppByID(ctx, id)
	})
}

// functionReader is the dataloader-backed reader for Function lookups by
// internal UUID. Shares the same architecture and follow-up trajectory as
// appReader (see appReader docstring).
type functionReader struct {
	reader cqrs.Manager
}

func (fr *functionReader) GetFunctions(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	return loadByUUID(ctx, keys, func(ctx context.Context, id uuid.UUID) (*cqrs.Function, error) {
		return fr.reader.GetFunctionByInternalUUID(ctx, id)
	})
}

// loadByUUID converts dataloader-style string keys into uuid.UUIDs, invokes
// `fetch` once per parsed key, and returns the results in the same order as
// the input keys. Errors on individual rows are surfaced per-result; an
// unparseable key produces a per-key error without aborting the rest of the
// batch.
func loadByUUID[V any](
	ctx context.Context,
	keys dataloader.Keys,
	fetch func(context.Context, uuid.UUID) (*V, error),
) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))
	for i, key := range keys.Keys() {
		id, err := uuid.Parse(key)
		if err != nil {
			results[i] = &dataloader.Result{Error: fmt.Errorf("invalid uuid key %q: %w", key, err)}
			continue
		}
		v, err := fetch(ctx, id)
		if err != nil {
			results[i] = &dataloader.Result{Error: err}
			continue
		}
		results[i] = &dataloader.Result{Data: v}
	}
	return results
}
