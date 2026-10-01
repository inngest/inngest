package redis_state

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// Execute the real lease Lua, then withhold its reply beyond the hint budget.
// Cancellation can lose the reply, but cannot roll back the committed lease.
type delayedHintLeaseReply struct {
	rueidis.Client
	itemID   string
	injected atomic.Bool
	result   chan error
}

func (c *delayedHintLeaseReply) Do(ctx context.Context, cmd rueidis.Completed) rueidis.RedisResult {
	parts := cmd.Commands()
	lease := len(parts) == 14 && (parts[0] == "EVALSHA" || parts[0] == "EVAL") && parts[2] == "4" && parts[7] == c.itemID
	res := c.Client.Do(ctx, cmd)
	if !lease {
		return res
	}
	status, err := res.ToInt64()
	if err != nil || status != 0 || !c.injected.CompareAndSwap(false, true) {
		return res
	}
	timer := time.NewTimer(1100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		c.result <- ctx.Err()
		// The client returns the canceled context error without sending a ping.
		return c.Client.Do(ctx, c.B().Ping().Build())
	case <-timer.C:
		c.result <- nil
		return res
	}
}

func TestItemHintDispatchesDelayedCommittedLease(t *testing.T) {
	for _, backlog := range []bool{false, true} {
		t.Run(fmt.Sprintf("backlog=%t", backlog), func(t *testing.T) {
			raw, err := rueidis.NewClient(rueidis.ClientOption{InitAddress: []string{miniredis.RunT(t).Addr()}, DisableCache: true})
			require.NoError(t, err)
			t.Cleanup(raw.Close)
			client := &delayedHintLeaseReply{Client: raw, result: make(chan error, 1)}
			opts := []osqueue.QueueOpt{osqueue.WithRunMode(osqueue.QueueRunMode{Partition: true}), osqueue.WithPollTick(time.Millisecond), osqueue.WithNumWorkers(2),
				osqueue.WithAllowKeyQueues(func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) bool { return backlog })}
			base := NewQueueShard("ss3", NewQueueClient(client, "{hint-deadline}"), opts...)
			reg, err := osqueue.NewSingleShardRegistry(hintOnlyRedisShard{base})
			require.NoError(t, err)
			offers := make(chan func(osqueue.QueueItem) bool, 1)
			opts = append(opts, osqueue.WithItemHints(osqueue.ItemHintOptions{BufferSize: 2, AttemptTimeout: time.Second, Source: func(ctx context.Context, _ osqueue.QueueShard, offer func(osqueue.QueueItem) bool) error {
				offers <- offer
				<-ctx.Done()
				return nil
			}}))
			proc, err := osqueue.New(t.Context(), "hint-deadline", reg, opts...)
			require.NoError(t, err)
			acct, env, fn := uuid.New(), uuid.New(), uuid.New()
			item, err := base.EnqueueItem(t.Context(), osqueue.QueueItem{ID: "delayed", FunctionID: fn, WorkspaceID: env,
				Data: osqueue.Item{Kind: osqueue.KindStart, WorkspaceID: env, Identifier: state.Identifier{AccountID: acct, WorkspaceID: env, WorkflowID: fn, RunID: ulid.Make()}}}, time.Now(), osqueue.EnqueueOpts{})
			require.NoError(t, err)
			client.itemID = item.ID
			var executions atomic.Int32
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() {
				done <- proc.Run(ctx, func(context.Context, osqueue.RunInfo, osqueue.Item) (osqueue.RunResult, error) {
					executions.Add(1)
					return osqueue.RunResult{}, nil
				})
			}()
			defer func() {
				cancel()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("processor did not stop")
				}
			}()
			var offer func(osqueue.QueueItem) bool
			select {
			case offer = <-offers:
			case <-time.After(time.Second):
				t.Fatal("hint source did not start")
			}
			require.True(t, offer(item))
			select {
			case err = <-client.result:
				require.NoError(t, err, "the hint budget must not abandon a committed lease")
			case <-time.After(3 * time.Second):
				t.Fatal("lease fault was not reached")
			}
			require.Eventually(t, func() bool {
				_, err := base.LoadQueueItem(t.Context(), item.ID)
				return err == osqueue.ErrQueueItemNotFound && proc.Semaphore().Available() == 2
			}, time.Second, time.Millisecond, "complete without waiting for lease expiry or scavenging")
			require.EqualValues(t, 1, executions.Load())
		})
	}
}
