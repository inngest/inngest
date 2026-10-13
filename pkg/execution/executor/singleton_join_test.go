package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/pauses"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/singleton"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// joinPauses records which correlation IDs the fast-resume path looked up and
// hands back a configurable pause.
type joinPauses struct {
	pauses.Manager

	mu       sync.Mutex
	lookups  []string
	pause    *state.Pause
	deleted  int
	lookupFn func(corrID string) (*state.Pause, error)
}

func (p *joinPauses) PauseByInvokeCorrelationID(_ context.Context, _ uuid.UUID, corrID string) (*state.Pause, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lookups = append(p.lookups, corrID)
	if p.lookupFn != nil {
		return p.lookupFn(corrID)
	}
	return nil, state.ErrPauseNotFound
}

func (p *joinPauses) Delete(context.Context, pauses.Index, state.Pause, ...state.DeletePauseOpt) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deleted++
	return nil
}

func (p *joinPauses) lookedUp() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.lookups...)
}

// joinRecorder captures everything the executor publishes.
type joinRecorder struct {
	mu        sync.Mutex
	finished  []event.Event
	invokeErr []execution.InvokeFailHandlerOpts
}

func (r *joinRecorder) finish(_ context.Context, _ sv2.ID, evts []event.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.finished = append(r.finished, evts...)
	return nil
}

func (r *joinRecorder) fail(_ context.Context, opts execution.InvokeFailHandlerOpts, _ []event.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invokeErr = append(r.invokeErr, opts)
	return nil
}

// byCorrelationID returns the published finished events that resolve corrID.
func (r *joinRecorder) byCorrelationID(corrID string) []event.Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	var out []event.Event
	for _, e := range r.finished {
		// Compare as the event stream would see it, i.e. after JSON.
		var wire event.Event
		byt, _ := json.Marshal(e)
		_ = json.Unmarshal(byt, &wire)
		if wire.Data[consts.InvokeCorrelationId] == corrID {
			out = append(out, wire)
		}
	}
	return out
}

// joinRunService stubs run state: schedule only needs Create and Delete.
type joinRunService struct {
	sv2.RunService
	mu      sync.Mutex
	created int
	deleted int

	// runs holds the metadata of created runs. It is only served by
	// LoadMetadata when serve is set.
	runs  map[ulid.ULID]sv2.Metadata
	serve bool
}

func (r *joinRunService) Create(_ context.Context, s sv2.CreateState) (sv2.State, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created++
	if r.runs == nil {
		r.runs = map[ulid.ULID]sv2.Metadata{}
	}
	r.runs[s.Metadata.ID.RunID] = s.Metadata
	return sv2.State{Metadata: s.Metadata}, nil
}

func (r *joinRunService) LoadMetadata(_ context.Context, id sv2.ID, _ ...sv2.LoadMetadataOption) (sv2.Metadata, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if md, ok := r.runs[id.RunID]; ok && r.serve {
		return md, nil
	}
	return sv2.Metadata{}, sv2.ErrMetadataNotFound
}

func (r *joinRunService) Delete(context.Context, sv2.ID, ...sv2.DeleteOption) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleted++
	return nil
}

// joinEnv is a "cluster": one Redis shared by any number of executors, exactly
// like multiple executor nodes sharing a queue.
type joinEnv struct {
	t       *testing.T
	redis   *miniredis.Miniredis
	rc      rueidis.Client
	shard   redis_state.RedisQueueShard
	reg     queue.ShardRegistryController
	store   singleton.Singleton
	queue   queue.Queue
	fn      inngest.Function
	account uuid.UUID
	env     uuid.UUID
	app     uuid.UUID
}

func newJoinEnv(t *testing.T, mode enums.SingletonMode) *joinEnv {
	t.Helper()

	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)

	opts := []queue.QueueOpt{
		queue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return true }),
		queue.WithPartitionConstraintConfigGetter(func(context.Context, queue.PartitionIdentifier) queue.PartitionConstraintConfig {
			return queue.PartitionConstraintConfig{}
		}),
	}
	shard := redis_state.NewQueueShard("default", redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), opts...)
	reg, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(context.Background(), "join-test", reg, opts...)
	require.NoError(t, err)

	return &joinEnv{
		t:       t,
		redis:   r,
		rc:      rc,
		shard:   shard,
		reg:     reg,
		store:   singleton.New(context.Background(), reg),
		queue:   q,
		account: uuid.New(),
		env:     uuid.New(),
		app:     uuid.New(),
		fn: inngest.Function{
			ID:              uuid.New(),
			FunctionVersion: 1,
			Name:            "Singleton join target",
			Slug:            "app-target",
			Singleton:       &inngest.Singleton{Mode: mode},
		},
	}
}

// node returns a new executor node over the shared cluster.
func (c *joinEnv) node(store singleton.Singleton) (*executor, *joinRecorder, *joinPauses, *joinRunService) {
	rec := &joinRecorder{}
	pm := &joinPauses{}
	runs := &joinRunService{}
	if store == nil {
		store = c.store
	}

	e := &executor{
		log:               logger.VoidLogger(),
		queue:             c.queue,
		smv2:              runs,
		pm:                pm,
		singletonMgr:      store,
		tracerProvider:    tracing.NewNoopTracerProvider(),
		conditionalTracer: telemetrytrace.NoopConditionalTracer(),
		finishHandler:     rec.finish,
		invokeFailHandler: rec.fail,
	}
	return e, rec, pm, runs
}

// invoke builds the event step.invoke publishes for an invoker run's step.
func (c *joinEnv) invoke(invokerRun ulid.ULID, step string, expires time.Time) event.TrackedEvent {
	corrID := invokerRun.String() + "." + step
	return event.NewInvocationEvent(event.NewInvocationEventOpts{
		AccountID:     c.account,
		EnvID:         c.env,
		Event:         event.Event{Data: map[string]any{"input": step}},
		FnID:          c.fn.Slug,
		CorrelationID: &corrID,
		ExpiresAt:     expires.UnixMilli(),
	})
}

func (c *joinEnv) plainEvent() event.TrackedEvent {
	id := ulid.Make()
	return event.InternalEvent{
		ID: id,
		Event: event.Event{
			ID:        id.String(),
			Name:      "test/plain",
			Timestamp: time.Now().UnixMilli(),
			Data:      map[string]any{},
		},
	}
}

func (c *joinEnv) schedule(e *executor, evt event.TrackedEvent) (*ulid.ULID, error) {
	c.t.Helper()

	req := execution.ScheduleRequest{
		AccountID:   c.account,
		WorkspaceID: c.env,
		AppID:       c.app,
		Function:    c.fn,
		Events:      []event.TrackedEvent{evt},
	}
	id, _, err := e.schedule(context.Background(), req, ulid.Make(), "key-"+evt.GetInternalID().String(), false, nil)
	return id, err
}

// lockHolder reads the run currently holding the singleton lock.
func (c *joinEnv) lockHolder() *ulid.ULID {
	c.t.Helper()
	val, rerr := c.redis.Get(c.lockKey())
	if rerr != nil {
		return nil
	}
	run := ulid.MustParse(val)
	return &run
}

func (c *joinEnv) lockKey() string {
	key, err := singleton.SingletonKey(context.Background(), c.fn.ID, *c.fn.Singleton, map[string]any{})
	require.NoError(c.t, err)
	return fmt.Sprintf("{%s}:singleton:%s", redis_state.QueueDefaultKey, key)
}

// finalize finalizes the active run as run `id`, as Finalize would, producing
// the events that resolve the run's invoker and any joiners.
func (c *joinEnv) finalize(e *executor, id ulid.ULID, resp execution.FinalizeResponse, cancel bool, trigger event.TrackedEvent) error {
	c.t.Helper()

	md := sv2.Metadata{
		ID: sv2.ID{
			RunID:      id,
			FunctionID: c.fn.ID,
			Tenant:     sv2.Tenant{AccountID: c.account, EnvID: c.env, AppID: c.app},
		},
		Config: *sv2.InitConfig(&sv2.Config{}),
	}
	md.Config.SetSingletonJoin(true)

	raw, err := json.Marshal(trigger.GetEvent())
	require.NoError(c.t, err)

	return e.finalizeEvents(context.Background(), execution.FinalizeOpts{
		Metadata: md,
		Response: resp,
		Optional: execution.FinalizeOptional{
			FnSlug:      c.fn.Slug,
			InputEvents: []json.RawMessage{raw},
			Cancel:      cancel,
		},
	}, nil)
}

func completed(result string) execution.FinalizeResponse {
	return execution.FinalizeResponse{
		Type:        execution.FinalizeResponseRunComplete,
		RunComplete: state.GeneratorOpcode{Op: enums.OpcodeRunComplete, Data: json.RawMessage(result)},
	}
}

func failedWith(message string) execution.FinalizeResponse {
	return execution.FinalizeResponse{
		Type: execution.FinalizeResponseDriver,
		DriverResponse: state.DriverResponse{
			Err:       &message,
			UserError: &state.UserError{Name: "Error", Message: message},
		},
	}
}

func requireSkipped(t *testing.T, err error) {
	t.Helper()
	var skipped SkippedError
	require.True(t, errors.As(err, &skipped), "expected a skip, got %v", err)
	require.Equal(t, enums.SkipReasonSingleton, skipped.Reason)
}

// startActive schedules the run that wins the singleton lock.
func (c *joinEnv) startActive(e *executor) (ulid.ULID, event.TrackedEvent) {
	c.t.Helper()

	trigger := c.plainEvent()
	runID, err := c.schedule(e, trigger)
	require.NoError(c.t, err)
	require.NotNil(c.t, runID)

	holder := c.lockHolder()
	require.NotNil(c.t, holder, "the active run must hold the singleton lock")
	require.Equal(c.t, *runID, *holder)
	return *runID, trigger
}

func TestSingletonJoin_JoinerResolvedWithActiveRunOutput(t *testing.T) {
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, rec, pm, _ := c.node(nil)

	active, trigger := c.startActive(e)

	// A caller invokes the function while it is active: it is skipped...
	corrID := ulid.Make().String() + ".build"
	evt := c.invoke(ulid.MustParse(corrID[:26]), "build", time.Now().Add(time.Hour))
	_, err := c.schedule(e, evt)
	requireSkipped(t, err)

	// ...but is NOT resolved yet: the active run is still going.
	require.Empty(t, rec.byCorrelationID(corrID))

	// The active run finishes.
	require.NoError(t, c.finalize(e, active, completed(`{"image":"sha256:abc"}`), false, trigger))

	got := rec.byCorrelationID(corrID)
	require.Len(t, got, 1, "the joiner is resolved exactly once")
	require.Equal(t, event.FnFinishedName, got[0].Name)
	require.Equal(t, active.String(), fmt.Sprint(got[0].Data["run_id"]), "joiner is pointed at the run it joined")
	require.Equal(t, c.fn.Slug, got[0].Data["function_id"])
	require.JSONEq(t, `{"image":"sha256:abc"}`, mustJSON(t, got[0].Data["result"]))
	require.NotContains(t, got[0].Data, "error")

	// And the invoker's pause is looked up for a fast resume.
	require.Eventually(t, func() bool {
		return contains(pm.lookedUp(), corrID)
	}, 2*time.Second, 5*time.Millisecond)
}

func TestSingletonJoin_RaceActiveRunFinishesBeforeJoin(t *testing.T) {
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, rec, pm, _ := c.node(nil)

	active, trigger := c.startActive(e)

	// The run finishes (and emits its finished events) while the singleton
	// lock is still held: the lock is only released when its last queue item is
	// dequeued, which is after finalize.
	require.NoError(t, c.finalize(e, active, completed(`"done"`), false, trigger))
	require.NotNil(t, c.lockHolder(), "lock outlives finalize")

	// A caller now invokes the function. The singleton check still says
	// "skip", but the join must resolve from the stored completion.
	corrID := ulid.Make().String() + ".late"
	_, err := c.schedule(e, c.invoke(ulid.MustParse(corrID[:26]), "late", time.Now().Add(time.Hour)))
	requireSkipped(t, err)

	got := rec.byCorrelationID(corrID)
	require.Len(t, got, 1, "a late joiner is resolved immediately from the stored result")
	require.JSONEq(t, `"done"`, mustJSON(t, got[0].Data["result"]))
	require.Equal(t, active.String(), fmt.Sprint(got[0].Data["run_id"]))
	require.Eventually(t, func() bool { return contains(pm.lookedUp(), corrID) }, 2*time.Second, 5*time.Millisecond)
}

func TestSingletonJoin_ActiveRunFailsOrIsCancelled(t *testing.T) {
	tests := []struct {
		name       string
		resp       execution.FinalizeResponse
		cancel     bool
		wantStatus enums.StepStatus
		wantError  bool
	}{
		{name: "failed", resp: failedWith("boom"), wantStatus: enums.StepStatusFailed, wantError: true},
		{name: "cancelled", resp: failedWith("cancelled"), cancel: true, wantStatus: enums.StepStatusCancelled, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newJoinEnv(t, enums.SingletonModeJoin)
			e, rec, _, _ := c.node(nil)
			active, trigger := c.startActive(e)

			corrID := ulid.Make().String() + ".build"
			_, err := c.schedule(e, c.invoke(ulid.MustParse(corrID[:26]), "build", time.Now().Add(time.Hour)))
			requireSkipped(t, err)

			require.NoError(t, c.finalize(e, active, tt.resp, tt.cancel, trigger))

			got := rec.byCorrelationID(corrID)
			require.Len(t, got, 1, "the joiner must never hang")
			if tt.wantError {
				require.Contains(t, got[0].Data, "error", "joiner gets the active run's error so it can fall back")
			}
			require.NotContains(t, got[0].Data, "result")

			// The resume payload the invoker sees is the error, as for a failed invoke.
			opcode := enums.OpcodeInvokeFunction.String()
			p := state.Pause{Opcode: &opcode, StepName: "build"}
			resume := p.GetResumeData(got[0])
			require.Contains(t, resume.With, "error")
			require.NotNil(t, resume.RunID)
			require.Equal(t, active, *resume.RunID)
		})
	}
}

func TestSingletonJoin_JoinedResultMatchesInvokeResume(t *testing.T) {
	// A joiner resumes with exactly the shape a normal finished invoke
	// resumes with, so memoization/replay of the invoker is unchanged.
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, rec, _, _ := c.node(nil)
	active, trigger := c.startActive(e)

	corrID := ulid.Make().String() + ".build"
	_, err := c.schedule(e, c.invoke(ulid.MustParse(corrID[:26]), "build", time.Now().Add(time.Hour)))
	requireSkipped(t, err)
	require.NoError(t, c.finalize(e, active, completed(`{"n":1}`), false, trigger))

	joined := rec.byCorrelationID(corrID)
	require.Len(t, joined, 1)

	opcode := enums.OpcodeInvokeFunction.String()
	p := state.Pause{Opcode: &opcode, StepName: "build"}

	// Round-trip through JSON like the event stream does.
	var wire event.Event
	require.NoError(t, json.Unmarshal([]byte(mustJSON(t, joined[0])), &wire))
	resume := p.GetResumeData(wire)
	require.Equal(t, map[string]any{"data": map[string]any{"n": float64(1)}}, resume.With)
}

func TestSingletonJoin_ManyJoinersResolveExactlyOnce(t *testing.T) {
	// N callers join; the active run finishes at an arbitrary point in the
	// middle of the joins. Every caller is resolved exactly once.
	const (
		rounds  = 10
		callers = 40
	)

	for round := range rounds {
		t.Run(fmt.Sprintf("round %d", round), func(t *testing.T) {
			c := newJoinEnv(t, enums.SingletonModeJoin)
			e, rec, _, _ := c.node(nil)
			active, trigger := c.startActive(e)

			corrIDs := make([]string, callers)
			start := make(chan struct{})
			var wg sync.WaitGroup

			for i := range callers {
				invoker := ulid.Make()
				corrIDs[i] = invoker.String() + ".build"
				evt := c.invoke(invoker, "build", time.Now().Add(time.Hour))

				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					_, err := c.schedule(e, evt)
					requireSkipped(t, err)
				}()
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				time.Sleep(time.Duration(round) * 300 * time.Microsecond)
				require.NoError(t, c.finalize(e, active, completed(`"ok"`), false, trigger))
			}()

			close(start)
			wg.Wait()

			for _, corrID := range corrIDs {
				require.Lenf(t, rec.byCorrelationID(corrID), 1, "caller %s must be resolved exactly once", corrID)
			}
		})
	}
}

func TestSingletonJoin_RetriedFinalizeIsIdempotent(t *testing.T) {
	// A finalize that is retried (e.g. the publish failed after completing)
	// re-resolves the same joiners; the resume is idempotent, and no joiner is
	// lost or invented.
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, rec, _, _ := c.node(nil)
	active, trigger := c.startActive(e)

	corrID := ulid.Make().String() + ".build"
	_, err := c.schedule(e, c.invoke(ulid.MustParse(corrID[:26]), "build", time.Now().Add(time.Hour)))
	requireSkipped(t, err)

	require.NoError(t, c.finalize(e, active, completed(`1`), false, trigger))
	require.NoError(t, c.finalize(e, active, completed(`1`), false, trigger))

	got := rec.byCorrelationID(corrID)
	require.NotEmpty(t, got)
	for _, g := range got {
		require.Equal(t, got[0].Data["result"], g.Data["result"], "retries carry the same outcome")
	}
}

func TestSingletonJoin_ExpiredJoinerIsNotResumed(t *testing.T) {
	// The caller's own invoke timeout still applies: a finished event for an
	// expired pause is dropped by the normal invoke path and the pause is
	// not resumed.
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, rec, pm, _ := c.node(nil)
	active, trigger := c.startActive(e)

	corrID := ulid.Make().String() + ".build"
	_, err := c.schedule(e, c.invoke(ulid.MustParse(corrID[:26]), "build", time.Now().Add(time.Hour)))
	requireSkipped(t, err)

	eventName := event.FnFinishedName
	pm.lookupFn = func(string) (*state.Pause, error) {
		// The caller's invoke timed out long ago.
		return &state.Pause{
			ID:          uuid.New(),
			WorkspaceID: c.env,
			Event:       &eventName,
			Expires:     state.Time(time.Now().Add(-48 * time.Hour)),
		}, nil
	}

	require.NoError(t, c.finalize(e, active, completed(`1`), false, trigger))

	require.Len(t, rec.byCorrelationID(corrID), 1, "the event is still published")
	require.Eventually(t, func() bool {
		pm.mu.Lock()
		defer pm.mu.Unlock()
		return pm.deleted > 0
	}, 2*time.Second, 5*time.Millisecond, "the expired pause is cleaned up and never resumed")
}

func TestSingletonJoin_RegistrationLifetimeFollowsInvokeTimeout(t *testing.T) {
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, _, _, _ := c.node(nil)
	active, _ := c.startActive(e)

	_, err := c.schedule(e, c.invoke(ulid.Make(), "build", time.Now().Add(2*time.Hour)))
	requireSkipped(t, err)

	setKey := fmt.Sprintf("{%s}:singleton-join:%s", redis_state.QueueDefaultKey, active)
	ttl := c.redis.TTL(setKey)
	require.Greater(t, ttl, 2*time.Hour, "registration outlives the caller's timeout by the grace period")
	require.LessOrEqual(t, ttl, 2*time.Hour+singletonJoinGrace+time.Minute)

	// The active run never finalizes (deleted, expired, lost): nothing is
	// orphaned once the caller's own timeout window has passed.
	c.redis.FastForward(2*time.Hour + singletonJoinGrace + time.Minute)
	require.False(t, c.redis.Exists(setKey))
}

func TestSingletonJoin_SelfJoinFailsFastInsteadOfDeadlocking(t *testing.T) {
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, rec, _, _ := c.node(nil)
	active, _ := c.startActive(e)

	// The active run invokes its own function with the same key: joining
	// itself would wait for itself.
	_, err := c.schedule(e, c.invoke(active, "recurse", time.Now().Add(time.Hour)))
	requireSkipped(t, err)

	require.Len(t, rec.invokeErr, 1)
	require.Contains(t, rec.invokeErr[0].Err["message"], "cannot join itself")
}

func TestSingletonJoin_NonInvokeTriggerIsAPlainSkip(t *testing.T) {
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, rec, _, _ := c.node(nil)
	active, _ := c.startActive(e)

	_, err := c.schedule(e, c.plainEvent())
	requireSkipped(t, err)

	setKey := fmt.Sprintf("{%s}:singleton-join:%s", redis_state.QueueDefaultKey, active)
	require.False(t, c.redis.Exists(setKey), "nothing to wait for, nothing registered")
	require.Empty(t, rec.invokeErr)
}

// countingStore counts how the store is used, and can hide the lock holder from
// the first lookup to force the enqueue-time (atomic) conflict path.
type countingStore struct {
	singleton.Singleton

	mu          sync.Mutex
	joins       int
	handles     int
	hideFirstN  int
	handleModes []enums.SingletonMode
}

func (s *countingStore) HandleSingleton(ctx context.Context, scope queue.Scope, key string, cfg inngest.Singleton) (*ulid.ULID, error) {
	s.mu.Lock()
	s.handles++
	s.handleModes = append(s.handleModes, cfg.Mode)
	hide := s.hideFirstN > 0
	if hide {
		s.hideFirstN--
	}
	s.mu.Unlock()

	if hide {
		return nil, nil
	}
	return s.Singleton.HandleSingleton(ctx, scope, key, cfg)
}

func (s *countingStore) Join(ctx context.Context, scope queue.Scope, run ulid.ULID, waiter string, ttl time.Duration) ([]byte, error) {
	s.mu.Lock()
	s.joins++
	s.mu.Unlock()
	return s.Singleton.Join(ctx, scope, run, waiter, ttl)
}

func TestSingletonJoin_LosingTheEnqueueRaceStillJoins(t *testing.T) {
	// Two schedulers pass the "is the lock free?" check, then race on the
	// atomic acquire at enqueue. The loser must find the winner and join it.
	c := newJoinEnv(t, enums.SingletonModeJoin)
	winner, wrec, _, _ := c.node(nil)
	active, trigger := c.startActive(winner)

	loserStore := &countingStore{Singleton: c.store, hideFirstN: 1}
	loser, _, _, loserRuns := c.node(loserStore)

	corrID := ulid.Make().String() + ".build"
	_, err := c.schedule(loser, c.invoke(ulid.MustParse(corrID[:26]), "build", time.Now().Add(time.Hour)))
	requireSkipped(t, err)

	require.Equal(t, 1, loserStore.joins, "the loser joined the winner")
	require.Equal(t, 1, loserRuns.deleted, "the loser's speculative run state was cleaned up")
	// The first lookup uses the function's own mode; the post-conflict
	// lookup is explicitly read-only.
	require.Equal(t, []enums.SingletonMode{enums.SingletonModeJoin, enums.SingletonModeSkip}, loserStore.handleModes,
		"join never releases the lock")

	require.NoError(t, c.finalize(winner, active, completed(`"x"`), false, trigger))
	require.Len(t, wrec.byCorrelationID(corrID), 1)
}

func TestSingletonJoin_LockReleasedBeforeLookupFailsInvokerFast(t *testing.T) {
	// The lock vanishes between the enqueue conflict and the lookup. The
	// caller must not hang until its invoke timeout: it fails fast and falls
	// back.
	c := newJoinEnv(t, enums.SingletonModeJoin)
	winner, _, _, _ := c.node(nil)
	c.startActive(winner)

	store := &releasingStore{Singleton: c.store, redis: c.redis, key: c.lockKey()}
	loser, lrec, _, _ := c.node(store)

	corrID := ulid.Make().String() + ".build"
	_, err := c.schedule(loser, c.invoke(ulid.MustParse(corrID[:26]), "build", time.Now().Add(time.Hour)))
	requireSkipped(t, err)

	require.Len(t, lrec.invokeErr, 1)
	require.Contains(t, lrec.invokeErr[0].Err["message"], "ended before it could be joined")
}

// releasingStore hides the holder on the first lookup (so the scheduler
// proceeds to enqueue) and releases the lock before the second (post-conflict)
// lookup.
type releasingStore struct {
	singleton.Singleton
	redis *miniredis.Miniredis
	key   string

	mu    sync.Mutex
	calls int
}

func (s *releasingStore) HandleSingleton(ctx context.Context, scope queue.Scope, key string, cfg inngest.Singleton) (*ulid.ULID, error) {
	s.mu.Lock()
	s.calls++
	call := s.calls
	s.mu.Unlock()

	switch call {
	case 1:
		return nil, nil
	case 2:
		s.redis.Del(s.key)
	}
	return s.Singleton.HandleSingleton(ctx, scope, key, cfg)
}

func TestSingletonJoin_TwoNodesRacingOnTheSameKey(t *testing.T) {
	// Two executor nodes schedule invocations for the same key at the same
	// time, many callers each. Exactly one run wins the lock; every other
	// caller joins it; when it finishes, every caller (winner's included) is
	// resolved exactly once.
	const (
		rounds       = 5
		callersTotal = 24
	)

	for round := range rounds {
		t.Run(fmt.Sprintf("round %d", round), func(t *testing.T) {
			c := newJoinEnv(t, enums.SingletonModeJoin)
			nodeA, recA, _, runsA := c.node(nil)
			nodeB, recB, _, runsB := c.node(nil)
			nodes := []*executor{nodeA, nodeB}

			type call struct {
				corrID string
				evt    event.TrackedEvent
			}
			calls := make([]call, callersTotal)
			for i := range calls {
				invoker := ulid.Make()
				calls[i] = call{
					corrID: invoker.String() + ".build",
					evt:    c.invoke(invoker, "build", time.Now().Add(time.Hour)),
				}
			}

			var (
				start   = make(chan struct{})
				wg      sync.WaitGroup
				mu      sync.Mutex
				winners []ulid.ULID
				winEvt  event.TrackedEvent
			)
			for i, cl := range calls {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					id, err := c.schedule(nodes[i%2], cl.evt)
					if err == nil {
						mu.Lock()
						winners = append(winners, *id)
						winEvt = cl.evt
						mu.Unlock()
						return
					}
					requireSkipped(t, err)
				}()
			}
			close(start)
			wg.Wait()

			require.Len(t, winners, 1, "the atomic enqueue admits exactly one run")

			runsCreated := runsA.created + runsB.created
			require.Equal(t, callersTotal, runsCreated, "every scheduler attempt created speculative state")
			require.Equal(t, callersTotal-1, runsA.deleted+runsB.deleted, "all losers cleaned up their state: no leaked runs")

			// The winner finishes. Everyone, on either node's recorder, is
			// resolved exactly once in total.
			require.NoError(t, c.finalize(nodeA, winners[0], completed(`"shared"`), false, winEvt))

			for _, cl := range calls {
				require.Lenf(t, append(recA.byCorrelationID(cl.corrID), recB.byCorrelationID(cl.corrID)...), 1, "caller %s resolved exactly once", cl.corrID)
			}
		})
	}
}

func TestSingletonJoin_SkipAndCancelModesAreUnchanged(t *testing.T) {
	for _, mode := range []enums.SingletonMode{enums.SingletonModeSkip, enums.SingletonModeCancel} {
		t.Run(mode.String(), func(t *testing.T) {
			c := newJoinEnv(t, mode)
			store := &countingStore{Singleton: c.store}
			e, rec, _, _ := c.node(store)

			active, trigger := c.startActive(e)
			metaCfg := sv2.InitConfig(&sv2.Config{})
			require.False(t, metaCfg.SingletonJoin())

			corrID := ulid.Make().String() + ".build"
			_, err := c.schedule(e, c.invoke(ulid.MustParse(corrID[:26]), "build", time.Now().Add(time.Hour)))

			switch mode {
			case enums.SingletonModeSkip:
				requireSkipped(t, err)
			case enums.SingletonModeCancel:
				// Cancel releases the lock and proceeds to start the new run.
				require.NoError(t, err)
			}

			require.Zero(t, store.joins, "only join mode ever registers waiters")
			setKey := fmt.Sprintf("{%s}:singleton-join:%s", redis_state.QueueDefaultKey, active)
			require.False(t, c.redis.Exists(setKey))
			require.Empty(t, rec.invokeErr)

			// Finalize does no join work for runs not scheduled in join mode.
			md := sv2.Metadata{
				ID:     sv2.ID{RunID: active, FunctionID: c.fn.ID, Tenant: sv2.Tenant{AccountID: c.account, EnvID: c.env}},
				Config: *sv2.InitConfig(&sv2.Config{}),
			}
			raw, _ := json.Marshal(trigger.GetEvent())
			require.NoError(t, e.finalizeEvents(context.Background(), execution.FinalizeOpts{
				Metadata: md,
				Response: completed(`1`),
				Optional: execution.FinalizeOptional{FnSlug: c.fn.Slug, InputEvents: []json.RawMessage{raw}},
			}, nil))
			doneKey := fmt.Sprintf("{%s}:singleton-join-done:%s", redis_state.QueueDefaultKey, active)
			require.False(t, c.redis.Exists(doneKey), "no completion payload is stored for non-join runs")
		})
	}
}

func TestSingletonJoin_ModeParsesFromSDKConfig(t *testing.T) {
	var cfg inngest.Singleton
	require.NoError(t, json.Unmarshal([]byte(`{"key":"event.data.user","mode":"join"}`), &cfg))
	require.Equal(t, enums.SingletonModeJoin, cfg.Mode)

	// Existing modes are unchanged.
	require.NoError(t, json.Unmarshal([]byte(`{"mode":"skip"}`), &cfg))
	require.Equal(t, enums.SingletonModeSkip, cfg.Mode)
	require.NoError(t, json.Unmarshal([]byte(`{"mode":"cancel"}`), &cfg))
	require.Equal(t, enums.SingletonModeCancel, cfg.Mode)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	byt, err := json.Marshal(v)
	require.NoError(t, err)
	return string(byt)
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func TestSingletonJoin_ScheduleFlagsOnlyJoinModeRuns(t *testing.T) {
	for _, mode := range []enums.SingletonMode{enums.SingletonModeSkip, enums.SingletonModeCancel, enums.SingletonModeJoin} {
		t.Run(mode.String(), func(t *testing.T) {
			c := newJoinEnv(t, mode)
			e, _, _, runs := c.node(nil)

			active, _ := c.startActive(e)
			created := runs.runs[active]
			require.Equal(t, mode == enums.SingletonModeJoin, created.Config.SingletonJoin(),
				"only join-mode runs resolve joiners when they finalize")
		})
	}
}

func TestSingletonJoin_ActiveRunStartedBeforeJoinModeFailsFast(t *testing.T) {
	// Joining a run that was scheduled before the function switched to join
	// mode could never be resolved at finalize. The caller must be failed
	// immediately instead of waiting out its invoke timeout.
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, rec, _, runs := c.node(nil)
	active, _ := c.startActive(e)

	legacy := runs.runs[active]
	legacy.Config = *sv2.InitConfig(&sv2.Config{})
	runs.runs[active] = legacy
	runs.serve = true

	_, err := c.schedule(e, c.invoke(ulid.Make(), "build", time.Now().Add(time.Hour)))
	requireSkipped(t, err)

	require.Len(t, rec.invokeErr, 1)
	require.Contains(t, rec.invokeErr[0].Err["message"], "before join mode was enabled")
}

func TestSingletonJoin_FlaggedActiveRunIsJoinableWhenStateIsServed(t *testing.T) {
	c := newJoinEnv(t, enums.SingletonModeJoin)
	e, rec, _, runs := c.node(nil)
	active, trigger := c.startActive(e)
	runs.serve = true

	corrID := ulid.Make().String() + ".build"
	_, err := c.schedule(e, c.invoke(ulid.MustParse(corrID[:26]), "build", time.Now().Add(time.Hour)))
	requireSkipped(t, err)
	require.Empty(t, rec.invokeErr, "a join-mode run is joined, not rejected")

	require.NoError(t, c.finalize(e, active, completed(`1`), false, trigger))
	require.Len(t, rec.byCorrelationID(corrID), 1)
}
