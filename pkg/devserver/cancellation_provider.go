package devserver

import (
	"context"

	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution/cancellation"
	"github.com/inngest/inngest/pkg/inngest"
)

type cancellationProvider struct{ writer cqrs.CancellationWriter }

func NewCancellationProvider(writer cqrs.CancellationWriter) apiv2.CancellationCreator {
	return &cancellationProvider{writer: writer}
}

func (p *cancellationProvider) CreateCancellation(ctx context.Context, fn inngest.DeployedFunction, opts apiv2.CreateCancellationOpts) (*cqrs.Cancellation, error) {
	return cancellation.CreateBulkRun(ctx, p.writer, cancellation.CreateBulkRunOpts{
		AccountID:     fn.AccountID,
		WorkspaceID:   fn.EnvironmentID,
		AppID:         fn.AppID,
		FunctionID:    fn.ID,
		FunctionSlug:  fn.Slug,
		StartedAfter:  &opts.From,
		StartedBefore: opts.Until,
		If:            opts.If,
		Name:          opts.Name,
	})
}
