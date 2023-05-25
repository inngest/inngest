package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

type Test struct {
	// Name is the human name of the test.
	Name string
	// Description is a description of the test.
	Description string
	// Function is the function to search for, expected to be registered during the SDK handshake.
	//
	// While tests can only check one function at a time, the SDK may register many functions
	// at once.
	Function function.Function
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
	requestCtx SDKCtx
	// requestSteps stores the "steps" field that must be present within executor requests
	requestSteps map[string]any
	// lastResponse stores the last response time
	lastResponse time.Time

	lastEventID *string
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
		client := inngestgo.NewClient(eventKey, inngestgo.WithEndpoint(eventURL.String()))
		id, err := client.Send(context.Background(), evt)
		require.NoError(t.test, err)
		t.lastEventID = &id
	}
}

// SetRequestEvent
func (t *Test) SetRequestEvent(event inngestgo.Event) func() {
	return func() {
		t.requestEvent = event
	}
}

func (t *Test) SetRequestContext(ctx SDKCtx) func() {
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

func (t *Test) ExpectRequest(name string, queryStepID string, timeout time.Duration) func() {
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

			require.EqualValues(t.test, t.requestEvent, er.Event, "Request event is incorrect", name)
			// Unset the run ID so that our unique run ID doesn't cause issues.
			t.requestCtx.RunID = er.Ctx.RunID
			require.EqualValues(t.test, t.requestCtx, er.Ctx, "Request ctx is incorrect", name)
			require.EqualValues(t.test, t.requestSteps, er.Steps, "Request steps are incorrect", name)

		case <-time.After(timeout):
			require.Failf(t.test, "Expected executor request but timed out", name)
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
					if reflect.DeepEqual(s, op[0]) {
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

func (t *Test) ExpectResponse(status int, body []byte) func() {
	return func() {
		select {
		case r := <-t.responses:
			t.lastResponse = time.Now()
			byt, err := io.ReadAll(r.Body)
			require.NoError(t.test, err)
			require.Equal(t.test, string(body), string(byt))
		case <-time.After(time.Second):
			require.Fail(t.test, "Expected SDK generator response but timed out")
		}
	}
}

func (t *Test) ExpectJSONResponse(status int, expected any) func() {
	return func() {
		select {
		case r := <-t.responses:
			t.lastResponse = time.Now()
			byt, err := io.ReadAll(r.Body)
			require.NoError(t.test, err)
			var actual any
			err = json.Unmarshal(byt, &actual)
			require.NoError(t.test, err)
			require.EqualValues(t.test, expected, actual)
		case <-time.After(time.Second):
			require.Fail(t.test, "Expected SDK generator response but timed out")
		}
	}
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
			require.EqualValues(t.test, expected, actual)
		case <-time.After(time.Second):
			require.Fail(t.test, "Expected SDK generator response but timed out")
		}
	}
}

func (t *Test) After(d time.Duration) func() {
	return func() {
		<-time.After(d)
	}
}

type ExecutorRequest struct {
	Event inngestgo.Event `json:"event"`
	Steps map[string]any  `json:"steps"`
	Ctx   SDKCtx          `json:"ctx"`
}

type SDKCtx struct {
	FnID   string               `json:"fn_id"`
	StepID string               `json:"step_id"`
	RunID  ulid.ULID            `json:"run_id"`
	Stack  driver.FunctionStack `json:"stack"`
}

func strptr(s string) *string {
	return &s
}
