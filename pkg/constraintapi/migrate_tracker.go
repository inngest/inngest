package constraintapi

import (
	"sync"

	"github.com/google/uuid"
)

// migrationDirtyTracker tracks Redis keys that have been modified during a migration.
// During the copy phase of a migration, Extend and Release operations continue on the
// source shard. Any keys they modify need to be re-copied in subsequent delta passes.
//
// This tracker is thread-safe and supports concurrent MarkDirty calls from multiple
// goroutines handling Extend/Release operations.
type migrationDirtyTracker struct {
	mu   sync.Mutex
	keys map[string]struct{}
}

func newMigrationDirtyTracker() *migrationDirtyTracker {
	return &migrationDirtyTracker{
		keys: make(map[string]struct{}),
	}
}

// MarkDirty records one or more Redis keys as modified during migration.
// These keys will need to be re-copied to the destination shard.
func (t *migrationDirtyTracker) MarkDirty(keys ...string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, k := range keys {
		t.keys[k] = struct{}{}
	}
}

// DrainAndReset atomically returns all dirty keys and resets the tracker.
// This is called between convergent delta passes to get the set of keys
// that need to be re-copied.
func (t *migrationDirtyTracker) DrainAndReset() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]string, 0, len(t.keys))
	for k := range t.keys {
		result = append(result, k)
	}

	t.keys = make(map[string]struct{})
	return result
}

// Len returns the current number of dirty keys.
func (t *migrationDirtyTracker) Len() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.keys)
}

// migrationAccountState tracks the per-account migration state including
// which accounts are being migrated and their dirty key trackers.
type migrationAccountState struct {
	mu       sync.RWMutex
	accounts map[uuid.UUID]*migrationDirtyTracker
}

func newMigrationAccountState() *migrationAccountState {
	return &migrationAccountState{
		accounts: make(map[uuid.UUID]*migrationDirtyTracker),
	}
}

// StartTracking begins dirty key tracking for an account.
// Returns the tracker for that account.
func (s *migrationAccountState) StartTracking(accountID uuid.UUID) *migrationDirtyTracker {
	s.mu.Lock()
	defer s.mu.Unlock()

	tracker := newMigrationDirtyTracker()
	s.accounts[accountID] = tracker
	return tracker
}

// StopTracking stops dirty key tracking for an account and removes it.
func (s *migrationAccountState) StopTracking(accountID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.accounts, accountID)
}

// GetTracker returns the dirty key tracker for an account, or nil if
// the account is not being migrated.
func (s *migrationAccountState) GetTracker(accountID uuid.UUID) *migrationDirtyTracker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accounts[accountID]
}

// IsMigrating returns whether an account is currently being migrated.
func (s *migrationAccountState) IsMigrating(accountID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.accounts[accountID]
	return ok
}
