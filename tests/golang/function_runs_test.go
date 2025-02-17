package golang

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FnRunTestEvtData struct {
	Success bool `json:"success"`
	Index   int  `json:"idx"`
}
type FnRunTestEvt inngestgo.GenericEvent[FnRunTestEvtData, any]

func TestFunctionRunList(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)

	// Unique names so the test can be retried without a Dev Server restart.
	appName := fmt.Sprintf("fnrun-%d", time.Now().UnixNano())
	okEventName := fmt.Sprintf("%s/ok", appName)
	failedEventName := fmt.Sprintf("%s/failed", appName)

	c := client.New(t)
	h, server, registerFuncs := NewSDKHandler(t, appName)
	defer server.Close()

	var (
		ok     int32
		failed int32
	)
	fn1 := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "fn-run-ok",
		},
		inngestgo.EventTrigger(okEventName, nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			atomic.AddInt32(&ok, 1)
			return map[string]any{"num": input.Event.Data.Index * 2}, nil
		},
	)

	fn2 := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "fn-run-err", Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger(failedEventName, nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			atomic.AddInt32(&failed, 1)
			return nil, fmt.Errorf("fail")
		},
	)

	h.Register(fn1, fn2)
	registerFuncs()

	start := time.Now()

	successTotal := 10
	failureTotal := 3

	// attempt to randomize the event send time a little
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < successTotal; i++ {
			_, _ = inngestgo.Send(ctx, inngestgo.Event{Name: okEventName, Data: map[string]any{"success": true, "idx": i}})
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < failureTotal; i++ {
			_, _ = inngestgo.Send(ctx, inngestgo.Event{Name: failedEventName, Data: map[string]any{"success": false, "idx": i}})
		}
	}()

	wg.Wait()

	r.EventuallyWithT(func(t *assert.CollectT) {
		a := assert.New(t)
		a.EqualValues(successTotal, ok)
		a.EqualValues(failureTotal, failed)
	}, 10*time.Second, 100*time.Millisecond)
	end := time.Now().Add(10 * time.Second)

	total := successTotal + failureTotal

	// tests
	t.Run("retrieve all runs", func(t *testing.T) {
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			edges, pageInfo, count := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start: start,
				End:   end,
			})

			assert.Equal(t, total, len(edges))
			assert.False(t, pageInfo.HasNextPage)
			assert.Equal(t, total, count)

			// sorted by queued_at desc order by default
			ts := time.Now()
			for _, e := range edges {
				queuedAt := e.Node.QueuedAt
				assert.True(t, queuedAt.UnixMilli() <= ts.UnixMilli())
				ts = queuedAt
			}
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("retrieve only successful runs sorted by started_at", func(t *testing.T) {
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			edges, pageInfo, _ := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:     start,
				End:       end,
				TimeField: models.RunsV2OrderByFieldStartedAt,
				Order: []models.RunsV2OrderBy{
					{Field: models.RunsV2OrderByFieldStartedAt, Direction: models.RunsOrderByDirectionDesc},
				},
				Status: []string{models.FunctionRunStatusCompleted.String()},
			})

			assert.Equal(t, successTotal, len(edges))
			assert.False(t, pageInfo.HasNextPage)

			// should be sorted by started_at desc order
			ts := time.Now()
			for _, e := range edges {
				startedAt := e.Node.StartedAt
				assert.True(t, startedAt.UnixMilli() <= ts.UnixMilli())
				ts = startedAt
			}
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("retrieve only failed runs sorted by ended_at", func(t *testing.T) {
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			edges, pageInfo, _ := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:     start,
				End:       end,
				TimeField: models.RunsV2OrderByFieldEndedAt,
				Order: []models.RunsV2OrderBy{
					{Field: models.RunsV2OrderByFieldEndedAt, Direction: models.RunsOrderByDirectionAsc},
				},
				Status: []string{models.FunctionRunStatusFailed.String()},
			})

			assert.Equal(t, failureTotal, len(edges))
			assert.False(t, pageInfo.HasNextPage)

			// should be sorted by ended_at asc order
			ts := start
			for _, e := range edges {
				endedAt := e.Node.EndedAt
				assert.True(t, endedAt.UnixMilli() >= ts.UnixMilli())
				ts = endedAt
			}
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("retrieve only failed runs", func(t *testing.T) {
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			edges, pageInfo, _ := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:  start,
				End:    end,
				Status: []string{models.FunctionRunStatusFailed.String()},
			})

			assert.Equal(t, failureTotal, len(edges))
			assert.False(t, pageInfo.HasNextPage)

			// should be sorted by queued_at desc order
			ts := time.Now()
			for _, e := range edges {
				queuedAt := e.Node.QueuedAt
				assert.True(t, queuedAt.UnixMilli() <= ts.UnixMilli())
				ts = queuedAt
			}
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("paginate without additional filter", func(t *testing.T) {
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			items := 10
			edges, pageInfo, _ := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start: start,
				End:   end,
				Items: items,
			})

			assert.Equal(t, items, len(edges))
			assert.True(t, pageInfo.HasNextPage)

			// there should be only 3 left
			edges, pageInfo, _ = c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:  start,
				End:    end,
				Items:  items,
				Cursor: *pageInfo.EndCursor,
			})
			remain := successTotal + failureTotal - items
			assert.Equal(t, remain, len(edges))
			assert.False(t, pageInfo.HasNextPage)
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("paginate with status filter", func(t *testing.T) {
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			items := 2
			edges, pageInfo, total := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start: start,
				// End:       end,
				End:       time.Now(),
				TimeField: models.RunsV2OrderByFieldEndedAt,
				Status:    []string{models.FunctionRunStatusFailed.String()},
				Order: []models.RunsV2OrderBy{
					{Field: models.RunsV2OrderByFieldEndedAt, Direction: models.RunsOrderByDirectionDesc},
				},
				Items: items,
			})

			assert.Equal(t, 2, len(edges))
			assert.True(t, pageInfo.HasNextPage)
			assert.Equal(t, failureTotal, total)

			// there are only 3 failed runs, so there shouldn't be anymore than 1
			edges, pageInfo, _ = c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start: start,
				// End:       end,
				End:       time.Now(),
				TimeField: models.RunsV2OrderByFieldEndedAt,
				Status:    []string{models.FunctionRunStatusFailed.String()},
				Items:     items,
				Order: []models.RunsV2OrderBy{
					{Field: models.RunsV2OrderByFieldEndedAt, Direction: models.RunsOrderByDirectionDesc},
				},
				Cursor: *pageInfo.EndCursor,
			})

			remain := failureTotal - items
			assert.Equal(t, remain, len(edges))
			assert.False(t, pageInfo.HasNextPage)
			assert.Equal(t, failureTotal, total)
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("filter with event CEL expression", func(t *testing.T) {
		min := 5
		cel := celBlob([]string{
			fmt.Sprintf("event.name == '%s'", okEventName),
			fmt.Sprintf("event.data.idx > %d", min),
		})

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			items := 3
			edges, pageInfo, total := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start: start,
				End:   end,
				Items: items,
				Query: &cel,
			})

			assert.Equal(t, items, len(edges))
			assert.Equal(t, successTotal-(min+1), total)
			assert.True(t, pageInfo.HasNextPage)
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("filter with output CEL expression", func(t *testing.T) {
		cel := celBlob([]string{
			"output.num > 11",
		})
		expectedCount := 4

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			edges, pageInfo, total := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start: start,
				End:   end,
				Query: &cel,
			})

			assert.Equal(t, expectedCount, len(edges))
			assert.Equal(t, expectedCount, total)
			assert.False(t, pageInfo.HasNextPage)
		}, 10*time.Second, 2*time.Second)
	})
}

func celBlob(cel []string) string {
	return strings.Join(cel, "\n")
}
