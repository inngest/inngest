package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/pkg/execution/driver"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/sdk"
	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

const (
	ENV_SDK_URL     = "SDK_URL"   // eg. http://127.0.0.1:3000/api/inngest
	ENV_API_URL     = "API_URL"   // eg http://127.0.0.1:8288 or https://api.inngest.com
	ENV_EVENT_URL   = "EVENT_URL" // eg http://127.0.0.1:8288 or https://inn.gs
	ENV_SIGNING_KEY = "INNGEST_SIGNING_KEY"
	ENV_EVENT_KEY   = "INNGEST_EVENT_KEY"
)

var (
	sdkURL               url.URL
	apiURL               url.URL
	eventURL             url.URL
	signingKey, eventKey string

	buffer = 5 * time.Second
)

func init() {
	sdkURL = parseEnvURL(ENV_SDK_URL)
	apiURL = parseEnvURL(ENV_API_URL)
	// If the ENV_EVENT_URL isn't specified default to the API url, assuming that this
	// is the dev server and runs both APIs.
	if os.Getenv(ENV_EVENT_URL) == "" {
		os.Setenv(ENV_EVENT_URL, apiURL.String())
	}
	eventURL = parseEnvURL(ENV_EVENT_URL)
	signingKey = os.Getenv(ENV_SIGNING_KEY)
	eventKey = os.Getenv(ENV_EVENT_KEY)
	if eventKey == "" {
		eventKey = "eventkey"
	}
}

func parseEnvURL(env string) url.URL {
	u, err := url.Parse(os.Getenv(env))
	if err != nil {
		panic(err.Error())
	}
	if u.Host == "" {
		panic(fmt.Sprintf("No %s env variable supplied", env))
	}
	return *u
}

type Test struct {
	// Name is the human name of the test.
	Name        string
	Description string

	// Function is the function to search for, expected to be registered during the SDK handshake.
	//
	// While tests can only check one function at a time, the SDK may register many functions
	// at once.
	Function function.Function

	// The event to send when testing this function.
	EventTrigger inngestgo.Event

	Timeout time.Duration

	Assertions []HTTPAssertion
}

type HTTPAssertion interface {
	Assert(t *testing.T, input []byte, req *http.Request, resp *http.Response)
}

type AfterHook interface {
	RunAfter()
}

type SDKResponse struct {
	Status int
	Data   any
	After  func()
}

func (s SDKResponse) RunAfter() {
	if s.After == nil {
		return
	}
	s.After()
}

func (s SDKResponse) Assert(t *testing.T, input []byte, req *http.Request, resp *http.Response) {
	status := resp.StatusCode

	if len(input) == 0 {
		require.Nil(t, s.Data)
		require.Equal(t, s.Status, status, "Unexpected SDK status")
		return
	}

	switch input[0] {
	case '[':
		var actual any

		if s.Status == 206 {
			// If we expect a 206, unmarshal into opcodes.
			data := []state.GeneratorOpcode{}
			err := json.Unmarshal(input, &data)
			require.NoError(t, err, "unable to marshal SDK response: %s", string(input))
			actual = data
		} else {
			actual = []map[string]any{}
			err := json.Unmarshal(input, &actual)
			require.NoError(t, err, "unable to marshal SDK response: %s", string(input))
		}

		require.EqualValues(t, s.Data, actual, "Unexpected SDK response: %s", string(input))
		require.Equal(t, s.Status, status, "Unexpected SDK status")
	case '{':
		actual := map[string]any{}
		err := json.Unmarshal(input, &actual)
		require.NoError(t, err, "unable to marshal SDK response: %s", string(input))
		require.EqualValues(t, s.Data, actual, "Unexpected SDK response: %s", string(input))
		require.Equal(t, s.Status, status, "Unexpected SDK status")
	default:
		expected, err := json.Marshal(s.Data)
		require.NoError(t, err, "unable to marshal SDK response: %s", string(input))
		require.EqualValues(t, string(expected), string(input), "Unexpected SDK response: %s", string(input))
		require.Equal(t, s.Status, status, "Unexpected SDK status")
	}
}

// ExecutorRequest
type ExecutorRequest struct {
	Event inngestgo.Event `json:"event"`
	Steps map[string]any  `json:"steps"`
	Ctx   SDKCtx          `json:"ctx"`

	// QueryStepID ensures we can validate that the executor's step ID is correct.
	QueryStepID string `json:"-"`
}

func (e ExecutorRequest) Assert(t *testing.T, input []byte, req *http.Request, resp *http.Response) {
	// Ensure the step ID is correct in the executor request
	if e.QueryStepID != "" {
		require.EqualValues(t, e.QueryStepID, req.URL.Query().Get("stepId"), "Invalid stepId query parameter from executor")
	}

	e.QueryStepID = ""

	// Assert that the input can be marshalled into a new executor request.
	actual := ExecutorRequest{}
	err := json.Unmarshal(input, &actual)
	require.NoError(t, err, "unable to marshal executor request")

	e.Ctx.RunID = actual.Ctx.RunID

	require.EqualValues(t, e, actual, "Unexpected executor request data")
}

type SDKCtx struct {
	FnID   string               `json:"fn_id"`
	StepID string               `json:"step_id"`
	RunID  ulid.ULID            `json:"run_id"`
	Stack  driver.FunctionStack `json:"stack"`
}

func run(t *testing.T, test Test) {
	t.Helper()

	// Ensure that the desired function exists within the SDK.
	rr, err := introspect(test)
	require.NoError(t, err, "Introspection error")

	var counter int32
	done := make(chan bool)

	// Start a new test server which will intercept all requests between the executor and the SDK.
	//
	// This allows us to assert that the messages passed are as we expect, including:
	//
	// - Responses from the SDK
	// - State injected via the executor.
	//
	// We can also randomly inject faults by disregarding the SDK's response and throwing a 500.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		byt, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		fmt.Printf(" ==> Received executor request:\n\t%s\n", string(byt))

		if int(counter) >= len(test.Assertions) {
			fmt.Println("")
			t.Fatalf("Received too many requests from executor:\n\n%s", string(byt))
		}

		// Assert that this request is correct.
		test.Assertions[counter].Assert(t, byt, r, nil)

		// Increase counter now that assertions have ran.
		atomic.AddInt32(&counter, 1)

		// Forward this request on to the SDK.
		url := sdkURL
		url.RawQuery = r.URL.RawQuery
		req, err := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(byt))
		require.NoError(t, err)
		req.ContentLength = r.ContentLength
		req.Header = r.Header
		sdkResponse, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		// Read the response.
		byt, err = io.ReadAll(sdkResponse.Body)
		require.NoError(t, err)

		fmt.Printf(" ==> Received SDK response:\n\t%s\n", string(byt))
		fmt.Println("")

		// Assert that the SDK response is correct
		test.Assertions[counter].Assert(t, byt, r, sdkResponse)

		// Forward the response from the SDK to the executor.
		w.WriteHeader(sdkResponse.StatusCode)
		w.Write(byt)

		if after, ok := test.Assertions[counter].(AfterHook); ok {
			after.RunAfter()
		}

		// Increase counter now that assertions have ran.
		atomic.AddInt32(&counter, 1)

		if atomic.LoadInt32(&counter) == int32(len(test.Assertions)) {
			// All and responses have come in.
			done <- true
		}
	}))
	defer srv.Close()
	localURL, err := url.Parse(srv.URL)
	require.NoError(t, err)

	// Register all functions with the SDK.
	err = register(*localURL, *rr)
	require.NoError(t, err, "Function registration error")

	// Trigger the function by sending an event.
	trigger := test.Function.Triggers[0]
	if trigger.EventTrigger == nil {
		t.Fatalf("Unable to trigger scheduled functions")
		return
	}

	client := inngestgo.NewClient(eventKey, inngestgo.WithEndpoint(eventURL.String()))
	err = client.Send(context.Background(), test.EventTrigger)
	require.NoError(t, err)

	select {
	case <-time.After(test.Timeout + buffer):
		t.Fatalf("Timed out before all request response chains were tested")
	case <-done:
		// Wait for an extra second and assert that no other requests come in.
		<-time.After(buffer)
		return
	}
}

// introspect asserts that the SDK is live and the expected function exists when calling
// the introspect handler.
func introspect(test Test) (*sdk.RegisterRequest, error) {
	url := sdkURL
	url.Path = "/api/inngest"
	url.RawQuery = "introspect"

	resp, err := http.Get(url.String())
	if err != nil {
		return nil, fmt.Errorf("unable to call introspect on SDK: %w", err)
	}
	defer resp.Body.Close()

	data := &sdk.RegisterRequest{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("invalid introspect response: unable to decode introspect response: %w", err)
	}

	expected, err := json.MarshalIndent(test.Function, "", "  ")
	if err != nil {
		return nil, err
	}

	found := false
	for _, f := range data.Functions {
		actual, _ := json.MarshalIndent(f, "", "  ")
		if bytes.Equal(expected, actual) {
			found = true
			break
		}
	}

	response, _ := json.MarshalIndent(data, "", "  ")
	if !found {
		return nil, fmt.Errorf("Expected function not found:\n%s\n\nIntrospection:\n%s", string(expected), string(response))
	}

	return data, nil
}

func register(serverURL url.URL, rr sdk.RegisterRequest) error {
	// Register functions using _this_ host and the introspection request
	for n, fn := range rr.Functions {
		for key, step := range fn.Steps {
			rt := step.Runtime.Runtime.(inngest.RuntimeHTTP)
			// Take the URL and replace the host with our server's URL.
			parsed, err := url.Parse(rt.URL)
			if err != nil {
				return err
			}
			serverURL.Path = "/"
			serverURL.RawQuery = parsed.RawQuery
			rt.URL = serverURL.String()
			step.Runtime.Runtime = rt
			fn.Steps[key] = step
		}
		rr.Functions[n] = fn
	}

	// Randomize the hash.
	hash := uuid.New().String()
	rr.Hash = &hash

	byt, err := json.Marshal(rr)
	if err != nil {
		return err
	}

	url := apiURL
	url.Path = "/fn/register"
	req, err := http.NewRequest(http.MethodPost, url.String(), bytes.NewReader(byt))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", signingKey))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		byt, _ := httputil.DumpResponse(resp, true)
		return fmt.Errorf("Error when registering functions: %s", string(byt))
	}

	return nil
}

func stepURL(fnID string, step string) string {
	url := sdkURL
	url.Path = "/api/inngest"
	q := url.Query()
	q.Add("fnId", fnID)
	q.Add("stepId", step)
	url.RawQuery = q.Encode()
	return url.String()
}
