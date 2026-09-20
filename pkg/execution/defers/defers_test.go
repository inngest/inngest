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
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// fakeTracerProvider records every CreateSpanOptions passed to CreateSpan so
// tests can assert on span timestamps/attributes without a real backend.
type fakeTracerProvider struct {
	tracing.TracerProvider
	createSpanCalls []*tracing.CreateSpanOptions
}

func (f *fakeTracerProvider) CreateSpan(_ context.Context, _ string, opts *tracing.CreateSpanOptions) (*meta.SpanReference, error) {
	f.createSpanCalls = append(f.createSpanCalls, opts)
	return &meta.SpanReference{}, nil
}

// recordingSyncListener records every OnDeferAdd/OnDeferAbort call.
type recordingSyncListener struct {
	execution.NoopSyncLifecycleListener
	deferAdds []time.Time
}

func (r *recordingSyncListener) OnDeferAdd(_ context.Context, _ statev2.Metadata, _ statev2.Defer, _ string, now time.Time) {
	r.deferAdds = append(r.deferAdds, now)
}

type fakeRunService struct {
	statev2.RunService

	saveDeferErr         error
	savedDefer           *statev2.Defer
	savedDeferCalls      int
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
	return nil
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
	oversizedMeta := json.RawMessage(`{"sessions": {"k": "` + strings.Repeat("a", consts.MaxEventMetaSize) + `"}}`)

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
			// Non-object Meta is caught at op receipt rather than at finalize,
			// where a bad blob could only be dropped silently. The sentinel means
			// that if the SDK sends the same defer again, it will dedupe rather than
			// re-rejecting.
			name: "non-object meta writes Rejected sentinel",
			opts: state.DeferAddOpts{
				FnSlug: "child-fn",
				Input:  validInput,
				// Wrap with an array, rather than a top-level object
				Meta: json.RawMessage(`[{"sessions":{}}]`),
			},
			wantSaveCalls: 1,
			wantStatus:    enums.DeferStatusRejected,
		},
		{
			name: "oversized meta writes Rejected sentinel",
			opts: state.DeferAddOpts{
				FnSlug: "child-fn",
				Input:  validInput,
				Meta:   oversizedMeta,
			},
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

			err := SaveFromOp(context.Background(), fake, nil, nil, logger.VoidLogger(), runMetadata(), op)

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

// TestSaveFromOp_MalformedOptsDoesNotPanic proves that a defer op whose Opts
// blob fails to unmarshal (DeferAddOpts returning (nil, err), distinct from
// unmarshal-succeeds-but-Validate-fails cases which return a non-nil opts)
// is soft-rejected without dereferencing the nil opts.
func TestSaveFromOp_MalformedOptsDoesNotPanic(t *testing.T) {
	r := require.New(t)
	fake := &fakeRunService{}
	op := state.GeneratorOpcode{
		Op:   enums.OpcodeDeferAdd,
		ID:   "hash-malformed",
		Opts: json.RawMessage(`not-json`),
	}

	r.NotPanics(func() {
		err := SaveFromOp(context.Background(), fake, nil, nil, logger.VoidLogger(), runMetadata(), op)
		r.NoError(err)
	})
	r.Zero(fake.savedDeferCalls)
}

// TestSaveFromOp_SpanAndListenerShareTimestamp proves the executor.defer
// span's StartTime/EndTime and the OnDeferAdd timestamp all come from the
// same instant, rather than each independently calling time.Now().
func TestSaveFromOp_SpanAndListenerShareTimestamp(t *testing.T) {
	r := require.New(t)
	fake := &fakeRunService{}
	tp := &fakeTracerProvider{}
	listener := &recordingSyncListener{}

	op := deferAddOp(t, "hash-timestamp", state.DeferAddOpts{
		FnSlug: "child-fn",
		Input:  json.RawMessage(`{"x":1}`),
	})

	err := SaveFromOp(context.Background(), fake, tp, []execution.SyncLifecycleListener{listener}, logger.VoidLogger(), runMetadata(), op)
	r.NoError(err)

	r.Len(tp.createSpanCalls, 1)
	r.Len(listener.deferAdds, 1)
	r.Equal(tp.createSpanCalls[0].StartTime, listener.deferAdds[0])
	r.Equal(tp.createSpanCalls[0].EndTime, listener.deferAdds[0])
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
		err := AbortFromOp(context.Background(), fake, nil, nil, logger.VoidLogger(), runMetadata(),
			abortOp(t, state.DeferAbortOpts{}))

		r.Error(err)
		r.Empty(fake.setDeferStatusHashed)
	})
}
