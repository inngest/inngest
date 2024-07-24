package golang

import (
	"context"
	"fmt"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
)

// TestFunctionStateSizeLimit tests step limit is enforced and surfaces the correct error message
func TestFunctionStateSizeLimit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	c := client.New(t)
	h, server, registerFuncs := NewSDKHandler(t, "fnrun")
	defer server.Close()

	var (
		ok        int32
		lastRunId string
	)
	fn1 := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "fn-state-size-limit",
		},
		inngestgo.EventTrigger("test/state-size.limit", nil),
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

	h.Register(fn1)
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
		setStateSizeLimit(t, 1)

		_, _ = inngestgo.Send(ctx, inngestgo.Event{Name: "test/state-size.limit", Data: map[string]any{"success": true}})

		<-time.After(3 * time.Second)

		removeStateSizeLimit(t)

		run := c.Run(ctx, lastRunId)
		assert.Equal(t, "FAILED", run.Status)
		assert.Equal(t, "{\"error\":{\"error\":\"InngestErrStateOverflowed: The function run exceeded the state size limit of 1 bytes.\",\"name\":\"InngestErrStateOverflowed\",\"message\":\"The function run exceeded the state size limit of 1 bytes.\"}}", run.Output)
	})
}
