// EXE-1552: covers the late-arriving stale-dispatch case for the duplicate
// step-execution bug — a Requeue that fires while a long-running step body
// is still executing on the prior dispatch, then the old SDK posts its
// checkpoint after the new dispatch is already in flight. The fence only
// engages once the old dispatch's RequestStartedAt is older than
// freshDispatchWindow (see pkg/execution/checkpoint/checkpoint.go); faster
// Requeue races are an accepted perf trade-off.
//
// Requires the pre-running dev server (see Makefile `test-integration`) to be
// started with `--redis-uri <addr>` AND `EXPERIMENTAL_ASYNC_DISPATCH_VALIDATION=true`,
// and the same Redis to be reachable from this test process via the REDIS_URI
// env var. Without REDIS_URI we fail loudly; without the validator gate the
// step2 dedup assertion will fire.

package golang

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/pkg/checkpoint"
	"github.com/inngest/inngestgo/step"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/require"
)

// stepLatch counts entries into a step body and blocks them until released.
type stepLatch struct {
	count   int32
	entered chan struct{}
	release chan struct{}
}

func newStepLatch() *stepLatch {
	return &stepLatch{entered: make(chan struct{}, 8), release: make(chan struct{})}
}

func (s *stepLatch) enter() {
	atomic.AddInt32(&s.count, 1)
	select {
	case s.entered <- struct{}{}:
	default:
	}
	<-s.release
}

func (s *stepLatch) unblock() { close(s.release) }

func TestEXE1552DuplicateStepExecutionOnRequeue(t *testing.T) {
	ctx := context.Background()

	redisURI := os.Getenv("REDIS_URI")
	require.NotEmpty(t, redisURI,
		"REDIS_URI must be set and point to the same redis the dev server is using "+
			"(start dev with --redis-uri <addr> and export REDIS_URI=<same addr>)",
	)
	opt, err := rueidis.ParseURL(redisURI)
	require.NoError(t, err, "REDIS_URI must be a valid redis URL")
	opt.DisableCache = true
	rc, err := rueidis.NewClient(opt)
	require.NoError(t, err)
	defer rc.Close()

	shard := redis_state.NewQueueShard(
		consts.DefaultQueueShardName,
		redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey),
	)
	finder, ok := shard.(interface {
		ItemsByRunID(ctx context.Context, runID ulid.ULID) ([]*osqueue.QueueItem, error)
	})
	require.True(t, ok, "shard does not implement ItemsByRunID")

	inngestClient, server, registerFuncs := NewSDKHandler(t, "exe1552")
	defer server.Close()

	step1 := newStepLatch()
	step2 := newStepLatch()
	rid := NewRunID()
	const trigger = "test/exe1552"

	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "exe1552-fn", Checkpoint: checkpoint.ConfigSafe},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			rid.Send(input.InputCtx.RunID)
			_, _ = step.Run(ctx, "step1", func(_ context.Context) (string, error) {
				step1.enter()
				return "step1-done", nil
			})
			_, _ = step.Run(ctx, "step2", func(_ context.Context) (string, error) {
				step2.enter()
				return "step2-done", nil
			})
			return "ok", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	_, err = inngestClient.Send(ctx, &event.Event{Name: trigger, Data: map[string]any{}})
	require.NoError(t, err)

	runID, err := ulid.Parse(rid.Wait(t))
	require.NoError(t, err)

	select {
	case <-step1.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for original SDK to enter step1")
	}

	// Force a requeue, mimicking the executor's HTTP-timeout path.
	leased := waitForLeasedItem(t, ctx, finder, runID)
	require.NoError(t, shard.Requeue(ctx, *leased, time.Now()))

	select {
	case <-step1.entered:
	case <-time.After(15 * time.Second):
		t.Fatalf("only %d SDK invocation(s) entered step1 after requeue; executor did not re-dispatch",
			atomic.LoadInt32(&step1.count))
	}

	// The validator's fast-path skips the queue-item load when
	// time.Since(RequestStartedAt) < freshDispatchWindow (10s — see
	// pkg/execution/checkpoint/checkpoint.go:freshDispatchWindow). To exercise
	// the entropy-comparison path that the fence relies on, wait past that
	// window (plus a small clock-skew buffer) before unblocking the OLD SDK so
	// its checkpoint POST is provably stale.
	time.Sleep(11 * time.Second)

	step1.unblock()

	select {
	case <-step2.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for any step2 entry")
	}
	// Brief grace window to catch a duplicate step2 entry if the fence didn't fire.
	select {
	case <-step2.entered:
	case <-time.After(2 * time.Second):
	}
	step2.unblock()

	// step1 is unavoidably duplicated; step2 must not be.
	require.Equalf(t, int32(1), atomic.LoadInt32(&step2.count),
		"step2 ran %d times; expected 1 (step1 ran %d)",
		atomic.LoadInt32(&step2.count), atomic.LoadInt32(&step1.count))
}

func waitForLeasedItem(
	t *testing.T,
	ctx context.Context,
	finder interface {
		ItemsByRunID(ctx context.Context, runID ulid.ULID) ([]*osqueue.QueueItem, error)
	},
	runID ulid.ULID,
) *osqueue.QueueItem {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		items, err := finder.ItemsByRunID(ctx, runID)
		require.NoError(t, err)
		for _, it := range items {
			if it != nil && it.LeaseID != nil {
				return it
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("did not find leased queue item for run within timeout")
	return nil
}
