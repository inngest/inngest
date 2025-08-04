package golang

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
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
			"name": eventName,
			"ts":   evt.OccurredAt.UnixMilli(),
			"v":    nil,
		})
		r.NoError(err)
		r.Equal(evt.Name, eventName)
		r.NotZero(evt.OccurredAt)
		r.Equal(evt.Raw, string(raw))
		r.NotZero(evt.ReceivedAt)
	})

	t.Run("not found", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()
		c := client.New(t)

		_, err := c.GetEvent(ctx, ulid.MustNew(ulid.Now(), rand.Reader))
		r.Error(err)
	})

	t.Run("runs", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()
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

		eventID, err := inngestClient.Send(ctx, &event.Event{Name: eventName})

		var evt *models.EventV2
		r.EventuallyWithT(func(t *assert.CollectT) {
			evt, err = c.GetEvent(ctx, ulid.MustParse(eventID))
			r.NoError(err)
			r.NotNil(evt) // TODO Delete once runs are completed
			//r.Len(evt.Runs, 1)
		}, time.Second*10, time.Second*1)
		//r.Equal("fn", evt.Runs[0].Function.Name)
		//r.Equal(models.FunctionRunStatusCompleted, evt.Runs[0].Status)
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
				res, err := c.GetEvent(ctx, ulid.MustParse(eventID))
				r.NoError(err)

				raw = res.Raw
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
	})
}

func TestEvent_UnsupportedContentTypes(t *testing.T) {
	t.Parallel()

	t.Run("multipart/form-data", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		ctx := context.Background()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		r.NoError(writer.WriteField("name", "my-event"))
		r.NoError(writer.Close())

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			"http://localhost:8288/e/test",
			body,
		)
		r.NoError(err)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		r.NoError(err)
		r.Equal(http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("x-www-form-urlencoded", func(t *testing.T) {
		t.Parallel()
		r := require.New(t)
		ctx := context.Background()

		formData := url.Values{}
		formData.Set("name", "my-event")
		body := strings.NewReader(formData.Encode())

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			"http://localhost:8288/e/test",
			body,
		)
		r.NoError(err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		r.NoError(err)
		r.Equal(http.StatusBadRequest, resp.StatusCode)
	})
}
