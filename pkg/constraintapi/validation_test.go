package constraintapi

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
			errMsgs: []string{"duration smaller than minimum"},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
			errMsgs: []string{"must provide constraints"},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
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
				"duration smaller than minimum",
				"missing maximum lifetime",
				"missing source service",
				"missing source location",
				"missing lease idempotency keys",
				"must provide constraints",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				BlockingThreshold:    0, // Zero blocking threshold should be valid since it's not required
				MaximumLifetime:      30 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             3 * time.Second,
				MaximumLifetime:      5 * time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             1 * time.Minute,
				MaximumLifetime:      24 * time.Hour,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"duration smaller than minimum"},
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
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
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceNewRuns,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationScheduleRun,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationPartitionLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceAPI,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          time.Unix(0, 1), // Minimum non-zero time
				Duration:             3 * time.Second, // Minimum allowed duration
				MaximumLifetime:      5 * time.Minute, // Minimum reasonable lifetime
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             1 * time.Minute, // Max allowed duration
				MaximumLifetime:      24 * time.Hour,  // Max allowed lifetime
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC),
				Duration:             1 * time.Minute,
				MaximumLifetime:      24 * time.Hour,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true, // Key exceeds maximum length of 128 characters
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
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
		name          string
		constraints   []ConstraintItem
		configuration ConstraintConfig
		mi            MigrationIdentifier
		wantErr       bool
		errMsgs       []string
	}{
		{
			name: "valid - only concurrency constraint",
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						InProgressItemKey: "test-key",
					},
				},
			},
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			wantErr: false,
		},
		{
			name: "valid - only throttle constraint",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{
					EvaluatedKeyHash:  "key-hash",
					KeyExpressionHash: "expr-hash",
				}},
			},
			configuration: ConstraintConfig{
				FunctionVersion: 1,
				Throttle: []ThrottleConfig{
					{
						ThrottleKeyExpressionHash: "expr-hash",
					},
				},
			},
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			wantErr: false,
		},
		{
			name: "valid - only rate limit constraint",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit, RateLimit: &RateLimitConstraint{
					KeyExpressionHash: "expr-hash",
					EvaluatedKeyHash:  "key-hash",
				}},
			},
			configuration: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []RateLimitConfig{
					{
						KeyExpressionHash: "expr-hash",
					},
				},
			},
			mi: MigrationIdentifier{
				IsRateLimit: true,
			},
			wantErr: false,
		},
		{
			name: "valid - multiple queue constraints (concurrency + throttle)",
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						InProgressItemKey: "test-key",
					},
				},
				{
					Kind: ConstraintKindThrottle, Throttle: &ThrottleConstraint{
						KeyExpressionHash: "expr-hash",
						EvaluatedKeyHash:  "key-hash",
					},
				},
			},
			configuration: ConstraintConfig{
				FunctionVersion: 1,
				Throttle: []ThrottleConfig{
					{
						ThrottleKeyExpressionHash: "expr-hash",
					},
				},
			},
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			wantErr: false,
		},
		{
			name: "valid - multiple concurrency constraints",
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						InProgressItemKey: "test-key-1",
					},
				},
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						InProgressItemKey: "test-key-2",
					},
				},
			},
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			wantErr: false,
		},
		{
			name: "invalid - multiple throttle constraints",
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindThrottle,
					Throttle: &ThrottleConstraint{
						KeyExpressionHash: "expr-hash-1",
						EvaluatedKeyHash:  "key-1",
					},
				},
				{
					Kind: ConstraintKindThrottle,
					Throttle: &ThrottleConstraint{
						KeyExpressionHash: "expr-hash-2",
						EvaluatedKeyHash:  "key-2",
					},
				},
			},
			configuration: ConstraintConfig{
				FunctionVersion: 1,
				Throttle: []ThrottleConfig{
					{
						ThrottleKeyExpressionHash: "expr-hash-1",
					},
					{
						ThrottleKeyExpressionHash: "expr-hash-2",
					},
				},
			},
			mi: MigrationIdentifier{
				QueueShard: "test",
			},
			wantErr: true,
			errMsgs: []string{
				"exceeded maximum of 1 throttles",
			},
		},
		{
			name: "invalid - multiple rate limit constraints",
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindRateLimit,
					RateLimit: &RateLimitConstraint{
						KeyExpressionHash: "expr-hash-1",
						EvaluatedKeyHash:  "key-1",
					},
				},
				{
					Kind: ConstraintKindRateLimit,
					RateLimit: &RateLimitConstraint{
						KeyExpressionHash: "expr-hash-2",
						EvaluatedKeyHash:  "key-2",
					},
				},
			},
			configuration: ConstraintConfig{
				FunctionVersion: 1,
				RateLimit: []RateLimitConfig{
					{
						KeyExpressionHash: "expr-hash-1",
					},
					{
						KeyExpressionHash: "expr-hash-2",
					},
				},
			},
			mi: MigrationIdentifier{
				IsRateLimit: true,
			},
			wantErr: true,
			errMsgs: []string{
				"exceeded maximum of 1 rate limits",
			},
		},
		{
			name: "invalid - rate limit + concurrency",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						InProgressItemKey: "test-key",
					},
				},
			},
			mi: MigrationIdentifier{
				QueueShard:  "test",
				IsRateLimit: true,
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
			mi: MigrationIdentifier{
				QueueShard:  "test",
				IsRateLimit: true,
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
		{
			name: "invalid - concurrency + rate limit",
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						InProgressItemKey: "test-key",
					},
				},
				{Kind: ConstraintKindRateLimit},
			},
			mi: MigrationIdentifier{
				IsRateLimit: true,
				QueueShard:  "test",
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
			mi: MigrationIdentifier{
				QueueShard:  "test",
				IsRateLimit: true,
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
		{
			name: "invalid - all three constraint types",
			constraints: []ConstraintItem{
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						InProgressItemKey: "test-key",
					},
				},
				{Kind: ConstraintKindThrottle},
				{Kind: ConstraintKindRateLimit},
			},
			mi: MigrationIdentifier{
				IsRateLimit: true,
				QueueShard:  "test",
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
		{
			name: "invalid - multiple mixed constraints",
			constraints: []ConstraintItem{
				{Kind: ConstraintKindRateLimit},
				{
					Kind: ConstraintKindConcurrency,
					Concurrency: &ConcurrencyConstraint{
						InProgressItemKey: "test-key",
					},
				},
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
				IdempotencyKey:       "test-key",
				AccountID:            accountID,
				EnvID:                envID,
				FunctionID:           functionID,
				Configuration:        tt.configuration,
				Constraints:          tt.constraints,
				Amount:               1,
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: tt.mi,
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

func TestConstraintItemValid(t *testing.T) {
	tests := []struct {
		name        string
		constraint  ConstraintItem
		wantErr     bool
		expectedMsg string
	}{
		{
			name: "valid concurrency constraint with InProgressItemKey",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					KeyExpressionHash: "test-key",
					EvaluatedKeyHash:  "eval-key",
					InProgressItemKey: "redis:concurrency:item:key123",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid concurrency constraint missing InProgressItemKey",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeFn,
					KeyExpressionHash: "test-key",
					EvaluatedKeyHash:  "eval-key",
					InProgressItemKey: "", // Missing required field
				},
			},
			wantErr:     true,
			expectedMsg: "concurrency constraint must specify InProgressItemKey",
		},
		{
			name: "valid throttle constraint with EvaluatedKeyHash",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "throttle-key",
					EvaluatedKeyHash:  "eval-throttle",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid throttle constraint missing EvaluatedKeyHash",
			constraint: ConstraintItem{
				Kind: ConstraintKindThrottle,
				Throttle: &ThrottleConstraint{
					Scope:             enums.ThrottleScopeFn,
					KeyExpressionHash: "throttle-key",
					EvaluatedKeyHash:  "", // Missing required field
				},
			},
			wantErr:     true,
			expectedMsg: "throttle constraint must include EvaluatedKeyHash",
		},
		{
			name: "valid rate limit constraint with EvaluatedKeyHash",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "rate-key",
					EvaluatedKeyHash:  "eval-rate",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid rate limit constraint missing EvaluatedKeyHash",
			constraint: ConstraintItem{
				Kind: ConstraintKindRateLimit,
				RateLimit: &RateLimitConstraint{
					Scope:             enums.RateLimitScopeFn,
					KeyExpressionHash: "rate-key",
					EvaluatedKeyHash:  "", // Missing required field
				},
			},
			wantErr:     true,
			expectedMsg: "rate limit constraint must include EvaluatedKeyHash",
		},
		{
			name: "concurrency constraint with nil struct is valid",
			constraint: ConstraintItem{
				Kind:        ConstraintKindConcurrency,
				Concurrency: nil, // nil constraint object
			},
			wantErr: false,
		},
		{
			name: "throttle constraint with nil struct is valid",
			constraint: ConstraintItem{
				Kind:     ConstraintKindThrottle,
				Throttle: nil, // nil constraint object
			},
			wantErr: false,
		},
		{
			name: "rate limit constraint with nil struct is valid",
			constraint: ConstraintItem{
				Kind:      ConstraintKindRateLimit,
				RateLimit: nil, // nil constraint object
			},
			wantErr: false,
		},
		{
			name: "concurrency constraint with empty string InProgressItemKey",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeStep,
					Scope:             enums.ConcurrencyScopeAccount,
					InProgressItemKey: "", // Explicitly empty
				},
			},
			wantErr:     true,
			expectedMsg: "concurrency constraint must specify InProgressItemKey",
		},
		{
			name: "concurrency constraint with run mode is invalid",
			constraint: ConstraintItem{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:              enums.ConcurrencyModeRun,
					Scope:             enums.ConcurrencyScopeAccount,
					InProgressItemKey: "test-key", // Valid key but mode is run
				},
			},
			wantErr:     true,
			expectedMsg: "run level concurrency is not implemented yet",
		},
		{
			name: "unknown constraint kind is valid (no specific validation)",
			constraint: ConstraintItem{
				Kind: ConstraintKind("unknown"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constraint.Valid()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCapacityExtendLeaseRequestValid(t *testing.T) {
	accountID := uuid.New()
	leaseID := ulid.Make()

	tests := []struct {
		name    string
		request CapacityExtendLeaseRequest
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid request",
			request: CapacityExtendLeaseRequest{
				IdempotencyKey: "extend-key",
				AccountID:      accountID,
				LeaseID:        leaseID,
				Duration:       15 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "missing idempotency key",
			request: CapacityExtendLeaseRequest{
				IdempotencyKey: "",
				AccountID:      accountID,
				LeaseID:        leaseID,
				Duration:       15 * time.Minute,
			},
			wantErr: true,
			errMsgs: []string{"missing idempotency key"},
		},
		{
			name: "missing account ID",
			request: CapacityExtendLeaseRequest{
				IdempotencyKey: "extend-key",
				AccountID:      uuid.Nil,
				LeaseID:        leaseID,
				Duration:       15 * time.Minute,
			},
			wantErr: true,
			errMsgs: []string{"missing accountID"},
		},
		{
			name: "missing lease ID",
			request: CapacityExtendLeaseRequest{
				IdempotencyKey: "extend-key",
				AccountID:      accountID,
				LeaseID:        ulid.ULID{},
				Duration:       15 * time.Minute,
			},
			wantErr: true,
			errMsgs: []string{"missing lease ID"},
		},
		{
			name: "invalid duration - zero",
			request: CapacityExtendLeaseRequest{
				IdempotencyKey: "extend-key",
				AccountID:      accountID,
				LeaseID:        leaseID,
				Duration:       0,
			},
			wantErr: true,
			errMsgs: []string{"invalid duration: must be positive"},
		},
		{
			name: "invalid duration - negative",
			request: CapacityExtendLeaseRequest{
				IdempotencyKey: "extend-key",
				AccountID:      accountID,
				LeaseID:        leaseID,
				Duration:       -5 * time.Minute,
			},
			wantErr: true,
			errMsgs: []string{"invalid duration: must be positive"},
		},
		{
			name: "multiple validation errors",
			request: CapacityExtendLeaseRequest{
				IdempotencyKey: "",
				AccountID:      uuid.Nil,
				LeaseID:        ulid.ULID{},
				Duration:       0,
			},
			wantErr: true,
			errMsgs: []string{
				"missing idempotency key",
				"missing accountID",
				"missing lease ID",
				"invalid duration: must be positive",
			},
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

func TestCapacityCheckRequestValid(t *testing.T) {
	accountID := uuid.New()
	envID := uuid.New()
	functionID := uuid.New()
	kindConcurrency := ConstraintKindConcurrency

	tests := []struct {
		name    string
		request CapacityCheckRequest
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid request with concurrency constraint",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "valid request with rate limit constraint",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
					RateLimit: []RateLimitConfig{
						{
							Scope:             enums.RateLimitScopeFn,
							Limit:             100,
							Period:            60,
							KeyExpressionHash: "rate-key",
						},
					},
				},
				Constraints: []ConstraintItem{
					{
						Kind: ConstraintKindRateLimit,
						RateLimit: &RateLimitConstraint{
							Scope:             enums.RateLimitScopeFn,
							KeyExpressionHash: "rate-key",
							EvaluatedKeyHash:  "rate-key-hash",
						},
					},
				},
				Migration: MigrationIdentifier{
					IsRateLimit: true,
				},
			},
			wantErr: false,
		},
		{
			name: "valid request with throttle constraint",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
					Throttle: []ThrottleConfig{
						{
							Scope:                     enums.ThrottleScopeFn,
							ThrottleKeyExpressionHash: "throttle-key",
							Limit:                     10,
							Burst:                     20,
							Period:                    60,
						},
					},
				},
				Constraints: []ConstraintItem{
					{
						Kind: ConstraintKindThrottle,
						Throttle: &ThrottleConstraint{
							Scope:             enums.ThrottleScopeFn,
							KeyExpressionHash: "throttle-key",
							EvaluatedKeyHash:  "throttle-key-hash",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "missing account ID",
			request: CapacityCheckRequest{
				AccountID:  uuid.Nil,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"missing accountID"},
		},
		{
			name: "missing env ID",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      uuid.Nil,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"missing envID"},
		},
		{
			name: "missing function ID",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: uuid.Nil,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"missing functionID"},
		},
		{
			name: "missing constraint config function version",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 0,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"missing constraint config workflow version"},
		},
		{
			name: "missing constraints",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{}, // Empty slice
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"must provide constraints"},
		},
		{
			name: "invalid constraint - concurrency missing InProgressItemKey",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "", // Missing required field
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"invalid constraint 0", "concurrency constraint must specify InProgressItemKey"},
		},
		{
			name: "invalid constraint - throttle missing EvaluatedKeyHash",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: ConstraintKindThrottle,
						Throttle: &ThrottleConstraint{
							EvaluatedKeyHash: "", // Missing required field
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"invalid constraint 0", "throttle constraint must include EvaluatedKeyHash"},
		},
		{
			name: "invalid constraint - rate limit missing EvaluatedKeyHash",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: ConstraintKindRateLimit,
						RateLimit: &RateLimitConstraint{
							EvaluatedKeyHash: "", // Missing required field
						},
					},
				},
				Migration: MigrationIdentifier{
					IsRateLimit: true,
				},
			},
			wantErr: true,
			errMsgs: []string{"invalid constraint 0", "rate limit constraint must include EvaluatedKeyHash"},
		},
		{
			name: "invalid - mixed queue and rate limit constraints",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: ConstraintKindRateLimit,
						RateLimit: &RateLimitConstraint{
							EvaluatedKeyHash: "rate-key-hash",
						},
					},
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard:  "test",
					IsRateLimit: true,
				},
			},
			wantErr: true,
			errMsgs: []string{"cannot mix queue and rate limit constraints for first stage"},
		},
		{
			name: "missing rate limit flag in migration identifier",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: ConstraintKindRateLimit,
						RateLimit: &RateLimitConstraint{
							EvaluatedKeyHash: "rate-key-hash",
						},
					},
				},
				Migration: MigrationIdentifier{
					IsRateLimit: false, // Should be true for rate limit constraints
				},
			},
			wantErr: true,
			errMsgs: []string{"missing rate limit flag in migration identifier"},
		},
		{
			name: "missing queue shard in migration identifier",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "", // Should be provided for queue constraints
				},
			},
			wantErr: true,
			errMsgs: []string{"missing queue shard in migration identifier"},
		},
		{
			name: "multiple validation errors",
			request: CapacityCheckRequest{
				AccountID:  uuid.Nil,
				EnvID:      uuid.Nil,
				FunctionID: uuid.Nil,
				Configuration: ConstraintConfig{
					FunctionVersion: 0,
				},
				Constraints: []ConstraintItem{}, // Empty slice
				Migration:   MigrationIdentifier{},
			},
			wantErr: true,
			errMsgs: []string{
				"missing accountID",
				"missing envID",
				"missing functionID",
				"missing constraint config workflow version",
				"must provide constraints",
			},
		},
		{
			name: "valid request with multiple concurrency constraints",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key-1",
						},
					},
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key-2",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "valid request with multiple throttle constraints",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
					Throttle: []ThrottleConfig{
						{
							Scope:                     enums.ThrottleScopeFn,
							ThrottleKeyExpressionHash: "throttle-key-1",
							Limit:                     10,
							Burst:                     20,
							Period:                    60,
						},
					},
				},
				Constraints: []ConstraintItem{
					{
						Kind: ConstraintKindThrottle,
						Throttle: &ThrottleConstraint{
							Scope:             enums.ThrottleScopeFn,
							KeyExpressionHash: "throttle-key-1",
							EvaluatedKeyHash:  "throttle-key-1",
						},
					},
					{
						Kind: ConstraintKindThrottle,
						Throttle: &ThrottleConstraint{
							Scope:             enums.ThrottleScopeFn,
							KeyExpressionHash: "throttle-key-1",
							EvaluatedKeyHash:  "throttle-key-2",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "valid request with concurrency and throttle constraints",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
					Throttle: []ThrottleConfig{
						{
							Scope:                     enums.ThrottleScopeFn,
							ThrottleKeyExpressionHash: "throttle-key",
							Limit:                     10,
							Burst:                     20,
							Period:                    60,
						},
					},
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
					{
						Kind: ConstraintKindThrottle,
						Throttle: &ThrottleConstraint{
							Scope:             enums.ThrottleScopeFn,
							KeyExpressionHash: "throttle-key",
							EvaluatedKeyHash:  "throttle-key",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "constraint with run mode concurrency",
			request: CapacityCheckRequest{
				AccountID:  accountID,
				EnvID:      envID,
				FunctionID: functionID,
				Configuration: ConstraintConfig{
					FunctionVersion: 1,
				},
				Constraints: []ConstraintItem{
					{
						Kind: kindConcurrency,
						Concurrency: &ConcurrencyConstraint{
							Mode:              enums.ConcurrencyModeRun,
							InProgressItemKey: "test-key",
						},
					},
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"invalid constraint 0", "run level concurrency is not implemented yet"},
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

func TestCapacityAcquireRequestValidAmountEdgeCases(t *testing.T) {
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
			name: "negative amount is invalid",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               -1, // Negative amount
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"must request at least one lease"},
		},
		{
			name: "zero amount is invalid",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               0, // Zero amount
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"must request at least one lease"},
		},
		{
			name: "amount greater than lease keys - too few keys",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               3, // Request 3 leases
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1", "lease-key-2"}, // Only 2 keys
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"must provide as many lease idempotency keys as amount"},
		},
		{
			name: "amount less than lease keys - too many keys",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               2, // Request 2 leases
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1", "lease-key-2", "lease-key-3"}, // 3 keys
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"must provide as many lease idempotency keys as amount"},
		},
		{
			name: "valid case - amount matches lease keys",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               3, // Request 3 leases
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1", "lease-key-2", "lease-key-3"}, // Exactly 3 keys
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "zero amount with empty lease keys (both invalid)",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               0, // Zero amount
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{}, // Empty
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"must request at least one lease", "missing lease idempotency keys"},
		},
		{
			name: "very large negative amount",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:               -999999, // Very large negative amount
				CurrentTime:          baseTime,
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				LeaseIdempotencyKeys: []string{"lease-key-1"},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"must request at least one lease"},
		},
		{
			name: "large positive amount with matching keys - should exceed maximum",
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
						Concurrency: &ConcurrencyConstraint{
							InProgressItemKey: "test-key",
						},
					},
				},
				Amount:          25, // Exceeds MaximumAmount of 20
				CurrentTime:     baseTime,
				Duration:        30 * time.Second,
				MaximumLifetime: time.Minute,
				LeaseIdempotencyKeys: []string{ // Create 25 keys to match amount
					"lease-key-0", "lease-key-1", "lease-key-2", "lease-key-3", "lease-key-4",
					"lease-key-5", "lease-key-6", "lease-key-7", "lease-key-8", "lease-key-9",
					"lease-key-10", "lease-key-11", "lease-key-12", "lease-key-13", "lease-key-14",
					"lease-key-15", "lease-key-16", "lease-key-17", "lease-key-18", "lease-key-19",
					"lease-key-20", "lease-key-21", "lease-key-22", "lease-key-23", "lease-key-24",
				},
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: LeaseLocationItemLease,
				},
				Migration: MigrationIdentifier{
					QueueShard: "test",
				},
			},
			wantErr: true,
			errMsgs: []string{"must request no more than 20 leases"},
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

func TestCapacityReleaseRequestValid(t *testing.T) {
	accountID := uuid.New()
	leaseID := ulid.Make()

	tests := []struct {
		name    string
		request CapacityReleaseRequest
		wantErr bool
		errMsgs []string
	}{
		{
			name: "valid request",
			request: CapacityReleaseRequest{
				IdempotencyKey: "release-key",
				AccountID:      accountID,
				LeaseID:        leaseID,
			},
			wantErr: false,
		},
		{
			name: "missing idempotency key",
			request: CapacityReleaseRequest{
				IdempotencyKey: "",
				AccountID:      accountID,
				LeaseID:        leaseID,
			},
			wantErr: true,
			errMsgs: []string{"missing idempotency key"},
		},
		{
			name: "missing account ID",
			request: CapacityReleaseRequest{
				IdempotencyKey: "release-key",
				AccountID:      uuid.Nil,
				LeaseID:        leaseID,
			},
			wantErr: true,
			errMsgs: []string{"missing accountID"},
		},
		{
			name: "missing lease ID",
			request: CapacityReleaseRequest{
				IdempotencyKey: "release-key",
				AccountID:      accountID,
				LeaseID:        ulid.ULID{},
			},
			wantErr: true,
			errMsgs: []string{"missing lease ID"},
		},
		{
			name: "multiple validation errors",
			request: CapacityReleaseRequest{
				IdempotencyKey: "",
				AccountID:      uuid.Nil,
				LeaseID:        ulid.ULID{},
			},
			wantErr: true,
			errMsgs: []string{
				"missing idempotency key",
				"missing accountID",
				"missing lease ID",
			},
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
