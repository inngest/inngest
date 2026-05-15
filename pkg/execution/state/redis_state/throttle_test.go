package redis_state

import (
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestOutdatedThrottle(t *testing.T) {
	cases := []struct {
		name       string
		constraint *osqueue.PartitionThrottle
		item       *osqueue.Throttle
		expected   enums.OutdatedThrottleReason
	}{
		{
			name:       "no throttle",
			constraint: nil,
			item:       nil,
			expected:   enums.OutdatedThrottleReasonNone,
		},
		{
			name: "missing key expression hash (old item)",
			constraint: &osqueue.PartitionThrottle{
				ThrottleKeyExpressionHash: util.XXHash("event.data.customerID"),
				Limit:                     10,
				Burst:                     1,
				Period:                    60,
			},
			item: &osqueue.Throttle{
				Key:               util.XXHash("customer1"),
				KeyExpressionHash: "", // old item; empty key expression hash
				Limit:             10,
				Burst:             1,
				Period:            60,
			},
			expected: enums.OutdatedThrottleReasonMissingKeyExpressionHash,
		},
		{
			name: "added throttle",
			constraint: &osqueue.PartitionThrottle{
				ThrottleKeyExpressionHash: util.XXHash("event.data.customerID"),
				Limit:                     10,
				Burst:                     1,
				Period:                    60,
			},
			item:     nil,
			expected: enums.OutdatedThrottleReasonMissingItemThrottle,
		},
		{
			name:       "removed throttle",
			constraint: nil,
			item: &osqueue.Throttle{
				Key:               util.XXHash("user1"),
				KeyExpressionHash: "", // old item; empty key expression hash
				Limit:             10,
				Burst:             1,
				Period:            60,
			},
			expected: enums.OutdatedThrottleReasonMissingConstraint,
		},
		{
			name: "changed throttle key",
			constraint: &osqueue.PartitionThrottle{
				ThrottleKeyExpressionHash: util.XXHash("event.data.customerID"),
				Limit:                     10,
				Burst:                     1,
				Period:                    60,
			},
			item: &osqueue.Throttle{
				Key:               util.XXHash("user1"),
				KeyExpressionHash: util.XXHash("event.data.userID"), // different key!
				Limit:             10,
				Burst:             1,
				Period:            60,
			},
			expected: enums.OutdatedThrottleReasonKeyExpressionMismatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			constraints := osqueue.PartitionConstraintConfig{
				Throttle: tc.constraint,
			}
			item := osqueue.QueueItem{
				Data: osqueue.Item{
					Kind:     osqueue.KindStart,
					Throttle: tc.item,
				},
			}
			want := tc.expected
			got := constraints.HasOutdatedThrottle(item)

			require.Equal(t, want, got)
		})
	}
}
