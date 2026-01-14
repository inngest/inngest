package constraintapi

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuplicateTracker_SingleRequest(t *testing.T) {
	tracker := newDuplicateTracker(100 * time.Millisecond)
	defer tracker.Close()

	accountID := uuid.New()
	fingerprint := "test-fingerprint-1"

	// Track a single request
	tracker.track(accountID, fingerprint)

	// Verify entry was created
	tracker.mu.RLock()
	acct, exists := tracker.accounts[accountID]
	tracker.mu.RUnlock()
	require.True(t, exists, "account should exist")

	acct.mu.Lock()
	entry, exists := acct.fingerprints[fingerprint]
	acct.mu.Unlock()
	require.True(t, exists, "fingerprint entry should exist")

	assert.Equal(t, int64(1), entry.count, "count should be 1 for single request")

	// Wait for timer to fire and cleanup
	time.Sleep(150 * time.Millisecond)

	// Verify cleanup occurred
	tracker.mu.RLock()
	_, exists = tracker.accounts[accountID]
	tracker.mu.RUnlock()
	assert.False(t, exists, "account should be cleaned up after timer expiry")
}

func TestDuplicateTracker_DuplicateRequests(t *testing.T) {
	tracker := newDuplicateTracker(100 * time.Millisecond)
	defer tracker.Close()

	accountID := uuid.New()
	fingerprint := "test-fingerprint-2"

	// Track multiple duplicate requests
	for i := 0; i < 5; i++ {
		tracker.track(accountID, fingerprint)
	}

	// Verify count
	tracker.mu.RLock()
	acct, exists := tracker.accounts[accountID]
	tracker.mu.RUnlock()
	require.True(t, exists, "account should exist")

	acct.mu.Lock()
	entry, exists := acct.fingerprints[fingerprint]
	count := entry.count
	acct.mu.Unlock()
	require.True(t, exists, "fingerprint entry should exist")

	assert.Equal(t, int64(5), count, "count should be 5 for duplicate requests")

	// Wait for timer to fire and cleanup
	time.Sleep(150 * time.Millisecond)

	// Verify cleanup occurred
	tracker.mu.RLock()
	_, exists = tracker.accounts[accountID]
	tracker.mu.RUnlock()
	assert.False(t, exists, "account should be cleaned up after timer expiry")
}

func TestDuplicateTracker_WindowExpiry(t *testing.T) {
	tracker := newDuplicateTracker(50 * time.Millisecond)
	defer tracker.Close()

	accountID := uuid.New()
	fingerprint := "test-fingerprint-3"

	// Track initial request
	tracker.track(accountID, fingerprint)

	// Wait for half the window
	time.Sleep(25 * time.Millisecond)

	// Track duplicate before window expires
	tracker.track(accountID, fingerprint)

	// Verify entry still exists
	tracker.mu.RLock()
	acct, exists := tracker.accounts[accountID]
	tracker.mu.RUnlock()
	require.True(t, exists)

	acct.mu.Lock()
	entry, exists := acct.fingerprints[fingerprint]
	count := entry.count
	acct.mu.Unlock()
	require.True(t, exists)
	assert.Equal(t, int64(2), count)

	// Wait for window to fully expire
	time.Sleep(50 * time.Millisecond)

	// Verify cleanup occurred
	tracker.mu.RLock()
	_, exists = tracker.accounts[accountID]
	tracker.mu.RUnlock()
	assert.False(t, exists, "entry should be cleaned up after window expiry")

	// Track new request after window expiry
	tracker.track(accountID, fingerprint)

	// Verify new entry was created with count=1
	tracker.mu.RLock()
	acct, exists = tracker.accounts[accountID]
	tracker.mu.RUnlock()
	require.True(t, exists)

	acct.mu.Lock()
	entry, exists = acct.fingerprints[fingerprint]
	newCount := entry.count
	acct.mu.Unlock()
	require.True(t, exists)
	assert.Equal(t, int64(1), newCount, "count should reset to 1 for new window")
}

func TestDuplicateTracker_ConcurrentAccess(t *testing.T) {
	tracker := newDuplicateTracker(200 * time.Millisecond)
	defer tracker.Close()

	accountID := uuid.New()
	fingerprint := "test-fingerprint-4"

	// Track requests concurrently from multiple goroutines
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			tracker.track(accountID, fingerprint)
		}()
	}

	wg.Wait()

	// Verify count
	tracker.mu.RLock()
	acct, exists := tracker.accounts[accountID]
	tracker.mu.RUnlock()
	require.True(t, exists)

	acct.mu.Lock()
	entry, exists := acct.fingerprints[fingerprint]
	count := entry.count
	acct.mu.Unlock()
	require.True(t, exists)

	assert.Equal(t, int64(numGoroutines), count, "count should match number of concurrent requests")

	// Wait for cleanup
	time.Sleep(250 * time.Millisecond)

	tracker.mu.RLock()
	_, exists = tracker.accounts[accountID]
	tracker.mu.RUnlock()
	assert.False(t, exists, "account should be cleaned up")
}

func TestDuplicateTracker_PerAccount(t *testing.T) {
	tracker := newDuplicateTracker(100 * time.Millisecond)
	defer tracker.Close()

	account1 := uuid.New()
	account2 := uuid.New()
	fingerprint := "test-fingerprint-5"

	// Track requests for two different accounts with same fingerprint
	tracker.track(account1, fingerprint)
	tracker.track(account1, fingerprint)
	tracker.track(account1, fingerprint)

	tracker.track(account2, fingerprint)
	tracker.track(account2, fingerprint)

	// Verify separate tracking per account
	tracker.mu.RLock()
	acct1, exists1 := tracker.accounts[account1]
	acct2, exists2 := tracker.accounts[account2]
	tracker.mu.RUnlock()

	require.True(t, exists1)
	require.True(t, exists2)

	acct1.mu.Lock()
	entry1 := acct1.fingerprints[fingerprint]
	count1 := entry1.count
	acct1.mu.Unlock()

	acct2.mu.Lock()
	entry2 := acct2.fingerprints[fingerprint]
	count2 := entry2.count
	acct2.mu.Unlock()

	assert.Equal(t, int64(3), count1, "account1 should have count 3")
	assert.Equal(t, int64(2), count2, "account2 should have count 2")
}

func TestDuplicateTracker_Disabled(t *testing.T) {
	tracker := newDuplicateTracker(0) // disabled
	defer tracker.Close()

	accountID := uuid.New()
	fingerprint := "test-fingerprint-6"

	// Track request - should be no-op
	tracker.track(accountID, fingerprint)

	// Verify no tracking occurred
	tracker.mu.RLock()
	_, exists := tracker.accounts[accountID]
	tracker.mu.RUnlock()

	assert.False(t, exists, "no tracking should occur when disabled")
}

func TestDuplicateTracker_EmptyFingerprint(t *testing.T) {
	tracker := newDuplicateTracker(100 * time.Millisecond)
	defer tracker.Close()

	accountID := uuid.New()

	// Track request with empty fingerprint
	tracker.track(accountID, "")

	// Verify no tracking occurred
	tracker.mu.RLock()
	_, exists := tracker.accounts[accountID]
	tracker.mu.RUnlock()

	assert.False(t, exists, "no tracking should occur for empty fingerprint")
}

func TestDuplicateTracker_Cleanup(t *testing.T) {
	tracker := newDuplicateTracker(50 * time.Millisecond)
	defer tracker.Close()

	accountID := uuid.New()

	// Track multiple fingerprints
	tracker.track(accountID, "fingerprint-1")
	tracker.track(accountID, "fingerprint-2")
	tracker.track(accountID, "fingerprint-3")

	// Verify all fingerprints are tracked
	tracker.mu.RLock()
	acct, exists := tracker.accounts[accountID]
	tracker.mu.RUnlock()
	require.True(t, exists)

	acct.mu.Lock()
	fpCount := len(acct.fingerprints)
	acct.mu.Unlock()
	assert.Equal(t, 3, fpCount, "should have 3 fingerprints")

	// Wait for all timers to fire
	time.Sleep(100 * time.Millisecond)

	// Verify complete cleanup
	tracker.mu.RLock()
	_, exists = tracker.accounts[accountID]
	tracker.mu.RUnlock()
	assert.False(t, exists, "account should be cleaned up when all fingerprints expire")
}

func TestDuplicateTracker_Close(t *testing.T) {
	tracker := newDuplicateTracker(500 * time.Millisecond)

	accountID := uuid.New()
	fingerprint := "test-fingerprint-7"

	// Track request
	tracker.track(accountID, fingerprint)

	// Verify entry exists
	tracker.mu.RLock()
	acct, exists := tracker.accounts[accountID]
	tracker.mu.RUnlock()
	require.True(t, exists)

	acct.mu.Lock()
	_, fpExists := acct.fingerprints[fingerprint]
	acct.mu.Unlock()
	require.True(t, fpExists)

	// Close tracker
	tracker.Close()

	// Verify cleanup
	tracker.mu.RLock()
	accountsNil := tracker.accounts == nil
	disabled := !tracker.enabled
	tracker.mu.RUnlock()

	assert.True(t, accountsNil, "accounts map should be nil after Close")
	assert.True(t, disabled, "tracker should be disabled after Close")
}

func TestDuplicateTracker_MultipleFingerprints(t *testing.T) {
	tracker := newDuplicateTracker(100 * time.Millisecond)
	defer tracker.Close()

	accountID := uuid.New()

	// Track different fingerprints
	tracker.track(accountID, "fingerprint-A")
	tracker.track(accountID, "fingerprint-A")

	tracker.track(accountID, "fingerprint-B")
	tracker.track(accountID, "fingerprint-B")
	tracker.track(accountID, "fingerprint-B")

	// Verify separate tracking per fingerprint
	tracker.mu.RLock()
	acct, exists := tracker.accounts[accountID]
	tracker.mu.RUnlock()
	require.True(t, exists)

	acct.mu.Lock()
	entryA := acct.fingerprints["fingerprint-A"]
	entryB := acct.fingerprints["fingerprint-B"]
	countA := entryA.count
	countB := entryB.count
	acct.mu.Unlock()

	assert.Equal(t, int64(2), countA, "fingerprint-A should have count 2")
	assert.Equal(t, int64(3), countB, "fingerprint-B should have count 3")
}
