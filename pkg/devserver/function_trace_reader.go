package devserver

import (
	"context"

	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type cqrsFunctionTraceReader struct {
	reader cqrs.TraceReader
}

func NewFunctionTraceReader(reader cqrs.TraceReader) apiv2.FunctionTraceReader {
	return &cqrsFunctionTraceReader{reader: reader}
}

func (r *cqrsFunctionTraceReader) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	return r.reader.GetSpansByRunID(ctx, runID)
}

func (r *cqrsFunctionTraceReader) GetSpanOutput(ctx context.Context, id cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	return r.reader.GetSpanOutput(ctx, id)
}
