package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// SleepMigrationDestination atomically handles legacy and run-scoped sleep IDs.
// Migration must verify the returned identity before removing the source item.
type SleepMigrationDestination interface {
	EnqueueSleepForMigration(context.Context, QueueItem, time.Time, EnqueueOpts) (QueueItem, error)
}

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
	if opts.LegacySleepID == "" {
		return nil
	}
	legacy := item
	legacy.ID = opts.LegacySleepID
	if item.Data.Kind != KindSleep || ScopeFromQueueItem(item).IsSystem || item.Data.Identifier.RunID == (ulid.ULID{}) || !opts.PassthroughJobId || opts.NormalizeFromBacklogID != "" || item.ID == legacy.ID || item.ID != MigrationSleepID(legacy) {
		return fmt.Errorf("legacy sleep compatibility requires a run-scoped function sleep ID")
	}
	return ScopeFromQueueItem(item).Validate()
}
