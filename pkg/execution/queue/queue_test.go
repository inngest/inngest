package queue

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type retryableError struct {
	error
	retry bool
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
