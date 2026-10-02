package executor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
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
		if s.calls == 1 && s.firstErr != nil {
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
	const sleepID = "sleep-id"
	storeErr := errors.New("state store unavailable")
	driverErr := errors.New("execution failed after saving sleep completion")
	for _, tc := range []struct {
		name         string
		firstSaveErr error
		firstExecErr error
	}{
		{name: "retry failed save", firstSaveErr: storeErr},
		{name: "retry after successful save", firstExecErr: driverErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := require.New(t)
			infra := newExecTestInfra(t, "step-sleep")
			stateStore := &failFirstSaveStepState{
				RunService: infra.smv2,
				stepID:     sleepID,
				firstErr:   tc.firstSaveErr,
			}
			worker, err := queue.New(infra.ctx, "sleep-retry", infra.shardRegistry,
				// The partition scanner requires at least five free workers.
				queue.WithNumWorkers(10),
				queue.WithPollTick(10*time.Millisecond),
				queue.WithRunMode(queue.QueueRunMode{Partition: true}),
				queue.WithBackoffFunc(func(int) time.Time { return time.Now().Add(50 * time.Millisecond) }),
			)
			r.NoError(err)
			exec := infra.newExecutorWithQueue(t, worker,
				executor.WithStateManager(stateStore),
				executor.WithDriverV1(&sleepRetryDriver{sleepID: sleepID, firstExecErr: tc.firstExecErr}),
			)
			run := infra.scheduleRun(t, exec)
			scope := queue.Scope{AccountID: infra.aID, EnvID: infra.wsID, FunctionID: infra.fnID}

			type attemptResult struct {
				jobID   string
				attempt int
				err     error
			}
			results := make(chan attemptResult, 10)
			ctx, cancel := context.WithCancel(infra.ctx)
			done := make(chan error, 1)
			t.Cleanup(func() {
				cancel()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("queue worker did not stop")
				}
			})
			go func() {
				done <- worker.Run(ctx, func(ctx context.Context, _ queue.RunInfo, item queue.Item) (queue.RunResult, error) {
					edge := item.Payload.(queue.PayloadEdge).Edge
					_, execErr := exec.Execute(ctx, item.Identifier, item, edge)
					if item.Kind == queue.KindSleep && edge.Outgoing == sleepID {
						select {
						case results <- attemptResult{queue.JobIDFromContext(ctx), item.Attempt, execErr}:
						case <-ctx.Done():
						}
					}
					return queue.RunResult{}, execErr
				})
			}()

			var attempts []attemptResult
			for range 2 {
				select {
				case result := <-results:
					attempts = append(attempts, result)
				case <-time.After(10 * time.Second):
					t.Fatal("sleep job was not retried by the queue worker")
				}
			}
			r.Equal(0, attempts[0].attempt)
			r.Equal(1, attempts[1].attempt)
			r.NotEmpty(attempts[0].jobID)
			r.Equal(attempts[0].jobID, attempts[1].jobID)
			if tc.firstSaveErr != nil {
				r.ErrorIs(attempts[0].err, tc.firstSaveErr)
			} else {
				r.ErrorIs(attempts[0].err, tc.firstExecErr)
			}
			r.NoError(attempts[1].err)

			// Wait for the worker's success path to remove the original sleep job.
			// The only remaining job must be the next sleep emitted by the SDK.
			r.Eventually(func() bool {
				jobs, err := worker.RunJobs(infra.ctx, infra.queueShard.Name(), scope, run.ID.RunID, 100, 0)
				if err != nil || len(jobs) != 1 || jobs[0].JobID == attempts[0].jobID {
					return false
				}
				item, ok := jobs[0].Raw.(*queue.QueueItem)
				return ok && item.Data.Kind == queue.KindSleep &&
					item.Data.Payload.(queue.PayloadEdge).Edge.Outgoing == "next-sleep" &&
					item.AtMS > time.Now().UnixMilli()
			}, 5*time.Second, 10*time.Millisecond, "resumed workflow must retain its next job, not become stranded")
			r.Equal(2, stateStore.saveCalls())
			steps, err := infra.smv2.LoadSteps(infra.ctx, run.ID)
			r.NoError(err)
			r.JSONEq("null", string(steps[sleepID]))
			md, err := infra.smv2.LoadMetadata(infra.ctx, run.ID)
			r.NoError(err)
			r.Equal([]string{sleepID}, md.Stack, "retry must not append the completed sleep twice")
		})
	}
}

// sleepRetryDriver models an SDK replay: without saved completion it emits the
// same sleep again; with completion it advances to the next piece of work.
type sleepRetryDriver struct {
	sleepID      string
	firstExecErr error
}

func (*sleepRetryDriver) Name() string { return "http" }

func (d *sleepRetryDriver) Execute(ctx context.Context, sl statev2.StateLoader, md statev2.Metadata, item queue.Item, _ inngest.Edge, _ inngest.Step, _ int, _ int) (*state.DriverResponse, error) {
	steps, err := sl.LoadSteps(ctx, md.ID)
	if err != nil {
		return nil, err
	}
	id, duration := d.sleepID, "0s"
	if _, complete := steps[d.sleepID]; complete {
		if item.Kind == queue.KindSleep && item.Attempt == 0 && d.firstExecErr != nil {
			return nil, d.firstExecErr
		}
		id, duration = "next-sleep", "1h"
	}
	return &state.DriverResponse{
		StatusCode: 206, RequestVersion: 2,
		Generator: []*state.GeneratorOpcode{{Op: enums.OpcodeSleep, ID: id, Name: duration}},
	}, nil
}
