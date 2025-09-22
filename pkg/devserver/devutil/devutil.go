// package devutil contains local interfaces for
package devutil

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/inngest/inngest/pkg/consts"
	loader "github.com/inngest/inngest/pkg/coreapi/graph/loaders"
	"github.com/inngest/inngest/pkg/coreapi/graph/resolvers"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

func NewLocalOutputReader(r *resolvers.Resolver, db cqrs.Manager, tr cqrs.TraceReader) apiv1.RunOutputReader {
	loaders := loader.NewLoaders(loader.LoaderParams{
		DB: db,
	})

	return runOutputReader{
		r:       r,
		tr:      tr,
		loaders: loaders,
	}
}

type runOutputReader struct {
	r       *resolvers.Resolver
	tr      cqrs.TraceReader
	loaders *loader.Loaders
}

func (r runOutputReader) RunOutput(ctx context.Context, envID uuid.UUID, runID ulid.ULID) ([]byte, error) {
	ctx = loader.ToCtx(ctx, r.loaders)

	fv2, err := r.r.Query().Run(ctx, runID.String())
	if err != nil {
		return nil, err
	}
	if fv2 == nil {
		return nil, fmt.Errorf("run not found: %s", runID)
	}

	run, err := r.tr.GetTraceRun(ctx, cqrs.TraceRunIdentifier{
		AccountID:   consts.DevServerAccountID,
		WorkspaceID: consts.DevServerEnvID,
		AppID:       fv2.AppID,
		FunctionID:  fv2.FunctionID,
		TraceID:     fv2.TraceID,
		RunID:       runID,
	})
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run not found: %s", runID)
	}

	if enums.RunStatusEnded(run.Status) {
		return run.Output, nil
	}

	return nil, fmt.Errorf("run output not found: %s", runID)
}
