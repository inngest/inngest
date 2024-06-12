package loader

import (
	"context"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
)

type traceReader struct {
	loaders *loaders
	reader  cqrs.TraceReader
}

func (tr *traceReader) GetRunTrace(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	return nil
}
