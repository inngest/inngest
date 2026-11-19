package sdk

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/inngest"
	"github.com/stretchr/testify/require"
)

func TestSDKFunctionLoop(t *testing.T) {
	loop := &inngest.Loop{
		Kind:    "loop",
		Key:     "event.data.id",
		Tick:    30 * time.Second,
		IdleTTL: time.Hour,
	}

	s := SDKFunction{
		Name: "loop",
		Triggers: []inngest.Trigger{
			{
				EventTrigger: &inngest.EventTrigger{
					Event: "loop/start",
				},
			},
		},
		Loop: loop,
		Steps: map[string]SDKStep{
			"step": {
				ID:   "step",
				Name: "step",
				Runtime: map[string]any{
					"url": "https://www.example.net/api/inngest",
				},
			},
		},
	}

	fn, err := s.Function()
	require.NoError(t, err)
	require.Equal(t, loop, fn.Loop)

	s.Loop = nil
	fn, err = s.Function()
	require.NoError(t, err)
	require.Nil(t, fn.Loop)
}
