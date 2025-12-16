package cron

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/redis/rueidis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateJitter(t *testing.T) {
	testcases := []struct {
		name      string
		min       time.Duration
		max       time.Duration
		expectMin time.Duration
		expectMax time.Duration
	}{
		{
			name:      "normal range 1-3 seconds",
			min:       1000 * time.Millisecond,
			max:       3000 * time.Millisecond,
			expectMin: 1000 * time.Millisecond,
			expectMax: 3000 * time.Millisecond,
		},
		{
			name:      "equal min and max",
			min:       2000 * time.Millisecond,
			max:       2000 * time.Millisecond,
			expectMin: 2000 * time.Millisecond,
			expectMax: 2000 * time.Millisecond,
		},
		{
			name:      "small range",
			min:       1000 * time.Millisecond,
			max:       1001 * time.Millisecond,
			expectMin: 1000 * time.Millisecond,
			expectMax: 1001 * time.Millisecond,
		},
		{
			name:      "min greater than max returns zero",
			min:       5000 * time.Millisecond,
			max:       1000 * time.Millisecond,
			expectMin: 0,
			expectMax: 0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.min > tc.max {
				// Special case: should return zero
				jitter := generateJitter(tc.min, tc.max)
				require.Equal(t, time.Duration(0), jitter)
				return
			}

			// Run multiple times to test range
			for range 50 {
				jitter := generateJitter(tc.min, tc.max)

				require.GreaterOrEqual(t, jitter, tc.expectMin, "jitter should be >= min")
				require.LessOrEqual(t, jitter, tc.expectMax, "jitter should be <= max")
			}

			// Test for some variation (unless min == max)
			if tc.min != tc.max {
				values := make(map[time.Duration]bool)
				for range 20 {
					jitter := generateJitter(tc.min, tc.max)
					values[jitter] = true
				}
				require.Greater(t, len(values), 1, "expected some variation in jitter values")
			}
		})
	}
}

func TestOptions(t *testing.T) {
	t.Run("WithJitterRange sets jitter correctly", func(t *testing.T) {
		opt := redisCronManagerOpt{}
		WithJitterRange(5*time.Millisecond, 15*time.Millisecond)(&opt)

		assert.Equal(t, 5*time.Millisecond, opt.jitterMin)
		assert.Equal(t, 15*time.Millisecond, opt.jitterMax)
	})

	t.Run("WithJitterRange ignores invalid range", func(t *testing.T) {
		opt := redisCronManagerOpt{}
		WithJitterRange(15*time.Millisecond, 5*time.Millisecond)(&opt)

		assert.Equal(t, time.Duration(0), opt.jitterMin)
		assert.Equal(t, time.Duration(0), opt.jitterMax)
	})

	t.Run("WithHealthCheckInterval sets interval correctly", func(t *testing.T) {
		opt := redisCronManagerOpt{}
		WithHealthCheckInterval(5 * time.Minute)(&opt)

		assert.Equal(t, 5*time.Minute, opt.healthCheckInterval)
	})

	t.Run("WithHealthCheckInterval ignores values < 1m", func(t *testing.T) {
		opt := redisCronManagerOpt{}
		WithHealthCheckInterval(10 * time.Second)(&opt)

		assert.Equal(t, time.Duration(0), opt.healthCheckInterval)
	})

	t.Run("WithHealthCheckLeadTimeSeconds sets lead time correctly", func(t *testing.T) {
		opt := redisCronManagerOpt{}
		WithHealthCheckLeadTimeSeconds(20)(&opt)

		assert.Equal(t, 20, opt.healthCheckLeadTimeSeconds)
	})

	t.Run("WithHealthCheckLeadTimeSeconds rejects negative values", func(t *testing.T) {
		opt := redisCronManagerOpt{}
		WithHealthCheckLeadTimeSeconds(-20)(&opt)

		assert.Equal(t, 0, opt.healthCheckLeadTimeSeconds)
	})

	t.Run("validate options", func(t *testing.T) {
		t.Run("valid configuration should not modify values", func(t *testing.T) {
			opt := redisCronManagerOpt{
				healthCheckLeadTimeSeconds: 19,
				healthCheckInterval:        time.Minute,
			}
			opt.validate()

			assert.Equal(t, 19, opt.healthCheckLeadTimeSeconds)
			assert.Equal(t, time.Minute, opt.healthCheckInterval)
		})

		t.Run("lead time equal to interval should reset to default", func(t *testing.T) {
			opt := redisCronManagerOpt{
				healthCheckLeadTimeSeconds: 60,
				healthCheckInterval:        time.Minute,
			}
			opt.validate()

			assert.Equal(t, defaultHealthCheckLeadTimeSeconds, opt.healthCheckLeadTimeSeconds)
			assert.Equal(t, time.Minute, opt.healthCheckInterval)
		})

		t.Run("lead time greater than interval should reset to default", func(t *testing.T) {
			opt := redisCronManagerOpt{
				healthCheckLeadTimeSeconds: 120,
				healthCheckInterval:        time.Minute,
			}
			opt.validate()

			assert.Equal(t, defaultHealthCheckLeadTimeSeconds, opt.healthCheckLeadTimeSeconds)
			assert.Equal(t, time.Minute, opt.healthCheckInterval)
		})

		t.Run("lead time one second less than interval should be valid", func(t *testing.T) {
			opt := redisCronManagerOpt{
				healthCheckLeadTimeSeconds: 59,
				healthCheckInterval:        time.Minute,
			}
			opt.validate()

			assert.Equal(t, 59, opt.healthCheckLeadTimeSeconds)
			assert.Equal(t, time.Minute, opt.healthCheckInterval)
		})

		t.Run("zero lead time should be valid", func(t *testing.T) {
			opt := redisCronManagerOpt{
				healthCheckLeadTimeSeconds: 0,
				healthCheckInterval:        time.Minute,
			}
			opt.validate()

			assert.Equal(t, 0, opt.healthCheckLeadTimeSeconds)
			assert.Equal(t, time.Minute, opt.healthCheckInterval)
		})

		t.Run("works with different intervals", func(t *testing.T) {
			testCases := []struct {
				name                    string
				leadTimeSeconds         int
				interval                time.Duration
				expectedLeadTimeSeconds int
			}{
				{
					name:                    "5 minute interval with valid lead time",
					leadTimeSeconds:         120,
					interval:                5 * time.Minute,
					expectedLeadTimeSeconds: 120,
				},
				{
					name:                    "5 minute interval with lead time equal to interval",
					leadTimeSeconds:         300,
					interval:                5 * time.Minute,
					expectedLeadTimeSeconds: defaultHealthCheckLeadTimeSeconds,
				},
				{
					name:                    "5 minute interval with lead time greater than interval",
					leadTimeSeconds:         400,
					interval:                5 * time.Minute,
					expectedLeadTimeSeconds: defaultHealthCheckLeadTimeSeconds,
				},
				{
					name:                    "1 hour interval with valid lead time",
					leadTimeSeconds:         1800,
					interval:                time.Hour,
					expectedLeadTimeSeconds: 1800,
				},
				{
					name:                    "1 hour interval with lead time equal to interval",
					leadTimeSeconds:         3600,
					interval:                time.Hour,
					expectedLeadTimeSeconds: defaultHealthCheckLeadTimeSeconds,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					opt := redisCronManagerOpt{
						healthCheckLeadTimeSeconds: tc.leadTimeSeconds,
						healthCheckInterval:        tc.interval,
					}
					opt.validate()

					assert.Equal(t, tc.expectedLeadTimeSeconds, opt.healthCheckLeadTimeSeconds)
					assert.Equal(t, tc.interval, opt.healthCheckInterval)
				})
			}
		})
	})
}

func TestJitterEdgeCases(t *testing.T) {
	t.Run("zero duration range", func(t *testing.T) {
		jitter := generateJitter(0, 0)
		assert.Equal(t, time.Duration(0), jitter)
	})

	t.Run("very small range", func(t *testing.T) {
		min := 1 * time.Nanosecond
		max := 2 * time.Nanosecond

		for range 100 {
			jitter := generateJitter(min, max)
			assert.True(t, jitter >= min && jitter <= max,
				"jitter %v should be between %v and %v", jitter, min, max)
		}
	})

	t.Run("large range", func(t *testing.T) {
		min := 1 * time.Second
		max := 1 * time.Hour

		for range 10 {
			jitter := generateJitter(min, max)
			assert.True(t, jitter >= min && jitter <= max,
				"jitter %v should be between %v and %v", jitter, min, max)
		}
	})
}

func TestNextHealthCheckTime(t *testing.T) {
	ctx := context.Background()
	r, rc := initRedis(t)
	defer rc.Close()

	defaultShard := redis_state.QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), Name: consts.DefaultQueueShardName}
	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))

	q := redis_state.NewQueue(
		defaultShard,
		redis_state.WithClock(clock),
	)

	t.Run("with default interval (1 minute) and default lead time (20 seconds)", func(t *testing.T) {
		cm := NewRedisCronManager(q, logger.StdlibLogger(ctx))
		cmTyped := cm.(*redisCronManager)

		testCases := []struct {
			name           string
			inputTime      time.Time
			expectedMinute int
			expectedSecond int
		}{
			{
				name:           "from middle of minute",
				inputTime:      time.Date(2025, 10, 26, 14, 30, 45, 0, time.UTC),
				expectedMinute: 31,
				expectedSecond: 40,
			},
			{
				name:           "from start of minute",
				inputTime:      time.Date(2025, 10, 26, 14, 30, 0, 0, time.UTC),
				expectedMinute: 30,
				expectedSecond: 40,
			},
			{
				name:           "from just before next slot",
				inputTime:      time.Date(2025, 10, 26, 14, 30, 39, 0, time.UTC),
				expectedMinute: 30,
				expectedSecond: 40,
			},
			{
				name:           "from exactly at slot time",
				inputTime:      time.Date(2025, 10, 26, 14, 30, 40, 0, time.UTC),
				expectedMinute: 31,
				expectedSecond: 40,
			},
			{
				name:           "from just after slot time",
				inputTime:      time.Date(2025, 10, 26, 14, 30, 42, 0, time.UTC),
				expectedMinute: 31,
				expectedSecond: 40,
			},
			{
				name:           "near end of hour",
				inputTime:      time.Date(2025, 10, 26, 14, 59, 50, 0, time.UTC),
				expectedMinute: 0,
				expectedSecond: 40,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				next := cmTyped.nextHealthCheckTime(tc.inputTime)

				// Should always be after input time
				assert.True(t, next.After(tc.inputTime),
					"next time %v should be after input time %v", next, tc.inputTime)

				assert.Equal(t, tc.expectedMinute, next.Minute(),
					"expected minute %d, got %d", tc.expectedMinute, next.Minute())
				assert.Equal(t, tc.expectedSecond, next.Second(),
					"expected second %d, got %d", tc.expectedSecond, next.Second())
			})
		}
	})

	t.Run("with 5 minute interval and 30 second lead time", func(t *testing.T) {
		cm := NewRedisCronManager(
			q,
			logger.StdlibLogger(ctx),
			WithHealthCheckInterval(5*time.Minute),
			WithHealthCheckLeadTimeSeconds(30),
		)
		cmTyped := cm.(*redisCronManager)

		testCases := []struct {
			name           string
			inputTime      time.Time
			expectedHour   int
			expectedMinute int
			expectedSecond int
		}{
			{
				name:           "from 14:32:45 should go to 14:34:30",
				inputTime:      time.Date(2025, 10, 26, 14, 32, 45, 0, time.UTC),
				expectedHour:   14,
				expectedMinute: 34,
				expectedSecond: 30,
			},
			{
				name:           "from 14:30:00 should go to 14:34:30",
				inputTime:      time.Date(2025, 10, 26, 14, 30, 0, 0, time.UTC),
				expectedHour:   14,
				expectedMinute: 34,
				expectedSecond: 30,
			},
			{
				name:           "from 14:34:30 should go to 14:39:30",
				inputTime:      time.Date(2025, 10, 26, 14, 34, 30, 0, time.UTC),
				expectedHour:   14,
				expectedMinute: 39,
				expectedSecond: 30,
			},
			{
				name:           "from 14:59:00 should go to 14:59:30",
				inputTime:      time.Date(2025, 10, 26, 14, 59, 0, 0, time.UTC),
				expectedHour:   14,
				expectedMinute: 59,
				expectedSecond: 30,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				next := cmTyped.nextHealthCheckTime(tc.inputTime)

				assert.True(t, next.After(tc.inputTime),
					"next time %v should be after input time %v", next, tc.inputTime)

				assert.Equal(t, tc.expectedHour, next.Hour(),
					"expected hour %d, got %d", tc.expectedHour, next.Hour())
				assert.Equal(t, tc.expectedMinute, next.Minute(),
					"expected minute %d, got %d", tc.expectedMinute, next.Minute())
				assert.Equal(t, tc.expectedSecond, next.Second(),
					"expected second %d, got %d", tc.expectedSecond, next.Second())
			})
		}
	})

	t.Run("with 1 hour interval and 5 minute lead time", func(t *testing.T) {
		cm := NewRedisCronManager(
			q,
			logger.StdlibLogger(ctx),
			WithHealthCheckInterval(1*time.Hour),
			WithHealthCheckLeadTimeSeconds(300),
		)
		cmTyped := cm.(*redisCronManager)

		testCases := []struct {
			name           string
			inputTime      time.Time
			expectedHour   int
			expectedMinute int
			expectedSecond int
		}{
			{
				name:           "from 14:32:00 should go to 14:55:00",
				inputTime:      time.Date(2025, 10, 26, 14, 32, 0, 0, time.UTC),
				expectedHour:   14,
				expectedMinute: 55,
				expectedSecond: 0,
			},
			{
				name:           "from 14:56:00 should go to 15:55:00",
				inputTime:      time.Date(2025, 10, 26, 14, 56, 0, 0, time.UTC),
				expectedHour:   15,
				expectedMinute: 55,
				expectedSecond: 0,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				next := cmTyped.nextHealthCheckTime(tc.inputTime)

				assert.True(t, next.After(tc.inputTime),
					"next time %v should be after input time %v", next, tc.inputTime)

				assert.Equal(t, tc.expectedHour, next.Hour(),
					"expected hour %d, got %d", tc.expectedHour, next.Hour())
				assert.Equal(t, tc.expectedMinute, next.Minute(),
					"expected minute %d, got %d", tc.expectedMinute, next.Minute())
				assert.Equal(t, tc.expectedSecond, next.Second(),
					"expected second %d, got %d", tc.expectedSecond, next.Second())
			})
		}
	})

	t.Run("consistency checks", func(t *testing.T) {
		cm := NewRedisCronManager(q, logger.StdlibLogger(ctx))
		cmTyped := cm.(*redisCronManager)

		t.Run("calling twice with same time should return same result", func(t *testing.T) {
			inputTime := time.Date(2025, 10, 26, 14, 30, 45, 0, time.UTC)
			next1 := cmTyped.nextHealthCheckTime(inputTime)
			next2 := cmTyped.nextHealthCheckTime(inputTime)

			assert.True(t, next1.Equal(next2),
				"calling nextHealthCheckTime twice with same input should return same result")
		})

		t.Run("calling twice with differnt time inside an interval should return same result", func(t *testing.T) {
			inputTime1 := time.Date(2025, 10, 26, 14, 30, 30, 0, time.UTC)
			next1 := cmTyped.nextHealthCheckTime(inputTime1)

			inputTime2 := time.Date(2025, 10, 26, 14, 30, 35, 0, time.UTC)
			next2 := cmTyped.nextHealthCheckTime(inputTime2)

			assert.True(t, next1.Equal(next2),
				"calling nextHealthCheckTime twice in same interval should return same result")
		})

		t.Run("result should always be aligned to interval", func(t *testing.T) {
			for i := 0; i < 100; i++ {
				// Random time
				inputTime := time.Date(2025, 10, 26, 14, i%60, i%60, 0, time.UTC)
				next := cmTyped.nextHealthCheckTime(inputTime)

				// With 1 minute interval and 20 second lead time,
				// result should always be at 40 seconds past the minute
				assert.Equal(t, 40, next.Second(),
					"for input %v, expected second to be 41, got %d", inputTime, next.Second())
			}
		})
	})

	r.FlushAll()
}

func TestCronHealthCheckJobID(t *testing.T) {
	ctx := context.Background()
	r, rc := initRedis(t)
	defer rc.Close()

	defaultShard := redis_state.QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), Name: consts.DefaultQueueShardName}
	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))

	q := redis_state.NewQueue(
		defaultShard,
		redis_state.WithClock(clock),
	)

	cm := NewRedisCronManager(q, logger.StdlibLogger(ctx))
	cmTyped := cm.(*redisCronManager)

	t.Run("should generate consistent JobID for same time", func(t *testing.T) {
		testTime := time.Date(2025, 10, 26, 14, 30, 41, 0, time.UTC)

		jobID1 := cmTyped.CronHealthCheckJobID(testTime)
		jobID2 := cmTyped.CronHealthCheckJobID(testTime)

		assert.Equal(t, jobID1, jobID2, "same time should generate same JobID")
	})

	t.Run("should generate different JobID for different times", func(t *testing.T) {
		time1 := time.Date(2025, 10, 26, 14, 30, 41, 0, time.UTC)
		time2 := time.Date(2025, 10, 26, 14, 31, 41, 0, time.UTC)

		jobID1 := cmTyped.CronHealthCheckJobID(time1)
		jobID2 := cmTyped.CronHealthCheckJobID(time2)

		assert.NotEqual(t, jobID1, jobID2, "different times should generate different JobIDs")
	})

	t.Run("should contain expected format", func(t *testing.T) {
		testTime := time.Date(2025, 10, 26, 14, 30, 41, 0, time.UTC)
		jobID := cmTyped.CronHealthCheckJobID(testTime)

		assert.Contains(t, jobID, ":cron:health-check", "JobID should contain ':cron:health-check'")
		assert.True(t, len(jobID) > 0, "JobID should not be empty")
	})

	t.Run("should handle different time formats", func(t *testing.T) {
		testCases := []struct {
			name string
			time time.Time
		}{
			{
				name: "standard time",
				time: time.Date(2025, 10, 26, 14, 30, 41, 0, time.UTC),
			},
			{
				name: "midnight",
				time: time.Date(2025, 10, 26, 0, 0, 0, 0, time.UTC),
			},
			{
				name: "noon",
				time: time.Date(2025, 10, 26, 12, 0, 0, 0, time.UTC),
			},
			{
				name: "end of day",
				time: time.Date(2025, 10, 26, 23, 59, 59, 0, time.UTC),
			},
			{
				name: "with nanoseconds",
				time: time.Date(2025, 10, 26, 14, 30, 41, 123456789, time.UTC),
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				jobID := cmTyped.CronHealthCheckJobID(tc.time)
				assert.NotEmpty(t, jobID)
				assert.Contains(t, jobID, ":cron:health-check")
			})
		}
	})

	r.FlushAll()
}

// CronItemEqualsIgnoreID compares two CronItems for equality, ignoring ID and JobID fields.
// This is useful when testing that a new CronItem was created with the same metadata.
func CronItemEqualsIgnoreIDAndOp(t *testing.T, expected, actual CronItem) {
	t.Helper()
	assert.Equal(t, expected.AccountID, actual.AccountID)
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	assert.Equal(t, expected.AppID, actual.AppID)
	assert.Equal(t, expected.FunctionID, actual.FunctionID)
	assert.Equal(t, expected.FunctionVersion, actual.FunctionVersion)
	assert.Equal(t, expected.Expression, actual.Expression)
}

// CronItemEquals compares two CronItems for complete equality, including ID and JobID fields.
func CronItemEquals(t *testing.T, expected, actual CronItem) {
	t.Helper()
	CronItemEqualsIgnoreIDAndOp(t, expected, actual)
	assert.True(t, expected.ID.Timestamp().Equal(actual.ID.Timestamp()))
	assert.Equal(t, expected.JobID, actual.JobID)
	assert.Equal(t, expected.Op, actual.Op)
}

func TestRedisCronManager(t *testing.T) {
	r, rc := initRedis(t)
	defer rc.Close()

	ctx := context.Background()

	defaultShard := redis_state.QueueShard{Kind: string(enums.QueueShardKindRedis), RedisClient: redis_state.NewQueueClient(rc, redis_state.QueueDefaultKey), Name: consts.DefaultQueueShardName}
	clock := clockwork.NewFakeClockAt(time.Now().Truncate(time.Second))

	q := redis_state.NewQueue(
		defaultShard,
		redis_state.WithClock(clock),
		redis_state.WithRunMode(redis_state.QueueRunMode{
			Sequential:    true,
			Scavenger:     true,
			Partition:     true,
			Account:       true,
			AccountWeight: 85,
		}),
	)

	cm := NewRedisCronManager(
		q,
		logger.StdlibLogger(ctx),
	)

	createCronItem := func(op enums.CronOp) CronItem {
		return CronItem{
			ID:              ulid.MustNew(ulid.Timestamp(clock.Now()), ulid.DefaultEntropy()),
			AccountID:       uuid.New(),
			WorkspaceID:     uuid.New(),
			AppID:           uuid.New(),
			FunctionID:      uuid.New(),
			FunctionVersion: 1,
			Expression:      "0 0 * * *",
			JobID:           uuid.NewString(),
			Op:              op,
		}
	}

	t.Run("ScheduleNext", func(t *testing.T) {
		r.FlushAll()

		t.Run("valid cron expression should create next item", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "0 * * * *" // Every hour

			nextItem, err := cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem)

			// Verify core fields match but ID and JobID are different
			CronItemEqualsIgnoreIDAndOp(t, cronItem, *nextItem)

			assert.NotEqual(t, cronItem.ID, nextItem.ID)
			assert.NotEmpty(t, nextItem.ID)
			assert.NotEqual(t, cronItem.JobID, nextItem.JobID)
			assert.NotEmpty(t, nextItem.JobID)
			assert.Equal(t, cronItem.Op, nextItem.Op)
		})

		t.Run("multiple schedulenext calls should be idempotent for same op", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "0 * * * *" // Every hour
			// current timestamp is 10 minutes past the hour
			cronItem.ID = ulid.MustNew(ulid.Timestamp(time.Date(2024, 1, 1, 0, 10, 0, 0, time.UTC)), ulid.DefaultEntropy())

			nextItem1, err := cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem1)

			// Verify core fields match but ID and JobID are different
			CronItemEqualsIgnoreIDAndOp(t, cronItem, *nextItem1)
			assert.NotEqual(t, cronItem.ID, nextItem1.ID)
			assert.NotEmpty(t, nextItem1.ID)
			assert.NotEqual(t, cronItem.JobID, nextItem1.JobID)
			assert.NotEmpty(t, nextItem1.JobID)

			// calling schedule next again should result in an identical nextItem
			nextItem2, err := cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem2)
			CronItemEquals(t, *nextItem1, *nextItem2)
		})

		t.Run("multiple schedulenext calls should be idempotent for different ops", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *" // Every hour
			// current timestamp is 10 minutes past the hour
			cronItem.ID = ulid.MustNew(ulid.Timestamp(time.Date(2024, 1, 1, 0, 10, 0, 0, time.UTC)), ulid.DefaultEntropy())

			nextItem1, err := cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem1)

			// Verify core fields match but ID and JobID are different
			CronItemEqualsIgnoreIDAndOp(t, cronItem, *nextItem1)
			assert.NotEqual(t, cronItem.ID, nextItem1.ID)
			assert.NotEmpty(t, nextItem1.ID)
			assert.NotEqual(t, cronItem.JobID, nextItem1.JobID)
			assert.NotEmpty(t, nextItem1.JobID)

			// call ScheduleNext again for init with same function version etc should result in an identical nextItem
			cronItem.Op = enums.CronInit
			nextItem2, err := cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem2)
			CronItemEquals(t, *nextItem1, *nextItem2)
		})

		t.Run("schedule next with new version should create new schedule", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *" // Every hour
			// current timestamp is 10 minutes past the hour
			cronItem.ID = ulid.MustNew(ulid.Timestamp(time.Date(2024, 1, 1, 0, 10, 0, 0, time.UTC)), ulid.DefaultEntropy())

			nextItem1, err := cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem1)

			// Verify core fields match but ID and JobID are different
			CronItemEqualsIgnoreIDAndOp(t, cronItem, *nextItem1)
			assert.NotEqual(t, cronItem.ID, nextItem1.ID)
			assert.NotEmpty(t, nextItem1.ID)
			assert.NotEqual(t, cronItem.JobID, nextItem1.JobID)
			assert.NotEmpty(t, nextItem1.JobID)

			cronItemUpdate := cronItem
			cronItemUpdate.Op = enums.CronOpUpdate
			cronItemUpdate.FunctionVersion++
			nextItem2, err := cm.ScheduleNext(ctx, cronItemUpdate)
			require.NoError(t, err)
			require.NotNil(t, nextItem2)
			assert.Greater(t, nextItem2.FunctionVersion, cronItem.FunctionVersion)
			assert.Greater(t, nextItem2.FunctionVersion, nextItem1.FunctionVersion)
		})

		t.Run("different valid cron expressions", func(t *testing.T) {
			testCases := []struct {
				name       string
				expression string
			}{
				{"every minute", "* * * * *"},
				{"daily at midnight", "0 0 * * *"},
				{"hourly descriptor", "@hourly"},
				{"daily descriptor", "@daily"},
				{"weekly descriptor", "@weekly"},
				{"monthly descriptor", "@monthly"},
				{"yearly descriptor", "@yearly"},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					cronItem := createCronItem(enums.CronOpProcess)
					cronItem.Expression = tc.expression

					nextItem, err := cm.ScheduleNext(ctx, cronItem)
					require.NoError(t, err)
					require.NotNil(t, nextItem)
					assert.Equal(t, tc.expression, nextItem.Expression)

					assert.NotEqual(t, cronItem.ID, nextItem.ID)
					assert.NotEqual(t, cronItem.JobID, nextItem.JobID)
				})
			}
		})

		t.Run("invalid cron expression should return error", func(t *testing.T) {
			testCases := []struct {
				name       string
				expression string
			}{
				{"too few fields", "* *"},
				{"invalid minute", "60 * * * *"},
				{"invalid hour", "0 25 * * *"},
				{"invalid day", "0 0 32 * *"},
				{"invalid month", "0 0 1 13 *"},
				{"invalid weekday", "0 0 * * 8"},
				{"empty expression", ""},
				{"invalid descriptor", "@invalid"},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					cronItem := createCronItem(enums.CronOpProcess)
					cronItem.Expression = tc.expression

					nextItem, err := cm.ScheduleNext(ctx, cronItem)
					assert.Error(t, err)
					assert.Nil(t, nextItem)
					assert.Contains(t, err.Error(), "failed to parse cron expression")
				})
			}
		})

		t.Run("valid expr but never ticks", func(t *testing.T) {
			testCases := []struct {
				name       string
				expression string
			}{
				{"Feb 30", "0 0 30 2 *"},
				{"Nov 31", "0 0 31 11 *"},
				{"31st on short months", "0 0 31 2,4,6,9,11 *"},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					cronItem := createCronItem(enums.CronOpProcess)
					cronItem.Expression = tc.expression

					nextItem, err := cm.ScheduleNext(ctx, cronItem)
					assert.NoError(t, err)
					assert.Nil(t, nextItem)
				})
			}
		})

		t.Run("all operations should use item timestamp directly", func(t *testing.T) {
			baseTime := time.Date(2025, 12, 25, 0, 59, 0, 0, time.UTC) // 12:59AM

			testOps := []enums.CronOp{
				enums.CronOpNew,
				enums.CronOpUpdate,
				enums.CronOpUnpause,
				enums.CronOpProcess,
				enums.CronInit,
			}

			for _, op := range testOps {
				t.Run(op.String(), func(t *testing.T) {
					cronItem := createCronItem(op)
					cronItem.ID = ulid.MustNew(ulid.Timestamp(baseTime), ulid.DefaultEntropy())
					cronItem.Expression = "0 * * * *"

					nextItem, err := cm.ScheduleNext(ctx, cronItem)
					require.NoError(t, err)
					require.NotNil(t, nextItem)

					nextTime := time.UnixMilli(int64(nextItem.ID.Time()))

					// Should be scheduled for 1AM
					assert.True(t, nextTime.Equal(time.Date(2025, 12, 25, 1, 0, 0, 0, time.UTC)))
				})
			}

		})

		t.Run("should create valid ULID with timestamp", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "0 * * * *"

			nextItem, err := cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem)

			// Verify ULID is valid and has future timestamp
			currentTime := time.UnixMilli(int64(cronItem.ID.Time()))
			nextTime := time.UnixMilli(int64(nextItem.ID.Time()))
			assert.True(t, nextTime.After(currentTime))
		})

		t.Run("should set operation to CronOpProcess", func(t *testing.T) {
			testOps := []enums.CronOp{
				enums.CronOpNew,
				enums.CronOpUpdate,
				enums.CronOpUnpause,
				enums.CronOpProcess,
				enums.CronInit,
			}

			for _, op := range testOps {
				t.Run(op.String(), func(t *testing.T) {
					cronItem := createCronItem(op)
					cronItem.Expression = "0 * * * *"

					nextItem, err := cm.ScheduleNext(ctx, cronItem)
					require.NoError(t, err)
					require.NotNil(t, nextItem)

					assert.Equal(t, enums.CronOpProcess, nextItem.Op)
				})
			}
		})

		t.Run("should generate different JobID", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "0 * * * *"

			nextItem, err := cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem)

			assert.NotEqual(t, cronItem.JobID, nextItem.JobID)
			assert.NotEmpty(t, nextItem.JobID)
		})

		t.Run("cancelled context should return error", func(t *testing.T) {
			cancelledCtx, cancel := context.WithCancel(ctx)
			cancel()

			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "0 * * * *"

			nextItem, err := cm.ScheduleNext(cancelledCtx, cronItem)
			assert.Error(t, err)
			assert.Nil(t, nextItem)
		})
		t.Run("unknown operation type will succeed", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Op = enums.CronOp(999) // Invalid operation type
			cronItem.Expression = "0 * * * *"

			nextItem, err := cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem)
			require.Equal(t, nextItem.Op, enums.CronOpProcess)
		})
	})

	t.Run("Sync", func(t *testing.T) {
		r.FlushAll()

		t.Run("should enqueue cron sync job successfully", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *"

			err := cm.Sync(ctx, cronItem)
			require.NoError(t, err)
		})

		t.Run("should handle duplicate sync jobs gracefully", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpUpdate)
			cronItem.Expression = "0 0 * * *"

			// Enqueue the same sync job multiple times
			err := cm.Sync(ctx, cronItem)
			require.NoError(t, err)

			err = cm.Sync(ctx, cronItem)
			require.NoError(t, err) // Should not error on duplicate
		})

		t.Run("should use correct queue parameters", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 12 * * *"

			err := cm.Sync(ctx, cronItem)
			require.NoError(t, err)

			// Verify the sync job ID format
			syncID := cronItem.SyncID()
			assert.Contains(t, syncID, ":sync")
			assert.Equal(t, fmt.Sprintf("%s:sync", cronItem.ID), syncID)
		})

		t.Run("should handle different cron operations", func(t *testing.T) {
			syncOperations := []enums.CronOp{
				enums.CronOpNew,
				enums.CronOpUpdate,
				enums.CronOpUnpause,
				enums.CronInit,
			}

			for _, op := range syncOperations {
				t.Run(op.String(), func(t *testing.T) {
					cronItem := createCronItem(op)
					cronItem.Expression = "0 * * * *"

					err := cm.Sync(ctx, cronItem)
					require.NoError(t, err)
				})
			}
		})

		t.Run("should skip CronOpProcess operations", func(t *testing.T) {
			r.FlushAll()
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "0 * * * *"

			err := cm.Sync(ctx, cronItem)
			require.NoError(t, err) // Should return nil without enqueueing
			require.Empty(t, r.Keys())
		})

		t.Run("should skip CronHealthCheck operations", func(t *testing.T) {
			r.FlushAll()
			cronItem := createCronItem(enums.CronHealthCheck)
			cronItem.Expression = "* * * * *"

			err := cm.Sync(ctx, cronItem)
			require.NoError(t, err) // Should return nil without enqueueing
			require.Empty(t, r.Keys())
		})

		t.Run("should handle context cancellation", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *"

			cancelledCtx, cancel := context.WithCancel(ctx)
			cancel()

			err := cm.Sync(cancelledCtx, cronItem)
			assert.Error(t, err)
		})

	})

	t.Run("HealthCheck", func(t *testing.T) {
		r.FlushAll()

		t.Run("should return next scheduled item with valid inputs", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *" // Every hour
			fnVersion := 1

			hc, err := cm.HealthCheck(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)

			// Verify Next time is set and is after now
			assert.True(t, hc.Next.After(time.Now()))

			// Verify JobID is set
			assert.NotEmpty(t, hc.JobID)
			// Verify Scheduled is false (no item exists in queue yet)
			assert.False(t, hc.Scheduled)
		})

		t.Run("should calculate next time from now", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *" // Every hour
			fnVersion := 1

			beforeCall := time.Now()
			hc, err := cm.HealthCheck(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)

			nextTime := hc.Next

			// The next scheduled time should be after now
			assert.True(t, nextTime.After(beforeCall),
				"Next time %v should be after call time %v", nextTime, beforeCall)

			// For an hourly cron, the next time should be at the top of the next hour
			assert.Equal(t, 0, nextTime.Minute())
			assert.Equal(t, 0, nextTime.Second())
		})

		t.Run("should work with different cron expressions", func(t *testing.T) {
			testCases := []struct {
				name       string
				expression string
				validate   func(t *testing.T, nextTime time.Time)
			}{
				{
					name:       "daily at midnight",
					expression: "0 0 * * *",
					validate: func(t *testing.T, nextTime time.Time) {
						assert.Equal(t, 0, nextTime.Hour())
						assert.Equal(t, 0, nextTime.Minute())
					},
				},
				{
					name:       "hourly descriptor",
					expression: "@hourly",
					validate: func(t *testing.T, nextTime time.Time) {
						assert.Equal(t, 0, nextTime.Minute())
					},
				},
				{
					name:       "daily descriptor",
					expression: "@daily",
					validate: func(t *testing.T, nextTime time.Time) {
						assert.Equal(t, 0, nextTime.Hour())
						assert.Equal(t, 0, nextTime.Minute())
					},
				},
				{
					name:       "weekly descriptor",
					expression: "@weekly",
					validate: func(t *testing.T, nextTime time.Time) {
						assert.Equal(t, time.Sunday, nextTime.Weekday())
						assert.Equal(t, 0, nextTime.Hour())
					},
				},
				{
					name:       "monthly descriptor",
					expression: "@monthly",
					validate: func(t *testing.T, nextTime time.Time) {
						assert.Equal(t, 1, nextTime.Day())
						assert.Equal(t, 0, nextTime.Hour())
					},
				},
				{
					name:       "yearly descriptor",
					expression: "@yearly",
					validate: func(t *testing.T, nextTime time.Time) {
						assert.Equal(t, time.January, nextTime.Month())
						assert.Equal(t, 1, nextTime.Day())
					},
				},
				{
					name:       "specific weekday",
					expression: "0 9 * * MON",
					validate: func(t *testing.T, nextTime time.Time) {
						assert.Equal(t, time.Monday, nextTime.Weekday())
						assert.Equal(t, 9, nextTime.Hour())
					},
				},
				{
					name:       "multiple times per day",
					expression: "0 6,12,18 * * *",
					validate: func(t *testing.T, nextTime time.Time) {
						assert.Contains(t, []int{6, 12, 18}, nextTime.Hour())
						assert.Equal(t, 0, nextTime.Minute())
					},
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					functionID := uuid.New()
					fnVersion := 1

					hc, err := cm.HealthCheck(ctx, functionID, tc.expression, fnVersion)
					require.NoError(t, err)

					nextTime := hc.Next
					tc.validate(t, nextTime)
				})
			}
		})

		t.Run("should return error for invalid cron expressions", func(t *testing.T) {
			testCases := []struct {
				name       string
				expression string
			}{
				{"too few fields", "* *"},
				{"invalid minute", "60 * * * *"},
				{"invalid hour", "0 25 * * *"},
				{"invalid day", "0 0 32 * *"},
				{"invalid month", "0 0 1 13 *"},
				{"invalid weekday", "0 0 * * 8"},
				{"empty expression", ""},
				{"invalid descriptor", "@invalid"},
				{"malformed expression", "* * * *"},
				{"invalid characters", "a b c d e"},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					functionID := uuid.New()
					fnVersion := 1

					hc, err := cm.HealthCheck(ctx, functionID, tc.expression, fnVersion)
					assert.Error(t, err)
					assert.Equal(t, CronHealthCheckStatus{}, hc)
					assert.Contains(t, err.Error(), "failed to get next schedule time for health check")
				})
			}
		})

		t.Run("should work with different function versions", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"

			versions := []int{1, 2, 5, 10, 100}
			for _, version := range versions {
				t.Run(fmt.Sprintf("version %d", version), func(t *testing.T) {
					hc, err := cm.HealthCheck(ctx, functionID, expr, version)
					require.NoError(t, err)
					assert.False(t, hc.Next.IsZero())
				})
			}
		})

		t.Run("should generate different JobIDs for different functions", func(t *testing.T) {
			expr := "0 * * * *"
			fnVersion := 1

			functionID1 := uuid.New()
			functionID2 := uuid.New()

			hc1, err := cm.HealthCheck(ctx, functionID1, expr, fnVersion)
			require.NoError(t, err)

			hc2, err := cm.HealthCheck(ctx, functionID2, expr, fnVersion)
			require.NoError(t, err)

			// Different function IDs should generate different job IDs
			assert.NotEqual(t, hc1.JobID, hc2.JobID)
		})

		t.Run("should generate different JobIDs for different expressions", func(t *testing.T) {
			functionID := uuid.New()
			fnVersion := 1

			hc1, err := cm.HealthCheck(ctx, functionID, "0 * * * *", fnVersion)
			require.NoError(t, err)

			hc2, err := cm.HealthCheck(ctx, functionID, "0 0 * * *", fnVersion)
			require.NoError(t, err)

			// Different expressions should generate different job IDs
			assert.NotEqual(t, hc1.JobID, hc2.JobID)
		})

		t.Run("should handle zero function ID", func(t *testing.T) {
			functionID := uuid.UUID{}
			expr := "0 * * * *"
			fnVersion := 1

			hc, err := cm.HealthCheck(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			// JobID should still be generated even with zero UUID
			assert.NotEmpty(t, hc.JobID)
			assert.False(t, hc.Next.IsZero())
		})

		t.Run("should handle zero function version", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := 0

			hc, err := cm.HealthCheck(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			// Version 0 should still work
			assert.NotEmpty(t, hc.JobID)
			assert.False(t, hc.Next.IsZero())
		})

		t.Run("should handle negative function version", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := -1

			hc, err := cm.HealthCheck(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			// Negative version should still work
			assert.NotEmpty(t, hc.JobID)
			assert.False(t, hc.Next.IsZero())
		})

		t.Run("should return valid next time", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := 1

			hc, err := cm.HealthCheck(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)

			// Verify Next time is valid and in the future
			assert.False(t, hc.Next.IsZero())
			assert.True(t, hc.Next.After(time.Now()))
		})

		t.Run("should check if item is scheduled", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := 1

			// First check should show not scheduled
			hc1, err := cm.HealthCheck(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			assert.False(t, hc1.Scheduled)

			// Create a CronItem and schedule it
			cronItem := CronItem{
				ID:              ulid.MustNew(ulid.Timestamp(clock.Now()), ulid.DefaultEntropy()),
				FunctionID:      functionID,
				FunctionVersion: fnVersion,
				Expression:      expr,
				Op:              enums.CronOpNew,
			}
			_, err = cm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)

			// Now the health check should show it as scheduled
			hc2, err := cm.HealthCheck(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			assert.True(t, hc2.Scheduled)
		})
	})

	t.Run("EnqueueNextHealthCheck", func(t *testing.T) {
		r.FlushAll()

		t.Run("should enqueue health check successfully and be idempotent", func(t *testing.T) {
			// Set fake clock to a specific time
			specificTime := time.Date(2025, 10, 26, 14, 30, 0, 0, time.UTC)
			clock.Advance(specificTime.Sub(clock.Now()))

			// Call enqueue next first time
			err := cm.EnqueueNextHealthCheck(ctx)
			require.NoError(t, err)

			// Get count of queue items in the partition
			cmd := rc.B().Zcard().Key("{queue}:queue:sorted:cron-health-check").Build()
			queueItemCount1, _ := rc.Do(ctx, cmd).AsInt64()

			// Call enqueue next second time (should be idempotent)
			err = cm.EnqueueNextHealthCheck(ctx)
			require.NoError(t, err)

			// Get count of queue items in the partition
			cmd = rc.B().Zcard().Key("{queue}:queue:sorted:cron-health-check").Build()
			queueItemCount2, _ := rc.Do(ctx, cmd).AsInt64()

			// The number of keys should be the same (no duplicates created)
			assert.Equal(t, queueItemCount1, queueItemCount2, "queue item count should not increase")
		})

	})

	t.Run("EnqueueHealthCheck", func(t *testing.T) {
		t.Run("not idempotent", func(t *testing.T) {
			r.FlushAll()
			cronItem := createCronItem(enums.CronHealthCheck)

			// Call enqueue should succeed
			err := cm.EnqueueHealthCheck(ctx, cronItem)
			assert.NoError(t, err)

			// Get count of queue items in the partition
			cmd := rc.B().Zcard().Key("{queue}:queue:sorted:cron-health-check").Build()
			queueItemCount1, _ := rc.Do(ctx, cmd).AsInt64()

			// Call enqueue should succeed: second time - should enqueue another item
			err = cm.EnqueueHealthCheck(ctx, cronItem)
			assert.NoError(t, err)

			// Get count of queue items in the partition
			cmd = rc.B().Zcard().Key("{queue}:queue:sorted:cron-health-check").Build()
			queueItemCount2, _ := rc.Do(ctx, cmd).AsInt64()

			assert.Equal(t, queueItemCount1+1, queueItemCount2, "queue item count should increase by 1")
		})

	})

}

func initRedis(t *testing.T) (*miniredis.Miniredis, rueidis.Client) {
	r := miniredis.RunT(t)
	rc, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:  []string{r.Addr()},
		DisableCache: true,
	})
	require.NoError(t, err)
	return r, rc
}
