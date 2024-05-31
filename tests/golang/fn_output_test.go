package golang

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/syscode"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestFnOutputTooLarge(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)
	c := client.New(t)

	appID := "TestFnOutputTooLarge"
	h, server, registerFuncs := NewSDKHandler(t, appID)
	defer server.Close()

	runID := ""
	evtName := "my-event"
	fn := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name:    "my-fn",
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger(evtName, nil),
		func(ctx context.Context, input inngestgo.Input[DebounceEvent]) (any, error) {
			runID = input.InputCtx.RunID
			return strings.Repeat("A", consts.MaxBodySize+1), nil
		},
	)

	h.Register(fn)
	registerFuncs()

	// Trigger the main function and successfully invoke the other function
	_, err := inngestgo.Send(ctx, &event.Event{Name: evtName})
	r.NoError(err)
	run := c.WaitForRunStatus(ctx, t, "FAILED", &runID)
	var output string
	err = json.Unmarshal([]byte(run.Output), &output)
	r.NoError(err)
	r.Equal(syscode.CodeOutputTooLarge, output)
}
