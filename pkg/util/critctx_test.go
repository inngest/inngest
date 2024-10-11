package util

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestCrit(t *testing.T) {
	bg := context.Background()

	t.Run("Plain ol contexts work", func(t *testing.T) {
		called := false
		err := Crit(bg, "foo", func(ctx context.Context) error {
			called = true
			return nil
		})
		require.True(t, called)
		require.Nil(t, err)
	})

	t.Run("Errors are passed back", func(t *testing.T) {
		called := false
		expectedErr := fmt.Errorf("no way")
		err := Crit(bg, "foo", func(ctx context.Context) error {
			called = true
			return expectedErr
		})
		require.True(t, called)
		require.Equal(t, err, expectedErr)
	})

	t.Run("With a context cancelled during execution", func(t *testing.T) {
		ctx, cancel := context.WithCancel(bg)

		go func() {
			<-time.After(10 * time.Millisecond)
			cancel()
		}()

		called := false
		expectedErr := fmt.Errorf("no way")
		err := Crit(ctx, "foo", func(ctx context.Context) error {
			<-time.After(20 * time.Millisecond)

			if ctx.Err() != nil {
				// Return the context cancelled error.
				return ctx.Err()
			}

			called = true
			return expectedErr
		})
		require.True(t, called)
		require.Equal(t, err, expectedErr)
	})

	t.Run("It should prevent the crit from running with a short deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(bg, 10*time.Millisecond)
		defer cancel()

		called := false
		err := Crit(ctx, "foo", func(ctx context.Context) error {
			if ctx.Err() != nil {
				// Return the context cancelled error.
				return ctx.Err()
			}
			called = true
			return nil
		}, time.Second)

		// Not called:  deadline too short.
		require.False(t, called)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "context deadline shorter than critical bounds")
	})

	t.Run("It should warn if the crit takes longer than ideal bounds", func(t *testing.T) {
		called := false

		buf := bytes.NewBuffer(nil)
		log := slog.New(slog.NewJSONHandler(buf, nil))
		ctx := logger.WithStdlib(bg, log)

		err := Crit(ctx, "foo", func(ctx context.Context) error {
			<-time.After(10 * time.Millisecond)
			if ctx.Err() != nil {
				// Return the context cancelled error.
				return ctx.Err()
			}
			called = true
			return nil
		}, time.Millisecond)

		require.True(t, called)
		require.Nil(t, err)

		require.Contains(t, string(buf.Bytes()), "critical section took longer than boundaries")
	})

	t.Run("With a cancelled context, the fn fails", func(t *testing.T) {
		ctx, cancel := context.WithCancel(bg)
		cancel()

		called := false
		err := Crit(ctx, "foo", func(ctx context.Context) error {
			called = true
			return nil
		})

		require.False(t, called)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "context canceled before entering crit")
	})

}
