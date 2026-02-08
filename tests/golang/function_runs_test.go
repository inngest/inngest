package golang

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
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
type FnRunTestEvt inngestgo.GenericEvent[FnRunTestEvtData]

func TestFunctionRunList(t *testing.T) {
	ctx := context.Background()
	r := require.New(t)

	// Unique names so the test can be retried without a Dev Server restart.
	appName := fmt.Sprintf("fnrun-%d", time.Now().UnixNano())
	okEventName := fmt.Sprintf("%s/ok", appName)
	failedEventName := fmt.Sprintf("%s/failed", appName)

	inngestClient, server, registerFuncs := NewSDKHandler(t, appName)
	defer server.Close()

	var (
		ok     int32
		failed int32

		// We want to constrain queries to only these function IDs.  In order to
		// do such a thing, we store a list of our function IDs in each execution.
		ids sync.Map
	)

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: fmt.Sprintf("fn-run-ok-%s", okEventName),
		},
		inngestgo.EventTrigger(okEventName, nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvtData]) (any, error) {
			atomic.AddInt32(&ok, 1)
			ids.Store(uuid.MustParse(input.InputCtx.FunctionID), true)
			return map[string]any{"num": input.Event.Data.Index * 2}, nil
		},
	)
	require.NoError(t, err)
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      fmt.Sprintf("fn-run-err-%s", failedEventName),
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger(failedEventName, nil),
		func(ctx context.Context, input inngestgo.Input[FnRunTestEvt]) (any, error) {
			atomic.AddInt32(&failed, 1)
			ids.Store(uuid.MustParse(input.InputCtx.FunctionID), true)
			// NOTE: If functions end at the same millisecond, this breaks dev server pagination.
			// Randomizing the duration means that we have less of a chance for functions to end
			// on the same millisecond, meaning pagination works as expected.
			<-time.After(time.Duration(rand.Intn(100)) * time.Millisecond)
			return nil, fmt.Errorf("fail")
		},
	)
	require.NoError(t, err)
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
			_, _ = inngestClient.Send(ctx, inngestgo.Event{Name: okEventName, Data: map[string]any{"success": true, "idx": i}})
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < failureTotal; i++ {
			_, _ = inngestClient.Send(ctx, inngestgo.Event{Name: failedEventName, Data: map[string]any{"success": false, "idx": i}})
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

	// Build function IDs list for filtering - needed to isolate from concurrent tests
	fnIDs := []uuid.UUID{}
	ids.Range(func(key any, value any) bool {
		if id, ok := key.(uuid.UUID); ok {
			fnIDs = append(fnIDs, id)
		}
		return true
	})

	// tests
	t.Run("retrieve all runs", func(t *testing.T) {
		c := client.New(t)
		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			edges, pageInfo, count := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:       start,
				End:         end,
				FunctionIDs: fnIDs,
			})

			// With parallel event processing, we should have at least the expected runs
			assert.GreaterOrEqual(ct, len(edges), total)
			assert.False(ct, pageInfo.HasNextPage)
			assert.GreaterOrEqual(ct, count, total)

			// sorted by queued_at desc order by default
			ts := time.Now()
			for _, e := range edges {
				queuedAt := e.Node.QueuedAt
				assert.True(ct, queuedAt.UnixMilli() <= ts.UnixMilli())
				ts = queuedAt
			}
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("retrieve only successful runs sorted by started_at", func(t *testing.T) {
		c := client.New(t)
		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			edges, pageInfo, _ := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:     start,
				End:       end,
				TimeField: models.RunsV2OrderByFieldStartedAt,
				Order: []models.RunsV2OrderBy{
					{Field: models.RunsV2OrderByFieldStartedAt, Direction: models.RunsOrderByDirectionDesc},
				},
				Status:      []string{models.FunctionRunStatusCompleted.String()},
				FunctionIDs: fnIDs,
			})

			// With parallel event processing, we should have at least the expected runs
			assert.GreaterOrEqual(ct, len(edges), successTotal)
			assert.False(ct, pageInfo.HasNextPage)

			// should be sorted by started_at desc order
			ts := time.Now()
			for _, e := range edges {
				startedAt := e.Node.StartedAt
				assert.True(ct, startedAt.UnixMilli() <= ts.UnixMilli())
				ts = startedAt
			}
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("retrieve only failed runs sorted by ended_at", func(t *testing.T) {
		c := client.New(t)
		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			edges, pageInfo, _ := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:     start,
				End:       end,
				TimeField: models.RunsV2OrderByFieldEndedAt,
				Order: []models.RunsV2OrderBy{
					{Field: models.RunsV2OrderByFieldEndedAt, Direction: models.RunsOrderByDirectionAsc},
				},
				Status:      []string{models.FunctionRunStatusFailed.String()},
				FunctionIDs: fnIDs,
			})

			assert.Equal(ct, failureTotal, len(edges))
			assert.False(ct, pageInfo.HasNextPage)

			// should be sorted by ended_at asc order
			ts := start
			for _, e := range edges {
				endedAt := e.Node.EndedAt
				assert.True(ct, endedAt.UnixMilli() >= ts.UnixMilli())
				ts = endedAt
			}
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("retrieve only failed runs", func(t *testing.T) {
		c := client.New(t)
		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			edges, pageInfo, _ := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:       start,
				End:         end,
				Status:      []string{models.FunctionRunStatusFailed.String()},
				FunctionIDs: fnIDs,
			})

			assert.Equal(ct, failureTotal, len(edges))
			assert.False(ct, pageInfo.HasNextPage)

			// should be sorted by queued_at desc order
			ts := time.Now()
			for _, e := range edges {
				queuedAt := e.Node.QueuedAt
				assert.True(ct, queuedAt.UnixMilli() <= ts.UnixMilli())
				ts = queuedAt
			}
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("paginate without additional filter", func(t *testing.T) {
		c := client.New(t)
		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			items := 10
			edges, pageInfo, totalCount := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:       start,
				End:         end,
				Items:       items,
				FunctionIDs: fnIDs, // Filter by function IDs to isolate from concurrent tests
			})

			// With parallel event processing, we should have at least the expected runs
			assert.GreaterOrEqual(ct, totalCount, successTotal+failureTotal)
			assert.Equal(ct, items, len(edges))
			assert.True(ct, pageInfo.HasNextPage)

			// Second page should have the remaining items
			edges, pageInfo, _ = c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start:       start,
				End:         end,
				Items:       items,
				Cursor:      *pageInfo.EndCursor,
				FunctionIDs: fnIDs,
			})
			// Remaining count depends on total, which may vary with parallel processing
			remain := totalCount - items
			assert.GreaterOrEqual(ct, len(edges), remain)
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("paginate with status filter", func(t *testing.T) {
		c := client.New(t)
		require.EventuallyWithT(t, func(ct *assert.CollectT) {
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
				FunctionIDs: fnIDs,
				Items:       items,
			})

			assert.Equal(ct, 2, len(edges))
			assert.True(ct, pageInfo.HasNextPage)
			assert.Equal(ct, failureTotal, total)

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
				FunctionIDs: fnIDs,
				Cursor:      *pageInfo.EndCursor,
			})

			remain := failureTotal - items // we should paginate and remove the 2 previous from the total.
			assert.False(ct, pageInfo.HasNextPage, "Failed with IDs: %s (%s)", fnIDs, fmt.Sprintf("fn-run-err-%s", failedEventName))
			assert.Equal(ct, failureTotal, total)
			assert.Equal(ct, remain, len(edges), "Got %#v and page info %#v", edges, pageInfo)
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("filter with event CEL expression", func(t *testing.T) {
		c := client.New(t)

		// test setup created 10 runs numbered 0-9, with the > min filter, we should get runs 6, 7, 8, 9 eventually
		// Because there are 4 runs after the filter, if we are requesting a page of 3 runs, we should get a page with
		// 3 items and with HasNextPage true
		min := 5
		cel := celBlob([]string{
			fmt.Sprintf("event.name == '%s'", okEventName),
			fmt.Sprintf("event.data.idx > %d", min),
		})

		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			items := 3
			edges, pageInfo, total := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start: start,
				End:   end,
				Items: items,
				Query: &cel,
			})

			assert.Equal(ct, items, len(edges))
			assert.Equal(ct, successTotal-(min+1), total)
			assert.True(ct, pageInfo.HasNextPage)
		}, 10*time.Second, 2*time.Second)
	})

	t.Run("filter with output CEL expression", func(t *testing.T) {
		c := client.New(t)

		// test setup created 10 runs numbered 0-9. The runs output double their run number
		// so runs 6, 7, 8, 9 should produce output.num greater than 11
		cel := celBlob([]string{
			"output.num > 11",
		})
		expectedCount := 4

		require.EventuallyWithT(t, func(ct *assert.CollectT) {
			edges, pageInfo, total := c.FunctionRuns(ctx, client.FunctionRunOpt{
				Start: start,
				End:   end,
				Query: &cel,
			})

			assert.Equal(ct, expectedCount, len(edges))
			assert.Equal(ct, expectedCount, total)
			assert.False(ct, pageInfo.HasNextPage)
		}, 10*time.Second, 2*time.Second)
	})
}

func celBlob(cel []string) string {
	return strings.Join(cel, "\n")
}
