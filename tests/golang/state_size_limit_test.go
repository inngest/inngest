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

// TestFunctionStateSizeLimit tests step limit is enforced and surfaces the correct error message
func TestFunctionStateSizeLimit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, randomSuffix("app"))
	defer server.Close()

	var runID atomic.Pointer[string]
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-state-size-limit",
		},
		inngestgo.EventTrigger("test/state-size.limit", nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			runID.Store(&input.InputCtx.RunID)

			_, _ = step.Run(ctx, "step1", func(ctx context.Context) (any, error) {
				return nil, nil
			})

			_, _ = step.Run(ctx, "step2", func(ctx context.Context) (any, error) {
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
		if f.Name == "fn-state-size-limit" {
			functionId = f.ID
		}
	}

	setStateSizeLimit := func(t *testing.T, limit int) {
		reqUrl, err := url.Parse(c.APIHost + "/fn/state-size-limit")
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

	removeStateSizeLimit := func(t *testing.T) {
		reqUrl, err := url.Parse(c.APIHost + "/fn/state-size-limit")
		require.NoError(t, err)

		fv := reqUrl.Query()
		fv.Add("functionId", functionId)

		req, err := http.NewRequest(http.MethodDelete, reqUrl.String()+"?"+fv.Encode(), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		resp.Body.Close()
	}

	t.Run("should fail due to state size limit reached", func(t *testing.T) {
		r := require.New(t)
		setStateSizeLimit(t, 1)

		_, _ = inngestClient.Send(ctx, inngestgo.Event{Name: "test/state-size.limit", Data: map[string]any{"success": true}})
		r.EventuallyWithT(func(t *assert.CollectT) {
			r := require.New(t)
			r.NotEmpty(runID.Load())
			run := c.Run(ctx, *runID.Load())
			r.Equal("FAILED", run.Status)
			r.Equal("{\"error\":{\"error\":\"InngestErrStateOverflowed: The function run exceeded the state size limit of 1 bytes.\",\"name\":\"InngestErrStateOverflowed\",\"message\":\"The function run exceeded the state size limit of 1 bytes.\"}}", run.Output)
		}, 10*time.Second, time.Second)

		removeStateSizeLimit(t)

	})
}
