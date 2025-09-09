package golang

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiTriggerExpressions tests that functions with multiple event triggers
// only evaluate expressions for triggers that match the incoming event name.
// This prevents incorrect cross-trigger expression evaluation.
func TestMultiTriggerExpressions(t *testing.T) {
	ctx := context.Background()

	inngestClient, server, registerFuncs := NewSDKHandler(t, "multi-trigger-test-app")
	defer server.Close()

	var executionCount int32
	var lastExecutedEvent string

	// Create function with multiple triggers having different expressions
	_, err := inngestgo.CreateFunction(
		inngestClient,
		inngestgo.FunctionOpts{ID: "multi-trigger-expression-test"},
		inngestgo.MultipleTriggers{
			inngestgo.EventTrigger("user.created", inngestgo.StrPtr("event.data.type == 'premium'")),
			inngestgo.EventTrigger("user.updated", inngestgo.StrPtr("event.data.type == 'standard'")),
		},
		func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
			atomic.AddInt32(&executionCount, 1)
			lastExecutedEvent = input.Event.Name
			return nil, nil
		},
	)
	require.NoError(t, err)

	registerFuncs()

	t.Run("matching trigger and expression executes", func(t *testing.T) {
		atomic.StoreInt32(&executionCount, 0)

		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "user.created",
			Data: map[string]any{"type": "premium"},
		})
		require.NoError(t, err)
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.Equal(t, int32(1), atomic.LoadInt32(&executionCount))
			assert.Equal(t, "user.created", lastExecutedEvent)
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("matching trigger but failing expression does not execute", func(t *testing.T) {
		atomic.StoreInt32(&executionCount, 0)

		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "user.created",
			Data: map[string]any{"type": "standard"}, // Doesn't match premium requirement
		})
		require.NoError(t, err)
		require.Never(t, func() bool {
			return atomic.LoadInt32(&executionCount) != 0
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("non-matching trigger does not execute even with satisfying data", func(t *testing.T) {
		atomic.StoreInt32(&executionCount, 0)

		// This event has data that would satisfy user.created expression (type: premium)
		// but it's sent as user.updated, which requires type: standard
		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "user.updated",
			Data: map[string]any{"type": "premium"},
		})
		require.NoError(t, err)
		require.Never(t, func() bool {
			return atomic.LoadInt32(&executionCount) != 0
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("second trigger with matching expression executes", func(t *testing.T) {
		atomic.StoreInt32(&executionCount, 0)

		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "user.updated",
			Data: map[string]any{"type": "standard"},
		})
		require.NoError(t, err)

		require.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.Equal(t, int32(1), atomic.LoadInt32(&executionCount))
			assert.Equal(t, "user.updated", lastExecutedEvent)
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("trigger without expression matches any data", func(t *testing.T) {
		var noExprCount int32
		var lastExecutedEvent string

		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{ID: "mixed-expression-test"},
			inngestgo.MultipleTriggers{
				inngestgo.EventTrigger("order.created", inngestgo.StrPtr("event.data.amount > 100")),
				inngestgo.EventTrigger("order.cancelled", nil), // No expression
			},
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				atomic.AddInt32(&noExprCount, 1)
				lastExecutedEvent = input.Event.Name
				return nil, nil
			},
		)
		require.NoError(t, err)
		registerFuncs()

		// Test that order.cancelled executes regardless of data
		atomic.StoreInt32(&noExprCount, 0)
		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "order.cancelled",
			Data: map[string]any{"amount": 50}, // Would fail order.created expression, but should still execute
		})
		require.NoError(t, err)
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.Equal(t, int32(1), atomic.LoadInt32(&noExprCount))
			assert.Equal(t, "order.cancelled", lastExecutedEvent)
		}, 5*time.Second, 100*time.Millisecond)

		// Test that order.created with insufficient amount does NOT execute
		atomic.StoreInt32(&noExprCount, 0)
		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "order.created",
			Data: map[string]any{"amount": 50}, // Fails expression requirement
		})

		require.NoError(t, err)
		require.Never(t, func() bool {
			return atomic.LoadInt32(&noExprCount) != 0
		}, 5*time.Second, 100*time.Millisecond)
	})

	t.Run("wildcard triggers work correctly", func(t *testing.T) {
		var wildcardCount int32
		var lastExecutedEvent string

		_, err := inngestgo.CreateFunction(
			inngestClient,
			inngestgo.FunctionOpts{ID: "wildcard-test"},
			inngestgo.EventTrigger("foo/*", nil), // Wildcard trigger
			func(ctx context.Context, input inngestgo.Input[any]) (any, error) {
				atomic.AddInt32(&wildcardCount, 1)
				lastExecutedEvent = input.Event.Name
				return nil, nil
			},
		)
		require.NoError(t, err)
		registerFuncs()

		// Test that foo/1 triggers the wildcard
		atomic.StoreInt32(&wildcardCount, 0)
		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "foo/1",
			Data: map[string]any{"test": "data"},
		})
		require.NoError(t, err)
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.Equal(t, int32(1), atomic.LoadInt32(&wildcardCount))
			assert.Equal(t, "foo/1", lastExecutedEvent)
		}, 5*time.Second, 100*time.Millisecond)

		// Test that foo/bar also triggers the wildcard
		atomic.StoreInt32(&wildcardCount, 0)
		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "foo/bar",
			Data: map[string]any{"test": "data"},
		})
		require.NoError(t, err)
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			assert.Equal(t, int32(1), atomic.LoadInt32(&wildcardCount))
			assert.Equal(t, "foo/bar", lastExecutedEvent)
		}, 5*time.Second, 100*time.Millisecond)

		// Test that bar/1 does NOT trigger the wildcard
		atomic.StoreInt32(&wildcardCount, 0)
		_, err = inngestClient.Send(ctx, inngestgo.Event{
			Name: "bar/1",
			Data: map[string]any{"test": "data"},
		})
		require.NoError(t, err)
		require.Never(t, func() bool {
			return atomic.LoadInt32(&wildcardCount) != 0
		}, 5*time.Second, 100*time.Millisecond)
	})
}
