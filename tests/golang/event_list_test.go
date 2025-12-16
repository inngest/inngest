package golang

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
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
		t.Run("3 functions 1-1 with events", func(t *testing.T) {
			// events 1-1 to functions is the simple case and we want to test that passing in different
			// sets of eventNames does proper filtering in the API
			r := require.New(t)
			c := client.New(t)
			inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-test")
			defer server.Close()

			numEvents := 3
			eventNames := make([]string, numEvents)
			fnNames := make([]string, numEvents)

			for i := 0; i < numEvents; i++ {
				eventName := randomSuffix("evt")
				fnName := fmt.Sprintf("fn-%d", i)

				eventNames[i] = eventName
				fnNames[i] = fnName

				_, err := inngestgo.CreateFunction(
					inngestClient,
					inngestgo.FunctionOpts{ID: fnName},
					inngestgo.EventTrigger(eventName, nil),
					func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
						return nil, nil
					},
				)
				r.NoError(err)
			}
			registerFuncs()

			ctx := context.Background()

			eventIDs := make([]string, numEvents)
			for i, eventName := range eventNames {
				eventID, err := inngestClient.Send(ctx, &event.Event{Name: eventName})
				r.NoError(err)
				eventIDs[i] = eventID
			}

			// If we filter for a single event name, we get the single associated function
			for i, eventName := range eventNames {
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

					r.Equal(1, res.TotalCount)
					node := res.Edges[0].Node
					r.Equal(ulid.MustParse(eventIDs[i]), node.ID)
					r.Len(node.Runs, 1)
					r.Equal(node.Runs[0].Function.Name, fnNames[i])
				}, time.Second*10, time.Second*1)
			}

			// filtering on a subset of event names works
			r.EventuallyWithT(func(t *assert.CollectT) {
				r := require.New(t)
				res, err := c.GetEvents(ctx, client.GetEventsOpts{
					PageSize: 10,
					Filter: models.EventsFilter{
						EventNames: []string{eventNames[0], eventNames[1]},
						From:       time.Now().Add(-time.Minute),
					},
				})
				r.NoError(err)

				r.Equal(2, res.TotalCount)
				for _, edge := range res.Edges {
					i := slices.Index(eventIDs, edge.Node.ID.String())
					r.NotEqual(-1, i) // make sure we found the event id
					r.Len(edge.Node.Runs, 1)
					r.Equal(edge.Node.Runs[0].Function.Name, fnNames[i])
				}
			}, time.Second*10, time.Second*1)

			// filtering on a different subset of event names works
			r.EventuallyWithT(func(t *assert.CollectT) {
				r := require.New(t)
				res, err := c.GetEvents(ctx, client.GetEventsOpts{
					PageSize: 10,
					Filter: models.EventsFilter{
						EventNames: []string{eventNames[1], eventNames[2]},
						From:       time.Now().Add(-time.Minute),
					},
				})
				r.NoError(err)

				r.Equal(2, res.TotalCount)
				for _, edge := range res.Edges {
					i := slices.Index(eventIDs, edge.Node.ID.String())
					r.NotEqual(-1, i) // make sure we found the event id
					r.Len(edge.Node.Runs, 1)
					r.Equal(edge.Node.Runs[0].Function.Name, fnNames[i])
				}
			}, time.Second*10, time.Second*1)
		})

		t.Run("1 event fans out to 2 functions", func(t *testing.T) {
			r := require.New(t)
			c := client.New(t)
			inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-test")
			defer server.Close()

			eventName := randomSuffix("evt")
			numFuncs := 2
			fnNames := make([]string, numFuncs)
			for i := 0; i < numFuncs; i++ {
				fnName := fmt.Sprintf("fn-%d", i)
				fnNames[i] = fnName

				_, err := inngestgo.CreateFunction(
					inngestClient,
					inngestgo.FunctionOpts{ID: fnName},
					inngestgo.EventTrigger(eventName, nil), // same event trigger
					func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
						return nil, nil
					},
				)
				r.NoError(err)
			}
			registerFuncs()

			ctx := context.Background()

			eventID, err := inngestClient.Send(ctx, &event.Event{Name: eventName})
			r.NoError(err)

			// We should see 2 runs associated with one event
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

				r.Equal(1, res.TotalCount)
				node := res.Edges[0].Node
				r.Equal(ulid.MustParse(eventID), node.ID)
				r.Len(node.Runs, 2)
				// Find each function name inside the runs, since we only have 2, both of these succeeding is sufficient
				for _, fnName := range fnNames {
					i := slices.IndexFunc(node.Runs, func(run *models.FunctionRunV2) bool {
						return run.Function.Name == fnName
					})
					r.NotEqual(-1, i) // make sure we found the function name
				}
			}, time.Second*10, time.Second*1)
		})

		t.Run("3 events non-batch to 1 function", func(t *testing.T) {
			r := require.New(t)
			c := client.New(t)
			inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-test")
			defer server.Close()

			eventName := randomSuffix("evt")
			fnName := "fn"

			_, err := inngestgo.CreateFunction(
				inngestClient,
				inngestgo.FunctionOpts{ID: fnName},
				inngestgo.EventTrigger(eventName, nil),
				func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
					return nil, nil
				},
			)
			r.NoError(err)
			registerFuncs()

			ctx := context.Background()

			numEvents := 3
			eventIDs := make([]string, numEvents)
			for i := 0; i < numEvents; i++ {
				eventID, err := inngestClient.Send(ctx, &event.Event{Name: eventName})
				r.NoError(err)
				eventIDs[i] = eventID
			}

			// We should see 3 events, all associated to the same function but different runs
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

				r.Equal(3, res.TotalCount)

				var runIDs []ulid.ULID
				for _, edge := range res.Edges {
					i := slices.Index(eventIDs, edge.Node.ID.String())
					r.NotEqual(-1, i) // make sure we found the event id

					r.Len(edge.Node.Runs, 1)
					r.Equal(edge.Node.Runs[0].Function.Name, fnName)

					// make sure the runID is unique
					runID := edge.Node.Runs[0].ID
					r.Equal(-1, slices.Index(runIDs, runID))
					runIDs = append(runIDs, runID)
				}
			}, time.Second*10, time.Second*1)
		})

		t.Run("3 events batch to 1 function", func(t *testing.T) {
			r := require.New(t)
			c := client.New(t)
			inngestClient, server, registerFuncs := NewSDKHandler(t, "fn-test")
			defer server.Close()

			eventName := randomSuffix("evt-batch")
			fnName := "fn-batch"

			_, err := inngestgo.CreateFunction(
				inngestClient,
				inngestgo.FunctionOpts{ID: fnName, BatchEvents: &inngestgo.ConfigBatchEvents{MaxSize: 5, Timeout: 5 * time.Second}},
				inngestgo.EventTrigger(eventName, nil),
				func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
					return nil, nil
				},
			)
			r.NoError(err)
			registerFuncs()

			ctx := context.Background()

			numEvents := 3
			eventIDs := make([]string, numEvents)
			for i := 0; i < numEvents; i++ {
				eventID, err := inngestClient.Send(ctx, &event.Event{Name: eventName})
				r.NoError(err)
				eventIDs[i] = eventID
			}

			// We should see 3 events, all associated to the same run
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

				r.Equal(3, res.TotalCount)

				var runID ulid.ULID
				for _, edge := range res.Edges {
					i := slices.Index(eventIDs, edge.Node.ID.String())
					r.NotEqual(-1, i) // make sure we found the event id

					r.Len(edge.Node.Runs, 1)
					r.Equal(edge.Node.Runs[0].Function.Name, fnName)
					// save the first runID we see and then check that the rest of the events returned the same run
					if runID.IsZero() {
						runID = edge.Node.Runs[0].ID
					} else {
						r.Equal(runID, edge.Node.Runs[0].ID)
					}
				}
			}, time.Second*10, time.Second*1)
		})
	})

	t.Run("internal events", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()
		c := client.New(t)

		inngestClient, server, registerFuncs := NewSDKHandler(t, "internal-events")
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

		// Send event and wait for it to show in the API.
		_, err = inngestClient.Send(ctx, event.Event{Name: eventName})
		r.NoError(err)

		// For extracting function_id from Raw
		type Raw struct {
			Data struct {
				FunctionID string `json:"function_id"`
			} `json:"data"`
		}
		expectedFunctionId := "internal-events-fn"
		r.EventuallyWithT(func(t *assert.CollectT) {
			// Explicitly include internal events.
			res, err := c.GetEvents(ctx, client.GetEventsOpts{
				PageSize: 50,
				Filter: models.EventsFilter{
					EventNames:            []string{eventName, "inngest/function.finished"},
					From:                  time.Now().Add(-time.Minute),
					IncludeInternalEvents: true,
				},
			})
			r.NoError(err)

			// Theoretically we want exactly equals to 2 but don't always get it due to poor test isolation
			r.GreaterOrEqual(res.TotalCount, 2)
			// Also due to poor test isolation, if there are more events in the last minute than a single page size
			// this might fail. Possibly remove this
			r.Equal(len(res.Edges), res.TotalCount)

			// Guarantee that at least one of the events we saw was from eventName and another from inngest/function.finished
			r.True(slices.ContainsFunc(res.Edges, func(e *models.EventsEdge) bool {
				return e.Node.Name == eventName
			}))
			r.True(slices.ContainsFunc(res.Edges, func(e *models.EventsEdge) bool {
				if e.Node.Name == "inngest/function.finished" {
					var raw Raw
					err := json.Unmarshal([]byte(e.Node.Raw), &raw)
					r.NoError(err)

					return raw.Data.FunctionID == expectedFunctionId
				}
				return false
			}))

		}, time.Second*10, time.Second*1)

		// Explicitly exclude internal events.
		res, err := c.GetEvents(ctx, client.GetEventsOpts{
			PageSize: 10,
			Filter: models.EventsFilter{
				EventNames:            []string{eventName, "inngest/function.finished"},
				From:                  time.Now().Add(-time.Minute),
				IncludeInternalEvents: false,
			},
		})
		r.NoError(err)
		r.Equal(1, res.TotalCount)
		r.Len(res.Edges, 1)
		// The inngest/function.finished event is omitted even though it's in the EventNames filter
		r.True(slices.ContainsFunc(res.Edges, func(e *models.EventsEdge) bool {
			return e.Node.Name == eventName
		}))

		// Implicitly exclude internal events (it has the same effect as
		// explicitly excluding them).
		res, err = c.GetEvents(ctx, client.GetEventsOpts{
			PageSize: 10,
			Filter: models.EventsFilter{
				EventNames: []string{eventName, "inngest/function.finished"},
				From:       time.Now().Add(-time.Minute),
			},
		})
		r.NoError(err)
		r.Equal(1, res.TotalCount)
		r.Len(res.Edges, 1)
		// The inngest/function.finished event is omitted even though it's in the EventNames filter
		r.True(slices.ContainsFunc(res.Edges, func(e *models.EventsEdge) bool {
			return e.Node.Name == eventName
		}))

	})

}
