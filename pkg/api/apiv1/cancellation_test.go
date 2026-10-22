package apiv1

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/api/apiv1/apiv1auth"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/publicerr"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type cancellationFunctionReader struct {
	cqrs.FunctionReader
	fn *cqrs.Function
}

func (r cancellationFunctionReader) GetFunctionByExternalID(_ context.Context, _ uuid.UUID, _, _ string) (*cqrs.Function, error) {
	return r.fn, nil
}

type cancellationReadWriter struct {
	cqrs.CancellationReadWriter
	got cqrs.Cancellation
	err error
}

func (w *cancellationReadWriter) CreateCancellation(_ context.Context, c cqrs.Cancellation) error {
	w.got = c
	return w.err
}

func TestCreateCancellationPreservesV1Record(t *testing.T) {
	until := time.Now().Add(-time.Hour)
	for _, from := range []*time.Time{nil, new(until.Add(-time.Hour))} {
		fn := &cqrs.Function{
			ID:    uuid.New(),
			AppID: uuid.New(),
			Slug:  "app-fn",
		}
		writer := &cancellationReadWriter{}
		api := API{opts: Opts{
			AuthFinder:             apiv1auth.NilAuthFinder,
			FunctionReader:         cancellationFunctionReader{fn: fn},
			CancellationReadWriter: writer,
		}}
		before := time.Now()
		c, err := api.CreateCancellation(t.Context(), CreateCancellationBody{
			AppID:         "app",
			FunctionID:    "fn",
			StartedAfter:  from,
			StartedBefore: until,
			If:            new(`event.data.cancel == true`),
		})
		require.NoError(t, err)
		require.False(t, c.ID.IsZero())
		require.WithinDuration(t, before, ulid.Time(c.ID.Time()), time.Second)
		require.WithinDuration(t, before, c.CreatedAt, time.Second)
		require.Equal(t, cqrs.Cancellation{
			ID:            c.ID,
			CreatedAt:     c.CreatedAt,
			AccountID:     consts.DevServerAccountID,
			WorkspaceID:   consts.DevServerEnvID,
			AppID:         fn.AppID,
			FunctionID:    fn.ID,
			FunctionSlug:  "app-fn",
			StartedAfter:  from,
			StartedBefore: until,
			If:            new(`event.data.cancel == true`),
			Kind:          enums.CancellationKindBulkRun,
			Type:          enums.CancellationTypeAPI,
		}, writer.got)
		require.Equal(t, writer.got, *c)
	}
}

func TestCreateCancellationPreservesV1WriterErrors(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{expressions.NewCompileError(errors.New("bad expression")), 400},
		{errors.New("storage unavailable"), 500},
	} {
		api := API{opts: Opts{
			AuthFinder:             apiv1auth.NilAuthFinder,
			FunctionReader:         cancellationFunctionReader{fn: &cqrs.Function{}},
			CancellationReadWriter: &cancellationReadWriter{err: tc.err},
		}}
		c, err := api.CreateCancellation(t.Context(), CreateCancellationBody{StartedBefore: time.Now()})
		require.Nil(t, c)
		var public publicerr.Error
		require.ErrorAs(t, err, &public)
		require.Equal(t, tc.status, public.Status)
	}
}
