package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
)

func TestCancelFunctionViaAPI(t *testing.T) {
	evt := inngestgo.Event{
		Name: "tests/cancel.test",
		Data: map[string]any{
			"request_id": "123",
		},
		User: map[string]interface{}{},
	}

	hashes := map[string]string{
		"Sleep 10s": "af731ad68b75abe9679cc9fc324a4ad3cd8075a2",
	}

	// This uses the ame
	fnID := "test-suite-cancel-test"
	retries := 10
	abstract := Test{
		Name: "Cancel via API test",
		Description: `
			This test asserts that the V0 cancellation API works as expected, cancelling functions.
		`,
		Function: inngest.Function{
			Name: "Cancel test",
			Slug: fnID,
			Triggers: []inngest.Trigger{
				{
					EventTrigger: &inngest.EventTrigger{
						Event: "tests/cancel.test",
					},
				},
			},
			Steps: []inngest.Step{
				{
					ID:      "step",
					Name:    "step",
					URI:     stepURL(fnID, "step"),
					Retries: &retries,
				},
			},
			Cancel: []inngest.Cancel{
				{
					Event:   "cancel/please",
					Timeout: strptr("1h"),
					If:      strptr("async.data.request_id == event.data.request_id"),
				},
			},
		},
		EventTrigger: evt,
		Timeout:      20 * time.Second,
	}

	t.Run("Cancel API cancels the function", func(t *testing.T) {
		copied := abstract
		test := &copied
		test.SetAssertions(
			// All executor requests should have this event.
			test.SetRequestEvent(evt),
			// And the executor should start its requests with this context.
			test.SetRequestContext(SDKCtx{
				FnID:   inngest.DeterministicUUID(abstract.Function).String(),
				StepID: "step",
				Stack: driver.FunctionStack{
					Current: 0,
				},
			}),
			test.SendTrigger(),

			// Execute the step again, get a wait
			test.ExpectRequest("Wait step run", "step", time.Second),
			test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
				Op:   enums.OpcodeSleep,
				ID:   hashes["Sleep 10s"],
				Name: "10s",
			}}),

			test.After(time.Second),

			// Run a function to hit the cancellation API
			test.Func(func() error {
				fmt.Println(" ==> Hitting cancel API")
				if test.lastEventID == nil {
					return fmt.Errorf("no event ID found")
				}

				// Get run ID from event
				route := fmt.Sprintf("%s/v0/events/%s/runs", apiURL.String(), *test.lastEventID)
				resp, err := http.Get(route)
				if err != nil {
					return err
				}
				defer resp.Body.Close()

				byt, _ := io.ReadAll(resp.Body)

				ids := []ulid.ULID{}
				if err := json.Unmarshal(byt, &ids); err != nil {
					return fmt.Errorf("cannot get event runs: %w\n\n%s", err, byt)
				}

				for _, id := range ids {
					// Cancel run
					route = fmt.Sprintf("%s/v0/runs/%s", apiURL.String(), id)
					req, _ := http.NewRequest(http.MethodDelete, route, nil)
					resp, err = http.DefaultClient.Do(req)
					if err != nil {
						return fmt.Errorf("error making delete request: %w", err)
					}
					defer resp.Body.Close()
					if resp.StatusCode != 204 {
						return fmt.Errorf("unexpected cancel status code: %d", resp.StatusCode)
					}
				}
				return nil
			}),
		)

		run(t, test)
	})
}
