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

// TestParallelFanIn is a regression test for parallel fan-in runs wedging.
// It drives the JS testParallelFanIn function (100 concurrent ops in one
// Promise.all: 50 step.run + 50 step.invoke of sleepRandom) and asserts the
// parent run completes with the expected aggregate.  If discovery dedup ever
// decouples from pending-step tracking again, the run hangs and this test
// fails on the wait-for-status timeout.  If a step runs the wrong number of
// times the aggregate assertions catch it.
func TestParallelFanIn(t *testing.T) {
	cli := client.New(t)
	ctx := context.Background()
	start := time.Now().Add(-2 * time.Second)

	evt := inngestgo.Event{
		Name: "tests/parallel-fan-in.test",
		Data: map[string]any{},
	}

	eventURLStr := eventURL.String()
	ic, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:    "test",
		EventKey: &eventKey,
		EventURL: &eventURLStr,
	})
	require.NoError(t, err)
	_, err = ic.Send(ctx, evt)
	require.NoError(t, err)

	// Generous timeout: invokes sleep up to 5s each, and there's queue
	// scheduling overhead for 100 concurrent ops.  If the wedge regresses
	// the run never completes and the test fails here.
	runID := waitForRecentRun(t, cli, ctx, "parallel-fan-in", start, "COMPLETED", 60*time.Second)
	run := cli.Run(ctx, runID)
	require.Equal(t, "COMPLETED", run.Status)

	var out struct {
		StepSquares int `json:"stepSquares"`
		InvokeCount int `json:"invokeCount"`
	}
	require.NoError(t, json.Unmarshal([]byte(run.Output), &out), "run output: %s", run.Output)

	// 0² + 1² + ... + 49² = 40425
	require.Equal(t, 40425, out.StepSquares, "sum of step.run squares — wrong value means a step ran the wrong number of times or returned the wrong value")
	require.Equal(t, 50, out.InvokeCount, "every step.invoke must return a {slept: number} response")
}
