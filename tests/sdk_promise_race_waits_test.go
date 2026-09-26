package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestPromiseRaceWaits(t *testing.T) {
	registerDirect(t)
	defer deregisterDirect(t)

	cli := client.New(t)
	ctx := context.Background()
	start := time.Now().Add(-2 * time.Second)

	id := uuid.New().String()
	eventURLStr := eventURL.String()

	ic, err := inngestgo.NewClient(inngestgo.ClientOpts{
		AppID:    "test",
		EventKey: &eventKey,
		EventURL: &eventURLStr,
	})
	require.NoError(t, err)

	_, err = ic.Send(ctx, inngestgo.Event{
		Name: "tests/promise-race-waits.test",
		Data: map[string]any{"id": id},
	})
	require.NoError(t, err)

	// Fire the answer event periodically until the run completes.  Each JS
	// waitForEvent uses a 5m timeout, so if Promise.race over parallel waits
	// regresses, the two unanswered calls won't expire within this window —
	// the run stays RUNNING and the test fails here instead of silently
	// passing minutes later.
	require.Eventually(t, func() bool {
		_, sendErr := ic.Send(ctx, inngestgo.Event{
			Name: "tests/promise-race-waits.answer",
			Data: map[string]any{"id": id},
		})
		require.NoError(t, sendErr)

		fnID, ok := lookupFunctionID(ctx, cli, "promise-race-waits")
		if !ok {
			return false
		}
		edges, _, _ := cli.FunctionRuns(ctx, client.FunctionRunOpt{
			Items:       1,
			Status:      []string{"COMPLETED"},
			Start:       start,
			End:         time.Now().Add(time.Minute),
			FunctionIDs: []uuid.UUID{fnID},
		})
		return len(edges) > 0
	}, 60*time.Second, 2*time.Second, "Promise.race over parallel step.waitForEvent regressed — run never completed")

	fnID, _ := lookupFunctionID(ctx, cli, "promise-race-waits")
	edges, _, _ := cli.FunctionRuns(ctx, client.FunctionRunOpt{
		Items:       1,
		Status:      []string{"COMPLETED"},
		Start:       start,
		End:         time.Now().Add(time.Minute),
		FunctionIDs: []uuid.UUID{fnID},
	})
	require.NotEmpty(t, edges)
	run := cli.Run(ctx, edges[0].Node.ID)

	var out struct {
		Winner string `json:"winner"`
	}
	require.NoError(t, json.Unmarshal([]byte(run.Output), &out), "run output: %s", run.Output)
	require.Equal(t, "tests/promise-race-waits.answer", out.Winner, "the fired event must be the winner; if other legs won, the race semantics regressed")
}
