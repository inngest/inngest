package loader

import (
	"context"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
)

type functionReader struct {
	reader cqrs.FunctionReader
}

// GetFunctionsBySlugs batches per-row function-by-slug lookups so list views
// don't issue one DB call per linkage row.
func (fr *functionReader) GetFunctionsBySlugs(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	slugs := keys.Keys()
	results := make([]*dataloader.Result, len(slugs))

	bySlug, err := fr.reader.GetFunctionsBySlugs(ctx, slugs)
	if err != nil {
		for i := range results {
			results[i] = &dataloader.Result{Error: err}
		}
		return results
	}

	for i, slug := range slugs {
		results[i] = &dataloader.Result{Data: bySlug[slug]}
	}
	return results
}
