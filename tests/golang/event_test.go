package golang

import (
	"context"
	"encoding/json"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvent(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()
		c := client.New(t)

		// Send 1 event and wait for it show in the API.
		inngestClient, err := inngestgo.NewClient(inngestgo.ClientOpts{
			AppID: "app",
			Dev:   toPtr(true),
		})
		r.NoError(err)
		eventName := randomSuffix("foo")

		eventID, err := inngestClient.Send(ctx, inngestgo.Event{
			Name: eventName,
			Data: map[string]any{"msg": "hi"},
		})
		r.NoError(err)

		var evt *models.EventV2
		r.EventuallyWithT(func(t *assert.CollectT) {
			var err error
			evt, err = c.GetEvent(ctx, ulid.MustParse(eventID))
			r.NoError(err)
		}, time.Second*10, time.Second)

		raw, err := json.Marshal(map[string]any{
			"data": map[string]any{"msg": "hi"},
			"id":   eventID,
			"name": "foo",
			"ts":   evt.OccurredAt.UnixMilli(),
			"v":    nil,
		})
		r.NoError(err)
		r.Equal(evt.Name, "foo")
		r.NotZero(evt.OccurredAt)
		r.Equal(evt.Raw, string(raw))
		r.NotZero(evt.ReceivedAt)
	})

	//t.Run("not found", func(t *testing.T) {
	//	r := require.New(t)
	//	c := clientutil.NewClient(t).Setup()
	//	defer c.Teardown()
	//
	//	_, err := c.GetEvent(ulid.MustNew(ulid.Now(), rand.Reader))
	//	r.Error(err)
	//})
	//
	//t.Run("runs", func(t *testing.T) {
	//	r := require.New(t)
	//	c := clientutil.NewClient(t).Setup()
	//	defer c.Teardown()
	//	ic, server, sync := test.NewSDKHandler(t, c)
	//	defer server.Close()
	//
	//	eventName := "evt"
	//	_, err := inngestgo.CreateFunction(
	//		ic,
	//		inngestgo.FunctionOpts{ID: "fn"},
	//		inngestgo.EventTrigger(eventName, nil),
	//		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
	//			return nil, nil
	//		},
	//	)
	//	r.NoError(err)
	//	sync()
	//
	//	eventID := ulid.MustParse(c.Send(&event.Event{Name: eventName})[0])
	//
	//	var evt *gqlmodels.EventV2
	//	r.EventuallyWithT(func(t *assert.CollectT) {
	//		a := assert.New(t)
	//		evt, err = c.GetEvent(eventID)
	//		if !a.NoError(err) {
	//			return
	//		}
	//		a.Len(evt.Runs, 1)
	//	}, time.Second*10, time.Second*1)
	//	r.Equal("fn", evt.Runs[0].Function.Name)
	//	r.Equal(gqlmodels.FunctionRunStatusCompleted, evt.Runs[0].Status)
	//})
}
