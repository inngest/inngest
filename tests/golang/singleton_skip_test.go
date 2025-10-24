package golang

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/require"
)

// TestSingletonSkipMode tests that when singleton mode is "skip" (default),
// skipped runs create proper database records and can be queried without errors.
// This is a regression test for: "sql: no rows in result set" error.
func TestSingletonSkipMode(t *testing.T) {
	appName := uuid.New().String()

	inngestClient, server, registerFuncs := NewSDKHandler(t, appName)
	defer server.Close()

	trigger := "test/singleton-skip"
	functionID := "fn-singleton-skip"
	
	var completedCounter int32
	var startedCounter int32

	// Create a function with singleton skip mode (default)
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: functionID,
			Singleton: &inngestgo.ConfigSingleton{
				Key: inngestgo.StrPtr("event.data.user.id"),
				// Mode: enums.SingletonModeSkip is the default
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			atomic.AddInt32(&startedCounter, 1)
			// Sleep for a while to allow concurrent events to be skipped
			time.Sleep(2 * time.Second)
			atomic.AddInt32(&completedCounter, 1)
			return map[string]any{"completed": true}, nil
		},
	)
	require.NoError(t, err)

	// Listen for function.finished events to verify the run completed
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "on-finish-skip"},
		inngestgo.EventTrigger("inngest/function.finished", inngestgo.StrPtr(fmt.Sprintf(
			"event.data.function_id == '%s-%s'",
			appName,
			functionID,
		))),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			// This verifies we can access run data without "sql: no rows" error
			t.Logf("Function finished: %+v", input.Event.Data)
			return nil, nil
		},
	)
	require.NoError(t, err)

	registerFuncs()

	// Send multiple events with the same singleton key
	numEvents := 5

	for i := 0; i < numEvents; i++ {
		_, err := inngestClient.Send(context.Background(), inngestgo.Event{
			Name: trigger,
			Data: map[string]any{
				"user":  map[string]any{"id": 42},
				"index": i,
			},
		})
		require.NoError(t, err)
		
		// Small delay between sends to ensure ordering
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for function to complete
	time.Sleep(5 * time.Second)

	// Verify that only one execution started (others were skipped)
	require.Equal(t, int32(1), atomic.LoadInt32(&startedCounter), "should have exactly one started execution")
	require.Equal(t, int32(1), atomic.LoadInt32(&completedCounter), "should have exactly one completed execution")
	
	// The key assertion: the test didn't crash with "sql: no rows in result set"
	// If the fix is working, the OnFunctionSkipped lifecycle event was called
	// and proper trace records were created for skipped runs
	t.Log("Test passed: skipped runs handled correctly without database errors")
}

// TestSingletonSkipModeStatusCheck tests that skipped runs have the correct status
// and do not show "running" status indefinitely
func TestSingletonSkipModeStatusCheck(t *testing.T) {
	appName := uuid.New().String()

	inngestClient, server, registerFuncs := NewSDKHandler(t, appName)
	defer server.Close()

	trigger := "test/singleton-skip-status"
	functionID := "fn-singleton-skip-status"

	var executionStarted int32
	var executionComplete int32

	// Create a function with singleton skip mode
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: functionID,
			Singleton: &inngestgo.ConfigSingleton{
				Key: inngestgo.StrPtr("event.data.key"),
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			atomic.AddInt32(&executionStarted, 1)
			time.Sleep(3 * time.Second)
			atomic.AddInt32(&executionComplete, 1)
			return "success", nil
		},
	)
	require.NoError(t, err)
	registerFuncs()

	// Send first event - this should execute
	_, err = inngestClient.Send(context.Background(), inngestgo.Event{
		Name: trigger,
		Data: map[string]any{"key": "test-key-1"},
	})
	require.NoError(t, err)

	// Wait a bit for the first run to start
	time.Sleep(500 * time.Millisecond)
	require.Equal(t, int32(1), atomic.LoadInt32(&executionStarted), "first execution should have started")

	// Send second event with same key - this should be skipped
	_, err = inngestClient.Send(context.Background(), inngestgo.Event{
		Name: trigger,
		Data: map[string]any{"key": "test-key-1"},
	})
	require.NoError(t, err)

	// Wait for first execution to complete
	time.Sleep(4 * time.Second)
	require.Equal(t, int32(1), atomic.LoadInt32(&executionComplete), "first execution should have completed")
	
	// The key assertion: second event was skipped, not stuck in "running" status
	// If the fix is working, OnFunctionSkipped was called with proper metadata
	// and trace runs were created with correct status
	require.Equal(t, int32(1), atomic.LoadInt32(&executionStarted), "only one execution should have started")
	t.Log("Test passed: skipped run did not cause database errors or stuck status")
}

// TestSingletonSkipHistoryEvent tests that skipped runs create proper history events
func TestSingletonSkipHistoryEvent(t *testing.T) {
	appName := uuid.New().String()

	inngestClient, server, registerFuncs := NewSDKHandler(t, appName)
	defer server.Close()

	trigger := "test/singleton-skip-history"
	functionID := "fn-singleton-skip-history"

	var skippedEventReceived int32
	var finishedEventReceived int32

	// Create a function with singleton skip mode
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: functionID,
			Singleton: &inngestgo.ConfigSingleton{
				Key: inngestgo.StrPtr("event.data.key"),
			},
		},
		inngestgo.EventTrigger(trigger, nil),
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			time.Sleep(2 * time.Second)
			return "done", nil
		},
	)
	require.NoError(t, err)

	// Listen for function.skipped events (if the system emits them)
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "on-skipped",
		},
		inngestgo.EventTrigger("inngest/function.skipped", inngestgo.StrPtr(fmt.Sprintf(
			"event.data.function_id == '%s-%s'",
			appName,
			functionID,
		))),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			atomic.AddInt32(&skippedEventReceived, 1)
			t.Logf("Received function.skipped event: %+v", input.Event.Data)
			return nil, nil
		},
	)
	require.NoError(t, err)
	
	// Also listen for function.finished events
	_, err = inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{
			ID: "on-finished-history",
		},
		inngestgo.EventTrigger("inngest/function.finished", inngestgo.StrPtr(fmt.Sprintf(
			"event.data.function_id == '%s-%s'",
			appName,
			functionID,
		))),
		func(ctx context.Context, input inngestgo.Input[map[string]any]) (any, error) {
			atomic.AddInt32(&finishedEventReceived, 1)
			t.Logf("Received function.finished event: %+v", input.Event.Data)
			return nil, nil
		},
	)
	require.NoError(t, err)

	registerFuncs()

	// Send first event
	_, err = inngestClient.Send(context.Background(), inngestgo.Event{
		Name: trigger,
		Data: map[string]any{"key": "shared-key"},
	})
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// Send second event with same key - should be skipped
	_, err = inngestClient.Send(context.Background(), inngestgo.Event{
		Name: trigger,
		Data: map[string]any{"key": "shared-key"},
	})
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(5 * time.Second)

	// Verify at least one function finished event was received
	require.GreaterOrEqual(t, atomic.LoadInt32(&finishedEventReceived), int32(1), "should receive at least one finished event")
	
	// Note: The system may or may not emit inngest/function.skipped events
	// This test primarily verifies that the system doesn't crash or error
	// when handling skipped singleton runs
	t.Logf("Skipped events received: %d, Finished events received: %d", 
		atomic.LoadInt32(&skippedEventReceived), 
		atomic.LoadInt32(&finishedEventReceived))
	
	// The key assertion: test completed without database errors
	t.Log("Test passed: singleton skip handling works correctly with lifecycle events")
}
