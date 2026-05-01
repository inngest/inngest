// Asserts that requeuing an in-flight async dispatch aborts the prior SDK
// process before it executes the next step, so an HTTP timeout no longer
// produces duplicate step executions (EXE-1552).

package golang

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/inngest/inngest/pkg/config"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/devserver"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mr := miniredis.RunT(t)
	startDevServer(t, ctx, "redis://"+mr.Addr())

	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{mr.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	defer rc.Close()
	shard := redis_state.NewQueueShard(
		consts.DefaultQueueShardName,
		redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey),
	)

	// ItemsByRunID is on *queue but not on the public RedisQueueShard interface,
	// so we type-assert via an inline interface.
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

	// Wait for the original SDK invocation to enter step1.
	select {
	case <-step1.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for original SDK to enter step1")
	}

	// Find the leased queue item and force a requeue, simulating the executor's
	// behavior on HTTP timeout (pkg/execution/queue/process.go:404-432).
	leased := waitForLeasedItem(t, ctx, finder, runID)
	require.NoError(t, shard.Requeue(ctx, *leased, time.Now()))

	// Wait for the second SDK invocation triggered by the requeue.
	select {
	case <-step1.entered:
	case <-time.After(15 * time.Second):
		t.Fatalf("only %d SDK invocation(s) entered step1 after requeue; executor did not re-dispatch",
			atomic.LoadInt32(&step1.count))
	}

	step1.unblock()

	// Wait for at least one step2 entry, then briefly for a possible second.
	select {
	case <-step2.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for any step2 entry")
	}
	select {
	case <-step2.entered:
	case <-time.After(2 * time.Second):
	}
	step2.unblock()

	// Load-bearing assertion: the in-flight step (step1) is unavoidably
	// duplicated, but step2 must run only once. Two step2 executions means the
	// original SDK process was not aborted before reaching it — which is what
	// DispatchID validation will fix (its checkpoint POST will get 409).
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

func startDevServer(t *testing.T, ctx context.Context, redisURI string) {
	t.Helper()
	conf, err := config.Dev(ctx)
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		errCh <- devserver.New(ctx, devserver.StartOpts{
			Config:             *conf,
			QueueWorkers:       devserver.DefaultQueueWorkers,
			Tick:               devserver.DefaultTickDuration,
			PollInterval:       devserver.DefaultPollInterval,
			ConnectGatewayPort: devserver.DefaultConnectGatewayPort,
			ConnectGatewayHost: conf.CoreAPI.Addr,
			NoUI:               true,
			RedisURI:           redisURI,
		})
	}()
	t.Cleanup(func() {
		select {
		case err := <-errCh:
			if err != nil && ctx.Err() == nil {
				t.Errorf("dev server exited with error: %v", err)
			}
		case <-time.After(2 * time.Second):
		}
	})

	require.NoError(t, waitForDevPort(ctx, "127.0.0.1:8288", 15*time.Second))
	time.Sleep(200 * time.Millisecond)
	_ = os.Setenv("INNGEST_DEV", "http://127.0.0.1:8288")
}

func waitForDevPort(ctx context.Context, addr string, timeout time.Duration) error {
	deadline, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		select {
		case <-deadline.Done():
			return fmt.Errorf("timed out waiting for %s", addr)
		default:
			if conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond); err == nil {
				_ = conn.Close()
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}
