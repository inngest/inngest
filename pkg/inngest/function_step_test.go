package inngest

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/require"
)

func TestGetRetryDelay(t *testing.T) {
	t.Run("returns nil when RetryDelay is not set", func(t *testing.T) {
		s := Step{}
		dur, err := s.GetRetryDelay()
		require.NoError(t, err)
		require.Nil(t, dur)
	})

	t.Run("parses a valid duration string", func(t *testing.T) {
		delay := "5m"
		s := Step{RetryDelay: &delay}
		dur, err := s.GetRetryDelay()
		require.NoError(t, err)
		require.NotNil(t, dur)
		require.Equal(t, 5*time.Minute, *dur)
	})

	t.Run("clamps to MinRetryDuration", func(t *testing.T) {
		delay := "100ms"
		s := Step{RetryDelay: &delay}
		dur, err := s.GetRetryDelay()
		require.NoError(t, err)
		require.NotNil(t, dur)
		require.Equal(t, consts.MinRetryDuration, *dur)
	})

	t.Run("clamps to MaxRetryDuration", func(t *testing.T) {
		delay := "48h"
		s := Step{RetryDelay: &delay}
		dur, err := s.GetRetryDelay()
		require.NoError(t, err)
		require.NotNil(t, dur)
		require.Equal(t, consts.MaxRetryDuration, *dur)
	})

	t.Run("returns error for invalid duration string", func(t *testing.T) {
		delay := "not-a-duration"
		s := Step{RetryDelay: &delay}
		dur, err := s.GetRetryDelay()
		require.Error(t, err)
		require.Nil(t, dur)
	})
}
