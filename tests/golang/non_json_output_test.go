package golang

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
)

func TestNonJSONOutput(t *testing.T) {
	t.Run("HTML response body", func(t *testing.T) {
		r := require.New(t)
		ctx := context.Background()

		c := client.New(t)

		// Start proxy which simulates a 504 response from the user's gateway
		proxy := NewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGatewayTimeout)
			_, _ = w.Write([]byte("<html>502 Bad Gateway</html>"))
		}))
		defer proxy.Close()
		proxyURL, err := url.Parse(proxy.URL())
		r.NoError(err)

		// Suffix with random string to avoid fanout to other tests
		eventName := fmt.Sprintf("my-event-%s", ulid.MustNew(ulid.Now(), nil).String())

		// Start app
		_ = os.Setenv("INNGEST_DEV", DEV_URL)
		ekey := "test"
		inngestClient, err := inngestgo.NewClient(
			inngestgo.ClientOpts{
				AppID:       "my-app",
				Logger:      slog.Default(),
				RegisterURL: inngestgo.StrPtr(fmt.Sprintf("%s/fn/register", DEV_URL)),
				URL:         proxyURL,
				EventKey:    &ekey,
			},
		)
		r.NoError(err)
		_, err = inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{
				ID:      "my-fn",
				Retries: inngestgo.IntPtr(0),
			},
			inngestgo.EventTrigger(eventName, nil),
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				return nil, nil
			},
		)
		r.NoError(err)
		server := NewHTTPServer(inngestClient.Serve())
		defer server.Close()

		// Sync
		r.EventuallyWithT(func(t *assert.CollectT) {
			a := assert.New(t)
			req, err := http.NewRequest(http.MethodPut, server.LocalURL(), nil)
			a.NoError(err)
			resp, err := http.DefaultClient.Do(req)
			a.NoError(err)
			a.Equal(200, resp.StatusCode)
			_ = resp.Body.Close()
		}, 5*time.Second, 100*time.Millisecond)

		eventID, err := inngestClient.Send(
			ctx,
			inngestgo.Event{Data: map[string]any{"foo": 1}, Name: eventName},
		)
		r.NoError(err)

		var runID string
		r.Eventually(func() bool {
			runs, err := c.RunsByEventID(ctx, eventID)
			if err != nil {
				return false
			}
			if len(runs) != 1 {
				return false
			}
			runID = runs[0].ID
			return true
		}, 5*time.Second, 100*time.Millisecond)

		run := c.WaitForRunStatus(ctx, t, "FAILED", runID)
		r.Equal("<html>502 Bad Gateway</html>", run.Output)
	})
}
