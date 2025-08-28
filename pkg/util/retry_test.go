package util

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithRetry(t *testing.T) {
	ctx := context.Background()

	t.Run("retry function if errors with default settings", func(t *testing.T) {
		attempt := 1

		_, err := WithRetry(
			ctx,
			"test",
			func(ctx context.Context) (bool, error) {
				if attempt%3 == 0 {
					return true, nil
				}

				attempt += 1
				return false, fmt.Errorf("failed")
			},
			NewRetryConf(),
		)

		require.NoError(t, err)
		require.Equal(t, 3, attempt)
	})

	t.Run("retry functions if errors are included in config", func(t *testing.T) {
		attempt := 1
		ioErr := errors.New("io timeout")
		notCoveredErr := errors.New("error not covered!!")

		_, err := WithRetry(
			ctx,
			"test",
			func(ctx context.Context) (bool, error) {
				if attempt%3 == 0 {
					return false, notCoveredErr
				}
				attempt += 1

				return false, ioErr
			},
			NewRetryConf(
				WithRetryConfRetryableErrors(func(err error) bool {
					return errors.Is(err, ioErr)
				}),
			),
		)

		require.Error(t, err)
		require.ErrorIs(t, err, notCoveredErr)
		require.Equal(t, 3, attempt)
	})

	t.Run("does not retry when max attempts is 1 and returns unwrapped error", func(t *testing.T) {
		idemErr := errors.New("idempotent error")

		_, err := WithRetry(
			ctx,
			"test",
			func(ctx context.Context) (bool, error) {
				return false, idemErr
			},
			NewRetryConf(
				WithRetryConfRetryableErrors(func(err error) bool {
					return false
				}),
				WithRetryConfMaxAttempts(1),
			),
		)

		require.Error(t, err)

		// It should be exact equality
		require.Equal(t, idemErr, err)
	})
}
