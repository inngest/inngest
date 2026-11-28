package cancellation

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

type CreateBulkRunOpts struct {
	AccountID, WorkspaceID, AppID, FunctionID uuid.UUID
	FunctionSlug                              string
	StartedAfter                              *time.Time
	StartedBefore                             time.Time
	If, Name                                  *string
}

func CreateBulkRun(ctx context.Context, writer cqrs.CancellationWriter, opts CreateBulkRunOpts) (*cqrs.Cancellation, error) {
	c := cqrs.Cancellation{
		CreatedAt:     time.Now(),
		ID:            ulid.MustNew(ulid.Now(), rand.Reader),
		AccountID:     opts.AccountID,
		WorkspaceID:   opts.WorkspaceID,
		AppID:         opts.AppID,
		FunctionID:    opts.FunctionID,
		FunctionSlug:  opts.FunctionSlug,
		StartedAfter:  opts.StartedAfter,
		StartedBefore: opts.StartedBefore,
		If:            opts.If,
		Name:          opts.Name,
		Kind:          enums.CancellationKindBulkRun,
		Type:          enums.CancellationTypeAPI,
	}
	if err := writer.CreateCancellation(ctx, c); err != nil {
		return nil, err
	}
	return &c, nil
}
