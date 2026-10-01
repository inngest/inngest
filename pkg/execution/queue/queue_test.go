package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type retryableError struct {
	error
	retry bool
}

func TestPartitionBacklogSizeConcurrencyOption(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		opts := NewQueueOptions()
		require.Equal(t, defaultPartitionBacklogSizeConcurrency, opts.PartitionBacklogSizeConcurrency())
	})

	t.Run("custom", func(t *testing.T) {
		opts := NewQueueOptions(WithPartitionBacklogSizeConcurrency(7))
		require.Equal(t, int64(7), opts.PartitionBacklogSizeConcurrency())
	})

	t.Run("invalid falls back to default", func(t *testing.T) {
		opts := NewQueueOptions(WithPartitionBacklogSizeConcurrency(0))
		require.Equal(t, defaultPartitionBacklogSizeConcurrency, opts.PartitionBacklogSizeConcurrency())

		opts = NewQueueOptions(WithPartitionBacklogSizeConcurrency(-1))
		require.Equal(t, defaultPartitionBacklogSizeConcurrency, opts.PartitionBacklogSizeConcurrency())
	})
}

func TestSemaphoreRequeueExtensionOption(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		opts := NewQueueOptions()
		require.Equal(t, PartitionSemaphoreLimitRequeueExtension, opts.SemaphoreRequeueExtension())
	})

	t.Run("custom", func(t *testing.T) {
		opts := NewQueueOptions(WithSemaphoreRequeueExtension(2 * time.Second))
		require.Equal(t, 2*time.Second, opts.SemaphoreRequeueExtension())
	})

	t.Run("non-positive falls back to default", func(t *testing.T) {
		opts := NewQueueOptions(WithSemaphoreRequeueExtension(0))
		require.Equal(t, PartitionSemaphoreLimitRequeueExtension, opts.SemaphoreRequeueExtension())

		opts = NewQueueOptions(WithSemaphoreRequeueExtension(-time.Second))
		require.Equal(t, PartitionSemaphoreLimitRequeueExtension, opts.SemaphoreRequeueExtension())
	})
}

func (r retryableError) Retryable() bool {
	return r.retry
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		err      error
		att      int
		max      int
		expected bool
	}{
		{
			fmt.Errorf("basic err retries"),
			1,
			5,
			true,
		},
		{
			fmt.Errorf("basic err fails at max attempts"),
			2, // 0, 1, 2 - off by one from zero index.
			3,
			false,
		},
		{
			retryableError{error: fmt.Errorf("retries if returns true"), retry: true},
			1,
			5,
			true,
		},
		{
			retryableError{error: fmt.Errorf("doesnt retry at max"), retry: true},
			5,
			5,
			false,
		},
		{
			retryableError{error: fmt.Errorf("doesnt retry if Retryable returns false"), retry: false},
			1,
			5,
			false,
		},
		{
			alwaysRetry{error: fmt.Errorf("always even if over max")},
			10,
			5,
			true,
		},
	}

	for _, test := range tests {
		actual := ShouldRetry(test.err, test.att, test.max)
		require.Equal(t, test.expected, actual)
	}
}

func TestAsRetryAt(t *testing.T) {
	now := time.Now()
	base := RetryAtError(fmt.Errorf("lol"), &now)
	wrapped := fmt.Errorf("wrap: %w", base)

	require.NotNil(t, AsRetryAtError(base))
	require.NotNil(t, AsRetryAtError(wrapped))
	require.Nil(t, AsRetryAtError(fmt.Errorf("no")))
}

func TestPartitionPeekLimit(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value int64
		want  int64
	}{
		{name: "custom", value: 750, want: 750},
		{name: "below minimum", value: 1, want: PartitionSelectionMax},
		{name: "just below minimum", value: PartitionSelectionMax - 1, want: PartitionSelectionMax},
		{name: "minimum", value: PartitionSelectionMax, want: PartitionSelectionMax},
		{name: "cap", value: 2000, want: AbsolutePartitionPeekMax},
		{name: "zero", value: 0, want: PartitionPeekMax},
		{name: "negative", value: -1, want: PartitionPeekMax},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewQueueOptions(WithPartitionPeekMaxGetter(func(ctx context.Context, shard string) int64 {
				require.Equal(t, "ss3", shard)
				return tt.value
			}))
			require.Equal(t, tt.want, opts.PartitionPeekLimit(context.Background(), "ss3"))
		})
	}
	t.Run("nil getter uses default", func(t *testing.T) {
		opts := NewQueueOptions(WithPartitionPeekMaxGetter(nil))
		require.Equal(t, int64(PartitionPeekMax), opts.PartitionPeekLimit(context.Background(), "ss3"))
	})
	t.Run("default", func(t *testing.T) {
		require.Equal(t, int64(PartitionPeekMax), NewQueueOptions().PartitionPeekLimit(context.Background(), "ss3"))
	})
	t.Run("runtime updates and shard targeting", func(t *testing.T) {
		limits := map[string]int64{"ss3": 750, "ss6": 500}
		opts := NewQueueOptions(WithPartitionPeekMaxGetter(func(_ context.Context, shard string) int64 { return limits[shard] }))
		require.Equal(t, int64(750), opts.PartitionPeekLimit(context.Background(), "ss3"))
		require.Equal(t, int64(500), opts.PartitionPeekLimit(context.Background(), "ss6"))
		limits["ss3"] = 1000
		require.Equal(t, int64(1000), opts.PartitionPeekLimit(context.Background(), "ss3"))
		require.Equal(t, int64(PartitionPeekMax), opts.PartitionPeekLimit(context.Background(), "other"))
	})
}
