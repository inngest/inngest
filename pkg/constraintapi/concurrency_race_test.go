package constraintapi

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/jonboulle/clockwork"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestConcurrencyAndRaces_ParallelOperations(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Concurrent Acquires on Same Constraint", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 10, // Allow multiple concurrent acquires
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		numGoroutines := 20
		numAcquiresPerGoroutine := 5

		var wg sync.WaitGroup
		var mu sync.Mutex
		var allLeases []CapacityLease
		var errors []error

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < numAcquiresPerGoroutine; j++ {
					resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
						IdempotencyKey:       fmt.Sprintf("concurrent-%d-%d", goroutineID, j),
						AccountID:            te.AccountID,
						EnvID:                te.EnvID,
						FunctionID:           te.FunctionID,
						Amount:               1,
						LeaseIdempotencyKeys: []string{fmt.Sprintf("concurrent-lease-%d-%d", goroutineID, j)},
						CurrentTime:          clock.Now(),
						Duration:             30 * time.Second,
						MaximumLifetime:      time.Minute,
						Configuration:        config,
						Constraints:          constraints,
						Source: LeaseSource{
							Service:  ServiceExecutor,
							Location: CallerLocationItemLease,
						},
								})

					mu.Lock()
					if err != nil {
						errors = append(errors, err)
					} else if len(resp.Leases) > 0 {
						allLeases = append(allLeases, resp.Leases...)
					}
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// Verify results
		require.Empty(t, errors, "Should not have errors in concurrent operations")
		require.NotEmpty(t, allLeases, "Should have acquired some leases")
		require.True(t, len(allLeases) <= 10, "Should not exceed capacity limit")

		// Verify no duplicate leases
		leaseIDSet := make(map[string]bool)
		for _, lease := range allLeases {
			leaseIDStr := lease.LeaseID.String()
			require.False(t, leaseIDSet[leaseIDStr], "Should not have duplicate lease IDs")
			leaseIDSet[leaseIDStr] = true
		}

		// Clean up all leases
		for _, lease := range allLeases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("cleanup-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
			})
			require.NoError(t, err)
		}

		// Verify final state consistency
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})
	})

	t.Run("Concurrent Acquire and Release", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		// First, acquire some leases to release later
		initialLeases := make([]CapacityLease, 3)
		for i := 0; i < 3; i++ {
			resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("initial-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("initial-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             30 * time.Second,
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: CallerLocationItemLease,
				},
				})
			require.NoError(t, err)
			require.Len(t, resp.Leases, 1)
			initialLeases[i] = resp.Leases[0]
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var newLeases []CapacityLease
		var errors []error

		// Concurrent releases
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(lease CapacityLease, index int) {
				defer wg.Done()

				_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
					IdempotencyKey: fmt.Sprintf("concurrent-release-%d", index),
					AccountID:      te.AccountID,
					LeaseID:        lease.LeaseID,
					})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				}
				mu.Unlock()
			}(initialLeases[i], i)
		}

		// Concurrent new acquires
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(acquireID int) {
				defer wg.Done()

				resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
					IdempotencyKey:       fmt.Sprintf("concurrent-new-%d", acquireID),
					AccountID:            te.AccountID,
					EnvID:                te.EnvID,
					FunctionID:           te.FunctionID,
					Amount:               1,
					LeaseIdempotencyKeys: []string{fmt.Sprintf("concurrent-new-lease-%d", acquireID)},
					CurrentTime:          clock.Now(),
					Duration:             30 * time.Second,
					MaximumLifetime:      time.Minute,
					Configuration:        config,
					Constraints:          constraints,
					Source: LeaseSource{
						Service:  ServiceExecutor,
						Location: CallerLocationItemLease,
					},
						})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else if len(resp.Leases) > 0 {
					newLeases = append(newLeases, resp.Leases...)
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Verify no errors occurred
		require.Empty(t, errors, "Should not have errors in concurrent acquire/release")

		// Verify capacity is consistent (should have at most 5 total)
		require.True(t, len(newLeases) <= 5, "Should not exceed total capacity")

		// Clean up remaining leases
		for _, lease := range newLeases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("final-cleanup-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
			})
			require.NoError(t, err)
		}

		// Verify final state
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})
	})

	t.Run("Concurrent Extend Operations", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		// Acquire a lease to extend
		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "extend-base",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"extend-base-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      5 * time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
		})

		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)
		originalLease := resp.Leases[0]

		var wg sync.WaitGroup
		var mu sync.Mutex
		var extendResults []string
		var errors []error

		// Multiple concurrent extends on the same lease
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(extendID int) {
				defer wg.Done()

				extendResp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
					IdempotencyKey: fmt.Sprintf("concurrent-extend-%d", extendID),
					AccountID:      te.AccountID,
					LeaseID:        originalLease.LeaseID,
					Duration:       time.Duration(60+extendID) * time.Second, // Different durations
					})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else if extendResp.LeaseID != nil {
					extendResults = append(extendResults, extendResp.LeaseID.String())
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Should handle concurrent extends gracefully
		require.Empty(t, errors, "Should not have errors in concurrent extends")
		require.NotEmpty(t, extendResults, "Should have some successful extends")

		// Clean up - use the latest lease ID from extend results
		if len(extendResults) > 0 {
			// Use any valid extended lease ID for cleanup
			for _, result := range extendResults {
				if result != "" {
					// Convert back to ULID for cleanup
					leaseID, err := ulid.Parse(result)
					if err == nil {
						_, _ = te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
							IdempotencyKey: "cleanup-extended",
							AccountID:      te.AccountID,
							LeaseID:        leaseID,
									})
						break // Only need to clean up once
					}
				}
			}
		}

		// Verify final state
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})
	})

	t.Run("Concurrent Operations During Scavenging", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		// Create some leases that will expire
		for i := 0; i < 3; i++ {
			resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
				IdempotencyKey:       fmt.Sprintf("expired-%d", i),
				AccountID:            te.AccountID,
				EnvID:                te.EnvID,
				FunctionID:           te.FunctionID,
				Amount:               1,
				LeaseIdempotencyKeys: []string{fmt.Sprintf("expired-lease-%d", i)},
				CurrentTime:          clock.Now(),
				Duration:             5 * time.Second, // Will expire soon
				MaximumLifetime:      time.Minute,
				Configuration:        config,
				Constraints:          constraints,
				Source: LeaseSource{
					Service:  ServiceExecutor,
					Location: CallerLocationItemLease,
				},
				})
			require.NoError(t, err)
			require.Len(t, resp.Leases, 1)
		}

		// Advance time to expire the leases
		clock.Advance(10 * time.Second)
		te.AdvanceTimeAndRedis(10 * time.Second)

		var wg sync.WaitGroup
		var mu sync.Mutex
		var scavengeCount int
		var newLeases []CapacityLease
		var errors []error

		// Start scavenging in background
		wg.Add(1)
		go func() {
			defer wg.Done()

			res, err := te.CapacityManager.Scavenge(context.Background())

			mu.Lock()
			if err != nil {
				errors = append(errors, err)
			} else {
				scavengeCount = res.ReclaimedLeases
			}
			mu.Unlock()
		}()

		// Concurrent new acquisitions while scavenging
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(acquireID int) {
				defer wg.Done()

				resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
					IdempotencyKey:       fmt.Sprintf("during-scavenge-%d", acquireID),
					AccountID:            te.AccountID,
					EnvID:                te.EnvID,
					FunctionID:           te.FunctionID,
					Amount:               1,
					LeaseIdempotencyKeys: []string{fmt.Sprintf("during-scavenge-lease-%d", acquireID)},
					CurrentTime:          clock.Now(),
					Duration:             30 * time.Second,
					MaximumLifetime:      time.Minute,
					Configuration:        config,
					Constraints:          constraints,
					Source: LeaseSource{
						Service:  ServiceExecutor,
						Location: CallerLocationItemLease,
					},
						})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else if len(resp.Leases) > 0 {
					newLeases = append(newLeases, resp.Leases...)
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Verify no errors and consistent state
		require.Empty(t, errors, "Should not have errors during concurrent scavenging")
		require.NotZero(t, scavengeCount, "Should have scavenged some leases")
		require.True(t, len(newLeases) <= 5, "Should respect capacity limits")

		// Clean up new leases
		for _, lease := range newLeases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("cleanup-concurrent-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
			})
			require.NoError(t, err)
		}
	})
}

func TestConcurrencyAndRaces_IdempotencyRaces(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Concurrent Idempotent Requests", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 10,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		numGoroutines := 20
		idempotencyKey := "concurrent-idempotent-test"

		var wg sync.WaitGroup
		var mu sync.Mutex
		var responses []*CapacityAcquireResponse
		var errors []error

		// Multiple goroutines making the same idempotent request
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()

				resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
					IdempotencyKey:       idempotencyKey, // Same key for all
					AccountID:            te.AccountID,
					EnvID:                te.EnvID,
					FunctionID:           te.FunctionID,
					Amount:               2, // Same parameters for all
					LeaseIdempotencyKeys: []string{"idempotent-lease-1", "idempotent-lease-2"},
					CurrentTime:          clock.Now(),
					Duration:             30 * time.Second,
					MaximumLifetime:      time.Minute,
					Configuration:        config,
					Constraints:          constraints,
					Source: LeaseSource{
						Service:  ServiceExecutor,
						Location: CallerLocationItemLease,
					},
						})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					responses = append(responses, resp)
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Verify all requests succeeded
		require.Empty(t, errors, "Should not have errors in idempotent requests")
		require.Len(t, responses, numGoroutines, "Should have responses from all goroutines")

		// Verify all responses are identical (idempotent)
		firstResponse := responses[0]
		require.Len(t, firstResponse.Leases, 2, "Should have 2 leases")

		for i, resp := range responses {
			require.Len(t, resp.Leases, len(firstResponse.Leases), "Response %d should have same lease count", i)
			for j, lease := range resp.Leases {
				require.Equal(t, firstResponse.Leases[j].LeaseID, lease.LeaseID, "Response %d lease %d should be identical", i, j)
				require.Equal(t, firstResponse.Leases[j].IdempotencyKey, lease.IdempotencyKey, "Response %d lease %d idempotency key should be identical", i, j)
			}
		}

		// Clean up
		for _, lease := range firstResponse.Leases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("cleanup-idempotent-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
			})
			require.NoError(t, err)
		}
	})

	t.Run("Race Between Idempotent Operations", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		// First acquire a lease
		resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
			IdempotencyKey:       "race-base",
			AccountID:            te.AccountID,
			EnvID:                te.EnvID,
			FunctionID:           te.FunctionID,
			Amount:               1,
			LeaseIdempotencyKeys: []string{"race-base-lease"},
			CurrentTime:          clock.Now(),
			Duration:             30 * time.Second,
			MaximumLifetime:      time.Minute,
			Configuration:        config,
			Constraints:          constraints,
			Source: LeaseSource{
				Service:  ServiceExecutor,
				Location: CallerLocationItemLease,
			},
		})

		require.NoError(t, err)
		require.Len(t, resp.Leases, 1)
		originalLease := resp.Leases[0]

		var wg sync.WaitGroup
		var mu sync.Mutex
		var extendResults []bool
		var releaseResults []bool
		var errors []error

		idempotentExtendKey := "race-extend"
		idempotentReleaseKey := "race-release"

		// Concurrent extend operations with same idempotency key
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				extendResp, err := te.CapacityManager.ExtendLease(context.Background(), &CapacityExtendLeaseRequest{
					IdempotencyKey: idempotentExtendKey, // Same key
					AccountID:      te.AccountID,
					LeaseID:        originalLease.LeaseID,
					Duration:       60 * time.Second,
					})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					extendResults = append(extendResults, extendResp.LeaseID != nil)
				}
				mu.Unlock()
			}()
		}

		// Concurrent release operations with same idempotency key (after a delay)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Small delay to let extends happen first
				time.Sleep(10 * time.Millisecond)

				_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
					IdempotencyKey: idempotentReleaseKey, // Same key
					AccountID:      te.AccountID,
					LeaseID:        originalLease.LeaseID,
					})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					releaseResults = append(releaseResults, true)
				}
				mu.Unlock()
			}()
		}

		wg.Wait()

		// Verify no errors and idempotent behavior
		require.Empty(t, errors, "Should not have errors in idempotent race operations")
		require.NotEmpty(t, extendResults, "Should have extend results")
		require.NotEmpty(t, releaseResults, "Should have release results")

		// All extend results should be identical (idempotent)
		if len(extendResults) > 1 {
			for i := 1; i < len(extendResults); i++ {
				require.Equal(t, extendResults[0], extendResults[i], "All extend results should be identical")
			}
		}
	})
}

func TestConcurrencyAndRaces_ClockSkew(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Concurrent Operations with Clock Skew", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 10,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var allLeases []CapacityLease
		var errors []error

		baseTime := clock.Now()
		numGoroutines := 10

		// Concurrent operations with different clock skew
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()

				// Each goroutine uses slightly different time (simulating clock skew)
				clientTime := baseTime.Add(time.Duration(routineID-5) * time.Second) // -5 to +4 seconds

				resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
					IdempotencyKey:       fmt.Sprintf("clock-skew-%d", routineID),
					AccountID:            te.AccountID,
					EnvID:                te.EnvID,
					FunctionID:           te.FunctionID,
					Amount:               1,
					LeaseIdempotencyKeys: []string{fmt.Sprintf("clock-skew-lease-%d", routineID)},
					CurrentTime:          clientTime, // Different time for each client
					Duration:             30 * time.Second,
					MaximumLifetime:      time.Minute,
					Configuration:        config,
					Constraints:          constraints,
					Source: LeaseSource{
						Service:  ServiceExecutor,
						Location: CallerLocationItemLease,
					},
						})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else if len(resp.Leases) > 0 {
					allLeases = append(allLeases, resp.Leases...)
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Should handle reasonable clock skew gracefully
		require.True(t, len(errors) < numGoroutines/2, "Should accept most requests with reasonable clock skew")
		require.NotEmpty(t, allLeases, "Should have some successful acquisitions")

		// Clean up
		for _, lease := range allLeases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("cleanup-skew-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
			})
			require.NoError(t, err)
		}
	})
}

func TestConcurrencyAndRaces_NetworkPartitions(t *testing.T) {
	te := NewTestEnvironment(t)
	defer te.Cleanup()

	clock := clockwork.NewFakeClock()
	te.CapacityManager.clock = clock

	t.Run("Simulated Network Delays", func(t *testing.T) {
		config := ConstraintConfig{
			FunctionVersion: 1,
			Concurrency: ConcurrencyConfig{
				FunctionConcurrency: 5,
			},
		}

		constraints := []ConstraintItem{
			{
				Kind: ConstraintKindConcurrency,
				Concurrency: &ConcurrencyConstraint{
					Mode:  enums.ConcurrencyModeStep,
					Scope: enums.ConcurrencyScopeFn,
				},
			},
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var allLeases []CapacityLease
		var errors []error

		// Simulate operations with random delays (network latency)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()

				// Simulate network delay
				delay := time.Duration(routineID%5) * 10 * time.Millisecond
				time.Sleep(delay)

				resp, err := te.CapacityManager.Acquire(context.Background(), &CapacityAcquireRequest{
					IdempotencyKey:       fmt.Sprintf("network-%d", routineID),
					AccountID:            te.AccountID,
					EnvID:                te.EnvID,
					FunctionID:           te.FunctionID,
					Amount:               1,
					LeaseIdempotencyKeys: []string{fmt.Sprintf("network-lease-%d", routineID)},
					CurrentTime:          clock.Now().Add(delay), // Account for delay in timestamp
					Duration:             30 * time.Second,
					MaximumLifetime:      time.Minute,
					Configuration:        config,
					Constraints:          constraints,
					Source: LeaseSource{
						Service:  ServiceExecutor,
						Location: CallerLocationItemLease,
					},
						})

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else if len(resp.Leases) > 0 {
					allLeases = append(allLeases, resp.Leases...)
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Should handle network delays gracefully
		require.Empty(t, errors, "Should handle network delays without errors")
		require.True(t, len(allLeases) <= 5, "Should respect capacity limits despite delays")
		require.NotEmpty(t, allLeases, "Should have some successful acquisitions")

		// Clean up
		for _, lease := range allLeases {
			_, err := te.CapacityManager.Release(context.Background(), &CapacityReleaseRequest{
				IdempotencyKey: fmt.Sprintf("cleanup-network-%s", lease.IdempotencyKey),
				AccountID:      te.AccountID,
				LeaseID:        lease.LeaseID,
			})
			require.NoError(t, err)
		}

		// Verify final consistency
		cv := te.NewConstraintVerifier()
		cv.VerifyInProgressCounts(constraints, map[string]int{"constraint_0": 0})
	})
}
