package golang

import (
	"context"
	"fmt"
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

type FnRunTestEvtData struct{}
type FnRunTestEvt inngestgo.GenericEvent[FnRunTestEvtData, any]

func TestFunctionRunList(t *testing.T) {
	ctx := context.Background()

	c := client.New(t)
	h, server, registerFuncs := NewSDKHandler(t, "fnrun")
	defer server.Close()

	var (
		ok     int32
		failed int32
	)
	fn1 := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "fn-run-ok",
		},
		inngestgo.EventTrigger("fnrun/ok", nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			atomic.AddInt32(&ok, 1)
			return nil, nil
		},
	)

	fn2 := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			Name: "fn-run-err", Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("fnrun/failed", nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			atomic.AddInt32(&failed, 1)
			return nil, fmt.Errorf("fail")
		},
	)

	h.Register(fn1, fn2)
	registerFuncs()

	// buy some time here so it doesn't collide with other runs happening :fingers_crossed:
	<-time.After(2 * time.Second)

	start := time.Now()

	successTotal := 10
	failureTotal := 3

	// attempt to randomize the event send time a little
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < successTotal; i++ {
			_, _ = inngestgo.Send(ctx, inngestgo.Event{Name: "fnrun/ok", Data: map[string]any{"success": true}})
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < failureTotal; i++ {
			_, _ = inngestgo.Send(ctx, inngestgo.Event{Name: "fnrun/failed", Data: map[string]any{"success": false}})
		}
	}()

	wg.Wait()

	<-time.After(3 * time.Second)
	end := time.Now()
	<-time.After(3 * time.Second)

	require.EqualValues(t, successTotal, ok)
	require.EqualValues(t, failureTotal, failed)

	total := successTotal + failureTotal

	// tests
	t.Run("retrieve all runs", func(t *testing.T) {

		require.Eventually(t, func() bool {
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

			return true
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("retrieve only successful runs sorted by started_at", func(t *testing.T) {
		require.Eventually(t, func() bool {
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

			return true
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("retrieve only failed runs sorted by ended_at", func(t *testing.T) {
		require.Eventually(t, func() bool {
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

			return true
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("retrieve only failed runs", func(t *testing.T) {
		require.Eventually(t, func() bool {
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

			return true
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("paginate without additional filter", func(t *testing.T) {
		require.Eventually(t, func() bool {
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

			return true
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("paginate with status filter", func(t *testing.T) {
		require.Eventually(t, func() bool {
			items := 2
			edges, pageInfo, total := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:     start,
				End:       end,
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
				Start:     start,
				End:       end,
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

			return true
		}, 10*time.Second, 2*time.Second)
	})
}
