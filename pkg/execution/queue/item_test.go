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

func TestDelay(t *testing.T) {
	type tableTest struct {
		name string

		now time.Time

		qi QueueItem

		expectedExpectedDelay time.Duration
		expectedRefillDelay   time.Duration
		expectedLeaseDelay    time.Duration
	}

	tests := []tableTest{
		{
			name: "old item should not have delays",
			qi: QueueItem{
				EnqueuedAt:   0,
				AtMS:         time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				WallTimeMS:   time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				RefilledFrom: "",
				RefilledAt:   0,
			},
			now:                   time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local),
			expectedExpectedDelay: 0,
			expectedRefillDelay:   0,
			expectedLeaseDelay:    0,
		},

		{
			name: "expected delay should never be negative",
			qi: QueueItem{
				AtMS:       time.Date(2025, 6, 11, 15, 7, 0, 0, time.Local).UnixMilli(),
				WallTimeMS: time.Date(2025, 6, 11, 15, 7, 0, 0, time.Local).UnixMilli(),
				EnqueuedAt: time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(), // enqueued 1s after expected AtMS
			},
			now:                   time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local),
			expectedExpectedDelay: 0,
			expectedRefillDelay:   0,
			expectedLeaseDelay:    0,
		},

		{
			name: "expected delay may be positive for future queue items",
			qi: QueueItem{
				EnqueuedAt: time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(), // enqueued 9s before expected AtMS
				AtMS:       time.Date(2025, 6, 11, 15, 7, 10, 0, time.Local).UnixMilli(),
				WallTimeMS: time.Date(2025, 6, 11, 15, 7, 10, 0, time.Local).UnixMilli(),
			},
			now:                   time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local),
			expectedExpectedDelay: 9 * time.Second,
			expectedRefillDelay:   0,
			expectedLeaseDelay:    0,
		},

		{
			name: "refill delay should be correct for items without delay",
			qi: QueueItem{
				EnqueuedAt: time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(), // enqueued at the same time as expected AtMS
				AtMS:       time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				WallTimeMS: time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),

				RefilledFrom: "backlog-1",
				RefilledAt:   time.Date(2025, 6, 11, 15, 7, 5, 0, time.Local).UnixMilli(), // 4s after enqueueing
			},
			now:                   time.Date(2025, 6, 11, 15, 7, 5, 0, time.Local),
			expectedExpectedDelay: 0,
			expectedRefillDelay:   4 * time.Second,
			expectedLeaseDelay:    0,
		},

		{
			name: "refill delay should ignore expected delay",
			qi: QueueItem{
				EnqueuedAt: time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(), // enqueued 9s before expected AtMS
				AtMS:       time.Date(2025, 6, 11, 15, 7, 10, 0, time.Local).UnixMilli(),
				WallTimeMS: time.Date(2025, 6, 11, 15, 7, 10, 0, time.Local).UnixMilli(),

				RefilledFrom: "backlog-1",
				RefilledAt:   time.Date(2025, 6, 11, 15, 7, 15, 0, time.Local).UnixMilli(), // 4s after enqueueing
			},
			now:                   time.Date(2025, 6, 11, 15, 7, 15, 0, time.Local),
			expectedExpectedDelay: 9 * time.Second,
			expectedRefillDelay:   14*time.Second - 9*time.Second,
			expectedLeaseDelay:    0,
		},

		{
			name: "lease delay should work",
			qi: QueueItem{
				EnqueuedAt: time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(), // enqueued at the same time as expected AtMS
				AtMS:       time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),
				WallTimeMS: time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(),

				RefilledFrom: "backlog-1",
				RefilledAt:   time.Date(2025, 6, 11, 15, 7, 1, 0, time.Local).UnixMilli(), // immediately refilled
			},
			now:                   time.Date(2025, 6, 11, 15, 7, 4, 0, time.Local),
			expectedExpectedDelay: 0,
			expectedRefillDelay:   0,
			expectedLeaseDelay:    3 * time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expectedExpectedDelay, test.qi.ExpectedDelay())
			require.Equal(t, test.expectedRefillDelay, test.qi.RefillDelay())
			require.Equal(t, test.expectedLeaseDelay, test.qi.LeaseDelay(test.now))
		})
	}
}
