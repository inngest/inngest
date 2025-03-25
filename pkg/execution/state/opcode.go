package state

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/dateutil"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/util/aigateway"
	"github.com/xhit/go-str2duration/v2"
)

var (
	ErrStepInputTooLarge  = fmt.Errorf("step input size is greater than the limit")
	ErrStepOutputTooLarge = fmt.Errorf("step output size is greater than the limit")
)

type GeneratorOpcode struct {
	// Op represents the type of operation invoked in the function.
	Op enums.Opcode `json:"op"`
	// ID represents a hashed unique ID for the operation.  This acts
	// as the generated step ID for the state store.
	ID string `json:"id"`
	// Name represents the name of the step, or the sleep duration for
	// sleeps.
	Name string `json:"name"`
	// Opts indicate options for the operation, eg. matching expressions
	// when setting up async event listeners via `waitForEvent`, or retry
	// policies for steps.
	Opts any `json:"opts"`
	// Data is the resulting data from the operation, eg. the step
	// output. Note that for gateway requests, this is initially the
	// request input.
	Data json.RawMessage `json:"data"`
	// Error is the failing result from the operation, e.g. an error thrown
	// from a step.  This MUST be in the shape of OpcodeError.
	Error *UserError `json:"error"`
	// SDK versions < 3.?.? don't respond with the display name.
	DisplayName *string `json:"displayName"`
}

func (g GeneratorOpcode) Validate() error {
	if input, _ := g.Input(); input != "" && len(input) > consts.MaxStepInputSize {
		return ErrStepOutputTooLarge
	}

	if output, _ := g.Output(); output != "" && len(output) > consts.MaxStepOutputSize {
		return ErrStepOutputTooLarge
	}

	return nil
}

// Get the name of the step as defined in code by the user.
func (g GeneratorOpcode) UserDefinedName() string {
	if g.DisplayName != nil {
		return *g.DisplayName
	}

	// SDK versions < 3.?.? don't respond with the display
	// name, so we we'll use the deprecated name field as a
	// fallback.
	return g.Name
}

// HasAI checks if this op is related to AI.
func (g GeneratorOpcode) HasAI() bool {
	if g.Op == enums.OpcodeAIGateway {
		return true
	}

	if g.RunType() == "step.ai.wrap" {
		return true
	}

	return false
}

// Get the stringified input of the step, if `step.Run` was passed inputs
// or if we're using request offloading.
func (g GeneratorOpcode) Input() (string, error) {
	// Only specific operations can have inputs.  These are currently limited
	// to OpcodeStepRun and OpcodeStepAIGateway.
	switch g.Op {
	case enums.OpcodeStepRun, enums.OpcodeStep:
		runOpts, _ := g.RunOpts()
		if runOpts != nil && runOpts.Input != nil {
			return string(runOpts.Input), nil
		}
	case enums.OpcodeAIGateway:
		req, _ := g.AIGatewayOpts()
		return string(req.Body), nil
	}

	return "", nil
}

// Get the stringified output of the step.
func (g GeneratorOpcode) Output() (string, error) {
	// OpcodeStepError MUST always wrap the output in an "error"
	// field, allowing the SDK to differentiate between an error and data.
	if g.Op == enums.OpcodeStepError {
		byt, err := json.Marshal(map[string]any{"error": g.Error})
		return string(byt), err
	}

	// If this is an OpcodeStepRun, we can guarantee that the data is unwrapped.
	//
	// We MUST wrap the data in a "data" object in the state store so that the
	// SDK can differentiate between "data" and "error";  per-step errors wraps the
	// error with "error" and updates step state on the final failure.
	if g.Op == enums.OpcodeStepRun {
		byt, err := json.Marshal(map[string]any{"data": g.Data})
		return string(byt), err
	}

	// Data is allowed to be `null` if no error is found and the op returned no data.
	if g.Data != nil {
		return string(g.Data), nil
	}
	return "", nil
}

// IsError returns whether this op represents an error, for example a
// `StepError` being passed back from an SDK.
func (g GeneratorOpcode) IsError() bool {
	return g.Error != nil
}

// Returns, if any, the type of a StepRun operation.
func (g GeneratorOpcode) RunType() string {
	opts, err := g.RunOpts()
	if err != nil {
		return ""
	}
	return opts.Type
}

func (g GeneratorOpcode) RunOpts() (*RunOpts, error) {
	opts := &RunOpts{}
	if err := opts.UnmarshalAny(g.Opts); err != nil {
		return nil, err
	}
	return opts, nil
}

func (g GeneratorOpcode) WaitForEventOpts() (*WaitForEventOpts, error) {
	if opts, ok := g.Opts.(*WaitForEventOpts); ok && opts != nil {
		return opts, nil
	}

	opts := &WaitForEventOpts{}
	if err := opts.UnmarshalAny(g.Opts); err != nil {
		return nil, err
	}
	if opts.Event == "" {
		// use the step name as a fallback, for v1/2 of the TS SDK.
		opts.Event = g.Name
	}
	if opts.Event == "" {
		return nil, fmt.Errorf("An event name must be provided when waiting for an event")
	}
	return opts, nil
}

func (g GeneratorOpcode) SleepDuration() (time.Duration, error) {
	if g.Op != enums.OpcodeSleep {
		return 0, fmt.Errorf("unable to return sleep duration for opcode %s", g.Op.String())
	}

	opts := &SleepOpts{}
	if err := opts.UnmarshalAny(g.Opts); err != nil {
		return 0, err
	}

	if opts.Duration == "" {
		// use step name as a fallback for v1/2 of the TS SDK
		opts.Duration = g.Name
	}
	if len(opts.Duration) == 0 {
		return 0, nil
	}

	// Quick heuristic to check if this is likely a date layout
	if len(opts.Duration) >= 10 {
		if parsed, err := dateutil.Parse(opts.Duration); err == nil {
			at := time.Until(parsed).Round(time.Second)
			if at < 0 {
				return time.Duration(0), nil
			}
			return at, nil
		}
	}

	return str2duration.ParseDuration(opts.Duration)
}

func (g GeneratorOpcode) InvokeFunctionOpts() (*InvokeFunctionOpts, error) {
	opts := &InvokeFunctionOpts{}
	if err := opts.UnmarshalAny(g.Opts); err != nil {
		return nil, err
	}
	return opts, nil
}

type InvokeFunctionOpts struct {
	FunctionID string       `json:"function_id"`
	Payload    *event.Event `json:"payload,omitempty"`
	Timeout    string       `json:"timeout"`
}

func (i *InvokeFunctionOpts) UnmarshalAny(a any) error {
	opts := InvokeFunctionOpts{}
	var mappedByt []byte
	switch typ := a.(type) {
	case []byte:
		mappedByt = typ
	default:
		byt, err := json.Marshal(a)
		if err != nil {
			return err
		}
		mappedByt = byt
	}
	if err := json.Unmarshal(mappedByt, &opts); err != nil {
		return err
	}
	*i = opts
	return nil
}

func (i InvokeFunctionOpts) Expires() (time.Time, error) {
	if i.Timeout == "" {
		return time.Now().AddDate(1, 0, 0), nil
	}

	dur, err := str2duration.ParseDuration(i.Timeout)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(dur), nil
}

type SleepOpts struct {
	Duration string `json:"duration"`
}

func (s *SleepOpts) UnmarshalAny(a any) error {
	opts := SleepOpts{}
	var mappedByt []byte
	switch typ := a.(type) {
	case []byte:
		mappedByt = typ
	default:
		byt, err := json.Marshal(a)
		if err != nil {
			return err
		}
		mappedByt = byt
	}
	if err := json.Unmarshal(mappedByt, &opts); err != nil {
		return err
	}
	*s = opts
	return nil
}

type RunOpts struct {
	Type  string          `json:"type,omitempty"`
	Input json.RawMessage `json:"input"`
}

func (r *RunOpts) UnmarshalAny(a any) error {
	opts := RunOpts{}
	var mappedByt []byte
	switch typ := a.(type) {
	case []byte:
		mappedByt = typ
	default:
		byt, err := json.Marshal(a)
		if err != nil {
			return err
		}
		mappedByt = byt
	}
	if err := json.Unmarshal(mappedByt, &opts); err != nil {
		return err
	}

	if len(opts.Input) > 0 && opts.Input[0] != '[' {
		return fmt.Errorf("input must be an array or undefined")
	}

	*r = opts
	return nil
}

type WaitForEventOpts struct {
	Timeout string  `json:"timeout"`
	If      *string `json:"if"`
	// Event is taken from GeneratorOpcode.Name if this is empty.
	Event string `json:"event"`
}

func (w *WaitForEventOpts) UnmarshalAny(a any) error {
	opts := WaitForEventOpts{}
	var mappedByt []byte
	switch typ := a.(type) {
	case []byte:
		mappedByt = typ
	default:
		byt, err := json.Marshal(a)
		if err != nil {
			return err
		}
		mappedByt = byt
	}
	if err := json.Unmarshal(mappedByt, &opts); err != nil {
		return err
	}
	*w = opts
	return nil
}

func (w WaitForEventOpts) Expires() (time.Time, error) {
	if w.Timeout == "" {
		// The TypeScript SDK sets timeout to an empty string when the duration
		// is negative
		return time.Now(), nil
	}

	dur, err := str2duration.ParseDuration(w.Timeout)
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(dur), nil
}

// AIGatewayOpts returns the AI gateway options within the driver.
func (g *GeneratorOpcode) AIGatewayOpts() (aigateway.Request, error) {
	req := aigateway.Request{}

	// Ensure we unmarshal g.Opts  into the request options.
	// This contains Inngest-related and auth-related options
	// that do not go in the API request body we make to the provider
	var optByt []byte
	switch typ := g.Opts.(type) {
	case []byte:
		optByt = typ
	default:
		var err error
		optByt, err = json.Marshal(g.Opts)
		if err != nil {
			return aigateway.Request{}, err
		}
	}
	if err := json.Unmarshal(optByt, &req); err != nil {
		return aigateway.Request{}, err
	}

	return req, nil
}
