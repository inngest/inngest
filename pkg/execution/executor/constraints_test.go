package executor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestRateLimitKeyExpressionHashConsistency(t *testing.T) {
	ptr := func(s string) *string { return &s }

	tests := []struct {
		name                      string
		rateLimitKey              *string
		expectedKeyExpressionHash string
	}{
		{
			name:                      "with key expression",
			rateLimitKey:              ptr("event.data.userId"),
			expectedKeyExpressionHash: util.XXHash("event.data.userId"),
		},
		{
			name:                      "without key expression",
			rateLimitKey:              nil,
			expectedKeyExpressionHash: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fnID := uuid.New()
			fn := inngest.Function{
				ID: fnID,
				RateLimit: &inngest.RateLimit{
					Limit:  1,
					Period: "1m",
					Key:    tt.rateLimitKey,
				},
			}

			// Get KeyExpressionHash from ConvertToConstraintConfiguration
			config, err := queue.ConvertToConstraintConfiguration(0, fn)
			require.NoError(t, err)
			require.Len(t, config.RateLimit, 1)
			configHash := config.RateLimit[0].KeyExpressionHash

			// Get KeyExpressionHash from getScheduleConstraints
			req := execution.ScheduleRequest{
				Function: fn,
				Events: []event.TrackedEvent{
					event.InternalEvent{
						Event: event.Event{
							Name: "test",
							Data: map[string]any{"userId": "test-user"},
						},
					},
				},
			}

			constraints, err := getScheduleConstraints(context.Background(), req)
			require.NoError(t, err)
			require.Len(t, constraints, 1)
			constraintHash := constraints[0].RateLimit.KeyExpressionHash

			// Both must match each other and the expected value
			require.Equal(t, tt.expectedKeyExpressionHash, configHash, "config KeyExpressionHash mismatch")
			require.Equal(t, tt.expectedKeyExpressionHash, constraintHash, "constraint KeyExpressionHash mismatch")
			require.Equal(t, configHash, constraintHash, "config and constraint KeyExpressionHash must be equal")
		})
	}
}
