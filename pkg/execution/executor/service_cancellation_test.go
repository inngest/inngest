package executor

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/cancellation"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

type bulkCancellationRecorder struct {
	execution.Executor
	ids           []ulid.ULID
	cancellations []ulid.ULID
	failRun       ulid.ULID
}

func (e *bulkCancellationRecorder) Cancel(_ context.Context, id sv2.ID, r execution.CancelRequest) error {
	if id.RunID == e.failRun {
		return errors.New("cancel failed")
	}
	e.ids = append(e.ids, id.RunID)
	e.cancellations = append(e.cancellations, *r.CancellationID)
	return nil
}

func TestEagerBulkCancellationQueuedTime(t *testing.T) {
	ctx := logger.WithStdlib(context.Background(), logger.VoidLogger())
	from := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	until := from.Add(30 * time.Second)
	for _, tc := range []struct {
		name         string
		runAt, score time.Time
		lower        *time.Time
		eager, lazy  bool
	}{
		{"older_run_sleep_wakes_inside_range", from.Add(-time.Hour), from.Add(time.Second), &from, false, false},
		{"queued_start_at_from", from, from, &from, true, true},
		{"queued_start_at_until", until, until, &from, true, true},
		{"sleeping_item_after_until", from.Add(time.Second), until.Add(time.Hour), &from, false, true},
		{"no_lower_bound_older_run", from.Add(-time.Hour), from.Add(time.Second), nil, true, true},
		{"no_lower_bound_at_until", until, until, nil, true, true},
		{"no_lower_bound_after_until", until.Add(time.Millisecond), until.Add(time.Millisecond), nil, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			rc, err := rueidis.NewClient(rueidis.ClientOption{
				InitAddress:  []string{mr.Addr()},
				DisableCache: true,
			})
			require.NoError(t, err)
			t.Cleanup(rc.Close)
			shard := redis_state.NewQueueShard("local", redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), queue.WithPartitionPausedGetter(func(context.Context, uuid.UUID) queue.PartitionPausedInfo {
				return queue.PartitionPausedInfo{Paused: true}
			}))
			registry, err := queue.NewSingleShardRegistry(shard)
			require.NoError(t, err)
			account, env, fn := uuid.New(), uuid.New(), uuid.New()
			run := ulid.MustNew(ulid.Timestamp(tc.runAt), rand.Reader)
			producer := queue.NewProducer(registry, queue.WithProducerClock(clockwork.NewFakeClockAt(tc.runAt)))
			jobID := run.String()
			kind := queue.KindStart
			if tc.score.After(tc.runAt) {
				kind = queue.KindSleep
			}
			err = producer.Enqueue(ctx, queue.Item{
				JobID:       &jobID,
				Kind:        kind,
				WorkspaceID: env,
				Identifier: state.Identifier{
					AccountID:   account,
					WorkspaceID: env,
					WorkflowID:  fn,
					RunID:       run,
				},
			}, tc.score, queue.EnqueueOpts{})
			require.NoError(t, err)
			c := cqrs.Cancellation{
				ID:            ulid.Make(),
				AccountID:     account,
				WorkspaceID:   env,
				FunctionID:    fn,
				StartedAfter:  tc.lower,
				StartedBefore: until,
			}
			require.NoError(t, cancellation.NewRedisWriter(rc, producer, "").CreateCancellation(ctx, c))
			recorder := &bulkCancellationRecorder{}
			s := &svc{
				shards: registry,
				exec:   recorder,
				log:    logger.VoidLogger(),
			}
			require.NoError(t, s.handleEagerCancelBulkRun(ctx, c))
			if tc.eager {
				require.Equal(t, []ulid.ULID{run}, recorder.ids)
				require.Equal(t, []ulid.ULID{c.ID}, recorder.cancellations)
			} else {
				require.Empty(t, recorder.ids)
			}
			matched, err := cancellation.NewChecker(cancellation.NewRedisReader(rc, "")).IsCancelled(ctx, env, fn, run, nil)
			require.NoError(t, err)
			if tc.lazy {
				require.NotNil(t, matched)
				require.Equal(t, c.ID, matched.ID)
			} else {
				require.Nil(t, matched)
			}
		})
	}
}

func newBulkCancellationQueue(t *testing.T) (rueidis.Client, queue.ShardRegistryController) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	t.Cleanup(rc.Close)
	shard := redis_state.NewQueueShard("local", redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey))
	registry, err := queue.NewSingleShardRegistry(shard)
	require.NoError(t, err)
	return rc, registry
}

func TestEagerBulkCancellationNoPartition(t *testing.T) {
	_, registry := newBulkCancellationQueue(t)
	s := &svc{
		shards: registry,
		log:    logger.VoidLogger(),
	}
	require.NoError(t, s.handleEagerCancelBulkRun(t.Context(), cqrs.Cancellation{
		AccountID:     uuid.New(),
		WorkspaceID:   uuid.New(),
		FunctionID:    uuid.New(),
		StartedBefore: time.Now(),
	}))
}

type bulkCancellationState struct {
	state.Manager
	failRun ulid.ULID
}

func (s bulkCancellationState) Load(_ context.Context, _ uuid.UUID, runID ulid.ULID) (state.State, error) {
	if runID == s.failRun {
		return nil, errors.New("load failed")
	}
	return state.NewStateInstance(state.Identifier{RunID: runID}, state.Metadata{}, []map[string]any{{"data": map[string]any{"cancel": true}}}, nil, nil), nil
}

func TestEagerBulkCancellationContinuesAfterItemError(t *testing.T) {
	for _, failure := range []string{"load", "cancel"} {
		t.Run(failure, func(t *testing.T) {
			_, registry := newBulkCancellationQueue(t)
			producer := queue.NewProducer(registry)
			from := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
			c := cqrs.Cancellation{
				ID:            ulid.Make(),
				AccountID:     uuid.New(),
				WorkspaceID:   uuid.New(),
				FunctionID:    uuid.New(),
				StartedAfter:  &from,
				StartedBefore: from.Add(time.Minute),
				If:            new("event.data.cancel == true"),
			}
			runs := []ulid.ULID{
				ulid.MustNew(ulid.Timestamp(from), rand.Reader),
				ulid.MustNew(ulid.Timestamp(from.Add(time.Second)), rand.Reader),
			}
			for i, runID := range runs {
				require.NoError(t, producer.Enqueue(t.Context(), queue.Item{
					JobID:       new(runID.String()),
					Kind:        queue.KindStart,
					WorkspaceID: c.WorkspaceID,
					Identifier: state.Identifier{
						AccountID:   c.AccountID,
						WorkspaceID: c.WorkspaceID,
						WorkflowID:  c.FunctionID,
						RunID:       runID,
					},
				}, from.Add(time.Duration(i)*time.Second), queue.EnqueueOpts{}))
			}
			recorder := &bulkCancellationRecorder{}
			loader := bulkCancellationState{}
			if failure == "load" {
				loader.failRun = runs[0]
			} else {
				recorder.failRun = runs[0]
			}
			s := &svc{
				shards: registry,
				state:  loader,
				exec:   recorder,
				log:    logger.VoidLogger(),
			}
			require.NoError(t, s.handleEagerCancelBulkRun(t.Context(), c))
			require.Equal(t, []ulid.ULID{runs[1]}, recorder.ids)
			require.Equal(t, []ulid.ULID{c.ID}, recorder.cancellations)
		})
	}
}

func TestCheckCancellationWithoutEvents(t *testing.T) {
	e := &executor{cancellationChecker: cancellation.NewChecker(nil)}
	for _, events := range [][]json.RawMessage{nil, {}} {
		cancelled, err := e.checkCancellation(t.Context(), sv2.Metadata{}, events)
		require.NoError(t, err)
		require.False(t, cancelled)
	}
}

func TestCheckCancellationStoreError(t *testing.T) {
	rc, _ := newBulkCancellationQueue(t)
	rc.Close()
	e := &executor{
		cancellationChecker: cancellation.NewChecker(cancellation.NewRedisReader(rc, "")),
		clock:               clockwork.NewRealClock(),
	}
	ctx := logger.WithStdlib(t.Context(), logger.VoidLogger())
	cancelled, err := e.checkCancellation(ctx, sv2.Metadata{}, []json.RawMessage{json.RawMessage(`{"name":"test"}`)})
	require.NoError(t, err)
	require.False(t, cancelled)
}
