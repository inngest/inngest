package cancellation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

type bulkRunWriter struct {
	cqrs.CancellationWriter
	records []cqrs.Cancellation
	err     error
}

func (w *bulkRunWriter) CreateCancellation(_ context.Context, c cqrs.Cancellation) error {
	w.records = append(w.records, c)
	return w.err
}

func TestCreateBulkRun(t *testing.T) {
	writer := &bulkRunWriter{}
	from := time.Now().Add(-time.Hour)
	opts := CreateBulkRunOpts{
		AccountID:     uuid.New(),
		WorkspaceID:   uuid.New(),
		AppID:         uuid.New(),
		FunctionID:    uuid.New(),
		FunctionSlug:  "app-fn",
		StartedAfter:  &from,
		StartedBefore: time.Now(),
		If:            new("event.data.cancel == true"),
		Name:          new("cleanup"),
	}
	first, err := CreateBulkRun(t.Context(), writer, opts)
	require.NoError(t, err)
	second, err := CreateBulkRun(t.Context(), writer, opts)
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)
	require.Equal(t, []cqrs.Cancellation{*first, *second}, writer.records)
	require.Equal(t, cqrs.Cancellation{
		ID:            first.ID,
		CreatedAt:     first.CreatedAt,
		AccountID:     opts.AccountID,
		WorkspaceID:   opts.WorkspaceID,
		AppID:         opts.AppID,
		FunctionID:    opts.FunctionID,
		FunctionSlug:  opts.FunctionSlug,
		StartedAfter:  &from,
		StartedBefore: opts.StartedBefore,
		If:            opts.If,
		Name:          opts.Name,
		Kind:          enums.CancellationKindBulkRun,
		Type:          enums.CancellationTypeAPI,
	}, *first)
	require.WithinDuration(t, time.Now(), first.CreatedAt, time.Second)
	writer.err = errors.New("write failed")
	result, err := CreateBulkRun(t.Context(), writer, opts)
	require.Nil(t, result)
	require.ErrorIs(t, err, writer.err)
}
