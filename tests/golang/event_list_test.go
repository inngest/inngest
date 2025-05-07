package golang

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventList(t *testing.T) {
	// t.Run("pagination", func(t *testing.T) {
	// 	r := require.New(t)
	// 	c := clientutil.NewClient(t).Setup()
	// 	defer c.Teardown()

	// 	var evts []*event.Event
	// 	for i := 0; i < 50; i++ {
	// 		evts = append(evts, &event.Event{Name: "evt"})
	// 	}
	// 	c.SendMany(evts)

	// 	var uniqueEventIDs types.Set[ulid.ULID]
	// 	var cursor string

	// 	// Get the first page.
	// 	r.EventuallyWithT(func(t *assert.CollectT) {
	// 		a := assert.New(t)
	// 		res, err := c.GetEvents()
	// 		if !a.NoError(err) {
	// 			return
	// 		}

	// 		// TODO: Assert that this is 50. Once we implement the totalCount
	// 		// resolver.
	// 		a.Equal(0, res.TotalCount)

	// 		// Default page size.
	// 		a.Len(res.Edges, 40)

	// 		if !a.NotNil(res.PageInfo.EndCursor) {
	// 			return
	// 		}

	// 		cursor = *res.PageInfo.EndCursor

	// 		for _, edge := range res.Edges {
	// 			uniqueEventIDs.Add(edge.Node.ID)
	// 		}
	// 	}, time.Second*10, time.Second*1)

	// 	// Get the next page.
	// 	r.EventuallyWithT(func(t *assert.CollectT) {
	// 		a := assert.New(t)
	// 		res, err := c.GetEvents(clientutil.GetEventsOpts{
	// 			After: &cursor,
	// 		})
	// 		if !a.NoError(err) {
	// 			return
	// 		}

	// 		// TODO: Assert that this is 50. Once we implement the totalCount
	// 		// resolver.
	// 		a.Equal(0, res.TotalCount)

	// 		a.Len(res.Edges, 10)

	// 		for _, edge := range res.Edges {
	// 			uniqueEventIDs.Add(edge.Node.ID)
	// 		}
	// 	}, time.Second*10, time.Second*1)

	// 	// Got all events across the 2 pages.
	// 	r.Equal(len(evts), uniqueEventIDs.Len())
	// })

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

		fmt.Println(event1Name, event2Name)

		// Send 1 event and wait for it show in the API.
		event1ID, err := ic.Send(ctx, inngestgo.Event{Name: event1Name})
		r.NoError(err)
		r.EventuallyWithT(func(t *assert.CollectT) {
			a := assert.New(t)
			res, err := c.GetEvents(ctx, client.GetEventsOpts{
				Filter: cqrs.EventsFilter{
					EventNames: []string{event1Name, event2Name},
					From:       time.Now().Add(-time.Minute).UTC(),
				},
			})
			if !a.NoError(err) {
				return
			}
			a.Len(res.Edges, 1)
		}, time.Second*10, time.Second*1)

		// Send a 2nd event and wait for it show in the API.
		event2ID, err := ic.Send(ctx, inngestgo.Event{Name: event2Name})
		r.NoError(err)
		var res *cqrs.EventsConnection
		r.EventuallyWithT(func(t *assert.CollectT) {
			a := assert.New(t)

			var err error
			res, err = c.GetEvents(ctx, client.GetEventsOpts{
				Filter: cqrs.EventsFilter{
					From: time.Now().Add(-time.Minute),
				},
			})
			if !a.NoError(err) {
				return
			}
			a.Len(res.Edges, 2)
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
			Filter: cqrs.EventsFilter{
				EventNames: []string{event1Name},
				From:       time.Now().Add(-time.Minute),
			},
		})
		r.NoError(err)
		r.Len(res.Edges, 1)
		r.Equal(res.Edges[0].Node.Name, event1Name)

		// Filter for "until" time works.
		res, err = c.GetEvents(ctx, client.GetEventsOpts{
			Filter: cqrs.EventsFilter{
				From:  time.Now().Add(-2 * time.Hour),
				Until: toPtr(time.Now().Add(-time.Hour)),
			},
		})
		r.NoError(err)
		r.Len(res.Edges, 0)
	})

	// t.Run("raw", func(t *testing.T) {
	// 	t.Run("minimal specified", func(t *testing.T) {
	// 		r := require.New(t)
	// 		c := clientutil.NewClient(t).Setup()
	// 		defer c.Teardown()

	// 		evt := event.Event{Name: "evt"}
	// 		eventID := c.Send(&evt)[0]

	// 		var raw string
	// 		r.EventuallyWithT(func(t *assert.CollectT) {
	// 			a := assert.New(t)
	// 			res, err := c.GetEvents(clientutil.GetEventsOpts{
	// 				Filter: gqlmodels.EventsFilter{
	// 					From: time.Now().Add(-time.Minute),
	// 				},
	// 			})
	// 			if !a.NoError(err) {
	// 				return
	// 			}

	// 			if !a.Len(res.Edges, 1) {
	// 				return
	// 			}
	// 			raw = res.Edges[0].Node.Raw
	// 		}, time.Second*10, time.Second*1)

	// 		var m map[string]any
	// 		r.NoError(json.Unmarshal([]byte(raw), &m))

	// 		// We don't know the exact timestamp, but we can check that it's
	// 		// recent.
	// 		ts, ok := m["ts"].(float64)
	// 		r.True(ok)
	// 		r.Greater(ts, float64(time.Now().Add(-time.Minute).UnixMilli()))

	// 		r.Equal(map[string]any{
	// 			"data": make(map[string]any),
	// 			"id":   eventID,
	// 			"name": evt.Name,
	// 			"ts":   ts,
	// 			"v":    nil,
	// 		}, m)
	// 	})

	// 	t.Run("everything specified", func(t *testing.T) {
	// 		r := require.New(t)
	// 		c := clientutil.NewClient(t).Setup()
	// 		defer c.Teardown()

	// 		evt := event.Event{
	// 			Data: map[string]any{
	// 				"foo": "bar",
	// 			},
	// 			ID:        ulid.Make().String(),
	// 			Name:      "evt",
	// 			Timestamp: time.Now().UnixMilli(),
	// 			Version:   "1",
	// 		}
	// 		c.Send(&evt)

	// 		var raw string
	// 		r.EventuallyWithT(func(t *assert.CollectT) {
	// 			a := assert.New(t)
	// 			res, err := c.GetEvents(clientutil.GetEventsOpts{
	// 				Filter: gqlmodels.EventsFilter{
	// 					From: time.Now().Add(-time.Minute),
	// 				},
	// 			})
	// 			if !a.NoError(err) {
	// 				return
	// 			}

	// 			if !a.Len(res.Edges, 1) {
	// 				return
	// 			}
	// 			raw = res.Edges[0].Node.Raw
	// 		}, time.Second*10, time.Second*1)

	// 		var m map[string]any
	// 		r.NoError(json.Unmarshal([]byte(raw), &m))
	// 		r.Equal(map[string]any{
	// 			"id":   evt.ID,
	// 			"name": evt.Name,
	// 			"data": evt.Data,
	// 			"ts":   float64(evt.Timestamp),
	// 			"v":    evt.Version,
	// 		}, m)
	// 	})
	// })

	// t.Run("runs", func(t *testing.T) {
	// 	r := require.New(t)
	// 	c := clientutil.NewClient(t).Setup()
	// 	defer c.Teardown()
	// 	ic, server, sync := test.NewSDKHandler(t, c)
	// 	defer server.Close()

	// 	eventName := "evt"
	// 	_, err := inngestgo.CreateFunction(
	// 		ic,
	// 		inngestgo.FunctionOpts{ID: "fn"},
	// 		inngestgo.EventTrigger(eventName, nil),
	// 		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
	// 			return nil, nil
	// 		},
	// 	)
	// 	r.NoError(err)
	// 	sync()

	// 	eventID := ulid.MustParse(c.Send(&event.Event{Name: eventName})[0])

	// 	var runs []*gqlmodels.FunctionRunV2
	// 	r.EventuallyWithT(func(t *assert.CollectT) {
	// 		a := assert.New(t)
	// 		res, err := c.GetEvents(clientutil.GetEventsOpts{
	// 			Filter: gqlmodels.EventsFilter{
	// 				From: time.Now().Add(-time.Minute),
	// 			},
	// 		})
	// 		if !a.NoError(err) {
	// 			return
	// 		}

	// 		// Includes "inngest/function.finished".
	// 		if !a.Len(res.Edges, 2) {
	// 			return
	// 		}

	// 		for _, edge := range res.Edges {
	// 			if edge.Node.ID == eventID {
	// 				runs = edge.Node.Runs
	// 			}
	// 		}
	// 		if !a.Len(runs, 1) {
	// 			return
	// 		}

	// 		a.Equal("fn", runs[0].Function.Name)
	// 		a.Equal(gqlmodels.FunctionRunStatusCompleted, runs[0].Status)
	// 	}, time.Second*10, time.Second*1)
	// })
}
