package executor

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type failedRunService struct {
	sv2.RunService
	mdErr     error
	evtsErr   error
	mdCalls   atomic.Int32
	evtsCalls atomic.Int32
	deleted   atomic.Bool
}

func (r *failedRunService) LoadMetadata(_ context.Context, id sv2.ID, _ ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	r.mdCalls.Add(1)
	return sv2.Metadata{ID: id, Config: *sv2.InitConfig(&sv2.Config{})}, r.mdErr
}

func (r *failedRunService) LoadEvents(context.Context, sv2.ID) ([]json.RawMessage, error) {
	r.evtsCalls.Add(1)
	if r.evtsErr != nil {
		return nil, r.evtsErr
	}
	return []json.RawMessage{json.RawMessage(`{"name":"test"}`)}, nil
}

func (r *failedRunService) LoadDefers(context.Context, sv2.ID) (map[string]sv2.Defer, error) {
	return nil, nil
}

func (r *failedRunService) Delete(context.Context, sv2.ID, ...sv2.DeleteOption) error {
	r.deleted.Store(true)
	return nil
}

type finishedRun struct {
	resp state.DriverResponse
	evts []json.RawMessage
}

type recordingFinishLifecycle struct {
	execution.NoopLifecyceListener
	finished chan finishedRun
}

func (l *recordingFinishLifecycle) OnFunctionFinished(_ context.Context, _ sv2.Metadata, _ queue.Item, evts []json.RawMessage, resp state.DriverResponse) {
	l.finished <- finishedRun{resp: resp, evts: evts}
}

func TestFinalizePermanentlyFailedRun(t *testing.T) {
	userErr := "Error"

	tests := []struct {
		name         string
		resp         *state.DriverResponse
		jobErr       error
		mdErr        error
		evtsErr      error
		wantErr      bool
		wantFinalize bool
		wantEvents   bool
	}{
		{
			name: "system error after op response",
			// eg. a 206 checkpoint whose next step failed to enqueue.
			resp:         &state.DriverResponse{StatusCode: 206},
			jobErr:       errors.New("error handling generator response: context canceled"),
			wantFinalize: true,
			wantEvents:   true,
		},
		{
			name:         "system error w/o response",
			jobErr:       errors.New("cannot load run events"),
			wantFinalize: true,
			wantEvents:   true,
		},
		{
			name:   "executor already finalized w/ final driver response",
			resp:   &state.DriverResponse{Err: &userErr, NoRetry: true},
			jobErr: errors.New("Error"),
		},
		{
			name:   "cancelled",
			jobErr: state.ErrFunctionCancelled,
		},
		{
			name:   "run not found",
			jobErr: state.ErrRunNotFound,
		},
		{
			name:   "event not found",
			jobErr: state.ErrEventNotFound,
		},
		{
			name:   "state already deleted",
			jobErr: errors.New("context canceled"),
			mdErr:  sv2.ErrMetadataNotFound,
		},
		{
			// The job must stay queued rather than strand the run.
			name:    "metadata load error",
			jobErr:  errors.New("context canceled"),
			mdErr:   errors.New("redis unavailable"),
			wantErr: true,
		},
		{
			// Finalizing w/o events would drop the function failed event, so
			// the job must stay queued instead.
			name:    "events load error",
			jobErr:  errors.New("context canceled"),
			evtsErr: errors.New("redis unavailable"),
			wantErr: true,
		},
		{
			name:         "events gone but metadata exists",
			jobErr:       errors.New("context canceled"),
			evtsErr:      state.ErrEventNotFound,
			wantFinalize: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runState := &failedRunService{mdErr: tc.mdErr, evtsErr: tc.evtsErr}
			lifecycle := &recordingFinishLifecycle{finished: make(chan finishedRun, 1)}
			e := &executor{
				log:            logger.VoidLogger(),
				smv2:           runState,
				shards:         missingShardRegistry{},
				tracerProvider: tracing.NewOtelTracerProvider(nil, time.Millisecond),
				lifecycles:     []execution.LifecycleListener{lifecycle},
			}
			item := queue.Item{Identifier: state.Identifier{
				RunID:      ulid.Make(),
				WorkflowID: uuid.New(),
				AccountID:  uuid.New(),
			}}

			err := e.FinalizePermanentlyFailedRun(context.Background(), item, tc.resp, tc.jobErr)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tc.mdErr != nil && !isRunStateGone(tc.mdErr) {
				require.Greater(t, runState.mdCalls.Load(), int32(1), "metadata read must be retried")
			}
			if tc.evtsErr != nil && !isRunStateGone(tc.evtsErr) {
				require.Greater(t, runState.evtsCalls.Load(), int32(1), "events read must be retried")
			}

			if !tc.wantFinalize {
				require.False(t, runState.deleted.Load())
				require.Empty(t, lifecycle.finished)
				return
			}

			require.True(t, runState.deleted.Load(), "run state must be deleted by Finalize")
			select {
			case fin := <-lifecycle.finished:
				resp := fin.resp
				if tc.wantEvents {
					require.Len(t, fin.evts, 1)
				} else {
					require.Empty(t, fin.evts)
				}
				require.NotNil(t, resp.Err)
				require.Equal(t, state.InngestErrInternal, *resp.Err)
				require.True(t, resp.Final())
				require.Contains(t, resp.Output, state.InngestErrInternal)
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for function finished lifecycle")
			}
		})
	}
}

type failingJobExecutor struct {
	execution.Executor
	resp        *state.DriverResponse
	err         error
	finalizeErr error
	finalized   []error
}

func (e *failingJobExecutor) Execute(context.Context, state.Identifier, queue.Item, inngest.Edge) (*state.DriverResponse, error) {
	return e.resp, e.err
}

func (e *failingJobExecutor) FinalizePermanentlyFailedRun(_ context.Context, _ queue.Item, _ *state.DriverResponse, jobErr error) error {
	e.finalized = append(e.finalized, jobErr)
	return e.finalizeErr
}

func TestHandleQueueItemFinalizesPermanentlyFailedRuns(t *testing.T) {
	jobErr := errors.New("error handling generator response: context canceled")
	maxAttempts := func(n int) *int { return &n }

	tests := []struct {
		name         string
		attempt      int
		maxAttempts  int
		err          error
		finalizeErr  error
		wantFinalize bool
		wantRetry    bool
	}{
		{
			name:         "last attempt",
			attempt:      0,
			maxAttempts:  1,
			err:          jobErr,
			wantFinalize: true,
		},
		{
			name:        "attempts remaining",
			attempt:     0,
			maxAttempts: 3,
			err:         jobErr,
		},
		{
			name:        "always retryable on last attempt",
			attempt:     0,
			maxAttempts: 1,
			err:         queue.AlwaysRetryError(jobErr),
			wantRetry:   true,
		},
		{
			// If the run can't be finalized the job must stay queued.
			name:         "finalize fails on last attempt",
			attempt:      0,
			maxAttempts:  1,
			err:          jobErr,
			finalizeErr:  errors.New("redis unavailable"),
			wantFinalize: true,
			wantRetry:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exec := &failingJobExecutor{resp: &state.DriverResponse{StatusCode: 206}, err: tc.err, finalizeErr: tc.finalizeErr}
			s := &svc{exec: exec}
			item := queue.Item{
				Attempt:     tc.attempt,
				MaxAttempts: maxAttempts(tc.maxAttempts),
				Payload:     queue.PayloadEdge{Edge: inngest.Edge{Incoming: "step"}},
			}

			_, err := s.handleQueueItem(context.Background(), item)
			// Not ErrorIs, as AlwaysRetryError doesn't unwrap.
			require.ErrorContains(t, err, jobErr.Error())
			require.Equal(t, tc.wantRetry || tc.maxAttempts > 1, queue.ShouldRetry(err, tc.attempt, tc.maxAttempts))

			if tc.wantFinalize {
				require.Len(t, exec.finalized, 1)
				require.ErrorIs(t, exec.finalized[0], jobErr)
			} else {
				require.Empty(t, exec.finalized)
			}
		})
	}
}
