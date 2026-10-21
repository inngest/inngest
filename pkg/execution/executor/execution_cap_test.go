package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/batch"
	"github.com/inngest/inngest/pkg/execution/debounce"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/logger"
	telemetrytrace "github.com/inngest/inngest/pkg/telemetry/trace"
	"github.com/inngest/inngest/pkg/tracing"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

func TestScheduleAccountExecutionCap(t *testing.T) {
	pausedAt := time.Now().Add(-time.Minute)

	tests := []struct {
		name       string
		capFn      ExecutionCapFn
		eventName  string
		pausedAt   *time.Time
		wantReason enums.SkipReason
	}{
		{
			name:       "exceeded and enforced skips with cap reason",
			capFn:      capDecision(ExecutionCapLimitDecision{Exceeded: true, Enforce: true}),
			wantReason: enums.SkipReasonAccountExecutionCapHit,
		},
		{
			name:       "enforced cap skips invokes without creating or deleting state",
			capFn:      capDecision(ExecutionCapLimitDecision{Exceeded: true, Enforce: true}),
			eventName:  event.InvokeFnName,
			wantReason: enums.SkipReasonAccountExecutionCapHit,
		},
		{
			name:       "no cap configured continues to normal skip checks",
			pausedAt:   &pausedAt,
			wantReason: enums.SkipReasonFunctionPaused,
		},
		{
			name:       "exceeded without enforce continues to normal skip checks",
			capFn:      capDecision(ExecutionCapLimitDecision{Exceeded: true, Enforce: false}),
			pausedAt:   &pausedAt,
			wantReason: enums.SkipReasonFunctionPaused,
		},
		{
			name:       "not exceeded continues to normal skip checks",
			capFn:      capDecision(ExecutionCapLimitDecision{Exceeded: false, Enforce: true}),
			pausedAt:   &pausedAt,
			wantReason: enums.SkipReasonFunctionPaused,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := newRecordingTracerProvider()
			// Leave smv2 nil so any state access, including Create or Delete, fails.
			e := &executor{
				log:                 logger.From(context.Background()),
				tracerProvider:      rec,
				conditionalTracer:   telemetrytrace.NoopConditionalTracer(),
				accountExecutionCap: tt.capFn,
			}

			req := capScheduleRequest(nil)
			req.FunctionPausedAt = tt.pausedAt
			if tt.eventName != "" {
				evt := req.Events[0].(event.InternalEvent)
				evt.Event.Name = tt.eventName
				req.Events[0] = evt
			}

			runID, md, err := e.schedule(context.Background(), req, ulid.Make(), "test-key", false, nil)
			require.ErrorIs(t, err, ErrFunctionSkipped)
			require.Nil(t, runID)
			require.Nil(t, md)

			var skipped SkippedError
			require.True(t, errors.As(err, &skipped))
			require.Equal(t, tt.wantReason, skipped.Reason)

			var runSpan *createSpanCall
			for _, call := range rec.createCalls {
				if call.name == meta.SpanNameRun {
					runSpan = call
					break
				}
			}
			require.NotNil(t, runSpan)
			spanReason, ok := runSpan.opts.Attributes.Get(meta.Attrs.SkipReason.Key()).(*enums.SkipReason)
			require.True(t, ok)
			require.Equal(t, tt.wantReason, *spanReason)
		})
	}
}

func TestScheduleAccountExecutionCapSkipsSingletonHandling(t *testing.T) {
	sm := &recordingSingletonManager{}
	e := &executor{
		log:                 logger.From(context.Background()),
		tracerProvider:      newRecordingTracerProvider(),
		conditionalTracer:   telemetrytrace.NoopConditionalTracer(),
		singletonMgr:        sm,
		accountExecutionCap: capDecision(ExecutionCapLimitDecision{Exceeded: true, Enforce: true}),
	}

	req := capScheduleRequest(&inngest.Singleton{Mode: enums.SingletonModeCancel})

	_, _, err := e.schedule(context.Background(), req, ulid.Make(), "test-key", false, nil)

	var skipped SkippedError
	require.True(t, errors.As(err, &skipped))
	require.Equal(t, enums.SkipReasonAccountExecutionCapHit, skipped.Reason)
	require.Equal(t, 0, sm.calls)
}

func capDecision(d ExecutionCapLimitDecision) ExecutionCapFn {
	return func(context.Context, uuid.UUID) ExecutionCapLimitDecision { return d }
}

func capScheduleRequest(singleton *inngest.Singleton) execution.ScheduleRequest {
	eventID := ulid.Make()
	return execution.ScheduleRequest{
		AccountID:   uuid.New(),
		WorkspaceID: uuid.New(),
		AppID:       uuid.New(),
		Function: inngest.Function{
			ID:              uuid.New(),
			FunctionVersion: 1,
			Name:            "capped-fn",
			Singleton:       singleton,
		},
		Events: []event.TrackedEvent{
			event.InternalEvent{
				ID: eventID,
				Event: event.Event{
					ID:        eventID.String(),
					Name:      "test/cap",
					Timestamp: time.Now().UnixMilli(),
					Data:      map[string]any{},
				},
			},
		},
	}
}

type recordingSingletonManager struct {
	calls int
}

func (r *recordingSingletonManager) HandleSingleton(context.Context, queue.Scope, string, inngest.Singleton) (*ulid.ULID, error) {
	r.calls++
	return nil, nil
}

func TestScheduleAccountExecutionCapInvokes(t *testing.T) {
	for _, tt := range []struct {
		name      string
		configure func(*execution.ScheduleRequest)
		wantSkip  enums.SkipReason
	}{
		{name: "step invoke"},
		{
			name: "historical event resend",
			configure: func(req *execution.ScheduleRequest) {
				// Event resend assigns a new internal ID but preserves the payload.
				encoded, err := json.Marshal(req.Events[0].GetEvent())
				require.NoError(t, err)
				var stored event.Event
				require.NoError(t, json.Unmarshal(encoded, &stored))
				req.Events[0] = event.NewBaseTrackedEventWithID(stored, ulid.Make())
			},
		},
		{
			name: "API or manual invoke",
			configure: func(req *execution.ScheduleRequest) {
				req.Events[0] = event.NewInvocationEvent(event.NewInvocationEventOpts{FnID: req.Function.ID.String()})
			},
			wantSkip: enums.SkipReasonAccountExecutionCapHit,
		},
		{
			name: "API invoke with supplied executor metadata",
			configure: func(req *execution.ScheduleRequest) {
				correlationID := ulid.Make().String() + ".supplied-step"
				req.Events[0] = event.NewInvocationEvent(event.NewInvocationEventOpts{
					FnID: req.Function.ID.String(), CorrelationID: &correlationID,
					Event: req.Events[0].GetEvent(),
				})
			},
			wantSkip: enums.SkipReasonAccountExecutionCapHit,
		},
		{
			name: "correlation alone",
			configure: func(req *execution.ScheduleRequest) {
				evt := req.Events[0].GetEvent()
				md, err := evt.InngestMetadata()
				require.NoError(t, err)
				md.SourceAppID, md.SourceFnID = "", ""
				evt.Data[consts.InngestEventDataPrefix] = *md
				req.Events[0] = event.NewBaseTrackedEventWithID(evt, ulid.Make())
			},
			wantSkip: enums.SkipReasonAccountExecutionCapHit,
		},
		{
			name: "invalid parent correlation",
			configure: func(req *execution.ScheduleRequest) {
				evt := req.Events[0].GetEvent()
				md, err := evt.InngestMetadata()
				require.NoError(t, err)
				md.InvokeCorrelationId = "not-a-run.call-child"
				evt.Data[consts.InngestEventDataPrefix] = *md
				req.Events[0] = event.NewBaseTrackedEventWithID(evt, ulid.Make())
			},
			wantSkip: enums.SkipReasonAccountExecutionCapHit,
		},
		{
			name: "ordinary event with copied metadata",
			configure: func(req *execution.ScheduleRequest) {
				evt := req.Events[0].GetEvent()
				evt.Name = "test/copied-invoke"
				req.Events[0] = event.NewBaseTrackedEventWithID(evt, ulid.Make())
			},
			wantSkip: enums.SkipReasonAccountExecutionCapHit,
		},
		{
			name: "malformed metadata",
			configure: func(req *execution.ScheduleRequest) {
				evt := req.Events[0].GetEvent()
				evt.Data[consts.InngestEventDataPrefix] = "invalid"
				req.Events[0] = event.NewBaseTrackedEventWithID(evt, ulid.Make())
			},
			wantSkip: enums.SkipReasonAccountExecutionCapHit,
		},
		{
			name: "exemption does not bypass pause",
			configure: func(req *execution.ScheduleRequest) {
				paused := time.Now().Add(-time.Minute)
				req.FunctionPausedAt = &paused
			},
			wantSkip: enums.SkipReasonFunctionPaused,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e, req := newExecutionCapScheduler(t)
			req.Events = []event.TrackedEvent{executionCapInvoke(req)}
			if tt.configure != nil {
				tt.configure(&req)
			}
			_, md, err := e.Schedule(t.Context(), req)
			if tt.wantSkip != enums.SkipReasonNone {
				var skipped SkippedError
				require.ErrorAs(t, err, &skipped)
				require.Equal(t, tt.wantSkip, skipped.Reason)
				require.Nil(t, md)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, md)
			events, err := e.smv2.LoadEvents(t.Context(), md.ID)
			require.NoError(t, err)
			require.Len(t, events, 1)
			jobs, err := e.queue.RunJobs(t.Context(), "local", queue.Scope{AccountID: req.AccountID, EnvID: req.WorkspaceID, FunctionID: req.Function.ID}, md.ID.RunID, 10, 0)
			require.NoError(t, err)
			require.Len(t, jobs, 1)
		})
	}
}

func TestScheduleAccountExecutionCapReruns(t *testing.T) {
	for _, identity := range []string{"original run", "replay", "schedule type"} {
		for _, debounced := range []bool{false, true} {
			for _, invoke := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/debounce=%t/invoke=%t", identity, debounced, invoke), func(t *testing.T) {
					e, req := newExecutionCapScheduler(t)
					switch identity {
					case "original run":
						id := ulid.Make()
						req.OriginalRunID = &id
					case "replay":
						id := uuid.New()
						req.ReplayID = &id
					case "schedule type":
						req.ScheduleType = enums.ScheduleTypeRerun
					}
					if invoke {
						req.Events = []event.TrackedEvent{executionCapInvoke(req)}
					}
					if debounced {
						req.Function.Debounce = &inngest.Debounce{Period: "10s"}
						_, md, err := e.Schedule(t.Context(), req)
						require.ErrorIs(t, err, ErrFunctionDebounced)
						require.Nil(t, md)
						scope := queue.Scope{AccountID: req.AccountID, EnvID: req.WorkspaceID, FunctionID: req.Function.ID}
						info, err := e.debouncer.GetDebounceInfo(t.Context(), scope, "")
						require.NoError(t, err)
						require.NotNil(t, info.Item)
						require.Equal(t, req.Events[0].GetInternalID(), info.Item.EventID)
						require.NoError(t, e.debouncer.StartExecution(t.Context(), *info.Item, req.Function, ulid.MustParse(info.DebounceID)))
						// Flush reconstructs a request from the selected event, without rerun identity.
						req = execution.ScheduleRequest{
							AccountID: req.AccountID, WorkspaceID: req.WorkspaceID, AppID: req.AppID,
							Function: req.Function, Events: []event.TrackedEvent{info.Item},
							PreventDebounce: true, PreventRateLimit: true,
						}
					}
					_, md, err := e.Schedule(t.Context(), req)
					if !invoke {
						var skipped SkippedError
						require.ErrorAs(t, err, &skipped)
						require.Equal(t, enums.SkipReasonAccountExecutionCapHit, skipped.Reason)
						require.Nil(t, md)
						return
					}
					require.NoError(t, err)
					require.NotNil(t, md)
					jobs, err := e.queue.RunJobs(t.Context(), "local", queue.Scope{AccountID: req.AccountID, EnvID: req.WorkspaceID, FunctionID: req.Function.ID}, md.ID.RunID, 10, 0)
					require.NoError(t, err)
					require.Len(t, jobs, 1)
				})
			}
		}
	}
}

func TestScheduleAccountExecutionCapInvokeRetry(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		t.Run(map[bool]string{false: "already enqueued", true: "interrupted enqueue"}[interrupted], func(t *testing.T) {
			e, req := newExecutionCapScheduler(t)
			req.Events = []event.TrackedEvent{executionCapInvoke(req)}
			e.accountExecutionCap = capDecision(ExecutionCapLimitDecision{})
			e.queue = &singletonRetryQueue{Queue: e.queue, failNext: interrupted}
			runID := ulid.Make()
			req.RunID = &runID
			_, _, err := e.Schedule(t.Context(), req)
			if interrupted {
				require.ErrorIs(t, err, errSingletonRetryEnqueue)
			} else {
				require.NoError(t, err)
			}
			e.accountExecutionCap = capDecision(ExecutionCapLimitDecision{Exceeded: true, Enforce: true})
			req.RunID = nil
			retryID, _, err := e.Schedule(t.Context(), req)
			if interrupted {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, state.ErrIdentifierExists)
			}
			require.Equal(t, &runID, retryID)
			jobs, err := e.queue.RunJobs(t.Context(), "local", queue.Scope{AccountID: req.AccountID, EnvID: req.WorkspaceID, FunctionID: req.Function.ID}, runID, 10, 0)
			require.NoError(t, err)
			require.Len(t, jobs, 1)
		})
	}
}

func TestScheduleAccountExecutionCapBatches(t *testing.T) {
	for _, flush := range []string{"full", "timeout", "bytes"} {
		for _, invokeIndex := range []int{-1, 0, 1, 2} {
			t.Run(flush+"/"+map[int]string{-1: "ordinary only", 0: "invoke first", 1: "invoke middle", 2: "invoke last"}[invokeIndex], func(t *testing.T) {
				e, req := newExecutionCapScheduler(t, batch.WithRedisBatchSizeLimit(10_000))
				req.Function.EventBatch = &inngest.EventBatchConfig{MaxSize: 3, Timeout: "1s"}
				if flush != "full" {
					req.Function.EventBatch.MaxSize = 10
				}
				var batchID ulid.ULID
				for i := range 3 {
					evt := req.Events[0]
					if i == invokeIndex {
						evt = executionCapInvoke(req)
					}
					input := evt.GetEvent()
					if flush == "bytes" && i == 2 {
						input.Data["large"] = strings.Repeat("x", 10_000)
					}
					require.NoError(t, e.AppendAndScheduleBatch(t.Context(), req.Function, batch.BatchItem{
						AccountID: req.AccountID, WorkspaceID: req.WorkspaceID, AppID: req.AppID,
						FunctionID: req.Function.ID, FunctionVersion: 1, EventID: ulid.Make(), Event: input,
					}, nil))
					if i == 0 {
						info, err := e.batcher.GetBatchInfo(t.Context(), req.Function.ID, "")
						require.NoError(t, err)
						batchID = ulid.MustParse(info.BatchID)
					}
				}
				if flush == "timeout" {
					jobID := queue.HashID(t.Context(), fmt.Sprintf("%s:%s", req.WorkspaceID, batchID))
					job, err := e.queue.LoadQueueItem(t.Context(), "local", jobID)
					require.NoError(t, err)
					var payload batch.ScheduleBatchPayload
					require.NoError(t, json.Unmarshal(job.Data.Payload.(json.RawMessage), &payload))
					_, err = e.batcher.StartExecution(t.Context(), req.Function.ID, payload.BatchID, payload.BatchPointer)
					require.NoError(t, err)
					require.NoError(t, e.RetrieveAndScheduleBatch(t.Context(), req.Function, payload, nil))
				}
				key := fmt.Sprintf("%s-%s", req.Function.ID, batchID)
				req.IdempotencyKey = &key
				id := sv2.ID{FunctionID: req.Function.ID, Tenant: sv2.Tenant{AccountID: req.AccountID, EnvID: req.WorkspaceID, AppID: req.AppID}}
				entry, err := e.smv2.LookupIdempotency(t.Context(), id, idempotencyKey(req, ulid.Make()))
				require.NoError(t, err)
				if invokeIndex < 0 {
					require.Nil(t, entry, "ordinary-only batch must not create a run")
				} else {
					require.NotNil(t, entry, "eligible batch must create a run")
					id.RunID = entry.RunID
					events, err := e.smv2.LoadEvents(t.Context(), id)
					require.NoError(t, err)
					require.Len(t, events, 3, "keep ordinary members of mixed batches")
					jobs, err := e.queue.RunJobs(t.Context(), "local", queue.Scope{AccountID: req.AccountID, EnvID: req.WorkspaceID, FunctionID: req.Function.ID}, id.RunID, 10, 0)
					require.NoError(t, err)
					require.Len(t, jobs, 1)
				}
				remaining, err := e.batcher.RetrieveItems(t.Context(), req.Function.ID, batchID)
				require.NoError(t, err)
				require.Empty(t, remaining)
			})
		}
	}
}

func TestScheduleAccountExecutionCapDebounce(t *testing.T) {
	for _, tt := range []struct {
		name          string
		invokeFirst   bool
		secondIsOlder bool
		wantInvoke    bool
	}{
		{name: "invoke replaces ordinary", wantInvoke: true},
		{name: "ordinary replaces invoke", invokeFirst: true},
		{name: "older ordinary cannot replace invoke", invokeFirst: true, secondIsOlder: true, wantInvoke: true},
		{name: "older invoke cannot replace ordinary", secondIsOlder: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e, req := newExecutionCapScheduler(t)
			req.Function.Debounce = &inngest.Debounce{Period: "10s"}
			inputs := []event.TrackedEvent{req.Events[0], executionCapInvoke(req)}
			if tt.invokeFirst {
				inputs[0], inputs[1] = inputs[1], inputs[0]
			}
			now := time.Now()
			for i, input := range inputs {
				evt := input.GetEvent()
				evt.Timestamp = now.Add(time.Duration(i) * time.Second).UnixMilli()
				if i == 1 && tt.secondIsOlder {
					evt.Timestamp = now.Add(-time.Second).UnixMilli()
				}
				req.Events = []event.TrackedEvent{event.NewBaseTrackedEventWithID(evt, input.GetInternalID())}
				_, _, err := e.Schedule(t.Context(), req)
				require.ErrorIs(t, err, ErrFunctionDebounced)
			}
			scope := queue.Scope{AccountID: req.AccountID, EnvID: req.WorkspaceID, FunctionID: req.Function.ID}
			info, err := e.debouncer.GetDebounceInfo(t.Context(), scope, "")
			require.NoError(t, err)
			require.NotNil(t, info.Item)
			selected := 1
			if tt.secondIsOlder {
				selected = 0
			}
			require.Equal(t, inputs[selected].GetInternalID(), info.Item.EventID)
			debounceID := ulid.MustParse(info.DebounceID)
			require.NoError(t, e.debouncer.StartExecution(t.Context(), *info.Item, req.Function, debounceID))
			req.Events = []event.TrackedEvent{info.Item}
			req.PreventDebounce, req.PreventRateLimit = true, true
			_, md, err := e.Schedule(t.Context(), req)
			if !tt.wantInvoke {
				var skipped SkippedError
				require.ErrorAs(t, err, &skipped)
				require.Equal(t, enums.SkipReasonAccountExecutionCapHit, skipped.Reason)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, md)
			jobs, err := e.queue.RunJobs(t.Context(), "local", scope, md.ID.RunID, 10, 0)
			require.NoError(t, err)
			require.Len(t, jobs, 1)
		})
	}
}

func newExecutionCapScheduler(t *testing.T, batchOpts ...batch.RedisBatchManagerOpt) (*executor, execution.ScheduleRequest) {
	t.Helper()
	ctx := t.Context()
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{r.Addr()}, DisableCache: true})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	uc := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)
	sc := redis_state.NewShardedClient(redis_state.ShardedClientOpts{
		UnshardedClient: uc, FunctionRunStateClient: rc, BatchClient: rc,
		StateDefaultKey: redis_state.StateDefaultKey, QueueDefaultKey: redis_state.QueueDefaultKey,
		FnRunIsSharded: redis_state.AlwaysShardOnRun,
	})
	mgr, err := redis_state.New(ctx, redis_state.WithShardedClient(sc), redis_state.WithPauseDeleter(redis_state.NewPauseStore(uc)))
	require.NoError(t, err)
	shard := redis_state.NewQueueShard("local", redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey))
	registry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	q, err := queue.New(ctx, "execution-cap", registry)
	require.NoError(t, err)
	d, err := debounce.NewDebouncer(registry, "local", q)
	require.NoError(t, err)
	bm := batch.NewRedisBatchManager(redis_state.NewBatchClient(rc, redis_state.QueueDefaultKey), q, append([]batch.RedisBatchManagerOpt{batch.WithoutBuffer()}, batchOpts...)...)
	t.Cleanup(func() { require.NoError(t, bm.Close()) })
	req := capScheduleRequest(nil)
	req.Function.Slug = "execution-cap"
	return &executor{
		log: logger.VoidLogger(), smv2: redis_state.MustRunServiceV2(mgr), queue: q, shards: registry,
		batcher: bm, debouncer: d,
		tracerProvider: tracing.NewNoopTracerProvider(), conditionalTracer: telemetrytrace.NoopConditionalTracer(),
		accountExecutionCap: capDecision(ExecutionCapLimitDecision{Exceeded: true, Enforce: true}),
	}, req
}

func executionCapInvoke(req execution.ScheduleRequest) event.BaseTrackedEvent {
	correlationID := ulid.Make().String() + ".call-child"
	return event.NewInvocationEvent(event.NewInvocationEventOpts{
		AccountID: req.AccountID, EnvID: req.WorkspaceID, FnID: req.Function.ID.String(),
		SourceAppID: req.AppID.String(), SourceFnID: uuid.New().String(), SourceFnVersion: 1,
		CorrelationID: &correlationID, ExpiresAt: time.Now().Add(-time.Hour).UnixMilli(),
		Event: event.Event{Data: map[string]any{"input": "child"}},
	})
}
