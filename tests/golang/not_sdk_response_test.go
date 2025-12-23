package golang

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
)

func TestNotSDKResponse(t *testing.T) {
	t.Setenv("INNGEST_DEV", "1")

	sync := func(t *testing.T, u string) {
		r := require.New(t)
		r.EventuallyWithT(func(t *assert.CollectT) {
			a := assert.New(t)
			req, err := http.NewRequest(http.MethodPut, u, nil)
			a.NoError(err)
			resp, err := http.DefaultClient.Do(req)
			a.NoError(err)
			a.Equal(200, resp.StatusCode)
			_ = resp.Body.Close()
		}, 5*time.Second, 100*time.Millisecond)
	}

	statusCodes := []int{
		http.StatusOK,
		http.StatusPartialContent,
	}

	for _, statusCode := range statusCodes {
		t.Run(fmt.Sprintf("%d status code", statusCode), func(t *testing.T) {
			r := require.New(t)
			ctx := context.Background()
			c := client.New(t)

			// Start proxy which simulates a 200 HTML response.
			var count int32
			proxy := NewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&count, 1)
				w.WriteHeader(statusCode)
				_, _ = w.Write([]byte("<html>hi</html>"))
			}))
			defer proxy.Close()
			proxyURL, err := url.Parse(proxy.URL())
			r.NoError(err)

			// Create and sync app.
			ic, err := inngestgo.NewClient(
				inngestgo.ClientOpts{
					AppID:       randomSuffix("app"),
					Dev:         inngestgo.BoolPtr(true),
					RegisterURL: inngestgo.StrPtr(fmt.Sprintf("%s/fn/register", DEV_URL)),
					URL:         proxyURL,
				},
			)
			r.NoError(err)
			eventName := randomSuffix("event")
			_, err = inngestgo.CreateFunction(
				ic,
				inngestgo.FunctionOpts{
					ID:      "fn",
					Retries: inngestgo.IntPtr(0),
				},
				inngestgo.EventTrigger(eventName, nil),
				func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
					return nil, nil
				},
			)
			r.NoError(err)
			server := NewHTTPServer(ic.Serve())
			defer server.Close()
			sync(t, server.LocalURL())

			// Trigger function.
			eventID, err := ic.Send(ctx, inngestgo.Event{Name: eventName})
			r.NoError(err)

			// Wait for 2 attempts.
			r.EventuallyWithT(func(t *assert.CollectT) {
				a := assert.New(t)
				a.Equal(int32(1), count)
			}, time.Minute, 100*time.Millisecond)

			// Assert status and output.
			var run client.Run
			r.EventuallyWithT(func(t *assert.CollectT) {
				runs, err := c.RunsByEventID(ctx, eventID)
				require.NoError(t, err)
				require.Len(t, runs, 1)
				run = c.WaitForRunStatus(ctx, t, "FAILED", &runs[0].ID)
			}, 20*time.Second, time.Second)

			if statusCode == http.StatusOK {
				// Function output includes the HTML response.
				r.Equal("<html>hi</html>", run.Output)
			} else {
				// Step output is empty. We should probably change this in the
				// future.
				r.Equal("", run.Output)
			}
		})
	}
}
