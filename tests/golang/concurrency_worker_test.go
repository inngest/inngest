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
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		inProgress, total, ready int32
		numEvents                = 3
		stepDuration             = 2
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
			if data, ok := input.Event.Data.(map[string]any); ok {
				if probe, _ := data["probe"].(bool); probe {
					atomic.StoreInt32(&ready, 1)
					return "ready", nil
				}
			}

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

	appName := inngestClient.AppID()
	var workerGroupID string

	// Wait for this app's worker group to be connected.
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		a := assert.New(collect)
		resp, err := http.Get(fmt.Sprintf("%s/v0/connect/envs/dev/conns", DEV_URL))
		if !a.NoError(err) {
			return
		}
		defer resp.Body.Close()

		var reply rest.ShowConnsReply
		err = json.NewDecoder(resp.Body).Decode(&reply)
		if !a.NoError(err) {
			return
		}

		for _, conn := range reply.Data {
			if groupID, ok := conn.AllWorkerGroups[appName]; ok {
				workerGroupID = groupID
				return
			}
		}
		a.Failf("worker group not connected", "no worker group found for app %q", appName)
	}, 10*time.Second, 500*time.Millisecond)

	// Wait for the worker group to be fully synced before enqueueing work.
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		a := assert.New(collect)
		resp, err := http.Get(fmt.Sprintf("%s/v0/connect/envs/dev/groups/%s", DEV_URL, workerGroupID))
		if !a.NoError(err) {
			return
		}
		defer resp.Body.Close()

		var reply rest.ShowWorkerGroupReply
		if !a.NoError(json.NewDecoder(resp.Body).Decode(&reply)) {
			return
		}
		if !a.NotNil(reply.Data) {
			return
		}

		a.True(reply.Data.Synced, "worker group should be synced")
	}, 10*time.Second, 500*time.Millisecond)

	// A synced worker group can still race with gateway routing metadata becoming
	// visible. Prove the route can execute before sending the workload under test.
	require.Eventually(t, func() bool {
		if atomic.LoadInt32(&ready) == 1 {
			return true
		}

		_, err := inngestClient.Send(context.Background(), inngestgo.Event{
			Name: trigger,
			Data: map[string]any{"probe": true},
		})
		if err != nil {
			return false
		}

		return atomic.LoadInt32(&ready) == 1
	}, 30*time.Second, 500*time.Millisecond, "worker route should become ready")

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
}
