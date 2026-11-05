package queue

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
)

// MigrationSleepID gives a sleep a stable identity for its run, whether created
// by the executor or copied during migration. Preserve already scoped IDs.
func MigrationSleepID(item QueueItem) string {
	if strings.HasPrefix(item.ID, "migration-sleep-") {
		return item.ID
	}
	scope := ScopeFromQueueItem(item)
	input := "migration-sleep:" + scope.AccountID.String() + ":" + scope.EnvID.String() + ":" + scope.FunctionID.String() + ":" + item.Data.Identifier.RunID.String() + ":" + item.ID
	digest := sha256.Sum256([]byte(input))
	return "migration-sleep-" + hex.EncodeToString(digest[:])
}

// ValidateSleepJobID limits legacy-ID compatibility to this run's function sleep.
func ValidateSleepJobID(item QueueItem, opts EnqueueOpts) error {
	// Normalization moves an existing item between indexes, not to another copy.
	if opts.NormalizeFromBacklogID != "" && opts.LegacySleepID == "" {
		return nil
	}
	if item.LegacySleepID != "" && item.LegacySleepID != opts.LegacySleepID {
		return fmt.Errorf("sleep legacy ID does not match its saved identity")
	}
	if opts.LegacySleepID == "" {
		if item.Data.Kind == KindSleep && strings.HasPrefix(item.ID, "migration-sleep-") {
			return fmt.Errorf("run-scoped sleep is missing its original legacy ID")
		}
		return nil
	}
	legacy := item
	legacy.ID = opts.LegacySleepID
	if item.Data.Kind != KindSleep || ScopeFromQueueItem(item).IsSystem || item.Data.Identifier.RunID == (ulid.ULID{}) || !opts.PassthroughJobId || opts.NormalizeFromBacklogID != "" || item.ID == legacy.ID || item.ID != MigrationSleepID(legacy) {
		return fmt.Errorf("legacy sleep compatibility requires a run-scoped function sleep ID")
	}
	return ScopeFromQueueItem(item).Validate()
}
