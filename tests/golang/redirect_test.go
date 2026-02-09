package golang

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution/exechttp"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

func TestRedirect(t *testing.T) {
	// Ensure that we follow redirects when syncing apps and executing functions

	ctx := context.Background()
	r := require.New(t)

	_ = os.Setenv("INNGEST_DEV", DEV_URL)

	inngestClient, server, _ := NewSDKHandler(t, "my-app")
	defer server.Close()

	// Create a server that 303 redirects to the SDK server
	redirectCounter := 0
	redirectServer := NewHTTPServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			redirectCounter++
			http.Redirect(w, r, server.URL(), http.StatusTemporaryRedirect)
		}),
	)
	defer redirectServer.Close()

	// Tell the SDK that it should use the redirect server URL. This is
	// necessary to ensure that the SDK syncs itself with the redirect server
	// URL
	u, err := url.Parse(redirectServer.URL())
	r.NoError(err)
	inngestClient.SetURL(u)

	var runID string
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:      "my-fn",
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("my-event", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			runID = input.InputCtx.RunID
			return nil, nil
		},
	)
	r.NoError(err)

	req, err := http.NewRequest(http.MethodPut, redirectServer.URL(), nil)
	r.NoError(err)
	resp, err := exechttp.Client(exechttp.SecureDialerOpts{AllowPrivate: true}).Do(req)
	r.NoError(err)
	r.Equal(200, resp.StatusCode)

	// Redirected during sync
	r.Equal(1, redirectCounter)

	evt := inngestgo.Event{
		Name: "my-event",
		Data: map[string]any{"foo": 1},
	}
	_, err = inngestClient.Send(ctx, evt)
	require.NoError(t, err)

	r.Eventually(func() bool {
		// Redirected during execution
		if redirectCounter != 2 {
			return false
		}

		// Function ran
		return runID != ""
	}, 10*time.Second, 50*time.Millisecond)
}
