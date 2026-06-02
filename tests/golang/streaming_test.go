package golang

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngest/tests/client"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreaming(t *testing.T) {
	t.Run("connection reset", func(t *testing.T) {
		ctx := context.Background()
		r := require.New(t)
		c := client.New(t)
		inngestClient, err := inngestgo.NewClient(inngestgo.ClientOpts{
			AppID:    "my-app",
			EventKey: toPtr("test"),
			EventURL: toPtr("http://localhost:8288"),
		})
		r.NoError(err)

		var appURL *string
		var runID *string

		// Create a fake SDK that replicates a connection reset during streaming.
		fakeSDK := NewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Execution
			if r.Method == http.MethodPost {
				byt, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "failed to read body", http.StatusInternalServerError)
					return
				}

				request := driver.SDKRequest{}
				err = json.Unmarshal(byt, &request)
				if err != nil {
					http.Error(w, "failed to unmarshal body", http.StatusInternalServerError)
					return
				}
				runID = toPtr(request.Context.RunID.String())

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)

				flusher, ok := w.(http.Flusher)
				if !ok {
					http.Error(w, "not a flusher", http.StatusInternalServerError)
					return
				}

				_, _ = w.Write([]byte(" "))
				flusher.Flush()
				<-time.After(1 * time.Second)

				// Reset connection using TCP reset.
				if hijacker, ok := w.(http.Hijacker); ok {
					conn, _, err := hijacker.Hijack()
					if err != nil {
						http.Error(w, "failed to reset connection", http.StatusInternalServerError)
						return
					}
					if tcpConn, ok := conn.(*net.TCPConn); ok {
						// Send RST instead of FIN.
						_ = tcpConn.SetLinger(0)
					}
					conn.Close()
				}
			}

			w.WriteHeader(http.StatusMethodNotAllowed)
		}))
		defer fakeSDK.Close()
		appURL = toPtr(fakeSDK.URL())

		// Simulate an SDK syncing itself.
		sync := func() error {
			byt, err := json.Marshal(sdk.RegisterRequest{
				AppName: "my-app",
				Functions: []sdk.SDKFunction{{
					Name: "my-fn",
					Triggers: []inngest.Trigger{
						{EventTrigger: &inngest.EventTrigger{Event: "my-event"}},
					},
					Steps: map[string]sdk.SDKStep{
						"step-1": {
							Name: "Step 1",
							Retries: &sdk.StepRetries{
								Attempts: 0,
							},
							Runtime: map[string]any{
								"url": fmt.Sprintf("%s/api/inngest?fnId=another-fn", *appURL),
							},
						},
					},
				}},
				URL: *appURL,
			})
			if err != nil {
				return err
			}

			req, err := http.NewRequest(
				http.MethodPost,
				"http://localhost:8288/fn/register",
				bytes.NewReader(byt),
			)
			if err != nil {
				return err
			}
			req.Header.Set("content-type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			if resp.StatusCode != http.StatusOK {
				// Print the body to make test failures easier to debug.
				byt, _ := io.ReadAll(resp.Body)
				fmt.Println(string(byt))
				resp.Body.Close()

				return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			return nil
		}
		r.NoError(sync())

		// _, err := inngestgo.Send(ctx, inngestgo.Event{
		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "my-event",
			Data: map[string]any{"foo": "bar"},
		})
		r.NoError(err)

		r.EventuallyWithT(func(ct *assert.CollectT) {
			a := assert.New(ct)
			a.NotNil(runID)
			if runID == nil {
				return
			}

			run, err := c.RunTraces(ctx, *runID, false)
			a.NoError(err)
			if err != nil {
				return
			}
			a.Equal("FAILED", run.Status)

			a.NotNil(run.Trace)
			if run.Trace == nil {
				return
			}
			a.NotNil(run.Trace.OutputID)
			if run.Trace.OutputID == nil {
				return
			}
			runOutput := c.RunSpanOutput(ctx, *run.Trace.OutputID)
			a.NotNil(runOutput)
			if runOutput == nil {
				return
			}
			a.NotNil(runOutput.Error)
			if runOutput.Error == nil {
				return
			}
			a.NotNil(runOutput.Error.Stack)
			if runOutput.Error.Stack == nil {
				return
			}
			a.Contains(*runOutput.Error.Stack, "Your server reset the connection while we were reading the reply")
		}, 10*time.Second, time.Second)
	})
}

func toPtr[T any](v T) *T {
	return &v
}
