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
			expectedSojourn: 5 * time.Second, // 0 -> 5
		},
		{
			name: "expected to match old sojourn time",
			qi: QueueItem{
				AtMS:             time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				WallTimeMS:       time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				EarliestPeekTime: time.Date(2025, 6, 11, 15, 7, 6, 0, time.Local).UnixMilli(),
			},
			now:             time.Date(2025, 6, 11, 15, 7, 9, 0, time.Local),
			expectedLatency: 5 * time.Second, // 5s between enqueue and first peek
			expectedSojourn: 3 * time.Second, // 3s between first peek and now (processing)
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expectedLatency, test.qi.Latency(test.now))
			require.Equal(t, test.expectedSojourn, test.qi.SojournLatency(test.now))
		})
	}
}
