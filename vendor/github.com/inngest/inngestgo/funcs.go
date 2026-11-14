package inngestgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/gosimple/slug"
	"github.com/inngest/inngestgo/internal/event"
	"github.com/inngest/inngestgo/internal/fn"
	"github.com/inngest/inngestgo/pkg/checkpoint"
)

// Slugify converts a string to a slug. This is only useful for replicating the
// legacy slugification logic for function IDs, aiding in migration to a newer
// SDK version.
func Slugify(s string) string {
	return slug.Make(s)
}

// Ptr converts the given type to a pointer.  Nil pointers are sometimes
// used for optional arguments within configuration, meaning we need pointers
// within struct values.  This util helps.
func Ptr[T any](i T) *T { return &i }

func BoolPtr(b bool) *bool { return &b }

func StrPtr(i string) *string { return &i }

func IntPtr(i int) *int { return &i }

// CreateFunction creates a new function which can be registered within a handler.
//
// This function uses generics, allowing you to supply the event that triggers the function.
// For example, if you have a signup event defined as a struct you can use this to strongly
// type your input:
//
//	type SignupEvent struct {
//		Name string
//		Data struct {
//			Email     string
//			AccountID string
//		}
//	}
//
//	f := CreateFunction(
//		inngestgo.FunctionOptions{Name: "Post-signup flow"},
//		inngestgo.EventTrigger("user/signed.up"),
//		func(ctx context.Context, input gosdk.Input[SignupEvent]) (any, error) {
//			// .. Your logic here.  input.Event will be strongly typed as a SignupEvent.
//			// step.Run(ctx, "Do some logic", func(ctx context.Context) (string, error) { return "hi", nil })
//		},
//	)
func CreateFunction[T any](
	c Client,
	fc FunctionOpts,
	trigger fn.Triggerable,
	f SDKFunction[T],
) (ServableFunction, error) {
	// Validate that the input type is a concrete type, and not an interface.
	//
	// The only exception is `any`, when users don't care about the input event
	// eg. for cron based functions.

	err := fc.Validate()
	if err != nil {
		return nil, err
	}
	if fc.Loop != nil {
		return nil, errors.New("loop is set, use CreateLoop or CreateFSM")
	}

	if fc.Checkpoint == nil {
		if v, ok := c.(*apiClient); ok {
			fc.Checkpoint = v.Checkpoint
		}
		if fc.Checkpoint == nil {
			fc.Checkpoint = &checkpoint.Config{}
		}
	}

	sf := servableFunc{
		appID:   c.AppID(),
		fc:      fc,
		trigger: trigger,
		f:       f,
	}

	zt := sf.ZeroType()
	eventDataField := zt.FieldByName("Data")
	err = event.ValidateEventDataType(eventDataField.Interface())
	if err != nil {
		return nil, err
	}

	// TODO: This feels wrong but is necessary since there isn't a
	// function-adding method on the client interface.
	if v, ok := c.(*apiClient); ok {
		v.h.Register(sf)
	}

	return sf, nil
}

func EventTrigger(name string, expression *string) fn.Trigger {
	return fn.Trigger{
		EventTrigger: &fn.EventTrigger{
			Event:      name,
			Expression: expression,
		},
	}
}

func CronTrigger(cron string) fn.Trigger {
	return fn.Trigger{
		CronTrigger: &fn.CronTrigger{
			Cron: cron,
		},
	}
}

// CronTriggerWithJitter returns a cron trigger with an optional jitter duration.
// Jitter delays the function execution by a random amount up to the specified duration.
// The duration is serialized as a string (e.g. "30s") for the server.
func CronTriggerWithJitter(cron string, jitter time.Duration) fn.Trigger {
	s := jitter.String()
	return fn.Trigger{
		CronTrigger: &fn.CronTrigger{
			Cron:   cron,
			Jitter: &s,
		},
	}
}

// SDKFunction represents a user-defined function to be called based off of events or
// on a schedule.
//
// The function is registered with the SDK by calling `CreateFunction` with the function
// name, the trigger, the event type for marshalling, and any options.
//
// This uses generics to strongly type input events:
//
//	func(ctx context.Context, input gosdk.Input[SignupEvent]) (any, error) {
//		// .. Your logic here.  input.Event will be strongly typed as a SignupEvent.
//	}
type SDKFunction[T any] func(ctx context.Context, input Input[T]) (any, error)

// LoopOpts configures a loop or fsm.  Key is a CEL expression over the
// event that names the instance.  Initial is the state an instance has
// before its first commit.
type LoopOpts[STATE any] struct {
	FunctionOpts
	Key     string
	Tick    time.Duration
	IdleTTL time.Duration
	Initial STATE
}

// LoopFunction is a loop or fsm handler.  the handler mutates the state
// through the pointer.  a nil return commits that state.  an error return
// commits nothing.  a new pointer assigned to the parameter is lost.
type LoopFunction[STATE any, EVENT any] func(ctx context.Context, input LoopInput[EVENT], state *STATE) error

// CreateLoop creates a loop function which can be registered within a handler.
// opts.Key picks the instance.  each run gets the instance state as the
// handler's third argument and the handler mutates it through the pointer.
// a nil return commits that state.  an error return commits nothing.  a
// new pointer assigned to the parameter is lost.
func CreateLoop[STATE any, EVENT any](
	c Client,
	opts LoopOpts[STATE],
	trigger fn.Triggerable,
	f LoopFunction[STATE, EVENT],
) (ServableFunction, error) {
	return createLoop(c, fn.LoopKindLoop, opts, trigger, f)
}

// CreateFSM creates an fsm function which can be registered within a
// handler.  an fsm is a loop that cannot subscribe to events.  the handler
// mutates the state through the pointer.  a nil return commits that state.
// an error return commits nothing.  a new pointer assigned to the
// parameter is lost.
func CreateFSM[STATE any, EVENT any](
	c Client,
	opts LoopOpts[STATE],
	trigger fn.Triggerable,
	f LoopFunction[STATE, EVENT],
) (ServableFunction, error) {
	return createLoop(c, fn.LoopKindFSM, opts, trigger, f)
}

func createLoop[STATE any, EVENT any](
	c Client,
	kind string,
	opts LoopOpts[STATE],
	trigger fn.Triggerable,
	f LoopFunction[STATE, EVENT],
) (ServableFunction, error) {
	fc := opts.FunctionOpts
	if err := fc.Validate(); err != nil {
		return nil, err
	}
	if fc.Loop != nil {
		return nil, errors.New("loop is set on FunctionOpts, set Key, Tick, and IdleTTL on LoopOpts")
	}
	if opts.Key == "" {
		return nil, errors.New("loop key is required")
	}
	if opts.Tick < 0 {
		return nil, errors.New("loop tick must not be negative")
	}
	if opts.IdleTTL < 0 {
		return nil, errors.New("loop idle ttl must not be negative")
	}
	if fc.Singleton != nil {
		return nil, errors.New("loops and singletons are mutually exclusive")
	}
	if fc.BatchEvents != nil {
		return nil, errors.New("loops and batching are mutually exclusive")
	}
	if fc.Debounce != nil {
		return nil, errors.New("loops and debouncing are mutually exclusive")
	}
	if fc.Concurrency != nil && len(fc.Concurrency.Fn) > 0 {
		return nil, errors.New("loops cannot set function concurrency, a loop always runs one input per key")
	}
	if fc.Concurrency != nil {
		for _, limit := range fc.Concurrency.Step {
			if limit.Key != nil {
				return nil, errors.New("loops and keyed step concurrency limits are mutually exclusive")
			}
		}
	}

	fc.Loop = &fn.Loop{
		Kind:    kind,
		Key:     opts.Key,
		Tick:    opts.Tick,
		IdleTTL: opts.IdleTTL,
	}

	if fc.Checkpoint == nil {
		if v, ok := c.(*apiClient); ok {
			fc.Checkpoint = v.Checkpoint
		}
		if fc.Checkpoint == nil {
			fc.Checkpoint = &checkpoint.Config{}
		}
	}

	// the handler gets a *STATE.  a pointer STATE would give it a double
	// pointer and a nil Initial that marshals to null.
	if reflect.TypeFor[STATE]().Kind() == reflect.Pointer {
		return nil, errors.New("STATE must not be a pointer type, use CreateLoop[State, Event] with the value type")
	}

	// marshal the pointer so a pointer receiver MarshalJSON on STATE is
	// used, which matches the form the handler returns from *STATE.
	initial, err := json.Marshal(&opts.Initial)
	if err != nil {
		return nil, fmt.Errorf("error marshalling initial loop state: %w", err)
	}
	if string(initial) == "null" {
		return nil, errors.New("Initial marshals to null, set a non-nil Initial value")
	}

	sf := servableFunc{
		appID:        c.AppID(),
		fc:           fc,
		trigger:      trigger,
		f:            f,
		initialState: initial,
	}

	zt := sf.ZeroType()
	eventDataField := zt.FieldByName("Data")
	if err := event.ValidateEventDataType(eventDataField.Interface()); err != nil {
		return nil, err
	}

	if v, ok := c.(*apiClient); ok {
		v.h.Register(sf)
	}

	return sf, nil
}

type servableFunc struct {
	appID   string
	fc      FunctionOpts
	trigger fn.Triggerable
	f       any
	// initialState is the marshalled LoopOpts.Initial.  nil for functions
	// made with CreateFunction.
	initialState json.RawMessage
}

// InitialState returns the state a loop instance has before its first
// commit.  it is nil for functions made with CreateFunction.
func (s servableFunc) InitialState() json.RawMessage {
	return s.initialState
}

func (s servableFunc) Config() FunctionOpts {
	return s.fc
}

func (s servableFunc) ID() string {
	return s.fc.ID
}

func (s servableFunc) FullyQualifiedID() string {
	return fmt.Sprintf("%s-%s", s.appID, s.ID())
}

func (s servableFunc) Name() string {
	if s.fc.Name == "" {
		return s.ID()
	}
	return s.fc.Name
}

func (s servableFunc) Trigger() fn.Triggerable {
	return s.trigger
}

func (s servableFunc) ZeroType() reflect.Value {
	// Grab the concrete type from the generic Input[T] type.  This lets us easily
	// initialize new values of this type at runtime.
	fVal := reflect.ValueOf(s.f)
	inputVal := reflect.New(fVal.Type().In(1)).Elem()
	return reflect.New(inputVal.FieldByName("Event").Type()).Elem()
}

func (s servableFunc) ZeroEvent() any {
	return s.ZeroType().Interface()
}

func (s servableFunc) Func() any {
	return s.f
}
