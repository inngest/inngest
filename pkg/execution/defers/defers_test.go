package defers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type fakeRunService struct {
	statev2.RunService

	saveDeferErr         error
	savedDefer           *statev2.Defer
	savedDeferCalls      int
	savedRejected        *statev2.Defer
	savedRejectedCalls   int
	setDeferStatusHashed string
	setDeferStatusValue  enums.DeferStatus
}

func (f *fakeRunService) SaveDefer(_ context.Context, _ statev2.ID, d statev2.Defer) error {
	f.savedDeferCalls++
	f.savedDefer = &d
	return f.saveDeferErr
}

func (f *fakeRunService) SaveRejectedDefer(_ context.Context, _ statev2.ID, d statev2.Defer) error {
	f.savedRejectedCalls++
	f.savedRejected = &d
	return nil
}

func (f *fakeRunService) SetDeferStatus(_ context.Context, _ statev2.ID, hashedID string, status enums.DeferStatus) error {
	f.setDeferStatusHashed = hashedID
	f.setDeferStatusValue = status
	return nil
}

func runID() statev2.ID {
	return statev2.ID{
		RunID:      ulid.MustNew(ulid.Now(), nil),
		FunctionID: uuid.New(),
	}
}

func deferAddOp(t *testing.T, hashedID, userlandID string, opts state.DeferAddOpts) state.GeneratorOpcode {
	t.Helper()
	raw, err := json.Marshal(opts)
	require.NoError(t, err)
	op := state.GeneratorOpcode{
		Op:   enums.OpcodeDeferAdd,
		ID:   hashedID,
		Opts: json.RawMessage(raw),
	}
	if userlandID != "" {
		op.Userland = &struct {
			ID    string `json:"id"`
			Index int    `json:"index,omitempty"`
		}{ID: userlandID}
	}
	return op
}

func TestSaveFromOp_Accepted(t *testing.T) {
	t.Run("propagates UserlandID", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-1", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 1, fake.savedDeferCalls)
		require.Nil(t, fake.savedRejected)
		require.Equal(t, "user-defer-id", fake.savedDefer.UserlandID)
		require.Equal(t, "hash-1", fake.savedDefer.HashedID)
		require.Equal(t, enums.DeferStatusAfterRun, fake.savedDefer.ScheduleStatus)
	})

	t.Run("Userland nil → empty UserlandID, still accepted", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-1", "", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 1, fake.savedDeferCalls)
		require.Empty(t, fake.savedDefer.UserlandID)
	})
}

func TestSaveFromOp_Rejected(t *testing.T) {
	t.Run("per_defer_size writes sentinel with UserlandID", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-too-big", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`"` + strings.Repeat("a", consts.MaxDeferInputSize+1) + `"`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 0, fake.savedDeferCalls)
		require.Equal(t, 1, fake.savedRejectedCalls)
		require.Equal(t, "user-defer-id", fake.savedRejected.UserlandID)
	})

	t.Run("invalid_opts with FnSlug present writes sentinel", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-noinput", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 0, fake.savedDeferCalls)
		require.Equal(t, 1, fake.savedRejectedCalls)
		require.Equal(t, "user-defer-id", fake.savedRejected.UserlandID)
	})

	t.Run("invalid_opts without FnSlug absorbs without sentinel", func(t *testing.T) {
		fake := &fakeRunService{}
		op := deferAddOp(t, "hash-empty", "user-defer-id", state.DeferAddOpts{
			Input: json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 0, fake.savedDeferCalls)
		require.Equal(t, 0, fake.savedRejectedCalls)
	})

	t.Run("SaveDefer sentinel error soft-rejects", func(t *testing.T) {
		fake := &fakeRunService{saveDeferErr: statev2.ErrDeferLimitExceeded}
		op := deferAddOp(t, "hash-1", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.NoError(t, err)
		require.Equal(t, 1, fake.savedDeferCalls)
		require.Equal(t, 0, fake.savedRejectedCalls)
	})

	t.Run("infra error surfaces", func(t *testing.T) {
		fake := &fakeRunService{saveDeferErr: errors.New("redis dead")}
		op := deferAddOp(t, "hash-1", "user-defer-id", state.DeferAddOpts{
			FnSlug: "child-fn",
			Input:  json.RawMessage(`{"x":1}`),
		})

		err := SaveFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op, tracing.NewNoopTracerProvider(), statev2.Metadata{}, time.Time{})

		require.Error(t, err)
	})
}

func TestAbortFromOp(t *testing.T) {
	t.Run("flips status to Aborted", func(t *testing.T) {
		fake := &fakeRunService{}
		raw, err := json.Marshal(state.DeferAbortOpts{TargetHashedID: "hash-abort"})
		require.NoError(t, err)
		op := state.GeneratorOpcode{
			Op:   enums.OpcodeDeferAbort,
			ID:   "step-id",
			Opts: json.RawMessage(raw),
		}

		err = AbortFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op)

		require.NoError(t, err)
		require.Equal(t, "hash-abort", fake.setDeferStatusHashed)
		require.Equal(t, enums.DeferStatusAborted, fake.setDeferStatusValue)
	})

	t.Run("surfaces parse error", func(t *testing.T) {
		fake := &fakeRunService{}
		op := state.GeneratorOpcode{
			Op:   enums.OpcodeDeferAbort,
			Opts: json.RawMessage(`{}`),
		}

		err := AbortFromOp(context.Background(), fake, logger.VoidLogger(), runID(), op)

		require.Error(t, err)
		require.Empty(t, fake.setDeferStatusHashed)
	})
}
