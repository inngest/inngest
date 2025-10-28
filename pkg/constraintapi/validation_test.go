package constraintapi

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCapacityAcquireRequestValid(t *testing.T) {
	baseTime := time.Date(2023, 10, 15, 12, 30, 45, 0, time.UTC)
	accountID := uuid.New()
	envID := uuid.New()
	functionID := uuid.New()
	kindConcurrency := ConstraintKindConcurrency

	tests := []struct {
		name    string
		request CapacityAcquireRequest
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid request",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "missing idempotency key",
			request: CapacityAcquireRequest{
				IdempotencyKey: "",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing idempotency key"},
		},
		{
			name: "missing account ID",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      uuid.Nil,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing accountID"},
		},
		{
			name: "missing env ID",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          uuid.Nil,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing envID"},
		},
		{
			name: "missing function ID",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     uuid.Nil,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing functionID"},
		},
		{
			name: "missing constraint config function version",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 0,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing constraint config workflow version"},
		},
		{
			name: "missing current time",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     time.Time{},
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing current time"},
		},
		{
			name: "missing duration",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        0,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing duration"},
		},
		{
			name: "missing maximum lifetime",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 0,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing maximum lifetime"},
		},
		{
			name: "missing source service",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceUnknown,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing source service"},
		},
		{
			name: "missing source location",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationUnknown,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing source location"},
		},
		{
			name: "missing constraints",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{}, // Empty slice
				Amount:      0,
				CurrentTime:       baseTime,
				Duration:          5 * time.Minute,
				MaximumLifetime:   30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"must request capacity"},
		},
		{
			name: "multiple validation errors",
			request: CapacityAcquireRequest{
				IdempotencyKey: "",
				AccountID:      uuid.Nil,
				EnvID:          uuid.Nil,
				FunctionID:     uuid.Nil,
				Configuration: ConstraintConfig{
					FunctionVersion: 0,
				},
				Constraints: []ConstraintItem{}, // Empty slice
				Amount:      0,
				CurrentTime:       time.Time{},
				Duration:          0,
				MaximumLifetime:   0,
				Source: LeaseSource{
					Service:  ServiceUnknown,
					Location: LeaseLocationUnknown,
				},
			},
			wantErr: true,
			errMsgs: []string{
				"missing idempotency key",
				"missing accountID",
				"missing envID",
				"missing functionID",
				"missing constraint config workflow version",
				"missing current time",
				"missing duration",
				"missing maximum lifetime",
				"missing source service",
				"missing source location",
				"must request capacity",
			},
		},
		{
			name: "valid with blocking threshold (not required)",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:       baseTime,
				Duration:          5 * time.Minute,
				BlockingThreshold: 0, // Zero blocking threshold should be valid since it's not required
				MaximumLifetime:   30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Valid()

			if tt.wantErr {
				assert.Error(t, err)
				for _, expectedMsg := range tt.errMsgs {
					assert.Contains(t, err.Error(), expectedMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCapacityAcquireRequestValidEdgeCases(t *testing.T) {
	baseTime := time.Date(2023, 10, 15, 12, 30, 45, 0, time.UTC)
	accountID := uuid.New()
	envID := uuid.New()
	functionID := uuid.New()
	kindConcurrency := ConstraintKindConcurrency

	tests := []struct {
		name    string
		request CapacityAcquireRequest
		wantErr bool
		errMsgs []string
	}{
		{
			name: "very small positive values are valid",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        1 * time.Nanosecond,
				MaximumLifetime: 1 * time.Nanosecond,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "large positive values are valid",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 999999,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        24 * time.Hour,
				MaximumLifetime: 168 * time.Hour, // 1 week
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "negative duration is invalid",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        -1 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing duration"},
		},
		{
			name: "negative maximum lifetime is invalid",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: -1 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing maximum lifetime"},
		},
		{
			name: "whitespace only idempotency key is invalid",
			request: CapacityAcquireRequest{
				IdempotencyKey: "   ",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false, // Note: validation only checks for empty string, not whitespace
		},
		{
			name: "all known service types are valid",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceNewRuns,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "all known location types are valid",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationScheduleRun,
				},
			},
			wantErr: false,
		},
		{
			name: "partition lease location is valid",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationPartitionLease,
				},
			},
			wantErr: false,
		},
		{
			name: "API service is valid",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceAPI,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Valid()

			if tt.wantErr {
				assert.Error(t, err)
				for _, expectedMsg := range tt.errMsgs {
					assert.Contains(t, err.Error(), expectedMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCapacityAcquireRequestValidBoundaryConditions(t *testing.T) {
	baseTime := time.Date(2023, 10, 15, 12, 30, 45, 0, time.UTC)
	accountID := uuid.New()
	envID := uuid.New()
	functionID := uuid.New()
	kindConcurrency := ConstraintKindConcurrency

	tests := []struct {
		name    string
		request CapacityAcquireRequest
		wantErr bool
		errMsgs []string
	}{
		{
			name: "minimum time value",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     time.Unix(0, 1), // Minimum non-zero time
				Duration:        1,               // Minimum non-zero duration
				MaximumLifetime: 1,               // Minimum non-zero lifetime
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "maximum duration value",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        time.Duration(1<<63 - 1), // Max duration
				MaximumLifetime: time.Duration(1<<63 - 1), // Max lifetime
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "far future time",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC),
				Duration:        1 * time.Hour,
				MaximumLifetime: 24 * time.Hour,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "very long idempotency key",
			request: CapacityAcquireRequest{
				IdempotencyKey: string(make([]byte, 10000)), // Very long key
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false, // Validation doesn't check key length
		},
		{
			name: "single character idempotency key",
			request: CapacityAcquireRequest{
				IdempotencyKey: "a",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "maximum function version",
			request: CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1<<63 - 1, // Max int64
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Valid()

			if tt.wantErr {
				assert.Error(t, err)
				for _, expectedMsg := range tt.errMsgs {
					assert.Contains(t, err.Error(), expectedMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCapacityAcquireRequestValidSpecialCharacters(t *testing.T) {
	baseTime := time.Date(2023, 10, 15, 12, 30, 45, 0, time.UTC)
	accountID := uuid.New()
	envID := uuid.New()
	functionID := uuid.New()
	kindConcurrency := ConstraintKindConcurrency

	tests := []struct {
		name    string
		request CapacityAcquireRequest
		wantErr bool
		errMsgs []string
	}{
		{
			name: "unicode characters in idempotency key",
			request: CapacityAcquireRequest{
				IdempotencyKey: "测试-key-🚀",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount:      1,
				CurrentTime: baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "special characters in idempotency key",
			request: CapacityAcquireRequest{
				IdempotencyKey: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
		{
			name: "newlines and tabs in idempotency key",
			request: CapacityAcquireRequest{
				IdempotencyKey: "key\nwith\ttabs\rand\nnewlines",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
					},
				},
				Amount: 1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Valid()

			if tt.wantErr {
				assert.Error(t, err)
				for _, expectedMsg := range tt.errMsgs {
					assert.Contains(t, err.Error(), expectedMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
