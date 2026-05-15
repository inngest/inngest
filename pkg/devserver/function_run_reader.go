package devserver

import (
	"context"

	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type cqrsFunctionRunReader struct {
	reader cqrs.APIV1FunctionRunReader
}

func NewFunctionRunReader(reader cqrs.APIV1FunctionRunReader) apiv2.FunctionRunReader {
	return &cqrsFunctionRunReader{reader: reader}
}

func (r *cqrsFunctionRunReader) GetFunctionRun(ctx context.Context, runID ulid.ULID) (*cqrs.FunctionRun, error) {
	return r.reader.GetFunctionRun(ctx, consts.DevServerAccountID, consts.DevServerEnvID, runID)
}
