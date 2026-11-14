package executor

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/loops"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/pubsub"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

// loopInputData answers the function lookup the loop input handler makes.
type loopInputData struct {
	cqrs.Manager
	fn *cqrs.Function
}

func (d loopInputData) GetFunctionByInternalUUID(context.Context, uuid.UUID) (*cqrs.Function, error) {
	return d.fn, nil
}

// loopInputExecutor records every schedule request the handler makes and
// passes it on to the real executor.
type loopInputExecutor struct {
	execution.Executor
	reqs []execution.ScheduleRequest
	mds  []*sv2.Metadata
}

func (e *loopInputExecutor) Schedule(ctx context.Context, req execution.ScheduleRequest) (*ulid.ULID, *sv2.Metadata, error) {
	e.reqs = append(e.reqs, req)
	runID, md, err := e.Executor.Schedule(ctx, req)
	e.mds = append(e.mds, md)
	return runID, md, err
}

type loopInputPublisher struct {
	mu   sync.Mutex
	msgs []pubsub.Message
	err  error
}

func (p *loopInputPublisher) Publish(_ context.Context, _ string, m pubsub.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.msgs = append(p.msgs, m)
	return p.err
}

func (p *loopInputPublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.msgs)
}

type loopInputTopic struct{}

func (loopInputTopic) Backend() string                        { return "test" }
func (loopInputTopic) TopicURL(string, config.URLType) string { return "" }
func (loopInputTopic) TopicName() string                      { return "events" }

// loopService builds the executor service around the harness executor.
func (h *loopHarness) loopService(t *testing.T, fn inngest.Function) (*Service, *loopInputExecutor, *loopInputPublisher) {
	t.Helper()
	conf, err := json.Marshal(fn)
	require.NoError(t, err)
	exec := &loopInputExecutor{Executor: h.e}
	pub := &loopInputPublisher{}
	s := &Service{
		exec:      exec,
		publisher: pub,
		log:       logger.VoidLogger(),
		data: loopInputData{
			fn: &cqrs.Function{
				ID:     fn.ID,
				EnvID:  h.req.WorkspaceID,
				AppID:  h.req.AppID,
				Slug:   fn.Slug,
				Name:   fn.Name,
				Config: conf,
			},
		},
	}
	s.config.EventStream.Service.Set(loopInputTopic{})
	return s, exec, pub
}

// timerItem enqueues one tick or wake item for the harness key and loads
// it back the way the queue hands it to the service.
func (h *loopHarness) timerItem(t *testing.T, kind string, input string, inputID string) queue.Item {
	t.Helper()
	id := h.loopID("user-1")
	jobID := "tick:" + id.InstanceID().String()
	if input == loops.InputWake {
		jobID = h.wakeJobID(inputID)
	}
	md := sv2.Metadata{
		ID: sv2.ID{
			RunID:      ulid.Make(),
			FunctionID: h.req.Function.ID,
			Tenant: sv2.Tenant{
				AccountID: h.req.AccountID,
				EnvID:     h.req.WorkspaceID,
				AppID:     h.req.AppID,
			},
		},
	}
	item := loopTimerItem(md, id, jobID, kind, input, inputID)
	require.NoError(t, h.q.Enqueue(h.ctx, item, time.Now(), queue.EnqueueOpts{}))
	loaded, err := h.shard.LoadQueueItem(h.ctx, queue.HashID(h.ctx, jobID))
	require.NoError(t, err)
	item = loaded.Data
	item.At = time.UnixMilli(loaded.AtMS)
	return item
}

// loopInputProcessor hands each item to the service's run function the way
// the queue does.
type loopInputProcessor struct {
	queue.QueueProcessor
	items []queue.Item
}

func (p loopInputProcessor) Run(ctx context.Context, f queue.RunFunc) error {
	for _, item := range p.items {
		if _, err := f(ctx, queue.RunInfo{}, item); err != nil {
			return err
		}
	}
	return nil
}

func TestHandleLoopInput(t *testing.T) {
	t.Run("tick starts a run with the tick event", func(t *testing.T) {
		h := newLoopHarness(t)
		h.openAndCommit(t)
		s, exec, pub := h.loopService(t, h.req.Function)

		item := h.timerItem(t, queue.KindLoopTick, loops.InputTick, "")
		payload := item.Payload.(queue.PayloadLoop)
		require.Equal(t, "user-1", payload.Key)
		require.Equal(t, loops.InputTick, payload.Kind)
		require.NotEmpty(t, payload.Nonce)
		require.NoError(t, s.handleLoopInput(h.ctx, item))

		require.Len(t, exec.reqs, 1)
		req := exec.reqs[0]
		require.Equal(t, h.req.AccountID, req.AccountID)
		require.Equal(t, h.req.WorkspaceID, req.WorkspaceID)
		require.Equal(t, h.req.AppID, req.AppID)
		require.Equal(t, h.req.Function.ID, req.Function.ID)
		require.Len(t, req.Events, 1)
		evt := req.Events[0].GetEvent()
		require.Equal(t, loops.EventTick, evt.Name)
		require.Equal(t, map[string]any{"key": "user-1"}, evt.Data)
		require.NotEmpty(t, evt.ID)

		require.NotNil(t, exec.mds[0])
		md, err := h.e.smv2.LoadMetadata(h.ctx, exec.mds[0].ID)
		require.NoError(t, err)
		h.requireStartSemaphore(t, md.ID, "user-1")
		evts, err := h.e.smv2.LoadEvents(h.ctx, md.ID)
		require.NoError(t, err)
		require.Len(t, evts, 1)
		stored := event.Event{}
		require.NoError(t, json.Unmarshal(evts[0], &stored))
		require.Equal(t, loops.EventTick, stored.Name)

		require.Eventually(t, func() bool { return pub.count() == 1 }, 2*time.Second, 10*time.Millisecond)
		require.Equal(t, event.EventReceivedName, pub.msgs[0].Name)
		require.Contains(t, pub.msgs[0].Data, loops.EventTick)
	})

	t.Run("wake starts a run with the wake event", func(t *testing.T) {
		h := newLoopHarness(t)
		h.openAndCommit(t)
		s, exec, _ := h.loopService(t, h.req.Function)

		item := h.timerItem(t, queue.KindLoopWake, loops.InputWake, "w1")
		require.NoError(t, s.handleLoopInput(h.ctx, item))

		require.Len(t, exec.reqs, 1)
		evt := exec.reqs[0].Events[0].GetEvent()
		require.Equal(t, loops.EventWake, evt.Name)
		require.Equal(t, map[string]any{
			"key": "user-1",
			"id":  "w1",
		}, evt.Data)
		require.NotNil(t, exec.mds[0])
		_, err := h.e.smv2.LoadMetadata(h.ctx, exec.mds[0].ID)
		require.NoError(t, err)
	})

	t.Run("tick for an ended instance schedules nothing", func(t *testing.T) {
		h := newLoopHarness(t)
		s, exec, pub := h.loopService(t, h.req.Function)

		item := h.timerItem(t, queue.KindLoopTick, loops.InputTick, "")
		require.NoError(t, s.handleLoopInput(h.ctx, item))

		require.Len(t, exec.reqs, 1)
		require.Nil(t, exec.mds[0])
		_, err := h.store.Load(h.ctx, h.loopID("user-1"))
		require.ErrorIs(t, err, loops.ErrNotFound)
		require.Equal(t, 0, pub.count())
	})

	t.Run("a publish error does not fail the item", func(t *testing.T) {
		h := newLoopHarness(t)
		h.openAndCommit(t)
		s, exec, pub := h.loopService(t, h.req.Function)
		pub.err = errors.New("publish failed")

		item := h.timerItem(t, queue.KindLoopTick, loops.InputTick, "")
		require.NoError(t, s.handleLoopInput(h.ctx, item))
		require.Len(t, exec.reqs, 1)
		require.Eventually(t, func() bool { return pub.count() == 1 }, 2*time.Second, 10*time.Millisecond)
	})

	t.Run("the same item twice starts one run", func(t *testing.T) {
		h := newLoopHarness(t)
		h.openAndCommit(t)
		s, exec, pub := h.loopService(t, h.req.Function)

		item := h.timerItem(t, queue.KindLoopWake, loops.InputWake, "w1")
		require.NoError(t, s.handleLoopInput(h.ctx, item))

		// a retry or a scavenge moves the item, so the rerun has a new time.
		item.At = item.At.Add(time.Minute)
		require.NoError(t, s.handleLoopInput(h.ctx, item))

		require.Len(t, exec.reqs, 2)
		first := exec.reqs[0].Events[0].GetEvent()
		second := exec.reqs[1].Events[0].GetEvent()
		require.Equal(t, first.ID, second.ID)
		require.Equal(t, exec.reqs[0].Events[0].GetInternalID(), exec.reqs[1].Events[0].GetInternalID())
		require.NotNil(t, exec.reqs[0].IdempotencyKey)
		require.Equal(t, first.ID, *exec.reqs[0].IdempotencyKey)
		require.NotNil(t, exec.mds[0])
		require.Nil(t, exec.mds[1])
		require.Eventually(t, func() bool { return pub.count() == 2 }, 2*time.Second, 10*time.Millisecond)

		// the same job enqueued by another run carries a new nonce and is
		// a new run.
		payload := item.Payload.(queue.PayloadLoop)
		payload.Nonce = ulid.Make().String()
		item.Payload = payload
		require.NoError(t, s.handleLoopInput(h.ctx, item))
		require.Len(t, exec.reqs, 3)
		third := exec.reqs[2].Events[0].GetEvent()
		require.NotEqual(t, first.ID, third.ID)
		require.NotNil(t, exec.mds[2])
	})

	t.Run("the service run loop dispatches both kinds", func(t *testing.T) {
		h := newLoopHarness(t)
		h.openAndCommit(t)
		s, exec, _ := h.loopService(t, h.req.Function)
		s.queueProcessor = loopInputProcessor{
			items: []queue.Item{
				h.timerItem(t, queue.KindLoopTick, loops.InputTick, ""),
				h.timerItem(t, queue.KindLoopWake, loops.InputWake, "w1"),
			},
		}

		require.NoError(t, s.Run(h.ctx))
		require.Len(t, exec.reqs, 2)
		require.Equal(t, loops.EventTick, exec.reqs[0].Events[0].GetEvent().Name)
		require.Equal(t, loops.EventWake, exec.reqs[1].Events[0].GetEvent().Name)
		require.NotNil(t, exec.mds[0])
		require.NotNil(t, exec.mds[1])
	})

	t.Run("function that is no longer a loop drops the item", func(t *testing.T) {
		h := newLoopHarness(t)
		h.openAndCommit(t)
		fn := h.req.Function
		fn.Loop = nil
		s, exec, _ := h.loopService(t, fn)

		item := h.timerItem(t, queue.KindLoopTick, loops.InputTick, "")
		require.NoError(t, s.handleLoopInput(h.ctx, item))
		require.Empty(t, exec.reqs)
	})

	t.Run("bad payload never retries", func(t *testing.T) {
		h := newLoopHarness(t)
		s, _, _ := h.loopService(t, h.req.Function)
		err := s.handleLoopInput(h.ctx, queue.Item{
			Kind:       queue.KindLoopTick,
			Identifier: state.Identifier{WorkflowID: h.req.Function.ID},
			Payload:    json.RawMessage(`{}`),
		})
		require.Error(t, err)
		require.False(t, queue.ShouldRetry(err, 0, 3))
	})
}
