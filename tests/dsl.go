package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

type Test struct {
	// ID is the ID of the test.
	ID   string
	Name string
	// Description is a description of the test.
	Description string
	// The event to send when testing this function.
	EventTrigger inngestgo.Event
	// Timeout is how long the tests take to run.
	Timeout time.Duration

	//
	// runner internals
	//

	// requests is a channel for incoming executor requests to be checked
	requests chan http.Request
	// responses is a channel for sdk responses to be checked
	responses chan http.Response
	// chain represents a chain of functions to run, some of which must assert the
	// requests and responses sent between the executor and SDK.
	chain []func()
	// test allows failing the unit test
	test *testing.T

	// requestEvent stores the event that must be sent within executor requests
	requestEvent inngestgo.Event
	// requestCtx stores the "ctx" field that must be present within executor requests
	requestCtx driver.SDKRequestContext
	// requestSteps stores the "steps" field that must be present within executor requests
	requestSteps map[string]any
	// lastResponse stores the last response time
	lastResponse time.Time

	lastEventID *string

	localURL *url.URL
}

func (t *Test) SetAssertions(items ...func()) {
	t.chain = items
	// Assert requestSteps is non-nil
	t.requestSteps = map[string]any{}
}

// SendTrigger sends the triggering event, kicking off the function run.
func (t *Test) SendTrigger() func() {
	return t.Send(t.EventTrigger)
}

func (t *Test) Func(f func() error) func() {
	return func() {
		t.test.Helper()
		err := f()
		require.NoError(t.test, err)
	}
}

func (t *Test) Send(evt inngestgo.Event) func() {
	return func() {
		url := eventURL.String()
		client, err := inngestgo.NewClient(inngestgo.ClientOpts{
			AppID:    "test",
			EventKey: &eventKey,
			EventURL: &url,
		})
		require.NoError(t.test, err)
		id, err := client.Send(context.Background(), evt)
		require.NoError(t.test, err)
		t.lastEventID = &id
	}
}

// SetRequestEvent
func (t *Test) SetRequestEvent(event inngestgo.Event) func() {
	return func() {
		t.requestEvent = event
		// Also reset context, as this happens at the start of tests.
		t.requestCtx = driver.SDKRequestContext{
			StepID: "step",
			Stack: &driver.FunctionStack{
				Current: 0,
			},
		}
		if t.requestCtx.Stack.Stack == nil {
			// Normalize to a non-nil slice
			t.requestCtx.Stack.Stack = []string{}
		}
	}
}

func (t *Test) SetRequestContext(ctx driver.SDKRequestContext) func() {
	return func() {
		t.requestCtx = ctx
		if t.requestCtx.Stack.Stack == nil {
			// Normalize to a non-nil slice
			t.requestCtx.Stack.Stack = []string{}
		}
	}
}

// SetRequestEvent
func (t *Test) SetRequestSteps(steps map[string]any) func() {
	return func() {
		t.requestSteps = steps
	}
}

// AddRequestStack pushes items onto the stack and updates current
func (t *Test) AddRequestStack(s driver.FunctionStack) func() {
	return func() {
		if t.requestCtx.Stack.Stack == nil {
			t.requestCtx.Stack.Stack = []string{}
		}
		t.requestCtx.Stack.Stack = append(t.requestCtx.Stack.Stack, s.Stack...)
		t.requestCtx.Stack.Current = s.Current
	}
}

// AddRequestSteps pushes items onto the stack and updates current
func (t *Test) AddRequestSteps(s map[string]any) func() {
	return func() {
		if t.requestSteps == nil {
			t.requestSteps = map[string]any{}
		}
		for k, v := range s {
			t.requestSteps[k] = v
		}
	}
}

func (t *Test) Printf(name string, args ...any) func() {
	return func() {
		fmt.Printf("\n\n===> "+name+"\n\n\n", args...)
	}
}

func (t *Test) ExpectRequest(name string, queryStepID string, timeout time.Duration, modifiers ...func(r *driver.SDKRequestContext)) func() {
	return func() {
		select {
		case r := <-t.requests:
			id := r.URL.Query().Get("stepId")
			require.EqualValues(t.test, queryStepID, id)

			byt, err := io.ReadAll(r.Body)
			require.NoError(t.test, err)
			require.NotEmpty(t.test, byt)

			// Asser that the request is well formatted.
			er := &ExecutorRequest{}
			err = json.Unmarshal(byt, er)
			require.NoError(t.test, err)

			require.NotNil(t.test, er.Ctx.Stack.Stack)

			require.NotZero(t.test, er.Event.Timestamp)
			// Zero out the TS and ID
			ts := er.Event.Timestamp
			evtID := er.Event.ID
			er.Event.Timestamp = 0
			er.Event.ID = nil
			require.EqualValues(t.test, t.requestEvent, er.Event, "Request event is incorrect")
			er.Event.Timestamp = ts
			er.Event.ID = evtID
			er.Ctx.MaxAttempts = 0

			for _, m := range modifiers {
				m(&t.requestCtx)
			}

			// Unset the run ID so that our unique run ID doesn't cause issues.
			t.requestCtx.RunID = er.Ctx.RunID
			t.requestCtx.FunctionID = uuid.UUID{}
			er.Ctx.FunctionID = uuid.UUID{}
			// Unset the queue ref, too
			t.requestCtx.QueueItemRef = ""
			er.Ctx.QueueItemRef = ""
			copyRequestContextFieldIfPresent(&t.requestCtx, er.Ctx, "RequestID")
			copyRequestContextFieldIfPresent(&t.requestCtx, er.Ctx, "JobID")
			copyRequestContextFieldIfPresent(&t.requestCtx, er.Ctx, "GenerationID")

			// For each error, remove the stack from our tests.
			for _, v := range er.Steps {
				data, ok := v.(map[string]any)
				if !ok {
					continue
				}
				if err, ok := data["error"].(map[string]any); ok {
					delete(err, "stack")
				}
			}

			require.EqualValues(t.test, t.requestCtx, er.Ctx, "Request ctx is incorrect")
			require.EqualValues(t.test, t.requestSteps, er.Steps, "Request steps are incorrect")
			// XXX: Assert req v

		case <-time.After(timeout):
			require.Failf(t.test, "Expected executor request but timed out", name)
		}
	}
}

func copyRequestContextFieldIfPresent(expected *driver.SDKRequestContext, actual driver.SDKRequestContext, name string) {
	expectedValue := reflect.ValueOf(expected).Elem().FieldByName(name)
	actualValue := reflect.ValueOf(actual).FieldByName(name)
	if !expectedValue.IsValid() || !actualValue.IsValid() || !expectedValue.CanSet() {
		return
	}
	if expectedValue.Type() != actualValue.Type() {
		return
	}
	expectedValue.Set(actualValue)
}

func (t *Test) ExpectResponse(status int, body []byte) func() {
	return t.ExpectResponseFunc(status, func(b []byte) error {
		require.Equal(t.test, string(body), string(b))
		return nil
	})
}

func (t *Test) ExpectResponseFunc(status int, f func(b []byte) error) func() {
	return func() {
		select {
		case r := <-t.responses:
			t.lastResponse = time.Now()

			rdr := r.Body

			byt, err := io.ReadAll(rdr)
			require.NoError(t.test, err)

			require.Equal(t.test, status, r.StatusCode)

			err = f(byt)
			require.NoError(t.test, err)
		case <-time.After(time.Second):
			require.Fail(t.test, "Expected SDK generator response but timed out")
		}
	}
}

func (t *Test) ExpectJSONResponse(status int, expected any) func() {
	return t.ExpectResponseFunc(status, func(byt []byte) error {
		var actual any
		err := json.Unmarshal(byt, &actual)
		require.NoError(t.test, err)
		require.EqualValues(t.test, expected, actual)
		return nil
	})
}

// opcodeFieldsEqual reports whether two opcodes match on the fields these tests
// assert. The SDK also populates fields we deliberately ignore because they
// drift across minor versions and no test depends on them: Timing (which
// carries non-deterministic timestamps), Userland, and an empty Opts map.
//
// Keep the field list in sync with the per-field assertions in
// ExpectGeneratorResponse.
func opcodeFieldsEqual(expected, actual state.GeneratorOpcode) bool {
	if expected.Op != actual.Op ||
		expected.ID != actual.ID ||
		expected.Name != actual.Name {
		return false
	}
	if !reflect.DeepEqual(expected.DisplayName, actual.DisplayName) ||
		!reflect.DeepEqual(expected.Error, actual.Error) ||
		!bytes.Equal(expected.Data, actual.Data) {
		return false
	}
	// Only compare Opts when the expected opcode specifies it (e.g.
	// waitForEvent's match expression); otherwise the SDK's empty Opts map is
	// ignored.
	if expected.Opts != nil && !reflect.DeepEqual(expected.Opts, actual.Opts) {
		return false
	}
	return true
}

func (t *Test) ExpectGeneratorResponse(expected []state.GeneratorOpcode) func() {
	return func() {
		select {
		case r := <-t.responses:
			t.lastResponse = time.Now()
			byt, err := io.ReadAll(r.Body)
			require.NoError(t.test, err)

			actual := []state.GeneratorOpcode{}
			err = json.Unmarshal(byt, &actual)
			require.NoError(t.test, err)

			require.Len(t.test, actual, len(expected))
			for i := range actual {
				exp, act := expected[i], actual[i]

				// For step errors/failures we ignore the Stack field: it
				// contains absolute paths, which vary by machine.
				if (act.Op == enums.OpcodeStepError || act.Op == enums.OpcodeStepFailed) && act.Error != nil {
					act.Error.Stack = "[proxy-redact]"
					if exp.Error != nil {
						exp.Error.Stack = "[proxy-redact]"
					}
				}

				// Assert only the fields these tests care about. The SDK also
				// populates Timing, Userland, and an empty Opts map that drift
				// across minor versions and are intentionally not checked.
				require.Equal(t.test, exp.Op, act.Op, "opcode Op mismatch")
				require.Equal(t.test, exp.ID, act.ID, "opcode ID mismatch")
				require.Equal(t.test, exp.Name, act.Name, "opcode Name mismatch")
				require.Equal(t.test, exp.DisplayName, act.DisplayName, "opcode DisplayName mismatch")
				require.Equal(t.test, exp.Error, act.Error, "opcode Error mismatch")
				require.Equal(t.test, exp.Data, act.Data, "opcode Data mismatch")
				// Only assert Opts when the test specifies it (e.g. waitForEvent).
				if exp.Opts != nil {
					require.Equal(t.test, exp.Opts, act.Opts, "opcode Opts mismatch")
				}
			}
		case <-time.After(time.Second):
			require.Fail(t.test, "Expected SDK generator response but timed out")
		}
	}
}

// ExpectParallelStepRuns is used to assert that step.run is called with the given number of steps
// in parallel.  This can be used for a single stpe or for multiple steps.
func (t *Test) ExpectParallelStepRuns(stepFunc func() []state.GeneratorOpcode, timeout time.Duration) func() {
	return func() {
		c := time.After(timeout)

		steps := stepFunc()

		for i := 0; i < len(steps); i++ {

			// Expect a request
			select {
			case <-t.requests:
				// TODO: expect a request for this opcode
				// Right now, let this pass through.
			case <-c:
				require.Fail(t.test, "Expected steps but timed out")
			}

			// And expect a response.
			select {
			case r := <-t.responses:
				t.lastResponse = time.Now()
				byt, err := io.ReadAll(r.Body)
				require.NoError(t.test, err)

				op := []state.GeneratorOpcode{}
				err = json.Unmarshal(byt, &op)
				require.NoError(t.test, err)

				if len(op) == 0 {
					// Equal to opcode none.
					op = append(op, state.GeneratorOpcode{})
				}

				found := false
				for _, s := range steps {
					if opcodeFieldsEqual(s, op[0]) {
						if s.Op == enums.OpcodeNone {
							// Do nothing.
							found = true
							break
						}

						// Update stack
						t.AddRequestStack(driver.FunctionStack{
							Stack:   []string{s.ID},
							Current: t.requestCtx.Stack.Current + 1,
						})()

						// wtf plz refactor
						var data interface{}
						switch op[0].Data[0] {
						case '"':
							data = ""
							err = json.Unmarshal(op[0].Data, &data)
							require.NoError(t.test, err)
						case '[':
							data = []map[string]any{}
							err = json.Unmarshal(op[0].Data, &data)
							require.NoError(t.test, err)
						case '{':
							data = map[string]any{}
							err = json.Unmarshal(op[0].Data, &data)
							require.NoError(t.test, err)
						}

						t.AddRequestSteps(map[string]any{
							s.ID: data,
						})()
						found = true
						break
					}
				}

				if !found {
					had, _ := json.Marshal(steps)
					require.Fail(
						t.test,
						"Found unexpected step output waiting for steps",
						"Got %s\nHad %#v",
						string(byt),
						string(had),
					)
				}
			case <-c:
				require.Fail(t.test, "Expected steps but timed out")
			}
		}
	}
}

func (t *Test) After(d time.Duration) func() {
	return func() {
		<-time.After(d)
	}
}

type ExecutorRequest struct {
	Event   inngestgo.Event          `json:"event"`
	Steps   map[string]any           `json:"steps"`
	Ctx     driver.SDKRequestContext `json:"ctx"`
	Version int                      `json:"version"`
}
