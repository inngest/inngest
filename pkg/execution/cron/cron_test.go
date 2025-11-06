package cron

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestCronItem(t *testing.T) {

	t.Run("SyncID", func(t *testing.T) {
		item := CronItem{
			ID:              ulid.MustNew(ulid.Timestamp(time.Now()), nil),
			AccountID:       uuid.New(),
			WorkspaceID:     uuid.New(),
			AppID:           uuid.New(),
			FunctionID:      uuid.New(),
			FunctionVersion: 1,
			Expression:      "0 0 * * *",
			JobID:           "test-job-id",
			Op:              enums.CronOpNew,
		}

		syncID := item.SyncID()
		assert.NotEmpty(t, syncID)
		assert.Contains(t, syncID, ":sync")

		syncID2 := item.SyncID()
		assert.Equal(t, syncID, syncID2, "SyncID should be consistent")
	})
}

func TestNext(t *testing.T) {
	curYear := time.Now().Year()
	sixYearsFromNow := curYear + 6
	tests := []struct {
		name        string
		cronExpr    string
		expectError bool
		nextIsZero  bool
	}{
		{
			name:        "valid minute expression",
			cronExpr:    "0 0 * * *",
			expectError: false,
		},
		{
			name:        "valid hourly expression",
			cronExpr:    "0 * * * *",
			expectError: false,
		},
		{
			name:        "valid daily expression",
			cronExpr:    "0 12 * * *",
			expectError: false,
		},
		{
			name:        "valid descriptor @hourly",
			cronExpr:    "@hourly",
			expectError: false,
		},
		{
			name:        "valid descriptor @daily",
			cronExpr:    "@daily",
			expectError: false,
		},
		{
			name:        "valid descriptor @weekly",
			cronExpr:    "@weekly",
			expectError: false,
		},
		{
			name:        "valid descriptor @monthly",
			cronExpr:    "@monthly",
			expectError: false,
		},
		{
			name:        "valid descriptor @yearly",
			cronExpr:    "@yearly",
			expectError: false,
		},
		{
			name:        "valid but never ticks Feb30",
			cronExpr:    "0 0 30 2 *",
			expectError: false,
			nextIsZero:  true,
		},
		{
			name:        "valid but never ticks Nov31",
			cronExpr:    "0 0 31 11 *",
			expectError: false,
			nextIsZero:  true,
		},
		{
			name:        "valid but never ticks 31st short months",
			cronExpr:    "0 0 31 2,4,6,9,11 *",
			nextIsZero:  true,
			expectError: false,
		},
		{
			name:        "invalid expression - too few fields",
			cronExpr:    "0 0 *",
			expectError: true,
		},
		{
			name:        "invalid expression - invalid minute",
			cronExpr:    "60 0 * * *",
			expectError: true,
		},
		{
			name:        "invalid expression - invalid hour",
			cronExpr:    "0 24 * * *",
			expectError: true,
		},
		{
			name:        "invalid expression - invalid day of month",
			cronExpr:    "0 0 32 * *",
			expectError: true,
		},
		{
			name:        "invalid expression - invalid month",
			cronExpr:    "0 0 * 13 *",
			expectError: true,
		},
		{
			name:        "invalid expression - invalid day of week",
			cronExpr:    "0 0 * * 8",
			expectError: true,
		},
		{
			name:        "invalid expression - 6 field format",
			cronExpr:    "* * * * * *",
			expectError: true,
		},
		{
			name:        "invalid expression - 7 field format",
			cronExpr:    "0 30 9 * * ? *",
			expectError: true,
		},
		{
			name:        "invalid expression - 7 field format with cur year",
			cronExpr:    fmt.Sprintf("0 30 9 * * ? %d", curYear),
			expectError: true,
		},
		{
			name:        "invalid expression - 7 field format with future year",
			cronExpr:    fmt.Sprintf("0 30 9 * * ? %d", sixYearsFromNow),
			expectError: true,
		},
		{
			name:        "empty expression",
			cronExpr:    "",
			expectError: true,
		},
		{
			name:        "invalid descriptor",
			cronExpr:    "@invalid",
			expectError: true,
		},
		{
			name:        "invalid string",
			cronExpr:    "invalid",
			expectError: true,
		},
	}

	from := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := Next(tt.cronExpr, from)
			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, next.IsZero())
				assert.Contains(t, err.Error(), "error parsing cron expression")
			} else {
				assert.NoError(t, err)
				if !tt.nextIsZero {
					assert.False(t, next.IsZero())
					assert.True(t, next.After(from))
				}

			}
		})
	}
}

func TestNextScheduleCalculation(t *testing.T) {
	tests := []struct {
		name     string
		cronExpr string
		from     time.Time
		expected time.Time
	}{
		{
			name:     "daily at noon from morning",
			cronExpr: "0 12 * * *",
			from:     time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "daily at noon from noon",
			cronExpr: "0 12 * * *",
			from:     time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "daily at noon from afternoon",
			cronExpr: "0 12 * * *",
			from:     time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC),
			expected: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			name:     "hourly from minute 30",
			cronExpr: "@hourly",
			from:     time.Date(2023, 1, 1, 10, 30, 0, 0, time.UTC),
			expected: time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC),
		},
		{
			name:     "every 5 minutes",
			cronExpr: "*/5 * * * *",
			from:     time.Date(2023, 1, 1, 10, 2, 0, 0, time.UTC),
			expected: time.Date(2023, 1, 1, 10, 5, 0, 0, time.UTC),
		},
		{
			name:     "every minute from top of minute",
			cronExpr: "* * * * *",
			from:     time.Date(2023, 1, 1, 10, 5, 0, 0, time.UTC),
			expected: time.Date(2023, 1, 1, 10, 6, 0, 0, time.UTC),
		},
		{
			name:     "weekly on monday",
			cronExpr: "0 0 * * 1",
			from:     time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC), // Sunday
			expected: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),  // Monday
		},
		{
			name:     "valid but never ticks Feb30",
			cronExpr: "0 0 30 2 *",
			from:     time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Time{}, // Zero time - never executes
		},
		{
			name:     "valid but never ticks Nov31",
			cronExpr: "0 0 31 11 *",
			from:     time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Time{}, // Zero time - never executes
		},
		{
			name:     "valid but never ticks 31 on short months",
			cronExpr: "0 0 31 2,4,6,9,11 *",
			from:     time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Time{}, // Zero time - never executes
		},
		{
			name:     "valid cron 31st of current month",
			cronExpr: "0 0 31 * *",
			from:     time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "valid cron skips 31st on short months",
			cronExpr: "0 0 31 * *",
			from:     time.Date(2023, 2, 1, 10, 0, 0, 0, time.UTC),
			expected: time.Date(2023, 3, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, err := Next(tt.cronExpr, tt.from)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, next)
		})
	}
}

func TestCronSyncerInterface(t *testing.T) {
	t.Run("CronManager implements CronSyncer", func(t *testing.T) {
		// This test verifies that CronManager satisfies the CronSyncer interface
		var _ CronSyncer = (*redisCronManager)(nil)

		// Also verify through CronManager interface
		var manager CronManager

		// If this compiles, the interface embedding is working correctly
		syncer := CronSyncer(manager)
		_ = syncer
	})
}
