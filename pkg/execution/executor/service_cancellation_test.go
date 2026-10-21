package executor

import (
	"context"
	"crypto/rand"
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
}

func (e *bulkCancellationRecorder) Cancel(_ context.Context, id sv2.ID, r execution.CancelRequest) error {
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
