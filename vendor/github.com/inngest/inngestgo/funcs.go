package inngestgo

import (
	"context"
	"fmt"
	"reflect"

	"github.com/gosimple/slug"
	"github.com/inngest/inngestgo/internal/event"
	"github.com/inngest/inngestgo/internal/fn"
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

type servableFunc struct {
	appID   string
	fc      FunctionOpts
	trigger fn.Triggerable
	f       any
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
