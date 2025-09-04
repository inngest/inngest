package cron

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state/redis_state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
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

	t.Run("WithScheduleForwardDuration sets duration correctly", func(t *testing.T) {
		opt := redisCronManagerOpt{}
		WithScheduleForwardDuration(30 * time.Second)(&opt)

		assert.Equal(t, 30*time.Second, opt.scheduleForwardDur)
	})

	t.Run("WithScheduleForwardDuration ignores negative duration", func(t *testing.T) {
		opt := redisCronManagerOpt{}
		WithScheduleForwardDuration(-5 * time.Second)(&opt)

		assert.Equal(t, time.Duration(0), opt.scheduleForwardDur)
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

			assert.Equal(t, cronItem.AccountID, nextItem.AccountID)
			assert.Equal(t, cronItem.WorkspaceID, nextItem.WorkspaceID)
			assert.Equal(t, cronItem.AppID, nextItem.AppID)
			assert.Equal(t, cronItem.FunctionID, nextItem.FunctionID)
			assert.Equal(t, cronItem.FunctionVersion, nextItem.FunctionVersion)
			assert.Equal(t, cronItem.Expression, nextItem.Expression)
			assert.Equal(t, enums.CronOpProcess, nextItem.Op)
			assert.NotEqual(t, cronItem.ID, nextItem.ID)
			assert.NotEqual(t, cronItem.JobID, nextItem.JobID)
		})

		t.Run("different cron expressions", func(t *testing.T) {
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

		t.Run("different operation types", func(t *testing.T) {
			baseTime := clock.Now()

			t.Run("CronOpProcess should add forward duration", func(t *testing.T) {
				cronItem := createCronItem(enums.CronOpProcess)
				cronItem.ID = ulid.MustNew(ulid.Timestamp(baseTime), ulid.DefaultEntropy())
				cronItem.Expression = "0 * * * *"

				nextItem, err := cm.ScheduleNext(ctx, cronItem)
				require.NoError(t, err)
				require.NotNil(t, nextItem)

				nextTime := time.UnixMilli(int64(nextItem.ID.Time()))

				// Should be scheduled for some time in the future after baseTime
				assert.True(t, nextTime.After(baseTime),
					"Next time %v should be after base time %v", nextTime, baseTime)
			})

			t.Run("other operations should use item timestamp directly", func(t *testing.T) {
				cronItem := createCronItem(enums.CronOpNew)
				cronItem.ID = ulid.MustNew(ulid.Timestamp(baseTime), ulid.DefaultEntropy())
				cronItem.Expression = "0 * * * *"

				nextItem, err := cm.ScheduleNext(ctx, cronItem)
				require.NoError(t, err)
				require.NotNil(t, nextItem)

				nextTime := time.UnixMilli(int64(nextItem.ID.Time()))

				// Should be scheduled for some time in the future after baseTime
				assert.True(t, nextTime.After(baseTime),
					"Next time %v should be after base time %v", nextTime, baseTime)
			})
		})

		t.Run("jitter should be applied", func(t *testing.T) {
			// Create manager with known jitter range for testing
			jitterCm := NewRedisCronManager(
				unshardedClient.Cron(),
				q,
				logger.StdlibLogger(ctx),
				WithJitterRange(1*time.Second, 2*time.Second),
			)

			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "0 * * * *"

			nextItem, err := jitterCm.ScheduleNext(ctx, cronItem)
			require.NoError(t, err)
			require.NotNil(t, nextItem)

			// Verify jitter was applied (item should be scheduled earlier than exact cron time)
			nextTime := time.UnixMilli(int64(nextItem.ID.Time()))
			baseTimeWithForward := time.UnixMilli(int64(cronItem.ID.Time())).Add(10 * time.Second)
			expectedTime := baseTimeWithForward.Truncate(time.Hour).Add(time.Hour)

			assert.True(t, nextTime.Before(expectedTime),
				"Expected jitter to schedule item before exact cron time. Next: %v, Expected: %v",
				nextTime, expectedTime)
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
				enums.CronOpArchive,
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
	})

	t.Run("CanRun", func(t *testing.T) {
		r.FlushAll()

		t.Run("identical items should return true", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)

			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			canRun, err := cm.CanRun(ctx, cronItem)
			require.NoError(t, err)
			assert.True(t, canRun)
		})

		t.Run("no scheduled item should return false", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)

			canRun, err := cm.CanRun(ctx, cronItem)
			require.NoError(t, err)
			assert.False(t, canRun)
		})

		t.Run("outdated function version should return false", func(t *testing.T) {
			scheduledItem := createCronItem(enums.CronOpProcess)
			scheduledItem.FunctionVersion = 3

			err := cm.UpdateSchedule(ctx, scheduledItem)
			require.NoError(t, err)

			testItem := createCronItem(enums.CronOpProcess)
			testItem.FunctionID = scheduledItem.FunctionID
			testItem.FunctionVersion = 2

			canRun, err := cm.CanRun(ctx, testItem)
			require.NoError(t, err)
			assert.False(t, canRun)
		})

		t.Run("same function version should return true", func(t *testing.T) {
			scheduledItem := createCronItem(enums.CronOpProcess)
			scheduledItem.FunctionVersion = 2

			err := cm.UpdateSchedule(ctx, scheduledItem)
			require.NoError(t, err)

			testItem := createCronItem(enums.CronOpProcess)
			testItem.FunctionID = scheduledItem.FunctionID
			testItem.FunctionVersion = 2
			testItem.ID = ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Second)), ulid.DefaultEntropy())

			canRun, err := cm.CanRun(ctx, testItem)
			require.NoError(t, err)
			assert.True(t, canRun)
		})

		t.Run("newer function version should return true", func(t *testing.T) {
			scheduledItem := createCronItem(enums.CronOpProcess)
			scheduledItem.FunctionVersion = 2

			err := cm.UpdateSchedule(ctx, scheduledItem)
			require.NoError(t, err)

			testItem := createCronItem(enums.CronOpProcess)
			testItem.FunctionID = scheduledItem.FunctionID
			testItem.FunctionVersion = 3
			testItem.ID = ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Second)), ulid.DefaultEntropy())

			canRun, err := cm.CanRun(ctx, testItem)
			require.NoError(t, err)
			assert.True(t, canRun)
		})
	})

	t.Run("UpdateSchedule", func(t *testing.T) {
		r.FlushAll()

		t.Run("CronOpNew should create new schedule", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *"

			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			// Verify schedule was created
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)
			require.NotNil(t, retrievedItem)

			assert.Equal(t, cronItem.FunctionID, retrievedItem.FunctionID)
			assert.Equal(t, cronItem.Expression, retrievedItem.Expression)
			assert.Equal(t, cronItem.FunctionVersion, retrievedItem.FunctionVersion)
			assert.Equal(t, enums.CronOpProcess, retrievedItem.Op)
		})

		t.Run("CronOpUpdate should update existing schedule", func(t *testing.T) {
			// Create initial schedule
			originalItem := createCronItem(enums.CronOpNew)
			originalItem.Expression = "0 * * * *"
			originalItem.FunctionVersion = 1

			err := cm.UpdateSchedule(ctx, originalItem)
			require.NoError(t, err)

			// Update the schedule
			updatedItem := createCronItem(enums.CronOpUpdate)
			updatedItem.FunctionID = originalItem.FunctionID
			updatedItem.Expression = "0 0 * * *"
			updatedItem.FunctionVersion = 2
			updatedItem.ID = ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Hour)), ulid.DefaultEntropy())

			err = cm.UpdateSchedule(ctx, updatedItem)
			require.NoError(t, err)

			// Verify schedule was updated
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, originalItem.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, "0 0 * * *", retrievedItem.Expression)
			assert.Equal(t, 2, retrievedItem.FunctionVersion)
		})

		t.Run("CronOpUpdate with older version should be ignored", func(t *testing.T) {
			// Create initial schedule with version 3
			originalItem := createCronItem(enums.CronOpNew)
			originalItem.Expression = "0 * * * *"
			originalItem.FunctionVersion = 3

			err := cm.UpdateSchedule(ctx, originalItem)
			require.NoError(t, err)

			// Try to update with older version (should be ignored)
			olderUpdateItem := createCronItem(enums.CronOpUpdate)
			olderUpdateItem.FunctionID = originalItem.FunctionID
			olderUpdateItem.Expression = "0 0 * * *"
			olderUpdateItem.FunctionVersion = 2

			err = cm.UpdateSchedule(ctx, olderUpdateItem)
			require.NoError(t, err) // Should succeed but be a no-op

			// Verify original schedule unchanged
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, originalItem.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, "0 * * * *", retrievedItem.Expression)
			assert.Equal(t, 3, retrievedItem.FunctionVersion)
		})

		t.Run("CronOpUpdate on non-existent function should succeed", func(t *testing.T) {
			nonExistentItem := createCronItem(enums.CronOpUpdate)
			nonExistentItem.Expression = "0 * * * *"

			// Should succeed even though no existing schedule
			err := cm.UpdateSchedule(ctx, nonExistentItem)
			require.NoError(t, err)

			// Should create new schedule
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, nonExistentItem.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, nonExistentItem.Expression, retrievedItem.Expression)
		})

		t.Run("CronOpPause should remove schedule", func(t *testing.T) {
			// Create initial schedule
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *"

			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			// Verify schedule exists
			_, err = cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)

			// Pause the function
			pauseItem := createCronItem(enums.CronOpPause)
			pauseItem.FunctionID = cronItem.FunctionID
			pauseItem.ID = ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Minute)), ulid.DefaultEntropy())

			err = cm.UpdateSchedule(ctx, pauseItem)
			require.NoError(t, err)

			// Verify schedule is removed
			_, err = cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "next schedule not found")
		})

		t.Run("CronOpArchive should remove schedule", func(t *testing.T) {
			// Create initial schedule
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *"

			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			// Verify schedule exists
			_, err = cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)

			// Archive the function
			archiveItem := createCronItem(enums.CronOpArchive)
			archiveItem.FunctionID = cronItem.FunctionID
			archiveItem.ID = ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Minute)), ulid.DefaultEntropy())

			err = cm.UpdateSchedule(ctx, archiveItem)
			require.NoError(t, err)

			// Verify schedule is removed
			_, err = cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "next schedule not found")
		})

		t.Run("CronOpUnpause should restore schedule", func(t *testing.T) {
			// Create and then pause a schedule
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *"

			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			pauseItem := createCronItem(enums.CronOpPause)
			pauseItem.FunctionID = cronItem.FunctionID

			err = cm.UpdateSchedule(ctx, pauseItem)
			require.NoError(t, err)

			none, err := cm.NextScheduledItemForFunction(ctx, pauseItem.FunctionID)
			require.ErrorIs(t, err, errNextScheduleNotFound)
			require.Nil(t, none)

			// Unpause the function
			unpauseItem := createCronItem(enums.CronOpUnpause)
			unpauseItem.FunctionID = cronItem.FunctionID
			unpauseItem.Expression = "0 0 * * *" // Different expression
			unpauseItem.FunctionVersion = 2

			err = cm.UpdateSchedule(ctx, unpauseItem)
			require.NoError(t, err)

			// Verify schedule is restored with new settings
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, "0 0 * * *", retrievedItem.Expression)
			assert.Equal(t, 2, retrievedItem.FunctionVersion)
		})

		t.Run("CronOpProcess should create schedule", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "* * * * *"

			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			// Verify schedule was created
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, cronItem.Expression, retrievedItem.Expression)
			assert.Equal(t, enums.CronOpProcess, retrievedItem.Op)
		})

		t.Run("invalid cron expression should return error", func(t *testing.T) {
			testCases := []struct {
				name       string
				expression string
				op         enums.CronOp
			}{
				{"CronOpNew with invalid expression", "invalid", enums.CronOpNew},
				{"CronOpUpdate with invalid expression", "60 * * * *", enums.CronOpUpdate},
				{"CronOpUnpause with invalid expression", "* * * *", enums.CronOpUnpause},
				{"CronOpProcess with invalid expression", "@invalid", enums.CronOpProcess},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					cronItem := createCronItem(tc.op)
					cronItem.Expression = tc.expression

					err := cm.UpdateSchedule(ctx, cronItem)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "failed to parse cron expression")
				})
			}
		})

		t.Run("cancelled context should return error", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *"

			cancelledCtx, cancel := context.WithCancel(ctx)
			cancel()

			err := cm.UpdateSchedule(cancelledCtx, cronItem)
			assert.Error(t, err)
		})

		t.Run("multiple operations on same function", func(t *testing.T) {
			functionID := uuid.New()

			// Create initial schedule
			createItem := createCronItem(enums.CronOpNew)
			createItem.FunctionID = functionID
			createItem.Expression = "0 * * * *"
			createItem.FunctionVersion = 1

			err := cm.UpdateSchedule(ctx, createItem)
			require.NoError(t, err)

			item, err := cm.NextScheduledItemForFunction(ctx, functionID)
			require.NoError(t, err)
			assert.Equal(t, item.Expression, createItem.Expression)

			// Update schedule
			updateItem := createCronItem(enums.CronOpUpdate)
			updateItem.FunctionID = functionID
			updateItem.Expression = "0 0 * * *"
			updateItem.FunctionVersion = 2

			err = cm.UpdateSchedule(ctx, updateItem)
			require.NoError(t, err)

			item, err = cm.NextScheduledItemForFunction(ctx, functionID)
			require.NoError(t, err)
			assert.Equal(t, item.Expression, updateItem.Expression)

			// Pause
			pauseItem := createCronItem(enums.CronOpPause)
			pauseItem.FunctionID = functionID

			err = cm.UpdateSchedule(ctx, pauseItem)
			require.NoError(t, err)

			item, err = cm.NextScheduledItemForFunction(ctx, functionID)
			assert.ErrorIs(t, err, errNextScheduleNotFound)
			assert.Nil(t, item)

			// Unpause with new settings
			unpauseItem := createCronItem(enums.CronOpUnpause)
			unpauseItem.FunctionID = functionID
			unpauseItem.Expression = "* * * * *"
			unpauseItem.FunctionVersion = 3

			err = cm.UpdateSchedule(ctx, unpauseItem)
			require.NoError(t, err)

			// Verify final state
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, functionID)
			require.NoError(t, err)
			assert.Equal(t, unpauseItem.Expression, retrievedItem.Expression)
			assert.Equal(t, 3, retrievedItem.FunctionVersion)

			// Archive the function
			archiveItem := createCronItem(enums.CronOpArchive)
			archiveItem.FunctionID = functionID

			err = cm.UpdateSchedule(ctx, archiveItem)
			require.NoError(t, err)

			// Verify schedule is removed after archive
			archivedItem, err := cm.NextScheduledItemForFunction(ctx, functionID)
			assert.ErrorIs(t, err, errNextScheduleNotFound)
			assert.Nil(t, archivedItem)
		})

		t.Run("CronInit should initialize schedule when none exists", func(t *testing.T) {
			cronItem := createCronItem(enums.CronInit)
			cronItem.Expression = "0 * * * *"

			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			// Verify schedule was created
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)
			require.NotNil(t, retrievedItem)

			assert.Equal(t, cronItem.FunctionID, retrievedItem.FunctionID)
			assert.Equal(t, cronItem.Expression, retrievedItem.Expression)
			assert.Equal(t, cronItem.FunctionVersion, retrievedItem.FunctionVersion)
			assert.Equal(t, enums.CronOpProcess, retrievedItem.Op)
		})

		t.Run("CronInit should do nothing when schedule already exists", func(t *testing.T) {
			// Create initial schedule
			originalItem := createCronItem(enums.CronOpNew)
			originalItem.Expression = "0 * * * *"
			originalItem.FunctionVersion = 1

			err := cm.UpdateSchedule(ctx, originalItem)
			require.NoError(t, err)

			// Get the original scheduled item
			originalScheduled, err := cm.NextScheduledItemForFunction(ctx, originalItem.FunctionID)
			require.NoError(t, err)

			// Try to initialize with CronInit (should be no-op)
			initItem := createCronItem(enums.CronInit)
			initItem.FunctionID = originalItem.FunctionID
			initItem.Expression = "0 0 * * *" // Different expression
			initItem.FunctionVersion = 2      // Different version

			err = cm.UpdateSchedule(ctx, initItem)
			require.NoError(t, err)

			// Verify original schedule is unchanged
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, originalItem.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, originalScheduled.ID, retrievedItem.ID)
			assert.Equal(t, originalScheduled.Expression, retrievedItem.Expression)
			assert.Equal(t, originalScheduled.FunctionVersion, retrievedItem.FunctionVersion)
		})

		t.Run("CronInit with invalid cron expression should return error", func(t *testing.T) {
			cronItem := createCronItem(enums.CronInit)
			cronItem.Expression = "invalid expression"

			err := cm.UpdateSchedule(ctx, cronItem)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse cron expression")
		})

		t.Run("unknown operation type should return error", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Op = enums.CronOp(999) // Invalid operation type
			cronItem.Expression = "0 * * * *"

			err := cm.UpdateSchedule(ctx, cronItem)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown cron operation provided")
		})

		t.Run("should handle different cron expressions correctly", func(t *testing.T) {
			testExpressions := []string{
				"* * * * *", // Every minute
				"0 * * * *", // Every hour
				"0 0 * * *", // Daily at midnight
				"0 0 * * 0", // Weekly on Sunday
				"0 0 1 * *", // Monthly on 1st
				"@hourly",   // Hourly descriptor
				"@daily",    // Daily descriptor
				"@weekly",   // Weekly descriptor
			}

			for i, expr := range testExpressions {
				t.Run(fmt.Sprintf("expression_%d", i), func(t *testing.T) {
					cronItem := createCronItem(enums.CronOpNew)
					cronItem.Expression = expr

					err := cm.UpdateSchedule(ctx, cronItem)
					require.NoError(t, err)

					retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
					require.NoError(t, err)
					assert.Equal(t, expr, retrievedItem.Expression)
				})
			}
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
				enums.CronOpArchive,
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

		t.Run("should preserve all cron item fields", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpUpdate)
			cronItem.Expression = "@daily"
			cronItem.FunctionVersion = 5

			err := cm.Sync(ctx, cronItem)
			require.NoError(t, err)
		})

		t.Run("should handle context cancellation", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.Expression = "0 * * * *"

			cancelledCtx, cancel := context.WithCancel(ctx)
			cancel()

			err := cm.Sync(cancelledCtx, cronItem)
			assert.Error(t, err)
		})

		t.Run("should use correct timing from ULID", func(t *testing.T) {
			baseTime := clock.Now()
			cronItem := createCronItem(enums.CronOpNew)
			cronItem.ID = ulid.MustNew(ulid.Timestamp(baseTime), ulid.DefaultEntropy())
			cronItem.Expression = "0 * * * *"

			err := cm.Sync(ctx, cronItem)
			require.NoError(t, err)

			// The sync job should be scheduled at the time from the ULID
			expectedTime := baseTime
			actualTime := ulid.Time(cronItem.ID.Time())
			assert.Equal(t, expectedTime.UnixMilli(), actualTime.UnixMilli())
		})
	})

	t.Run("NextScheduleItemForFunction", func(t *testing.T) {
		r.FlushAll()

		t.Run("should return scheduled item for existing function", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)

			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)
			require.NotNil(t, retrievedItem)

			// UpdateSchedule calls ScheduleNext which creates a new item, so compare core fields
			assert.Equal(t, cronItem.AccountID, retrievedItem.AccountID)
			assert.Equal(t, cronItem.WorkspaceID, retrievedItem.WorkspaceID)
			assert.Equal(t, cronItem.AppID, retrievedItem.AppID)
			assert.Equal(t, cronItem.FunctionID, retrievedItem.FunctionID)
			assert.Equal(t, cronItem.FunctionVersion, retrievedItem.FunctionVersion)
			assert.Equal(t, cronItem.Expression, retrievedItem.Expression)
			assert.Equal(t, enums.CronOpProcess, retrievedItem.Op)
		})

		t.Run("should return error for non-existent function", func(t *testing.T) {
			nonExistentFunctionID := uuid.New()

			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, nonExistentFunctionID)
			assert.Error(t, err)
			assert.Nil(t, retrievedItem)
			assert.Contains(t, err.Error(), "next schedule not found")
		})

		t.Run("should handle multiple functions independently", func(t *testing.T) {
			// Create multiple cron items for different functions with different expressions
			cronItem1 := createCronItem(enums.CronOpProcess)
			cronItem1.Expression = "0 * * * *" // Every hour

			cronItem2 := createCronItem(enums.CronOpProcess)
			cronItem2.Expression = "0 0 * * *" // Daily at midnight

			cronItem3 := createCronItem(enums.CronOpProcess)
			cronItem3.Expression = "* * * * *" // Every minute

			// Ensure different function IDs
			assert.NotEqual(t, cronItem1.FunctionID, cronItem2.FunctionID)
			assert.NotEqual(t, cronItem1.FunctionID, cronItem3.FunctionID)
			assert.NotEqual(t, cronItem2.FunctionID, cronItem3.FunctionID)

			// Schedule all items
			err := cm.UpdateSchedule(ctx, cronItem1)
			require.NoError(t, err)

			err = cm.UpdateSchedule(ctx, cronItem2)
			require.NoError(t, err)

			err = cm.UpdateSchedule(ctx, cronItem3)
			require.NoError(t, err)

			// Retrieve each independently and verify correct mapping
			retrieved1, err := cm.NextScheduledItemForFunction(ctx, cronItem1.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, cronItem1.Expression, retrieved1.Expression)
			assert.Equal(t, cronItem1.FunctionID, retrieved1.FunctionID)

			retrieved2, err := cm.NextScheduledItemForFunction(ctx, cronItem2.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, cronItem2.Expression, retrieved2.Expression)
			assert.Equal(t, cronItem2.FunctionID, retrieved2.FunctionID)

			retrieved3, err := cm.NextScheduledItemForFunction(ctx, cronItem3.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, cronItem3.Expression, retrieved3.Expression)
			assert.Equal(t, cronItem3.FunctionID, retrieved3.FunctionID)

			// Verify each has unique IDs and JobIDs
			assert.NotEqual(t, retrieved1.ID, retrieved2.ID)
			assert.NotEqual(t, retrieved1.ID, retrieved3.ID)
			assert.NotEqual(t, retrieved2.ID, retrieved3.ID)

			assert.NotEqual(t, retrieved1.JobID, retrieved2.JobID)
			assert.NotEqual(t, retrieved1.JobID, retrieved3.JobID)
			assert.NotEqual(t, retrieved2.JobID, retrieved3.JobID)
		})

		t.Run("should reflect updated schedule", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "0 * * * *"

			// Schedule initial item
			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			// Retrieve and verify initial schedule
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, "0 * * * *", retrievedItem.Expression)

			// Update the schedule
			updatedItem := cronItem
			updatedItem.Expression = "0 0 * * *"
			updatedItem.FunctionVersion = 2
			updatedItem.ID = ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Hour)), ulid.DefaultEntropy())

			err = cm.UpdateSchedule(ctx, updatedItem)
			require.NoError(t, err)

			// Retrieve and verify updated schedule
			retrievedUpdated, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)
			assert.Equal(t, "0 0 * * *", retrievedUpdated.Expression)
			assert.Equal(t, 2, retrievedUpdated.FunctionVersion)
		})

		t.Run("should handle paused functions", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)

			// Schedule initial item
			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			// Verify item exists
			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)
			require.NotNil(t, retrievedItem)

			// Pause the function
			pauseItem := cronItem
			pauseItem.Op = enums.CronOpPause
			pauseItem.ID = ulid.MustNew(ulid.Timestamp(clock.Now().Add(time.Minute)), ulid.DefaultEntropy())

			err = cm.UpdateSchedule(ctx, pauseItem)
			require.NoError(t, err)

			// Now paused functions have their schedule mapping removed completely
			retrievedAfterPause, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			assert.Error(t, err)
			assert.Nil(t, retrievedAfterPause)
			assert.Contains(t, err.Error(), "next schedule not found")
		})

		t.Run("cancelled context should return error", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)

			// Schedule item first
			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			// Try to retrieve with cancelled context
			cancelledCtx, cancel := context.WithCancel(ctx)
			cancel()

			retrievedItem, err := cm.NextScheduledItemForFunction(cancelledCtx, cronItem.FunctionID)
			assert.Error(t, err)
			assert.Nil(t, retrievedItem)
		})

		t.Run("should handle different operation types correctly", func(t *testing.T) {
			testCases := []struct {
				name          string
				op            enums.CronOp
				needsExisting bool
			}{
				{"CronOpNew", enums.CronOpNew, false},
				{"CronOpUpdate", enums.CronOpUpdate, true},
				{"CronOpUnpause", enums.CronOpUnpause, false},
				{"CronOpProcess", enums.CronOpProcess, false},
				{"CronInit", enums.CronInit, false},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					cronItem := createCronItem(tc.op)

					// For CronOpUpdate, we need to create an existing scheduled item first
					if tc.needsExisting {
						initialItem := cronItem
						initialItem.Op = enums.CronOpNew
						err := cm.UpdateSchedule(ctx, initialItem)
						require.NoError(t, err)
					}

					err := cm.UpdateSchedule(ctx, cronItem)
					require.NoError(t, err)

					retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
					require.NoError(t, err)
					require.NotNil(t, retrievedItem)

					// All operations should result in CronOpProcess being stored
					assert.Equal(t, enums.CronOpProcess, retrievedItem.Op)
				})
			}

			// Test CronOpArchive separately since it removes schedule (like CronOpPause)
			t.Run("CronOpArchive", func(t *testing.T) {
				// Create initial schedule first
				cronItem := createCronItem(enums.CronOpNew)
				err := cm.UpdateSchedule(ctx, cronItem)
				require.NoError(t, err)

				// Archive the function
				archiveItem := createCronItem(enums.CronOpArchive)
				archiveItem.FunctionID = cronItem.FunctionID

				err = cm.UpdateSchedule(ctx, archiveItem)
				require.NoError(t, err)

				// Should not be retrievable after archiving
				retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
				assert.Error(t, err)
				assert.Nil(t, retrievedItem)
				assert.Contains(t, err.Error(), "next schedule not found")
			})
		})

		t.Run("should preserve all cron item fields", func(t *testing.T) {
			cronItem := createCronItem(enums.CronOpProcess)
			cronItem.Expression = "@daily"
			cronItem.FunctionVersion = 42

			err := cm.UpdateSchedule(ctx, cronItem)
			require.NoError(t, err)

			retrievedItem, err := cm.NextScheduledItemForFunction(ctx, cronItem.FunctionID)
			require.NoError(t, err)
			require.NotNil(t, retrievedItem)

			// Verify all important fields are preserved
			assert.Equal(t, cronItem.AccountID, retrievedItem.AccountID)
			assert.Equal(t, cronItem.WorkspaceID, retrievedItem.WorkspaceID)
			assert.Equal(t, cronItem.AppID, retrievedItem.AppID)
			assert.Equal(t, cronItem.FunctionID, retrievedItem.FunctionID)
			assert.Equal(t, cronItem.FunctionVersion, retrievedItem.FunctionVersion)
			assert.Equal(t, cronItem.Expression, retrievedItem.Expression)
			assert.Equal(t, enums.CronOpProcess, retrievedItem.Op)

			// ID and JobID should be different (generated by ScheduleNext)
			assert.NotEqual(t, cronItem.ID, retrievedItem.ID)
			assert.NotEqual(t, cronItem.JobID, retrievedItem.JobID)
			assert.NotEmpty(t, retrievedItem.JobID)
		})
	})
}
