package loader

import (
	"context"
	"fmt"

	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/cqrs"
)

type TraceRequestKey struct {
	cqrs.TraceRunIdentifier
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
	return []*dataloader.Result{}
}
