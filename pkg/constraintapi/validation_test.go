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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:          1,
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
				Amount:          1,
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
				Amount:          1,
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
				Amount:          1,
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
				Amount:          1,
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
				Amount:          1,
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
				Amount:          1,
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
				Amount:          1,
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
				Amount:          1,
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
				Amount:          1,
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
			name: "missing resource kind",
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
				Amount:          1,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing resource kind"},
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
				Constraints:     []ConstraintItem{}, // Empty slice
				Amount:          0,
				CurrentTime:     baseTime,
				Duration:        5 * time.Minute,
				MaximumLifetime: 30 * time.Minute,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"must request capacity"},
		},
		{
			name: "missing lease idempotency keys",
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{}, // Empty slice
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			},
			wantErr: true,
			errMsgs: []string{"missing lease idempotency keys"},
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
				Constraints:     []ConstraintItem{}, // Empty slice
				Amount:          0,
				CurrentTime:     time.Time{},
				Duration:        0,
				MaximumLifetime: 0,
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
				"missing resource kind",
				"missing lease idempotency keys",
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				BlockingThreshold:    0, // Zero blocking threshold should be valid since it's not required
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             1 * time.Nanosecond,
				MaximumLifetime:      1 * time.Nanosecond,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             24 * time.Hour,
				MaximumLifetime:      168 * time.Hour, // 1 week
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:          1,
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
				Amount:          1,
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          time.Unix(0, 1), // Minimum non-zero time
				Duration:             1,               // Minimum non-zero duration
				MaximumLifetime:      1,               // Minimum non-zero lifetime
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             time.Duration(1<<63 - 1), // Max duration
				MaximumLifetime:      time.Duration(1<<63 - 1), // Max lifetime
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC),
				Duration:             1 * time.Hour,
				MaximumLifetime:      24 * time.Hour,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
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

func TestRolloutNoMixedConstraints(t *testing.T) {
	baseTime := time.Date(2023, 10, 15, 12, 30, 45, 0, time.UTC)
	accountID := uuid.New()
	envID := uuid.New()
	functionID := uuid.New()

	tests := []struct {
		name        string
		constraints []ConstraintItem
		wantErr     bool
		errMsgs     []string
	}{
		{
			name: "valid - only concurrency constraint",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
			},
			wantErr: false,
		},
		{
			name: "valid - only throttle constraint",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindThrottle},
			},
			wantErr: false,
		},
		{
			name: "valid - only rate limit constraint",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
			},
			wantErr: false,
		},
		{
			name: "valid - multiple queue constraints (concurrency + throttle)",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindThrottle},
			},
			wantErr: false,
		},
		{
			name: "valid - multiple concurrency constraints",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindConcurrency},
			},
			wantErr: false,
		},
		{
			name: "valid - multiple throttle constraints",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindThrottle},
				{Kind: ConstraintKindThrottle},
			},
			wantErr: false,
		},
		{
			name: "valid - multiple rate limit constraints",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindRateLimit},
			},
			wantErr: false,
		},
		{
			name: "invalid - rate limit + concurrency",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindConcurrency},
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
		{
			name: "invalid - rate limit + throttle",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindThrottle},
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
		{
			name: "invalid - concurrency + rate limit",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindRateLimit},
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
		{
			name: "invalid - throttle + rate limit",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindThrottle},
				{Kind: ConstraintKindRateLimit},
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
		{
			name: "invalid - all three constraint types",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindThrottle},
				{Kind: ConstraintKindRateLimit},
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
		{
			name: "invalid - multiple mixed constraints",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindConcurrency},
				{Kind: ConstraintKindRateLimit},
				{Kind: ConstraintKindThrottle},
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := CapacityAcquireRequest{
				IdempotencyKey: "test-key",
				AccountID:      accountID,
				EnvID:          envID,
				FunctionID:     functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints:          tt.constraints,
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             5 * time.Minute,
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
			}

			err := request.Valid()

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
