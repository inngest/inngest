package executor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// resumeSignalStubPauseMgr stubs the pauses.Manager surface ResumeSignal and
// Resume touch: looking up a pause by signal ID, deleting an expired pause,
// and consuming a pause once Resume is reached.
type resumeSignalStubPauseMgr struct {
	pauses.Manager
	pauseBySignalID func(ctx context.Context, workspaceID uuid.UUID, signalID string) (*state.Pause, error)
	consumePause    func(ctx context.Context, rs sv2.RunService, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error)
	deleteCalls     []state.Pause
}

func (m *resumeSignalStubPauseMgr) PauseBySignalID(ctx context.Context, workspaceID uuid.UUID, signalID string) (*state.Pause, error) {
	return m.pauseBySignalID(ctx, workspaceID, signalID)
}

func (m *resumeSignalStubPauseMgr) Delete(ctx context.Context, index pauses.Index, pause state.Pause, opts ...state.DeletePauseOpt) error {
	m.deleteCalls = append(m.deleteCalls, pause)
	return nil
}

func (m *resumeSignalStubPauseMgr) ConsumePause(ctx context.Context, rs sv2.RunService, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error) {
	return m.consumePause(ctx, rs, pause, opts)
}

// resumeSignalStubRunService stubs the single RunService method Resume calls
// before reaching ConsumePause: LoadMetadata.
type resumeSignalStubRunService struct {
	sv2.RunService
	loadMetadataErr error
}

func (s *resumeSignalStubRunService) LoadMetadata(ctx context.Context, id sv2.ID, opts ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	if s.loadMetadataErr != nil {
		return sv2.Metadata{}, s.loadMetadataErr
	}
	return sv2.Metadata{ID: id}, nil
}

func nonExpiredSignalPause(workspaceID uuid.UUID, signalID string) state.Pause {
	return state.Pause{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Identifier: state.PauseIdentifier{
			RunID:      ulid.MustNew(ulid.Now(), nil),
			FunctionID: uuid.New(),
		},
		SignalID: &signalID,
		Expires:  state.Time(time.Now().Add(time.Hour)),
	}
}

// TestResumeSignal_NoMatch_Expired_RacedLease pins ResumeSignal's contract for
// every path that does *not* produce a matched signal: no pause found, an
// expired pause (both sides of the grace-period delete decision), and the
// three Resume errors that must be swallowed into a plain "no match" result.
func TestResumeSignal_NoMatch_Expired_RacedLease(t *testing.T) {
	workspaceID := uuid.New()
	signalID := "signal-id"

	t.Run("no pause found for signal", func(t *testing.T) {
		pm := &resumeSignalStubPauseMgr{
			pauseBySignalID: func(ctx context.Context, workspaceID uuid.UUID, signalID string) (*state.Pause, error) {
				return nil, nil
			},
		}
		e := &executor{pm: pm, log: logger.From(context.Background())}

		res, err := e.ResumeSignal(context.Background(), workspaceID, signalID, nil)
		require.NoError(t, err)
		require.False(t, res.MatchedSignal)
		require.Empty(t, pm.deleteCalls, "an absent pause must never trigger a delete")
	})

	t.Run("expired pause past the grace period is deleted", func(t *testing.T) {
		clock := clockwork.NewFakeClockAt(time.Now())
		expiredPause := nonExpiredSignalPause(workspaceID, signalID)
		expiredPause.Expires = state.Time(clock.Now().Add(-consts.PauseExpiredDeletionGracePeriod - time.Minute))

		pm := &resumeSignalStubPauseMgr{
			pauseBySignalID: func(ctx context.Context, workspaceID uuid.UUID, signalID string) (*state.Pause, error) {
				return &expiredPause, nil
			},
		}
		e := &executor{pm: pm, log: logger.From(context.Background()), clock: clock}

		res, err := e.ResumeSignal(context.Background(), workspaceID, signalID, nil)
		require.NoError(t, err)
		require.False(t, res.MatchedSignal)
		require.Len(t, pm.deleteCalls, 1, "a pause expired beyond the grace period must be deleted")
	})

	t.Run("expired pause within the grace period is left in place", func(t *testing.T) {
		clock := clockwork.NewFakeClockAt(time.Now())
		expiredPause := nonExpiredSignalPause(workspaceID, signalID)
		expiredPause.Expires = state.Time(clock.Now().Add(-consts.PauseExpiredDeletionGracePeriod + time.Minute))

		pm := &resumeSignalStubPauseMgr{
			pauseBySignalID: func(ctx context.Context, workspaceID uuid.UUID, signalID string) (*state.Pause, error) {
				return &expiredPause, nil
			},
		}
		e := &executor{pm: pm, log: logger.From(context.Background()), clock: clock}

		res, err := e.ResumeSignal(context.Background(), workspaceID, signalID, nil)
		require.NoError(t, err)
		require.False(t, res.MatchedSignal)
		require.Empty(t, pm.deleteCalls, "a pause still within the grace period must not be deleted yet")
	})

	swallowedResumeErrorCases := []struct {
		name            string
		loadMetadataErr error
		consumePauseErr error
	}{
		{name: "pause leased by another handler", consumePauseErr: state.ErrPauseLeased},
		{name: "pause already consumed", consumePauseErr: state.ErrPauseNotFound},
		{name: "run no longer exists", loadMetadataErr: state.ErrRunNotFound},
	}

	for _, tc := range swallowedResumeErrorCases {
		t.Run(tc.name, func(t *testing.T) {
			pause := nonExpiredSignalPause(workspaceID, signalID)

			pm := &resumeSignalStubPauseMgr{
				pauseBySignalID: func(ctx context.Context, workspaceID uuid.UUID, signalID string) (*state.Pause, error) {
					return &pause, nil
				},
				consumePause: func(ctx context.Context, rs sv2.RunService, pause state.Pause, opts state.ConsumePauseOpts) (state.ConsumePauseResult, func() error, error) {
					return state.ConsumePauseResult{}, nil, tc.consumePauseErr
				},
			}
			e := &executor{
				pm:    pm,
				smv2:  &resumeSignalStubRunService{loadMetadataErr: tc.loadMetadataErr},
				queue: &stubQueue{},
				log:   logger.From(context.Background()),
			}

			res, err := e.ResumeSignal(context.Background(), workspaceID, signalID, nil)
			require.NoError(t, err, "Resume errors of this class must be swallowed, not surfaced")
			require.False(t, res.MatchedSignal)
		})
	}
}
