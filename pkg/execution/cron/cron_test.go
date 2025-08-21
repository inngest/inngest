package cron

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestCronItem(t *testing.T) {
	t.Run("Equal", func(t *testing.T) {
		now := time.Now()
		id1 := ulid.MustNew(ulid.Timestamp(now), nil)
		id2 := ulid.MustNew(ulid.Timestamp(now.Add(time.Millisecond)), nil)

		accountID := uuid.New()
		workspaceID := uuid.New()
		appID := uuid.New()
		functionID := uuid.New()

		item1 := CronItem{
			ID:              id1,
			AccountID:       accountID,
			WorkspaceID:     workspaceID,
			AppID:           appID,
			FunctionID:      functionID,
			FunctionVersion: 1,
			Expression:      "0 0 * * *",
			JobID:           "job1",
			Op:              enums.CronOpProcess,
		}

		t.Run("identical items", func(t *testing.T) {
			item2 := item1
			assert.True(t, item1.Equal(item2))
		})

		t.Run("different ID", func(t *testing.T) {
			item2 := CronItem{
				ID:              id2,
				AccountID:       accountID,
				WorkspaceID:     workspaceID,
				AppID:           appID,
				FunctionID:      functionID,
				FunctionVersion: 1,
				Expression:      "0 0 * * *",
				JobID:           "job1",
				Op:              enums.CronOpProcess,
			}
			assert.False(t, item1.Equal(item2))
		})

		t.Run("different AccountID", func(t *testing.T) {
			item2 := item1
			item2.AccountID = uuid.New()
			assert.False(t, item1.Equal(item2))
		})

		t.Run("different WorkspaceID", func(t *testing.T) {
			item2 := item1
			item2.WorkspaceID = uuid.New()
			assert.False(t, item1.Equal(item2))
		})

		t.Run("different AppID", func(t *testing.T) {
			item2 := item1
			item2.AppID = uuid.New()
			assert.False(t, item1.Equal(item2))
		})

		t.Run("different FunctionID", func(t *testing.T) {
			item2 := item1
			item2.FunctionID = uuid.New()
			assert.False(t, item1.Equal(item2))
		})

		t.Run("different FunctionVersion", func(t *testing.T) {
			item2 := item1
			item2.FunctionVersion = 2
			assert.False(t, item1.Equal(item2))
		})

		t.Run("different JobID", func(t *testing.T) {
			item2 := item1
			item2.JobID = "different-job"
			assert.False(t, item1.Equal(item2))
		})

		t.Run("different Expression", func(t *testing.T) {
			item2 := item1
			item2.Expression = "0 30 * * *"
			assert.False(t, item1.Equal(item2))
		})

		t.Run("different Op", func(t *testing.T) {
			item2 := item1
			item2.Op = enums.CronOpNew
			assert.False(t, item1.Equal(item2))
		})
	})

	t.Run("ProcessID", func(t *testing.T) {
		item := CronItem{
			ID:              ulid.MustNew(ulid.Timestamp(time.Now()), nil),
			AccountID:       uuid.New(),
			WorkspaceID:     uuid.New(),
			AppID:           uuid.New(),
			FunctionID:      uuid.New(),
			FunctionVersion: 1,
			Expression:      "0 0 * * *",
			JobID:           "test-job-id",
			Op:              enums.CronOpProcess,
		}

		processID := item.ProcessID()
		assert.NotEmpty(t, processID)

		processID2 := item.ProcessID()
		assert.Equal(t, processID, processID2, "ProcessID should be consistent")
	})

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

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		cronExpr    string
		expectError bool
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
			name:        "empty expression",
			cronExpr:    "",
			expectError: true,
		},
		{
			name:        "invalid descriptor",
			cronExpr:    "@invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := Parse(tt.cronExpr)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, schedule)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schedule)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		cronExpr    string
		expectError bool
	}{
		{
			name:        "valid expression",
			cronExpr:    "0 0 * * *",
			expectError: false,
		},
		{
			name:        "valid descriptor",
			cronExpr:    "@hourly",
			expectError: false,
		},
		{
			name:        "invalid expression",
			cronExpr:    "0 0 32 * *",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cronExpr)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsAt(t *testing.T) {
	schedule, err := Parse("0 12 * * *")
	assert.NoError(t, err)
	assert.NotNil(t, schedule)

	tests := []struct {
		name     string
		time     time.Time
		expected bool
	}{
		{
			name:     "exact time match",
			time:     time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "within allowed variance - 10 seconds after",
			time:     time.Date(2023, 1, 1, 12, 0, 10, 0, time.UTC),
			expected: true,
		},
		{
			name:     "within allowed variance - 30 seconds after",
			time:     time.Date(2023, 1, 1, 12, 0, 30, 0, time.UTC),
			expected: true,
		},
		{
			name:     "within allowed variance - 49 seconds after",
			time:     time.Date(2023, 1, 1, 12, 0, 49, 0, time.UTC),
			expected: true,
		},
		{
			name:     "outside allowed variance - 51 seconds after",
			time:     time.Date(2023, 1, 1, 12, 0, 51, 0, time.UTC),
			expected: false,
		},
		{
			name:     "outside allowed variance - 1 minute after",
			time:     time.Date(2023, 1, 1, 12, 1, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "before scheduled time",
			time:     time.Date(2023, 1, 1, 11, 59, 59, 0, time.UTC),
			expected: false,
		},
		{
			name:     "different day, same time",
			time:     time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAt(schedule, tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAtWithHourlySchedule(t *testing.T) {
	schedule, err := Parse("@hourly")
	assert.NoError(t, err)
	assert.NotNil(t, schedule)

	tests := []struct {
		name     string
		time     time.Time
		expected bool
	}{
		{
			name:     "exact hour start",
			time:     time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "within variance",
			time:     time.Date(2023, 1, 1, 12, 0, 30, 0, time.UTC),
			expected: true,
		},
		{
			name:     "outside variance",
			time:     time.Date(2023, 1, 1, 12, 1, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "different hour",
			time:     time.Date(2023, 1, 1, 13, 0, 0, 0, time.UTC),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAt(schedule, tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}
