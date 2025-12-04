package golang

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
)

// TestFunctionStepLimit tests step limit is enforced and surfaces the correct error message
func TestFunctionStepLimit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("app"))
	defer server.Close()

	var (
		ok        int32
		lastRunId string
	)
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-step-limit",
		},
		inngestgo.EventTrigger("test/step.limit", nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			_, _ = step.Run(ctx, "step1", func(ctx context.Context) (any, error) {
				if atomic.LoadInt32(&ok) == 0 {
					lastRunId = input.InputCtx.RunID
				}
				atomic.AddInt32(&ok, 1)

				return nil, nil
			})

			_, _ = step.Run(ctx, "step2", func(ctx context.Context) (any, error) {
				if atomic.LoadInt32(&ok) == 0 {
					lastRunId = input.InputCtx.RunID
				}
				atomic.AddInt32(&ok, 1)

				return nil, nil
			})

			return nil, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	functions, err := c.Functions(ctx)
	require.NoError(t, err)

	var functionId string
	// use last function with matching name
	for _, f := range functions {
		if f.Name == "fn-step-limit" {
			functionId = f.ID
		}
	}

	setStepLimit := func(t *testing.T, limit int) {
		reqUrl, err := url.Parse(c.APIHost + "/fn/step-limit")
		require.NoError(t, err)

		fv := reqUrl.Query()
		fv.Add("functionId", functionId)
		fv.Add("limit", fmt.Sprintf("%d", limit))

		req, err := http.NewRequest(http.MethodPost, reqUrl.String()+"?"+fv.Encode(), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		resp.Body.Close()
	}

	removeStepLimit := func(t *testing.T) {
		reqUrl, err := url.Parse(c.APIHost + "/fn/step-limit")
		require.NoError(t, err)

		fv := reqUrl.Query()
		fv.Add("functionId", functionId)

		req, err := http.NewRequest(http.MethodDelete, reqUrl.String()+"?"+fv.Encode(), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		resp.Body.Close()
	}

	t.Run("should fail due to step limit reached", func(t *testing.T) {
		setStepLimit(t, 1)

		_, _ = inngestClient.Send(ctx, inngestgo.Event{Name: "test/step.limit", Data: map[string]any{"success": true}})

		<-time.After(3 * time.Second)

		removeStepLimit(t)

		run := c.Run(ctx, lastRunId)
		assert.Equal(t, "FAILED", run.Status)
		assert.Equal(t, "{\"error\":{\"error\":\"function has too many steps\",\"name\":\"InngestErrFunctionOverflowed\",\"message\":\"The function run exceeded the step limit of 1 steps.\"}}", run.Output)

		// TODO Fix this
		//runTraces := c.WaitForRunTraces(ctx, t, &lastRunId, client.WaitForRunTracesOptions{
		//	Status: models.FunctionStatusFailed,
		//})
		//output := c.RunSpanOutput(ctx, *runTraces.Trace.OutputID)
		//
		//require.NotNil(t, output.Data)
		//
		//require.Equal(t, "{\"error\":{\"error\":\"function has too many steps\",\"name\":\"InngestErrFunctionOverflowed\",\"message\":\"The function run exceeded the step limit of 1 steps.\"}}", output)
	})
}
