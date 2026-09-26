package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

// TestParallelFanIn is a regression test for parallel fan-in runs hanging.
// It drives the JS testParallelFanIn function (40 concurrent ops in one
// Promise.all: 20 step.run + 20 step.invoke of sleepRandom) and asserts the
// parent run completes with the expected aggregate.  If discovery dedup ever
// decouples from pending-step tracking again, the run stalls and this test
// fails on the wait-for-status timeout.  If a step runs the wrong number of
// times the aggregate assertions catch it.
func TestParallelFanIn(t *testing.T) {
	registerDirect(t)
	defer deregisterDirect(t)

	cli := client.New(t)
	ctx := context.Background()
	start := time.Now().Add(-2 * time.Second)

	eventURLStr := eventURL.String()
	ic, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:    "test",
		EventKey: &eventKey,
		EventURL: &eventURLStr,
	})
	require.NoError(t, err)
	_, err = ic.Send(ctx, inngestgo.Event{
		Name: "tests/parallel-fan-in.test",
		Data: map[string]any{},
	})
	require.NoError(t, err)

	runID := waitForRecentRun(t, cli, ctx, "parallel-fan-in", start, "COMPLETED", 60*time.Second)
	run := cli.Run(ctx, runID)
	require.Equal(t, "COMPLETED", run.Status)

	var out struct {
		StepSquares int `json:"stepSquares"`
		InvokeCount int `json:"invokeCount"`
	}
	require.NoError(t, json.Unmarshal([]byte(run.Output), &out), "run output: %s", run.Output)

	// 0² + 1² + ... + 19² = 2470
	require.Equal(t, 2470, out.StepSquares, "sum of step.run squares — wrong value means a step ran the wrong number of times or returned the wrong value")
	require.Equal(t, 20, out.InvokeCount, "every step.invoke must return a {slept: number} response")
}
