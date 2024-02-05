package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/errors"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/require"
)

func TestStepErrors(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t, "fail-app")
	defer server.Close()

	require := require.New(t)

	var aCount int32
	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			ID:      "always-fail",
			Name:    "Always fail",
			Retries: inngestgo.IntPtr(0),
		},
		inngestgo.EventTrigger("test/fail", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			val, err := step.Run(ctx, "fails", func(ctx context.Context) (any, error) {
				return "some-data", fmt.Errorf("this step fails")
			})

			// Typically, you'd...
			// if err != nil {
			// }

			// Assert multilpe returns work as expected.
			require.Equal("some-data", val)

			// We also returned an error>
			require.NotNil(err)
			require.Contains(err.Error(), "this step fails")
			require.True(errors.IsStepError(err))

			val, err = step.Run(ctx, "succeeds", func(ctx context.Context) (any, error) {
				return []any{"ok", true}, nil
			})
			// This step should succeed
			atomic.AddInt32(&aCount, 1)
			return val, err
		},
	)
	h.Register(a)
	registerFuncs()

	_, err := inngestgo.Send(context.Background(), inngestgo.Event{
		Name: "test/fail",
		Data: map[string]any{
			"test": true,
			"id":   "1",
		},
	})
	require.NoError(err)

	<-time.After(3 * time.Second)

	require.Eventually(func() bool {
		return atomic.LoadInt32(&aCount) == 1
	}, 15*time.Second, time.Second)
}

func TestStepErrorCalledOnce(t *testing.T) {
	h, server, registerFuncs := NewSDKHandler(t, "fail-app")
	defer server.Close()

	require := require.New(t)

	var stepCount, fnCount int32
	a := inngestgo.CreateFunction(
		inngestgo.FunctionOpts{
			ID:      "always-fail",
			Name:    "Always fail",
			Retries: inngestgo.IntPtr(2),
		},
		inngestgo.EventTrigger("test/fail", nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			val, err := step.Run(ctx, "fails", func(ctx context.Context) (any, error) {
				atomic.AddInt32(&stepCount, 1)
				fmt.Println("step hit")
				return "some-data", errors.RetryAtError(fmt.Errorf("this step fails"), time.Now())
			})
			// This should only be called once, even with retries.  Unhandled step errors
			// trigger an immediate function failure.
			fmt.Println("fn hit")
			atomic.AddInt32(&fnCount, 1)
			<-time.After(10 * time.Millisecond)
			return val, err
		},
	)
	h.Register(a)
	registerFuncs()

	_, err := inngestgo.Send(context.Background(), inngestgo.Event{
		Name: "test/fail",
		Data: map[string]any{
			"test": true,
			"id":   "1",
		},
	})
	require.NoError(err)

	require.Eventually(func() bool {
		return atomic.LoadInt32(&fnCount) == 1
	}, 5*time.Second, 5*time.Millisecond)

	// The step should have been called 3 times, and the fn once.
	require.EqualValues(3, atomic.LoadInt32(&stepCount))

	// Ensure that the fn is not retried.
	<-time.After(30 * time.Second)
	require.EqualValues(1, atomic.LoadInt32(&fnCount))
}
