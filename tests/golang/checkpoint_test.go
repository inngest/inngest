package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/pkg/checkpoint"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

func TestFnCheckpoint(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	inngestClient, server, registerFuncs := NewSDKHandler(t, "checkpoint")
	defer server.Close()

	configs := []*checkpoint.Config{
		checkpoint.ConfigSafe,
		checkpoint.ConfigPerformant,
		{
			BatchSteps:    3,
			BatchInterval: time.Second,
		},
	}

	delays := []time.Duration{
		time.Millisecond,
		time.Second,
		2 * time.Second,
	}

	for _, cfg := range configs {
		// For each config, add a delay after the second and third step.  This is because a config
		// will always checkpoint after a second, and we want to assert that this happens.
		for _, delay := range delays {
			runID := ""
			evtName := fmt.Sprintf("invoke-checkpoint-delay-%v", delay.Milliseconds())

			_, err := inngestgo.CreateFunction(
				inngestClient,
				inngestgo.FunctionOpts{
					ID:         fmt.Sprintf("checkpoint-myfn-%v", cfg),
					Checkpoint: cfg,
				},
				inngestgo.EventTrigger(evtName, nil),
				func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
					_, _ = step.Run(ctx, "a", func(ctx context.Context) (string, error) { return "a", nil })
					_, _ = step.Run(ctx, "b", func(ctx context.Context) (string, error) {
						<-time.After(delay)
						return "b", nil
					})
					_, _ = step.Run(ctx, "c", func(ctx context.Context) (string, error) {
						<-time.After(delay)
						return "c", nil
					})
					runID = input.InputCtx.RunID
					return nil, nil
				},
			)
			r.NoError(err)
			registerFuncs()

			// Trigger the main function and successfully invoke the other function
			_, err = inngestClient.Send(ctx, &event.Event{Name: evtName})
			r.NoError(err)

			// Wait a moment for runID to be populated
			<-time.After(2 * time.Second)
			run := c.WaitForRunStatus(ctx, t, "COMPLETED", &runID)
			var output string
			err = json.Unmarshal([]byte(run.Output), &output)
			require.NotEmpty(t, runID)
			r.NoError(err)
		}
	}
}
