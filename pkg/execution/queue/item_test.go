package queue

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLatency(t *testing.T) {
	type tableTest struct {
		name string

		now time.Time

		qi QueueItem

		expectedLatency time.Duration
		expectedSojourn time.Duration
	}

	tests := []tableTest{
		{
			name: "expected time between refill and lease",
			qi: QueueItem{
				EnqueuedAt:       time.Date(2025, 6, 11, 15, 7, 0, 0, time.Local).UnixMilli(),
				AtMS:             time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				WallTimeMS:       time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				RefilledFrom:     "backlog-1",
				RefilledAt:       time.Date(2025, 6, 11, 15, 7, 5, 0, time.Local).UnixMilli(),
				EarliestPeekTime: time.Date(2025, 6, 11, 15, 7, 8, 0, time.Local).UnixMilli(),
			},
			now:             time.Date(2025, 6, 11, 15, 7, 9, 0, time.Local),
			expectedLatency: 4 * time.Second, // 5 -> 9
		},
		{
			name: "expected sojourn delay to match",
			qi: QueueItem{
				EnqueuedAt:       time.Date(2025, 6, 11, 15, 7, 0, 0, time.Local).UnixMilli(),
				AtMS:             time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				WallTimeMS:       time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				RefilledFrom:     "backlog-1",
				RefilledAt:       time.Date(2025, 6, 11, 15, 7, 5, 0, time.Local).UnixMilli(),
				EarliestPeekTime: time.Date(2025, 6, 11, 15, 7, 8, 0, time.Local).UnixMilli(),
			},
			now:             time.Date(2025, 6, 11, 15, 7, 9, 0, time.Local),
			expectedSojourn: 5 * time.Second, // 0 -> 5
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expectedLatency, test.qi.Latency(test.now))
			require.Equal(t, test.expectedSojourn, test.qi.SojournLatency(test.now))
		})
	}
}
