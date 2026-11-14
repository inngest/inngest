package executor

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
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

// terminalRunSvc records every state call in order so tests can assert what
// a response that ends the run writes and in which order.
type terminalRunSvc struct {
	sv2.RunService
	mu    sync.Mutex
	calls []string
}

func (s *terminalRunSvc) record(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, name)
}

func (s *terminalRunSvc) count(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.calls {
		if c == name {
			n++
		}
	}
	return n
}

func (s *terminalRunSvc) SaveStep(_ context.Context, _ sv2.ID, _ string, _ []byte) (bool, error) {
	s.record("SaveStep")
	return false, nil
}

func (s *terminalRunSvc) UpdateMetadata(_ context.Context, _ sv2.ID, _ sv2.MutableConfig) error {
	s.record("UpdateMetadata")
	return nil
}

func (s *terminalRunSvc) SavePending(_ context.Context, _ sv2.ID, _ []string) error {
	s.record("SavePending")
	return nil
}

func (s *terminalRunSvc) SaveDefer(_ context.Context, _ sv2.ID, _ sv2.Defer) error {
	s.record("SaveDefer")
	return nil
}

func (s *terminalRunSvc) LoadEvents(_ context.Context, _ sv2.ID) ([]json.RawMessage, error) {
	s.record("LoadEvents")
	return nil, nil
}

func (s *terminalRunSvc) LoadDefers(_ context.Context, _ sv2.ID) (map[string]sv2.Defer, error) {
	s.record("LoadDefers")
	return nil, nil
}

func (s *terminalRunSvc) Delete(_ context.Context, _ sv2.ID, _ ...sv2.DeleteOption) error {
	s.record("Delete")
	return nil
}

// noShards makes finalizeRemoveJobs return before touching the queue.
type noShards struct {
	queue.ShardRegistry
}

func (noShards) Resolve(context.Context, queue.Scope, *string) (queue.QueueShard, error) {
	return nil, errors.New("no shard registry in test")
}

// countingSyncListener records the sync lifecycle hooks the terminal path
// must still fire, and the order they fire in.
type countingSyncListener struct {
	execution.NoopSyncLifecycleListener
	mu               sync.Mutex
	events           []string
	stepRunFinished  []string
	stepRunFailed    []string
	functionFinished int
	hasAIAtFinish    bool
}

func (l *countingSyncListener) OnStepRunFinished(_ context.Context, _ sv2.Metadata, _ queue.Item, _ inngest.Edge, gen state.GeneratorOpcode, _ time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, "OnStepRunFinished:"+gen.ID)
	l.stepRunFinished = append(l.stepRunFinished, gen.ID)
}

func (l *countingSyncListener) OnStepRunFailed(_ context.Context, _ sv2.Metadata, _ queue.Item, _ inngest.Edge, gen state.GeneratorOpcode, _ enums.StepStatus, _ int, _ time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, "OnStepRunFailed:"+gen.ID)
	l.stepRunFailed = append(l.stepRunFailed, gen.ID)
}

func (l *countingSyncListener) OnFunctionFinished(_ context.Context, md sv2.Metadata, _ queue.Item, _ []json.RawMessage, _ state.DriverResponse, _ time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, "OnFunctionFinished")
	l.functionFinished++
	l.hasAIAtFinish = md.Config.HasAI
}

type terminalFixture struct {
	e        *executor
	svc      *terminalRunSvc
	q        *dedupeQueue
	listener *countingSyncListener
	i        *runInstance
}

func newTerminalFixture(gen ...*state.GeneratorOpcode) *terminalFixture {
	svc := &terminalRunSvc{}
	q := &dedupeQueue{}
	listener := &countingSyncListener{}
	e := &executor{
		smv2:           svc,
		queue:          q,
		shards:         noShards{},
		log:            logger.From(context.Background()),
		tracerProvider: tracing.NewNoopTracerProvider(),
		syncLifecycles: []execution.SyncLifecycleListener{listener},
	}

	md := sv2.Metadata{
		ID: sv2.ID{
			RunID:      ulid.MustNew(ulid.Now(), nil),
			FunctionID: uuid.New(),
			Tenant: sv2.Tenant{
				EnvID:     uuid.New(),
				AccountID: uuid.New(),
			},
		},
		Config: *sv2.InitConfig(&sv2.Config{}),
	}
	edge := inngest.Edge{Incoming: "step"}
	resp := &state.DriverResponse{
		StatusCode: 200,
		Generator:  gen,
	}
	i := &runInstance{
		md:   md,
		edge: edge,
		item: queue.Item{Payload: queue.PayloadEdge{Edge: edge}},
		resp: resp,
	}

	return &terminalFixture{
		e:        e,
		svc:      svc,
		q:        q,
		listener: listener,
		i:        i,
	}
}

func (f *terminalFixture) handle(t *testing.T) error {
	t.Helper()
	return f.e.HandleGeneratorResponse(context.Background(), f.i, f.i.resp)
}

func stepRunOp(id string) *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeStepRun,
		ID:   id,
		Data: json.RawMessage(`{"ok":true}`),
	}
}

func runCompleteOp() *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeRunComplete,
		ID:   "done",
		Data: json.RawMessage(`"result"`),
	}
}

func syncRunCompleteOp() *state.GeneratorOpcode {
	return &state.GeneratorOpcode{
		Op:   enums.OpcodeSyncRunComplete,
		ID:   "done",
		Data: json.RawMessage(`{"data":{"status":200,"headers":{"content-type":"application/json"},"body":"{\"ok\":true}","duration":1000000}}`),
	}
}

// assertTerminalSteps checks the shared outcome of a response that ends the
// run next to step-a and step-b.
func assertTerminalSteps(t *testing.T, f *terminalFixture) {
	t.Helper()
	require.Equal(t, 0, f.svc.count("SaveStep"), "step output must not be saved for a run that ends")
	require.Equal(t, 0, f.svc.count("UpdateMetadata"), "metadata must not be written for a run that ends")
	require.Equal(t, 0, f.svc.count("SavePending"))
	require.Empty(t, f.q.enqueued, "no discovery item may be enqueued for a run that ends")
	require.Equal(t, 1, f.svc.count("Delete"), "the run must be finalized exactly once")

	require.Equal(t, []string{"step-a", "step-b"}, f.listener.stepRunFinished)
	require.Equal(t, 1, f.listener.functionFinished)
	require.Equal(t, []string{
		"OnStepRunFinished:step-a",
		"OnStepRunFinished:step-b",
		"OnFunctionFinished",
	}, f.listener.events, "every step must be reported before the run finishes")
}

func TestHandleTerminalResponse_StepsNextToRunComplete(t *testing.T) {
	f := newTerminalFixture(stepRunOp("step-a"), stepRunOp("step-b"), runCompleteOp())
	require.NoError(t, f.handle(t))
	assertTerminalSteps(t, f)
}

func TestHandleTerminalResponse_StepsNextToSyncRunComplete(t *testing.T) {
	f := newTerminalFixture(stepRunOp("step-a"), stepRunOp("step-b"), syncRunCompleteOp())
	require.NoError(t, f.handle(t))
	assertTerminalSteps(t, f)
}

func TestHandleTerminalResponse_HasAISetInMemory(t *testing.T) {
	ai := stepRunOp("infer")
	ai.Op = enums.OpcodeStep
	ai.Opts = json.RawMessage(`{"type":"step.ai.wrap"}`)
	f := newTerminalFixture(ai, runCompleteOp())
	require.True(t, f.i.resp.HasAI(), "fixture must produce an AI response")

	require.NoError(t, f.handle(t))

	require.True(t, f.i.md.Config.HasAI)
	require.True(t, f.listener.hasAIAtFinish, "OnFunctionFinished must see HasAI on the metadata it is given")
	require.Equal(t, 0, f.svc.count("UpdateMetadata"))
}

func TestHandleTerminalResponse_DeferAddBeforeFinalize(t *testing.T) {
	deferAdd := &state.GeneratorOpcode{
		Op:   enums.OpcodeDeferAdd,
		ID:   "defer-1",
		Opts: json.RawMessage(`{"fn_slug":"app-child","input":{}}`),
	}
	f := newTerminalFixture(deferAdd, runCompleteOp())

	require.NoError(t, f.handle(t))

	require.Equal(t, 1, f.svc.count("SaveDefer"))
	require.Equal(t, 1, f.svc.count("Delete"))
	require.Empty(t, f.q.enqueued)

	var saveDeferIdx, deleteIdx int
	for n, c := range f.svc.calls {
		switch c {
		case "SaveDefer":
			saveDeferIdx = n
		case "Delete":
			deleteIdx = n
		}
	}
	require.Less(t, saveDeferIdx, deleteIdx, "the defer must be saved before state is deleted")
}

func TestHandleTerminalResponse_SleepIsSkipped(t *testing.T) {
	sleep := &state.GeneratorOpcode{
		Op:   enums.OpcodeSleep,
		ID:   "sleep-1",
		Opts: json.RawMessage(`{"duration":"1h"}`),
	}
	f := newTerminalFixture(sleep, runCompleteOp())

	require.NoError(t, f.handle(t))

	require.Empty(t, f.q.enqueued, "a sleep next to a terminal op must not enqueue anything")
	require.Equal(t, 1, f.svc.count("Delete"))
	require.Equal(t, 1, f.listener.functionFinished)
}

func TestHandleTerminalResponse_StepFailedIsRecorded(t *testing.T) {
	failed := &state.GeneratorOpcode{
		Op: enums.OpcodeStepFailed,
		ID: "step-x",
		Error: &state.UserError{
			Name:    "Error",
			Message: "boom",
		},
	}
	f := newTerminalFixture(failed, runCompleteOp())

	require.NoError(t, f.handle(t))

	require.Equal(t, []string{"step-x"}, f.listener.stepRunFailed, "the failed lifecycle must fire once for the step")
	require.Empty(t, f.listener.stepRunFinished)
	require.Equal(t, 0, f.svc.count("SaveStep"))
	require.Empty(t, f.q.enqueued)
	require.Equal(t, 1, f.svc.count("Delete"))
	require.Equal(t, []string{
		"OnStepRunFailed:step-x",
		"OnFunctionFinished",
	}, f.listener.events)
}

func TestHandleTerminalResponse_TwoTerminalOpsUsesFirst(t *testing.T) {
	f := newTerminalFixture(stepRunOp("step-a"), runCompleteOp(), runCompleteOp())

	require.NoError(t, f.handle(t))

	require.Equal(t, 1, f.svc.count("Delete"), "the first terminal op must finalize the run exactly once")
	require.Equal(t, 1, f.listener.functionFinished)
	require.Equal(t, 0, f.svc.count("SaveStep"))
	require.Empty(t, f.q.enqueued)
	require.Equal(t, []string{"step-a"}, f.listener.stepRunFinished)
}

func TestHandleTerminalResponse_NoTerminalOpUsesOldPath(t *testing.T) {
	f := newTerminalFixture(stepRunOp("step-a"))

	require.NoError(t, f.handle(t))

	require.Equal(t, 1, f.svc.count("SaveStep"))
	require.Len(t, f.q.enqueued, 1, "a plain step response must still enqueue discovery")
	require.Equal(t, 0, f.svc.count("Delete"))
	require.Equal(t, []string{"step-a"}, f.listener.stepRunFinished)
}

func TestHandleTerminalResponse_ParallelStepsWithoutTerminalOpUseOldPath(t *testing.T) {
	f := newTerminalFixture(stepRunOp("step-a"), stepRunOp("step-b"))

	require.NoError(t, f.handle(t))

	require.Equal(t, 1, f.svc.count("UpdateMetadata"), "two steps must still write ForceStepPlan")
	require.Equal(t, 2, f.svc.count("SaveStep"))
	require.Len(t, f.q.enqueued, 2, "each step must still enqueue discovery")
	require.Equal(t, 0, f.svc.count("Delete"))
	require.ElementsMatch(t, []string{"step-a", "step-b"}, f.listener.stepRunFinished)
}
