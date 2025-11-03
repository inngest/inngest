package queue

import (
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/constraintapi"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestConvertToConstraintConfiguration(t *testing.T) {
	tests := []struct {
		name               string
		accountConcurrency int
		fn                 inngest.Function
		expected           constraintapi.ConstraintConfig
	}{
		{
			name:               "minimal function",
			accountConcurrency: 100,
			fn: inngest.Function{
				FunctionVersion: 1,
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 1,
				RateLimit:       nil,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:    100,
					FunctionConcurrency:   0,
					CustomConcurrencyKeys: nil,
				},
				Throttle: nil,
			},
		},
		{
			name:               "function with rate limit",
			accountConcurrency: 50,
			fn: inngest.Function{
				FunctionVersion: 2,
				RateLimit: &inngest.RateLimit{
					Limit:  10,
					Period: "60s",
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 2,
				RateLimit: []constraintapi.RateLimitConfig{
					{
						Scope:             enums.RateLimitScopeFn,
						Limit:             10,
						KeyExpressionHash: util.XXHash(""),
					},
				},
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:    50,
					FunctionConcurrency:   0,
					CustomConcurrencyKeys: nil,
				},
				Throttle: nil,
			},
		},
		{
			name:               "function with rate limit and key",
			accountConcurrency: 50,
			fn: inngest.Function{
				FunctionVersion: 2,
				RateLimit: &inngest.RateLimit{
					Limit:  10,
					Period: "60s",
					Key:    stringPtr("event.user.id"),
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 2,
				RateLimit: []constraintapi.RateLimitConfig{
					{
						Scope:             enums.RateLimitScopeFn,
						Limit:             10,
						KeyExpressionHash: util.XXHash("event.user.id"),
					},
				},
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:    50,
					FunctionConcurrency:   0,
					CustomConcurrencyKeys: nil,
				},
				Throttle: nil,
			},
		},
		{
			name:               "function with basic concurrency",
			accountConcurrency: 100,
			fn: inngest.Function{
				FunctionVersion: 3,
				Concurrency: &inngest.ConcurrencyLimits{
					Limits: []inngest.Concurrency{
						{
							Limit: 5,
							Scope: enums.ConcurrencyScopeFn,
						},
					},
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 3,
				RateLimit:       nil,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:    100,
					FunctionConcurrency:   5,
					CustomConcurrencyKeys: nil,
				},
				Throttle: nil,
			},
		},
		{
			name:               "function with custom concurrency limits",
			accountConcurrency: 200,
			fn: inngest.Function{
				FunctionVersion: 4,
				Concurrency: &inngest.ConcurrencyLimits{
					Limits: []inngest.Concurrency{
						{
							Limit: 10,
							Scope: enums.ConcurrencyScopeFn,
						},
						{
							Limit: 3,
							Key:   stringPtr("event.user.id"),
							Scope: enums.ConcurrencyScopeAccount,
							Hash:  "user-key-hash",
						},
						{
							Limit: 2,
							Key:   stringPtr("event.organization.id"),
							Scope: enums.ConcurrencyScopeEnv,
							Hash:  "org-key-hash",
						},
					},
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 4,
				RateLimit:       nil,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  200,
					FunctionConcurrency: 10,
					CustomConcurrencyKeys: []constraintapi.CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeAccount,
							Limit:             3,
							KeyExpressionHash: "user-key-hash",
						},
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeEnv,
							Limit:             2,
							KeyExpressionHash: "org-key-hash",
						},
					},
				},
				Throttle: nil,
			},
		},
		{
			name:               "function with throttle",
			accountConcurrency: 100,
			fn: inngest.Function{
				FunctionVersion: 5,
				Throttle: &inngest.Throttle{
					Limit:  20,
					Burst:  5,
					Period: 60 * time.Second,
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 5,
				RateLimit:       nil,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:    100,
					FunctionConcurrency:   0,
					CustomConcurrencyKeys: nil,
				},
				Throttle: []constraintapi.ThrottleConfig{
					{
						Limit:                     20,
						Burst:                     5,
						Period:                    60,
						Scope:                     enums.ThrottleScopeFn,
						ThrottleKeyExpressionHash: util.XXHash(""),
					},
				},
			},
		},
		{
			name:               "function with throttle and key",
			accountConcurrency: 100,
			fn: inngest.Function{
				FunctionVersion: 6,
				Throttle: &inngest.Throttle{
					Limit:  15,
					Burst:  3,
					Period: 30 * time.Second,
					Key:    stringPtr("event.tenant.id"),
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 6,
				RateLimit:       nil,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:    100,
					FunctionConcurrency:   0,
					CustomConcurrencyKeys: nil,
				},
				Throttle: []constraintapi.ThrottleConfig{
					{
						Limit:                     15,
						Burst:                     3,
						Period:                    30,
						Scope:                     enums.ThrottleScopeFn,
						ThrottleKeyExpressionHash: util.XXHash("event.tenant.id"),
					},
				},
			},
		},
		{
			name:               "complete function configuration",
			accountConcurrency: 500,
			fn: inngest.Function{
				FunctionVersion: 7,
				RateLimit: &inngest.RateLimit{
					Limit:  25,
					Period: "120s",
					Key:    stringPtr("event.api_key"),
				},
				Concurrency: &inngest.ConcurrencyLimits{
					Limits: []inngest.Concurrency{
						{
							Limit: 20,
							Scope: enums.ConcurrencyScopeFn,
						},
						{
							Limit: 5,
							Key:   stringPtr("event.user.id"),
							Scope: enums.ConcurrencyScopeAccount,
							Hash:  "complete-user-hash",
						},
					},
				},
				Throttle: &inngest.Throttle{
					Limit:  50,
					Burst:  10,
					Period: 90 * time.Second,
					Key:    stringPtr("event.organization.slug"),
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 7,
				RateLimit: []constraintapi.RateLimitConfig{
					{
						Scope:             enums.RateLimitScopeFn,
						Limit:             25,
						KeyExpressionHash: util.XXHash("event.api_key"),
					},
				},
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:  500,
					FunctionConcurrency: 20,
					CustomConcurrencyKeys: []constraintapi.CustomConcurrencyLimit{
						{
							Mode:              enums.ConcurrencyModeStep,
							Scope:             enums.ConcurrencyScopeAccount,
							Limit:             5,
							KeyExpressionHash: "complete-user-hash",
						},
					},
				},
				Throttle: []constraintapi.ThrottleConfig{
					{
						Limit:                     50,
						Burst:                     10,
						Period:                    90,
						Scope:                     enums.ThrottleScopeFn,
						ThrottleKeyExpressionHash: util.XXHash("event.organization.slug"),
					},
				},
			},
		},
		{
			name:               "empty account concurrency",
			accountConcurrency: 0,
			fn: inngest.Function{
				FunctionVersion: 8,
				Concurrency: &inngest.ConcurrencyLimits{
					Limits: []inngest.Concurrency{
						{
							Limit: 1,
							Scope: enums.ConcurrencyScopeFn,
						},
					},
				},
			},
			expected: constraintapi.ConstraintConfig{
				FunctionVersion: 8,
				RateLimit:       nil,
				Concurrency: constraintapi.ConcurrencyConfig{
					AccountConcurrency:    0,
					FunctionConcurrency:   1,
					CustomConcurrencyKeys: nil,
				},
				Throttle: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToConstraintConfiguration(tt.accountConcurrency, tt.fn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
