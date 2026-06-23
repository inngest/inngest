package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/connect/rest"
	"github.com/inngest/inngest/pkg/consts"
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestWorkerConcurrency tests that connect worker concurrency limits the number of
// concurrent steps across an app. Each step acquires and releases a slot from the
// app semaphore (auto-release). With MaxWorkerConcurrency=1, only 1 step should
// execute at a time across all functions in the app.
func TestWorkerConcurrency(t *testing.T) {
	os.Setenv("INNGEST_EVENT_KEY", "abc123")
	os.Setenv("INNGEST_SIGNING_KEY", "signkey-test-12345678")
	os.Setenv("INNGEST_SIGNING_KEY_FALLBACK", "signkey-test-00000000")

	ctx := context.Background()
	c := client.New(t)
	c.ResetAll(t)

	inngestClient := NewSDKConnectHandler(t, "worker-concurrency")

	var (
		inProgress, total int32
		numEvents         = 3
		stepDuration      = 2
	)

	trigger := "test/worker-concurrency"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      "worker-concurrency-test",
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running worker concurrency test", *input.Event.ID)

			next := atomic.AddInt32(&inProgress, 1)
			// With worker concurrency=1, only 1 step should be active at a time
			require.LessOrEqual(t, next, int32(1), "worker concurrency violated: more than 1 step active")

			<-time.After(time.Duration(stepDuration) * time.Second)

			atomic.AddInt32(&inProgress, -1)
			atomic.AddInt32(&total, 1)
			return "done", nil
		},
	)
	require.NoError(t, err)

	connectCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Connect with MaxWorkerConcurrency=1
	maxConcurrency := int64(1)
	_, err = inngestgo.Connect(connectCtx, inngestgo.ConnectOpts{
		InstanceID:           inngestgo.StrPtr("worker-concurrency-test"),
		Apps:                 []inngestgo.Client{inngestClient},
		MaxWorkerConcurrency: &maxConcurrency,
	})
	require.NoError(t, err)

	// Wait for the worker to be connected and synced
	var appID string
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		a := assert.New(collect)
		resp, err := http.Get(fmt.Sprintf("%s/v0/connect/envs/dev/conns", DEV_URL))
		a.NoError(err)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		var reply rest.ShowConnsReply
		err = json.NewDecoder(resp.Body).Decode(&reply)
		a.NoError(err)
		a.GreaterOrEqual(len(reply.Data), 1, "worker should be connected")
		if len(reply.Data) > 0 {
			// Verify at least one worker group is synced
			a.GreaterOrEqual(len(reply.Data[0].SyncedWorkerGroups), 1, "worker should be synced")
			for id := range reply.Data[0].SyncedWorkerGroups {
				appID = id
				break
			}
		}
	}, 10*time.Second, 500*time.Millisecond)
	require.NotEmpty(t, appID, "worker connection should expose synced app ID")

	// Give time for semaphore capacity to propagate and gateway routing to stabilize
	debugClient, debugConn := newDebugClient(t)
	defer debugConn.Close()

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		a := assert.New(collect)
		level, err := debugClient.GetAppSemaphoreLevel(ctx, &dbgpb.AppSemaphoreLevelRequest{
			AccountId: consts.DevServerAccountID.String(),
			AppId:     appID,
		})
		a.NoError(err)
		if err != nil || level.GetLevel() == nil {
			return
		}
		a.Equal(maxConcurrency, level.GetLevel().GetCapacity())
		a.Equal(int64(0), level.GetLevel().GetUsage())
		a.Equal(maxConcurrency, level.GetLevel().GetRemaining())
	}, 10*time.Second, 500*time.Millisecond)

	// Give time for semaphore capacity to propagate and gateway routing to stabilize
	<-time.After(10 * time.Second)

	// Send multiple events
	for i := 0; i < numEvents; i++ {
		_, err := inngestClient.Send(context.Background(), inngestgo.Event{
			Name: trigger,
			Data: map[string]any{"test": true},
		})
		require.NoError(t, err)
		<-time.After(50 * time.Millisecond)
	}

	// Eventually the first fn starts
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgress) == 1
	}, 60*time.Second, 100*time.Millisecond, "function should start")
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		a := assert.New(collect)
		level, err := debugClient.GetAppSemaphoreLevel(ctx, &dbgpb.AppSemaphoreLevelRequest{
			AccountId: consts.DevServerAccountID.String(),
			AppId:     appID,
		})
		a.NoError(err)
		if err != nil || level.GetLevel() == nil {
			return
		}
		a.Equal(maxConcurrency, level.GetLevel().GetCapacity())
		a.Equal(int64(1), level.GetLevel().GetUsage())
		a.Equal(int64(0), level.GetLevel().GetRemaining())
	}, 10*time.Second, 100*time.Millisecond)

	// During execution, never exceed limit
	totalDuration := time.Duration(numEvents*stepDuration+5) * time.Second
	deadline := time.Now().Add(totalDuration)
	for time.Now().Before(deadline) {
		<-time.After(200 * time.Millisecond)
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(1),
			"worker concurrency violated: more than 1 step active")
	}

	// All runs should eventually complete
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&total) == int32(numEvents)
	}, 5*time.Second, 100*time.Millisecond, "all runs should complete")
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		a := assert.New(collect)
		level, err := debugClient.GetAppSemaphoreLevel(ctx, &dbgpb.AppSemaphoreLevelRequest{
			AccountId: consts.DevServerAccountID.String(),
			AppId:     appID,
		})
		a.NoError(err)
		if err != nil || level.GetLevel() == nil {
			return
		}
		a.Equal(maxConcurrency, level.GetLevel().GetCapacity())
		a.Equal(int64(0), level.GetLevel().GetUsage())
		a.Equal(maxConcurrency, level.GetLevel().GetRemaining())
	}, 10*time.Second, 100*time.Millisecond)
}

func newDebugClient(t *testing.T) (dbgpb.DebugClient, *grpc.ClientConn) {
	t.Helper()

	addr := os.Getenv("DEBUG_API_ADDR")
	if addr == "" {
		addr = "127.0.0.1:7777"
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return dbgpb.NewDebugClient(conn), conn
}
