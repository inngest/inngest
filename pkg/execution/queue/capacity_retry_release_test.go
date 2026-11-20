package queue

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/util/errs"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// Real queue admission and ProcessItem cleanup must release both attempts within
// the capacity API's operation-idempotency window. The next job, rather than a
// mocked Release call, proves that no concurrency slot remains stranded.
func TestCapacityRetryReleasesEachLease(t *testing.T) {
	for _, earlyRelease := range []bool{false, true} {
		t.Run(fmt.Sprintf("early_release_%v", earlyRelease), func(t *testing.T) {
			ctx := context.Background()
			addr := os.Getenv("INNGEST_REPRO_REDIS_ADDR")
			if addr == "" {
				addr = miniredis.RunT(t).Addr()
			}
			client, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{addr}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(client.Close)
			clock := clockwork.NewFakeClockAt(time.Now())
			manager, err := constraintapi.NewRedisCapacityManager(constraintapi.WithClient(client), constraintapi.WithClock(clock), constraintapi.WithShardName("test"))
			require.NoError(t, err)
			observedManager := &capacityReleaseEvidence{CapacityManager: manager, t: t}
			events := make(chan string, 8)
			shard := &capacityReleaseNotifierShard{mockShardForIterator: &mockShardForIterator{name: "test"}, events: events}
			registry, err := NewSingleShardRegistry(shard)
			require.NoError(t, err)
			processor, err := New(ctx, "test", registry, WithCapacityManager(observedManager), WithClock(clock))
			require.NoError(t, err)
			account, env, fn := uuid.New(), uuid.New(), uuid.New()
			partition := &QueueShadowPartition{PartitionID: fn.String(), AccountID: &account, EnvID: &env, FunctionID: &fn}
			backlog := &QueueBacklog{BacklogID: "test", ShadowPartitionID: fn.String()}
			config := PartitionConstraintConfig{FunctionVersion: 1, Concurrency: PartitionConcurrency{AccountConcurrency: NoConcurrencyLimit, FunctionConcurrency: 1}}
			item := QueueItem{ID: "same-job", FunctionID: fn, Data: Item{Kind: KindEdge, Identifier: state.Identifier{AccountID: account, WorkspaceID: env, WorkflowID: fn, AppID: uuid.New(), RunID: ulid.Make()}}}
			acquire := func(item *QueueItem) *CapacityLease {
				result, err := processor.ItemLeaseConstraintCheck(ctx, partition, backlog, config, item, clock.Now())
				require.NoError(t, err)
				return result.CapacityLease
			}
			var previousLease ulid.ULID
			for attempt := 0; attempt < 2; attempt++ {
				item.Data.Attempt = attempt
				itemLease := ulid.Make()
				item.LeaseID = &itemLease
				capacity := acquire(&item)
				require.NotNil(t, capacity)
				require.NotEqual(t, previousLease, capacity.LeaseID)
				previousLease = capacity.LeaseID
				t.Logf("attempt=%d admitted lease=%s", attempt, capacity.LeaseID)
				contender := item
				contender.ID = fmt.Sprintf("blocked-%d", attempt)
				require.Nil(t, acquire(&contender), "the single slot must be occupied during execution")
				_, err := processor.ProcessItem(ctx, ProcessItem{I: item, CapacityLease: capacity}, func(_ context.Context, info RunInfo, _ Item) (RunResult, error) {
					if earlyRelease {
						require.NoError(t, info.CapacityLease.Release())
					}
					if attempt == 0 {
						return RunResult{}, context.DeadlineExceeded
					}
					return RunResult{}, nil
				})
				require.NoError(t, err)
				for receiveCapacityReleaseEvent(t, events) != "notify" {
				}
				// Keep Redis TTLs and the capacity-manager clock unchanged: release
				// idempotency must not hide a different lease in the next attempt.
			}
			item.ID = "next-job"
			next := acquire(&item)
			t.Logf("post_retry_admission=%t", next != nil)
			require.NotNil(t, next, "a completed fast retry must immediately free its concurrency slot")
		})
	}
}

// Observe real release responses without replacing capacity behavior.
type capacityReleaseEvidence struct {
	constraintapi.CapacityManager
	t *testing.T
}

func (m *capacityReleaseEvidence) Release(ctx context.Context, req *constraintapi.CapacityReleaseRequest) (*constraintapi.CapacityReleaseResponse, errs.InternalError) {
	response, err := m.CapacityManager.Release(ctx, req)
	if err != nil {
		m.t.Logf("release key=%s lease=%s error=%v", req.IdempotencyKey, req.LeaseID, err)
	} else if response != nil {
		m.t.Logf("release key=%s lease=%s operation_idempotency_hit=%t", req.IdempotencyKey, req.LeaseID, response.OperationIdempotencyHit)
	}
	return response, err
}
