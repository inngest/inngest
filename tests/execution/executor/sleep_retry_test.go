package executor

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/inngest/inngest/pkg/execution/executor"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	statev2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

type failFirstSaveStepState struct {
	statev2.RunService

	mu       sync.Mutex
	stepID   string
	calls    int
	firstErr error
}

func (s *failFirstSaveStepState) SaveStep(ctx context.Context, id statev2.ID, stepID string, data []byte) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if stepID == s.stepID {
		s.calls++
		if s.calls == 1 {
			return false, s.firstErr
		}
	}

	return s.RunService.SaveStep(ctx, id, stepID, data)
}

func (s *failFirstSaveStepState) saveCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestSleepCompletionSaveRetries(t *testing.T) {
	r := require.New(t)
	infra := newExecTestInfra(t, "step-sleep")

	const sleepID = "sleep-id"
	storeErr := errors.New("state store unavailable")
	driverErr := errors.New("stop after loading state")
	stateStore := &failFirstSaveStepState{
		RunService: infra.smv2,
		stepID:     sleepID,
		firstErr:   storeErr,
	}
	driver := &mockDriverV1{
		t:   t,
		err: driverErr,
	}
	exec := infra.newExecutor(t,
		executor.WithStateManager(stateStore),
		executor.WithDriverV1(driver),
	)
	run := infra.scheduleRun(t, exec)
	identifier := state.Identifier{
		RunID:       run.ID.RunID,
		WorkflowID:  infra.fnID,
		WorkspaceID: infra.wsID,
		AppID:       infra.appID,
		AccountID:   infra.aID,
	}
	edge := inngest.Edge{Incoming: "step-sleep", Outgoing: sleepID}
	item := queue.Item{
		Identifier:  identifier,
		WorkspaceID: infra.wsID,
		Kind:        queue.KindSleep,
		Payload:     queue.PayloadEdge{Edge: edge},
	}

	_, err := exec.Execute(infra.ctx, identifier, item, edge)
	r.ErrorIs(err, storeErr)
	r.Equal(1, stateStore.saveCalls())

	item.Attempt = 1
	_, err = exec.Execute(infra.ctx, identifier, item, edge)
	r.ErrorIs(err, driverErr)
	r.Equal(2, stateStore.saveCalls())

	steps, err := infra.smv2.LoadSteps(infra.ctx, run.ID)
	r.NoError(err)
	r.JSONEq("null", string(steps[sleepID]))
}
