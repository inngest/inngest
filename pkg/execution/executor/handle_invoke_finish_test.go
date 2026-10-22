package executor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type missingRunService struct{ sv2.RunService }

func (missingRunService) LoadMetadata(context.Context, sv2.ID, ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	return sv2.Metadata{}, state.ErrRunNotFound
}

type invokeFinishPauseManager struct {
	pauses.Manager
	pause               *state.Pause
	deleteErr           error
	deleteCalls         int
	lookupWorkspaceID   uuid.UUID
	lookupCorrelationID string
	deletedIndex        pauses.Index
	deletedPause        state.Pause
}

func (m *invokeFinishPauseManager) PauseByInvokeCorrelationID(_ context.Context, workspaceID uuid.UUID, correlationID string) (*state.Pause, error) {
	m.lookupWorkspaceID = workspaceID
	m.lookupCorrelationID = correlationID
	return m.pause, nil
}

func (m *invokeFinishPauseManager) Delete(_ context.Context, index pauses.Index, pause state.Pause, _ ...state.DeletePauseOpt) error {
	m.deleteCalls++
	m.deletedIndex = index
	m.deletedPause = pause
	return m.deleteErr
}

func TestHandleInvokeFinishDeletesPauseWhenRunIsMissing(t *testing.T) {
	deleteErr := errors.New("pause store unavailable")
	for _, tc := range []struct {
		name      string
		deleteErr error
	}{
		{name: "pause deleted", deleteErr: nil},
		{name: "delete error is returned", deleteErr: deleteErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspaceID := uuid.New()
			accountID := uuid.New()
			functionID := uuid.New()
			correlationID := "invoke-correlation"
			eventName := event.FnFinishedName
			pause := &state.Pause{
				ID:          uuid.New(),
				WorkspaceID: workspaceID,
				Identifier: state.PauseIdentifier{
					RunID:      ulid.MustNew(ulid.Now(), nil),
					FunctionID: functionID,
					AccountID:  accountID,
				},
				Expires: state.Time(time.Now().Add(time.Hour)),
				Event:   &eventName,
			}
			pm := &invokeFinishPauseManager{pause: pause, deleteErr: tc.deleteErr}
			e := &executor{
				log:   logger.VoidLogger(),
				smv2:  missingRunService{},
				pm:    pm,
				queue: &stubQueue{},
			}
			finishedEvent := event.BaseTrackedEvent{
				ID:          ulid.MustNew(ulid.Now(), nil),
				AccountID:   accountID,
				WorkspaceID: workspaceID,
				Event: event.Event{
					Name: event.FnFinishedName,
					Data: map[string]any{consts.InvokeCorrelationId: correlationID},
				},
			}

			err := e.HandleInvokeFinish(context.Background(), finishedEvent)

			require.ErrorIs(t, err, state.ErrRunNotFound)
			require.Equal(t, workspaceID, pm.lookupWorkspaceID)
			require.Equal(t, correlationID, pm.lookupCorrelationID)
			require.Equal(t, 1, pm.deleteCalls)
			require.Equal(t, pauses.Index{WorkspaceID: workspaceID, EventName: eventName}, pm.deletedIndex)
			require.Equal(t, pause.ID, pm.deletedPause.ID)
			if tc.deleteErr != nil {
				require.ErrorIs(t, err, tc.deleteErr)
				require.ErrorContains(t, err, "failed to delete orphaned invoke pause")
			}
		})
	}
}
