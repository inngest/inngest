package defers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type fakeRunService struct {
	statev2.RunService

	saveDeferErr         error
	savedDefer           *statev2.Defer
	savedDeferCalls      int
	setDeferStatusErr    error
	setDeferStatusHashed string
	setDeferStatusValue  enums.DeferStatus
}

func (f *fakeRunService) SaveDefer(_ context.Context, _ statev2.ID, d statev2.Defer) error {
	f.savedDeferCalls++
	f.savedDefer = &d
	return f.saveDeferErr
}

func (f *fakeRunService) SetDeferStatus(_ context.Context, _ statev2.ID, hashedID string, status enums.DeferStatus) error {
	f.setDeferStatusHashed = hashedID
	f.setDeferStatusValue = status
	return f.setDeferStatusErr
}

func runMetadata() *statev2.Metadata {
	return &statev2.Metadata{ID: statev2.ID{
		RunID:      ulid.MustNew(ulid.Now(), nil),
		FunctionID: uuid.New(),
	}}
}

func deferAddOp(t *testing.T, hashedID string, opts state.DeferAddOpts) state.GeneratorOpcode {
	t.Helper()
	raw, err := json.Marshal(opts)
	require.NoError(t, err)
	return state.GeneratorOpcode{
		Op:   enums.OpcodeDeferAdd,
		ID:   hashedID,
		Opts: json.RawMessage(raw),
	}
}

func TestSaveFromOp_Rejected(t *testing.T) {
	validInput := json.RawMessage(`{"x":1}`)
	oversizedInput := json.RawMessage(`{"msg": "` + strings.Repeat("a", consts.MaxDeferInputSize+1) + `"}`)

	cases := []struct {
		name           string
		opts           state.DeferAddOpts
		saveDeferErr   error
		wantSaveCalls  int
		wantStatus     enums.DeferStatus
		wantSurfaceErr bool
	}{
		{
			name:          "oversized input writes Rejected sentinel",
			opts:          state.DeferAddOpts{FnSlug: "child-fn", Input: oversizedInput},
			wantSaveCalls: 1,
			wantStatus:    enums.DeferStatusRejected,
		},
		{
			name:          "invalid opts with FnSlug writes Rejected sentinel",
			opts:          state.DeferAddOpts{FnSlug: "child-fn"},
			wantSaveCalls: 1,
			wantStatus:    enums.DeferStatusRejected,
		},
		{
			name:          "invalid opts without FnSlug",
			opts:          state.DeferAddOpts{Input: validInput},
			wantSaveCalls: 1,
		},
		{
			// ErrDeferLimitExceeded is the soft-reject signal from the
			// underlying state store. The original AfterRun save was attempted;
			// no follow-up Rejected sentinel is written.
			name:          "soft-reject from state store keeps AfterRun status",
			opts:          state.DeferAddOpts{FnSlug: "child-fn", Input: validInput},
			saveDeferErr:  statev2.ErrDeferLimitExceeded,
			wantSaveCalls: 1,
			wantStatus:    enums.DeferStatusAfterRun,
		},
		{
			name:           "infra error surfaces to caller",
			opts:           state.DeferAddOpts{FnSlug: "child-fn", Input: validInput},
			saveDeferErr:   errors.New("redis dead"),
			wantSurfaceErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			fake := &fakeRunService{saveDeferErr: tc.saveDeferErr}
			op := deferAddOp(t, "hash-"+tc.name, tc.opts)

			err := SaveFromOp(context.Background(), fake, nil, logger.VoidLogger(), runMetadata(), op)

			if tc.wantSurfaceErr {
				r.Error(err)
				return
			}
			r.NoError(err)
			r.Equal(tc.wantSaveCalls, fake.savedDeferCalls)
			if tc.wantSaveCalls > 0 && tc.wantStatus != enums.DeferStatusUnknown {
				r.Equal(tc.wantStatus, fake.savedDefer.ScheduleStatus)
			}
		})
	}
}

func TestAbortFromOp(t *testing.T) {
	abortOp := func(t *testing.T, opts state.DeferAbortOpts) state.GeneratorOpcode {
		t.Helper()
		raw, err := json.Marshal(opts)
		require.NoError(t, err)
		return state.GeneratorOpcode{
			Op:   enums.OpcodeDeferAbort,
			ID:   "step-id",
			Opts: json.RawMessage(raw),
		}
	}

	t.Run("surfaces parse error from missing TargetHashedID", func(t *testing.T) {
		r := require.New(t)
		fake := &fakeRunService{}
		err := AbortFromOp(context.Background(), fake, nil, logger.VoidLogger(), runMetadata(),
			abortOp(t, state.DeferAbortOpts{}))

		r.Error(err)
		r.Empty(fake.setDeferStatusHashed)
	})

	t.Run("unknown-target abort is benign", func(t *testing.T) {
		r := require.New(t)
		fake := &fakeRunService{
			setDeferStatusErr: fmt.Errorf("%w for hashedID %q", state.ErrDeferNotFound, "never-added"),
		}
		err := AbortFromOp(context.Background(), fake, nil, logger.VoidLogger(), runMetadata(),
			abortOp(t, state.DeferAbortOpts{TargetHashedID: "never-added"}))

		r.NoError(err)
		r.Equal("never-added", fake.setDeferStatusHashed)
	})

	t.Run("surfaces infra error from SetDeferStatus", func(t *testing.T) {
		r := require.New(t)
		fake := &fakeRunService{
			setDeferStatusErr: fmt.Errorf("redis unavailable"),
		}
		err := AbortFromOp(context.Background(), fake, nil, logger.VoidLogger(), runMetadata(),
			abortOp(t, state.DeferAbortOpts{TargetHashedID: "some-defer"}))

		r.Error(err)
	})
}
