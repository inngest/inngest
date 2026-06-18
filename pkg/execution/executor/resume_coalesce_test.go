package executor

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type stubRunServiceMD struct {
	sv2.RunService
	saveStepHasPending bool
	saveStepErr        error
	md                 sv2.Metadata
}

func (s *stubRunServiceMD) LoadMetadata(_ context.Context, _ sv2.ID, _ ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	return s.md, nil
}

func (s *stubRunServiceMD) SaveStep(_ context.Context, _ sv2.ID, _ string, _ []byte) (bool, error) {
	return s.saveStepHasPending, s.saveStepErr
}

type stubPauseMgr struct {
	pauses.Manager
	consumeResult state.ConsumePauseResult
}

func (s *stubPauseMgr) ConsumePause(_ context.Context, _ sv2.RunService, _ state.Pause, _ state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error) {
	return s.consumeResult, func() error { return nil }, nil
}

func (s *stubPauseMgr) Delete(_ context.Context, _ pauses.Index, _ state.Pause, _ ...state.DeletePauseOpt) error {
	return nil
}

func TestResumePauseTimeoutCoalesceJobID(t *testing.T) {
	cases := []struct {
		name string
		ck   string
	}{
		{name: "coalesce key set", ck: "abc123"},
		{name: "no coalesce key", ck: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runID := ulid.MustNew(ulid.Now(), nil)
			wsID, fnID, aID := uuid.New(), uuid.New(), uuid.New()

			md := sv2.Metadata{
				ID: sv2.ID{
					RunID:      runID,
					FunctionID: fnID,
					Tenant:     sv2.Tenant{EnvID: wsID, AccountID: aID},
				},
				Config: *sv2.InitConfig(&sv2.Config{}),
			}

			q := &stubQueue{}
			e := &executor{
				smv2:           &stubRunServiceMD{md: md},
				pm:             &stubPauseMgr{},
				queue:          q,
				log:            logger.From(context.Background()),
				tracerProvider: tracing.NewNoopTracerProvider(),
			}

			pause := state.Pause{
				ID:          uuid.New(),
				WorkspaceID: wsID,
				DataKey:     "wait-step",
				Identifier: state.PauseIdentifier{
					RunID:      runID,
					FunctionID: fnID,
					AccountID:  aID,
				},
				ParallelCoalesceKey: tc.ck,
			}

			err := e.ResumePauseTimeout(context.Background(), pause, execution.ResumeRequest{})
			require.NoError(t, err)
			require.Len(t, q.enqueued, 1)

			got := *q.enqueued[0].JobID
			if tc.ck != "" {
				require.Equal(t, fmt.Sprintf("%s-%s-discover", runID, tc.ck), got)
			} else {
				require.Equal(t, fmt.Sprintf("%s-%s-timeout", md.IdempotencyKey(), pause.DataKey), got)
			}
		})
	}
}

func TestResumeCoalesceJobID(t *testing.T) {
	cases := []struct {
		name string
		ck   string
	}{
		{name: "coalesce key set", ck: "def456"},
		{name: "no coalesce key", ck: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runID := ulid.MustNew(ulid.Now(), nil)
			wsID, fnID, aID := uuid.New(), uuid.New(), uuid.New()

			md := sv2.Metadata{
				ID: sv2.ID{
					RunID:      runID,
					FunctionID: fnID,
					Tenant:     sv2.Tenant{EnvID: wsID, AccountID: aID},
				},
				Config: *sv2.InitConfig(&sv2.Config{}),
			}

			q := &stubQueue{}
			e := &executor{
				smv2: &stubRunServiceMD{md: md},
				pm: &stubPauseMgr{consumeResult: state.ConsumePauseResult{
					DidConsume:      true,
					HasPendingSteps: false,
				}},
				queue:          q,
				log:            logger.From(context.Background()),
				tracerProvider: tracing.NewNoopTracerProvider(),
			}

			pause := state.Pause{
				ID:          uuid.New(),
				WorkspaceID: wsID,
				DataKey:     "wait-step",
				Identifier: state.PauseIdentifier{
					RunID:      runID,
					FunctionID: fnID,
					AccountID:  aID,
				},
				ParallelCoalesceKey: tc.ck,
			}

			err := e.Resume(context.Background(), pause, execution.ResumeRequest{})
			require.NoError(t, err)
			require.Len(t, q.enqueued, 1)

			got := *q.enqueued[0].JobID
			if tc.ck != "" {
				require.Equal(t, fmt.Sprintf("%s-%s-discover", runID, tc.ck), got)
			} else {
				require.Equal(t, fmt.Sprintf("%s-%s-event", md.IdempotencyKey(), pause.DataKey), got)
			}
		})
	}
}
