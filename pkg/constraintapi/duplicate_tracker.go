package constraintapi

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
)

// duplicateTracker tracks duplicate Acquire requests within a configurable debounce window
// to detect retry storms and client-side issues. When the window expires, it emits a histogram
// metric with the total duplicate count.
type duplicateTracker struct {
	mu       sync.RWMutex
	accounts map[uuid.UUID]*accountTracking

	debounceWindow time.Duration
	enabled        bool
}

// accountTracking holds fingerprint tracking state for a single account
type accountTracking struct {
	mu           sync.Mutex
	fingerprints map[string]*fingerprintEntry
}

// fingerprintEntry tracks duplicate requests for a single fingerprint
type fingerprintEntry struct {
	count       int64
	firstSeen   time.Time
	timer       *time.Timer
	accountID   uuid.UUID
	fingerprint string
}

// newDuplicateTracker creates a new duplicate request tracker with the specified debounce window.
// If window is 0 or negative, tracking is disabled.
func newDuplicateTracker(window time.Duration) *duplicateTracker {
	return &duplicateTracker{
		accounts:       make(map[uuid.UUID]*accountTracking),
		debounceWindow: window,
		enabled:        window > 0,
	}
}

// track records an Acquire request for the given account and fingerprint.
// If this is the first request for this fingerprint, it starts a timer that will
// emit a metric when the debounce window expires. If this is a duplicate request,
// it increments the counter.
func (dt *duplicateTracker) track(accountID uuid.UUID, fingerprint string) {
	// Fast path: skip if disabled or invalid fingerprint
	if !dt.enabled || dt.debounceWindow == 0 || fingerprint == "" {
		return
	}

	// Get or create account-level tracking (double-checked locking pattern)
	dt.mu.RLock()
	acct, exists := dt.accounts[accountID]
	dt.mu.RUnlock()

	if !exists {
		dt.mu.Lock()
		// Double-check after acquiring write lock
		acct, exists = dt.accounts[accountID]
		if !exists {
			acct = &accountTracking{
				fingerprints: make(map[string]*fingerprintEntry),
			}
			dt.accounts[accountID] = acct
		}
		dt.mu.Unlock()
	}

	// Track at fingerprint level
	acct.mu.Lock()
	defer acct.mu.Unlock()

	entry, exists := acct.fingerprints[fingerprint]
	if exists {
		// Duplicate - increment counter
		entry.count++
	} else {
		// New fingerprint - create entry with timer
		entry = &fingerprintEntry{
			count:       1,
			firstSeen:   time.Now(),
			accountID:   accountID,
			fingerprint: fingerprint,
		}

		// Set timer to emit metric and cleanup
		entry.timer = time.AfterFunc(dt.debounceWindow, func() {
			dt.emitAndCleanup(accountID, fingerprint, entry)
		})

		acct.fingerprints[fingerprint] = entry
	}
}

// emitAndCleanup is called when the debounce window expires for a fingerprint.
// It emits a histogram metric if there were duplicates (count > 1) and cleans up
// the tracking entry.
func (dt *duplicateTracker) emitAndCleanup(accountID uuid.UUID, fingerprint string, entry *fingerprintEntry) {
	// Emit histogram only for duplicates (count > 1)
	if entry.count > 1 {
		metrics.HistogramConstraintAPIDuplicateAcquireRequests(
			context.Background(),
			entry.count,
			metrics.HistogramOpt{
				PkgName: "constraintapi",
				Tags: map[string]any{
					"account_id": accountID.String(),
				},
			},
		)
	}

	// Cleanup fingerprint entry
	dt.mu.RLock()
	acct, exists := dt.accounts[accountID]
	dt.mu.RUnlock()

	if exists {
		acct.mu.Lock()
		delete(acct.fingerprints, fingerprint)
		isEmpty := len(acct.fingerprints) == 0
		acct.mu.Unlock()

		// Clean up empty account map to prevent memory leaks
		if isEmpty {
			dt.mu.Lock()
			// Double-check before deleting account
			if len(acct.fingerprints) == 0 {
				delete(dt.accounts, accountID)
			}
			dt.mu.Unlock()
		}
	}
}

// Close stops all active timers and disables tracking.
// This should be called when the capacity manager is shutting down.
func (dt *duplicateTracker) Close() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	for _, acct := range dt.accounts {
		acct.mu.Lock()
		for _, entry := range acct.fingerprints {
			if entry.timer != nil {
				entry.timer.Stop()
			}
		}
		acct.mu.Unlock()
	}

	dt.accounts = nil
	dt.enabled = false
}
