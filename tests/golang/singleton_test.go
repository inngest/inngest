package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngestgo"
	"github.com/inngest/inngestgo/step"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingletonFunction(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "singleton")
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 50
		fnDuration = 10
	)

	trigger := "test/singleton-fn"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:        "fn-singleton",
			Singleton: &inngestgo.ConfigSingleton{Key: inngestgo.StrPtr("event.data.user.id")},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&total, 1)

			next := atomic.AddInt32(&inProgress, 1)

			// We should never exceed more than one function running
			require.Less(t, next, int32(2))

			<-time.After(time.Duration(fnDuration) * time.Second)

			atomic.AddInt32(&inProgress, -1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	for i := 0; i < numEvents; i++ {
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"user": map[string]any{"id": 42},
				},
			})
			require.NoError(t, err)
		}()
	}

	// Eventually the fn starts.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgress) == 1
	}, 2*time.Second, 50*time.Millisecond)

	for i := 0; i < fnDuration; i++ {
		<-time.After(time.Second)
		require.LessOrEqual(t, atomic.LoadInt32(&inProgress), int32(1))
	}

	// The function executed only once, the other two events were ignored
	require.EqualValues(t, 1, atomic.LoadInt32(&total))
}

func TestSingletonFunctionWithKeyResolvingToFalse(t *testing.T) {
	inngestClient, server, registerFuncs := NewSDKHandler(t, "singleton")
	defer server.Close()

	var (
		inProgress, total int32

		numEvents  = 2
		fnDuration = 5
	)

	trigger := "test/singleton-fn"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID:        "fn-singleton",
			Singleton: &inngestgo.ConfigSingleton{Key: inngestgo.StrPtr("event.data.decision")},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			fmt.Println("Running func", *input.Event.ID, input.Event.Data)
			atomic.AddInt32(&total, 1)

			atomic.AddInt32(&inProgress, 1)

			<-time.After(time.Duration(fnDuration) * time.Second)

			atomic.AddInt32(&inProgress, -1)
			return true, nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	for i := 0; i < numEvents; i++ {
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"decision": false,
				},
			})
			require.NoError(t, err)
		}()
	}

	// Eventually both fn start because they are not running as a singleton
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&inProgress) == 2
	}, 2*time.Second, 50*time.Millisecond)

	// The function executed twice
	require.EqualValues(t, 2, atomic.LoadInt32(&total))
}

func TestSingletonCancelMode(t *testing.T) {
	appName := uuid.New().String()

	inngestClient, server, registerFuncs := NewSDKHandler(t, appName)
	defer server.Close()

	var successCounter int32
	var cancelCounter int32
	var startedCounter int32

	trigger := "test/singleton-cancel"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-singleton-cancel",
			Singleton: &inngestgo.ConfigSingleton{
				Key:  inngestgo.StrPtr("event.data.user.id"),
				Mode: enums.SingletonModeCancel,
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			_, err := step.Run(ctx, "counter1", func(ctx context.Context) (any, error) {
				atomic.AddInt32(&startedCounter, 1)
				return "done", nil
			})
			require.NoError(t, err)
			step.Sleep(ctx, "sleep", 10*time.Second)

			return true, nil
		},
	)

	require.NoError(t, err)

	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "on-cancel",
		},
		inngestgo.EventTrigger("inngest/function.cancelled",
			inngestgo.StrPtr(fmt.Sprintf(
				"event.data.function_id == '%s-fn-singleton-cancel'",
				appName,
			))),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			atomic.AddInt32(&cancelCounter, 1)
			return nil, nil
		},
	)

	require.NoError(t, err)

	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "on-finish",
		},
		inngestgo.EventTrigger("inngest/function.finished", inngestgo.StrPtr(fmt.Sprintf(
			"event.data.function_id == '%s-fn-singleton-cancel' && event.data.result == true",
			appName,
		))),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			atomic.AddInt32(&successCounter, 1)
			return nil, nil
		},
	)

	require.NoError(t, err)
	registerFuncs()

	numEvents := 50

	// send an event immediately prior to the goroutine to ensure one func is cancelled.
	_, err = inngestClient.Send(context.Background(), inngestgo.Event{
		Name: trigger,
		Data: map[string]any{
			"user": map[string]any{"id": 42},
		},
	})
	require.NoError(t, err)
	<-time.After(5 * time.Millisecond)

	for i := 0; i < (numEvents - 1); i++ {
		go func() {
			_, err := inngestClient.Send(context.Background(), inngestgo.Event{
				Name: trigger,
				Data: map[string]any{
					"user": map[string]any{"id": 42},
				},
			})
			require.NoError(t, err)
		}()
	}

	<-time.After(time.Second)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		require.Equal(c, int32(1), atomic.LoadInt32(&successCounter), "success counter should be 1")
		// We run events in a goroutine, meaning many funcs can be running then cancelled.
		// there must be at least one cancel func.
		loaded := atomic.LoadInt32(&cancelCounter)
		require.GreaterOrEqual(c, loaded, int32(1), "cancel counter should be at least 1, got %d", loaded)
	}, 15*time.Second, 100*time.Millisecond)
}

func TestSingletonDifferentKeysBothRun(t *testing.T) {
	appName := uuid.New().String()

	inngestClient, server, registerFuncs := NewSDKHandler(t, appName)
	defer server.Close()

	var successCounter int32
	var cancelCounter int32

	trigger := "test/singleton-cancel-different-keys"

	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "fn-singleton-cancel-different-keys",
			Singleton: &inngestgo.ConfigSingleton{
				Key:  inngestgo.StrPtr("event.data.user.id"),
				Mode: enums.SingletonModeCancel,
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			step.Sleep(ctx, "sleep", 5*time.Second)
			return true, nil
		},
	)
	require.NoError(t, err)

	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "on-cancel-different-keys"},
		inngestgo.EventTrigger("inngest/function.cancelled", inngestgo.StrPtr(fmt.Sprintf(
			"event.data.function_id == '%s-fn-singleton-cancel-different-keys'",
			appName,
		))),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			atomic.AddInt32(&cancelCounter, 1)
			return nil, nil
		},
	)
	require.NoError(t, err)

	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "on-success-different-keys"},
		inngestgo.EventTrigger("inngest/function.finished", inngestgo.StrPtr(fmt.Sprintf(
			"event.data.function_id == '%s-fn-singleton-cancel-different-keys' && event.data.result == true",
			appName,
		))),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			atomic.AddInt32(&successCounter, 1)
			return nil, nil
		},
	)
	require.NoError(t, err)

	registerFuncs()

	_, err = inngestClient.Send(context.Background(), inngestgo.Event{
		Name: trigger,
		Data: map[string]any{"user": map[string]any{"id": 1}},
	})
	require.NoError(t, err)

	_, err = inngestClient.Send(context.Background(), inngestgo.Event{
		Name: trigger,
		Data: map[string]any{"user": map[string]any{"id": 2}},
	})
	require.NoError(t, err)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		require.Equal(c, int32(2), atomic.LoadInt32(&successCounter))
		require.Equal(c, int32(0), atomic.LoadInt32(&cancelCounter))
	}, 10*time.Second, 100*time.Millisecond)
}
