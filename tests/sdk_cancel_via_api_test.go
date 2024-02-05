package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngestgo"
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
		"Sleep 10s": "c3ca5f787365eae0dea86250e27d476406956478",
	}

	// This uses the ame
	fnID := "test-suite-cancel-test"
	abstract := Test{
		ID:   fnID,
		Name: "Cancel via API test",
		Description: `
			This test asserts that the V0 cancellation API works as expected, cancelling functions.
		`,
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
			test.SetRequestContext(driver.SDKRequestContext{
				StepID: "step",
				Stack: &driver.FunctionStack{
					Current: 0,
				},
			}),
			test.SendTrigger(),

			// Execute the step again, get a wait
			test.ExpectRequest("Wait step run", "step", time.Second),
			test.ExpectGeneratorResponse([]state.GeneratorOpcode{{
				Op:          enums.OpcodeSleep,
				ID:          hashes["Sleep 10s"],
				DisplayName: inngestgo.StrPtr("sleep"),
				Data:        json.RawMessage("null"),
				Name:        "10s",
			}}),

			test.After(time.Second),

			// Run a function to hit the cancellation API
			test.Func(func() error {
				fmt.Println(" ==> Hitting cancel API")
				if test.lastEventID == nil {
					return fmt.Errorf("no event ID found")
				}

				route := fmt.Sprintf("%s/v1/runs/%s", apiURL.String(), test.requestCtx.RunID)
				req, _ := http.NewRequest(http.MethodDelete, route, nil)
				req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", eventKey))
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return fmt.Errorf("error making delete request: %w", err)
				}

				defer resp.Body.Close()
				if resp.StatusCode > 299 {
					return fmt.Errorf("unexpected cancel status code: %d", resp.StatusCode)
				}

				return nil
			}),
		)

		run(t, test)
	})
}
