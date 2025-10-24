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
	unshardedClient := redis_state.NewUnshardedClient(rc, redis_state.StateDefaultKey, redis_state.QueueDefaultKey)

	cm := NewRedisCronManager(
		unshardedClient.Cron(),
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

		t.Run("all operations should use item timestamp directly", func(t *testing.T) {
			baseTime := time.Date(2025, 12, 25, 0, 59, 0, 0, time.UTC) // 12:59AM

			testOps := []enums.CronOp{
				enums.CronOpNew,
				enums.CronOpUpdate,
				enums.CronOpPause,
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
				enums.CronOpPause,
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
				enums.CronOpPause,
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
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "0 * * * *"

			err := cm.Sync(ctx, cronItem)
			require.NoError(t, err) // Should return nil without enqueueing
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

	t.Run("NextScheduledItemIDForFunction", func(t *testing.T) {
		r.FlushAll()

		t.Run("should return next scheduled item with valid inputs", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *" // Every hour
			fnVersion := 1

			item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			require.NotNil(t, item)

			// Verify basic fields are set correctly
			assert.Equal(t, functionID, item.FunctionID)
			assert.Equal(t, expr, item.Expression)
			assert.Equal(t, fnVersion, item.FunctionVersion)

			// Verify ID is set with non empty timestamp
			assert.NotEqual(t, ulid.ULID{}, item.ID)

			// Verify JobID is set
			assert.NotEmpty(t, item.JobID)
		})

		t.Run("should calculate next time from now", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *" // Every hour
			fnVersion := 1

			beforeCall := time.Now()
			item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)

			nextTime := item.ID.Timestamp()

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

					item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, tc.expression, fnVersion)
					require.NoError(t, err)
					require.NotNil(t, item)

					assert.Equal(t, tc.expression, item.Expression)
					nextTime := item.ID.Timestamp()
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

					item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, tc.expression, fnVersion)
					assert.Error(t, err)
					assert.Nil(t, item)
					assert.Contains(t, err.Error(), "failed to parse cron expression")
				})
			}
		})

		t.Run("should work with different function versions", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"

			versions := []int{1, 2, 5, 10, 100}
			for _, version := range versions {
				t.Run(fmt.Sprintf("version %d", version), func(t *testing.T) {
					item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, version)
					require.NoError(t, err)
					assert.Equal(t, version, item.FunctionVersion)
				})
			}
		})

		t.Run("should generate different JobIDs for different functions", func(t *testing.T) {
			expr := "0 * * * *"
			fnVersion := 1

			functionID1 := uuid.New()
			functionID2 := uuid.New()

			item1, err := cm.NextScheduledItemIDForFunction(ctx, functionID1, expr, fnVersion)
			require.NoError(t, err)

			item2, err := cm.NextScheduledItemIDForFunction(ctx, functionID2, expr, fnVersion)
			require.NoError(t, err)

			// Different function IDs should generate different job IDs
			assert.NotEqual(t, item1.JobID, item2.JobID)
		})

		t.Run("should generate different JobIDs for different expressions", func(t *testing.T) {
			functionID := uuid.New()
			fnVersion := 1

			item1, err := cm.NextScheduledItemIDForFunction(ctx, functionID, "0 * * * *", fnVersion)
			require.NoError(t, err)

			item2, err := cm.NextScheduledItemIDForFunction(ctx, functionID, "0 0 * * *", fnVersion)
			require.NoError(t, err)

			// Different expressions should generate different job IDs
			assert.NotEqual(t, item1.JobID, item2.JobID)
		})

		t.Run("should generate different IDs on subsequent calls", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := 1

			item1, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)

			// Small delay to ensure different entropy
			time.Sleep(1 * time.Millisecond)

			item2, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)

			// IDs should be different due to entropy even if timestamps might be the same
			// Note: The timestamp might be the same since it's based on the next cron schedule
			// but the random part of the ULID should differ
			if item1.ID.Timestamp().Equal(item2.ID.Timestamp()) {
				// If timestamps are equal, the random portion should differ
				assert.NotEqual(t, item1.ID, item2.ID, "ULIDs should differ in their random portion")
			}
		})

		t.Run("should handle zero function ID", func(t *testing.T) {
			functionID := uuid.UUID{}
			expr := "0 * * * *"
			fnVersion := 1

			item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			assert.Equal(t, functionID, item.FunctionID)
			assert.NotEmpty(t, item.JobID)
		})

		t.Run("should handle zero function version", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := 0

			item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			assert.Equal(t, 0, item.FunctionVersion)
		})

		t.Run("should handle negative function version", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := -1

			item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			assert.Equal(t, -1, item.FunctionVersion)
		})

		t.Run("should create valid ULID", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := 1

			item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)

			// Verify ULID is valid
			assert.NotEqual(t, ulid.ULID{}, item.ID)

		})

		t.Run("should leave tenant fields empty", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := 1

			item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)

			// These fields are not populated since they're not provided to the function
			assert.Equal(t, uuid.UUID{}, item.AccountID)
			assert.Equal(t, uuid.UUID{}, item.WorkspaceID)
			assert.Equal(t, uuid.UUID{}, item.AppID)
		})

		t.Run("should handle context", func(t *testing.T) {
			functionID := uuid.New()
			expr := "0 * * * *"
			fnVersion := 1

			// Normal context should work
			item, err := cm.NextScheduledItemIDForFunction(ctx, functionID, expr, fnVersion)
			require.NoError(t, err)
			require.NotNil(t, item)
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
