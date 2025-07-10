package golang

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventList(t *testing.T) {
	t.Run("pagination", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()
		c := client.New(t)

		ic, err := inngestgo.NewClient(inngestgo.ClientOpts{
			AppID: "app",
			Dev:   toPtr(true),
		})
		r.NoError(err)

		eventName := randomSuffix("evt")
		const numEvents = 50
		evts := make([]any, 0, numEvents)
		for i := 0; i < numEvents; i++ {
			evts = append(evts, inngestgo.Event{Name: eventName})
		}
		sentEventIds, err := ic.SendMany(ctx, evts)
		r.NoError(err)
		r.Equal(numEvents, len(sentEventIds))

		uniqueEventIDs := make(map[ulid.ULID]bool)
		var cursor string

		// Get the first page.
		const pageSize = 40
		eventsFilter := models.EventsFilter{
			EventNames: []string{eventName},
			From:       time.Now().Add(-time.Minute).UTC(),
		}
		r.EventuallyWithT(func(t *assert.CollectT) {
			r := require.New(t)
			res, err := c.GetEvents(ctx, client.GetEventsOpts{
				PageSize: pageSize,
				Filter:   eventsFilter,
			})
			r.NoError(err)

			r.Equal(numEvents, res.TotalCount)
			r.True(res.PageInfo.HasNextPage)
			r.Len(res.Edges, pageSize)

			r.NotNil(res.PageInfo.EndCursor)

			cursor = *res.PageInfo.EndCursor

			for _, edge := range res.Edges {
				// TODO Do I need to protect with mutex?
				uniqueEventIDs[edge.Node.ID] = true
			}
		}, time.Second*10, time.Second*1)

		// Get the next page.
		r.EventuallyWithT(func(t *assert.CollectT) {
			r := require.New(t)
			res, err := c.GetEvents(ctx, client.GetEventsOpts{
				PageSize: pageSize,
				Cursor:   &cursor,
				Filter:   eventsFilter,
			})
			r.NoError(err)

			r.Equal(50, res.TotalCount)
			r.False(res.PageInfo.HasNextPage)
			r.Len(res.Edges, 10)

			for _, edge := range res.Edges {
				uniqueEventIDs[edge.Node.ID] = true
			}
		}, time.Second*10, time.Second*1)

		// Got all events across the 2 pages.
		r.Equal(len(evts), len(uniqueEventIDs))
	})

	t.Run("2 event types", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()
		c := client.New(t)

		ic, err := inngestgo.NewClient(inngestgo.ClientOpts{
			AppID: "app",
			Dev:   toPtr(true),
		})
		r.NoError(err)
		event1Name := randomSuffix("foo")
		event2Name := randomSuffix("bar")

		// Send 1 event and wait for it show in the API.
		event1ID, err := ic.Send(ctx, inngestgo.Event{Name: event1Name})
		r.NoError(err)
		r.EventuallyWithT(func(t *assert.CollectT) {
			r := require.New(t)
			res, err := c.GetEvents(ctx, client.GetEventsOpts{
				PageSize: 10,
				Filter: models.EventsFilter{
					EventNames: []string{event1Name, event2Name},
					From:       time.Now().Add(-time.Minute).UTC(),
				},
			})
			r.NoError(err)
			r.Equal(1, res.TotalCount)
			r.Len(res.Edges, 1)
		}, time.Second*10, time.Second*1)

		// Send a 2nd event and wait for it show in the API.
		event2ID, err := ic.Send(ctx, inngestgo.Event{Name: event2Name})
		r.NoError(err)
		var res *models.EventsConnection
		r.EventuallyWithT(func(t *assert.CollectT) {
			r := require.New(t)

			var err error
			res, err = c.GetEvents(ctx, client.GetEventsOpts{
				PageSize: 10,
				Filter: models.EventsFilter{
					EventNames: []string{event1Name, event2Name},
					From:       time.Now().Add(-time.Minute),
				},
			})
			r.NoError(err)
			r.Equal(2, res.TotalCount)
			r.Len(res.Edges, 2)
		}, time.Second*10, time.Second*1)

		// 1st event in the API response is the 2nd event sent, since it's in
		// descending order.
		event := res.Edges[0].Node
		r.Equal(event2ID, event.ID.String())
		r.Equal(event2Name, event.Name)
		r.NotZero(event.OccurredAt)
		r.NotZero(event.ReceivedAt)

		// 2nd event.
		event = res.Edges[1].Node
		r.Equal(event1ID, event.ID.String())
		r.Equal(event1Name, event.Name)

		// Filtering for 1 event name works.
		res, err = c.GetEvents(ctx, client.GetEventsOpts{
			PageSize: 10,
			Filter: models.EventsFilter{
				EventNames: []string{event1Name},
				From:       time.Now().Add(-time.Minute),
			},
		})
		r.NoError(err)
		r.Len(res.Edges, 1)
		r.Equal(res.Edges[0].Node.Name, event1Name)

		// Filter for "until" time works.
		res, err = c.GetEvents(ctx, client.GetEventsOpts{
			PageSize: 10,
			Filter: models.EventsFilter{
				From:  time.Now().Add(-2 * time.Hour),
				Until: toPtr(time.Now().Add(-time.Hour)),
			},
		})
		r.NoError(err)
		r.Equal(0, res.TotalCount)
		r.Len(res.Edges, 0)
	})

	t.Run("raw", func(t *testing.T) {
		t.Run("minimal specified", func(t *testing.T) {
			r := require.New(t)
			ctx := context.Background()
			c := client.New(t)

			inngestClient, err := inngestgo.NewClient(inngestgo.ClientOpts{
				AppID: "app",
				Dev:   toPtr(true),
			})
			r.NoError(err)
			eventName := randomSuffix("evt")
			evt := event.Event{Name: eventName}
			eventID, err := inngestClient.Send(ctx, &evt)
			r.NoError(err)

			var raw string
			r.EventuallyWithT(func(t *assert.CollectT) {
				r := require.New(t)
				res, err := c.GetEvents(ctx, client.GetEventsOpts{
					PageSize: 10,
					Filter: models.EventsFilter{
						EventNames: []string{eventName},
						From:       time.Now().Add(-time.Minute),
					},
				})
				r.NoError(err)

				r.Len(res.Edges, 1)
				raw = res.Edges[0].Node.Raw
			}, time.Second*10, time.Second*1)

			var m map[string]any
			r.NoError(json.Unmarshal([]byte(raw), &m))

			// We don't know the exact timestamp, but we can check that it's
			// recent.
			ts, ok := m["ts"].(float64)
			r.True(ok)
			r.Greater(ts, float64(time.Now().Add(-time.Minute).UnixMilli()))

			r.Equal(map[string]any{
				"data": make(map[string]any),
				"id":   eventID,
				"name": evt.Name,
				"ts":   ts,
				"v":    nil,
			}, m)
		})

		t.Run("everything specified", func(t *testing.T) {
			r := require.New(t)
			ctx := context.Background()
			c := client.New(t)

			inngestClient, err := inngestgo.NewClient(inngestgo.ClientOpts{
				AppID: "app",
				Dev:   toPtr(true),
			})
			r.NoError(err)

			eventName := randomSuffix("evt")
			evt := event.Event{
				Data: map[string]any{
					"foo": "bar",
				},
				ID:        ulid.Make().String(),
				Name:      eventName,
				Timestamp: time.Now().UnixMilli(),
				Version:   "1",
			}
			_, err = inngestClient.Send(ctx, &evt)
			r.NoError(err)

			var raw string
			r.EventuallyWithT(func(t *assert.CollectT) {
				r := require.New(t)
				res, err := c.GetEvents(ctx, client.GetEventsOpts{
					PageSize: 10,
					Filter: models.EventsFilter{
						EventNames: []string{eventName},
						From:       time.Now().Add(-time.Minute),
					},
				})
				r.NoError(err)

				r.Len(res.Edges, 1)
				raw = res.Edges[0].Node.Raw
			}, time.Second*10, time.Second*1)

			var m map[string]any
			r.NoError(json.Unmarshal([]byte(raw), &m))
			r.Equal(map[string]any{
				"id":   evt.ID,
				"name": evt.Name,
				"data": evt.Data,
				"ts":   float64(evt.Timestamp),
				"v":    evt.Version,
			}, m)
		})
	})

	t.Run("runs", func(t *testing.T) {

		r := require.New(t)
		c := client.New(t)
		inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-test")
		defer server.Close()

		eventName := randomSuffix("evt")
		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{ID: "fn"},
			inngestgo.EventTrigger(eventName, nil),
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				return nil, nil
			},
		)
		r.NoError(err)
		registerFuncs()

		ctx := context.Background()
		eventID, err := inngestClient.Send(ctx, &event.Event{Name: eventName})
		r.NoError(err)

		var runs []*models.FunctionRunV2
		r.EventuallyWithT(func(t *assert.CollectT) {
			r := require.New(t)
			res, err := c.GetEvents(ctx, client.GetEventsOpts{
				PageSize: 10,
				Filter: models.EventsFilter{
					EventNames: []string{eventName, "inngest/function.finished"},
					From:       time.Now().Add(-time.Minute),
				},
			})
			r.NoError(err)

			// Includes "inngest/function.finished".
			// r.Len(res.Edges, 2)

			for _, edge := range res.Edges {
				if edge.Node.ID == ulid.MustParse(eventID) {
					runs = edge.Node.Runs
				}
			}
			// TODO: punting on FunctionRunsV2 for now
			r.Len(runs, 0)
			//r.Len(runs, 1)
		}, time.Second*10, time.Second*1)
	})
}
