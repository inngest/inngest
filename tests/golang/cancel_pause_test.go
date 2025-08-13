package golang

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/oklog/ulid/v2"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPauseCancelFunction(t *testing.T) {
	ctx := context.Background()

	randomSuffix := ulid.MustNew(ulid.Now(), rand.Reader).String()

	appName := "app-test-pause-cancel" + randomSuffix
	c := client.New(t)
	inngestClient, server, registerFuncs := NewSDKHandler(t, appName)
	defer server.Close()

	var (
		runCounter   int32
		runCancelled int32
		runID        string
	)

	triggerEvtName := uuid.New().String()

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "function-test-pause-cancel",
		},
		inngestgo.EventTrigger(triggerEvtName, nil),
		func(ctx context.Context, input inngestgo.Input[testCancelEvt]) (any, error) {
			_, _ = step.Run(ctx, "do something", func(ctx context.Context) (any, error) {
				runID = input.InputCtx.RunID
				fmt.Println("HELLO")

				atomic.AddInt32(&runCounter, 1)
				return nil, nil
			})

			step.Sleep(ctx, "stop", 30*time.Second)

			_, _ = step.Run(ctx, "should not happen", func(ctx context.Context) (any, error) {
				atomic.AddInt32(&runCounter, 1)
				return nil, nil
			})

			return true, nil
		},
	)
	require.NoError(t, err)

	fnSlug := appName + "-function-test-pause-cancel"

	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "handle-cancel"},
		inngestgo.EventTrigger(
			"inngest/function.cancelled",
			inngestgo.StrPtr(fmt.Sprintf("event.data.function_id == '%s'", fnSlug)),
		),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("CANCELLED")

			atomic.AddInt32(&runCancelled, 1)

			return true, nil
		},
	)

	require.NoError(t, err)
	registerFuncs()

	fnId := ""
	require.Eventually(t, func() bool {
		functions, err := c.Functions(ctx)
		if err != nil {
			return false
		}
		for _, function := range functions {
			if function.App.ExternalID != appName {
				continue
			}

			fnId = function.ID
			return true
		}
		return false
	}, 10*time.Second, 250*time.Millisecond)

	// Ensure that the runs are actually cancelled in the queue
	getQueueSize := func(accountId uuid.UUID, fnId uuid.UUID) int {
		reqUrl, err := url.Parse(c.APIHost + "/test/queue/function-queue-size")
		require.NoError(t, err)

		fv := reqUrl.Query()
		fv.Add("accountId", consts.DevServerAccountID.String())
		fv.Add("fnId", fnId.String())

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl.String()+"?"+fv.Encode(), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		byt, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		r := map[string]any{}
		err = json.Unmarshal(byt, &r)
		require.NoError(t, err, "Test API may not be enabled! Error unmarshalling %s", byt)

		count, ok := r["count"].(float64)
		require.True(t, ok)

		return int(count)
	}

	// Ensure that the runs are actually cancelled in the queue
	pauseFn := func(accountId uuid.UUID, fnId uuid.UUID) {
		reqUrl, err := url.Parse(c.APIHost + "/test/function/pause")
		require.NoError(t, err)

		fv := reqUrl.Query()
		fv.Add("accountId", consts.DevServerAccountID.String())
		fv.Add("fnId", fnId.String())

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUrl.String()+"?"+fv.Encode(), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		_ = resp.Body.Close()
	}

	// Ensure that the runs are actually cancelled in the queue
	cancelFnRun := func(accountId uuid.UUID, fnId uuid.UUID, runId ulid.ULID) {
		reqUrl, err := url.Parse(c.APIHost + "/test/function/runs/cancel")
		require.NoError(t, err)

		fv := reqUrl.Query()
		fv.Add("accountId", consts.DevServerAccountID.String())
		fv.Add("fnId", fnId.String())
		fv.Add("runId", runId.String())

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUrl.String()+"?"+fv.Encode(), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		_ = resp.Body.Close()
	}

	evt := inngestgo.Event{
		Name: triggerEvtName,
		Data: map[string]any{"cancel": 1},
	}
	_, err = inngestClient.Send(ctx, evt)
	require.NoError(t, err)

	t.Run("check run", func(t *testing.T) {
		r := require.New(t)
		r.EventuallyWithT(func(t *assert.CollectT) {
			a := assert.New(t)
			a.Equal(int32(1), atomic.LoadInt32(&runCounter))
			a.Equal(int32(0), atomic.LoadInt32(&runCancelled))
		}, 20*time.Second, 10*time.Millisecond)
	})

	t.Run("should cancel run", func(t *testing.T) {
		pauseFn(consts.DevServerAccountID, uuid.MustParse(fnId))
		cancelFnRun(consts.DevServerAccountID, uuid.MustParse(fnId), ulid.MustParse(runID))

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			r := require.New(t)
			r.Equal(int32(1), atomic.LoadInt32(&runCounter))
			r.Equal(int32(1), atomic.LoadInt32(&runCancelled))
		}, 10*time.Second, 10*time.Millisecond)

		require.Equal(t, 0, getQueueSize(consts.DevServerAccountID, uuid.MustParse(fnId)))
	})

	t.Run("trace run should have appropriate data", func(t *testing.T) {
		run := c.WaitForRunTraces(ctx, t, &runID, client.WaitForRunTracesOptions{
			Status:         models.FunctionStatusCancelled,
			Timeout:        10 * time.Second,
			Interval:       500 * time.Millisecond,
			ChildSpanCount: 1, // at least 1
		})

		require.Equal(t, models.RunTraceSpanStatusCancelled.String(), run.Trace.Status)
	})
}
