package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// SleepMigrationDestination atomically handles legacy and run-scoped sleep IDs.
// Migration must verify the returned identity before removing the source item.
type SleepMigrationDestination interface {
	EnqueueSleepForMigration(context.Context, QueueItem, time.Time, EnqueueOpts) (QueueItem, error)
}

// MigrationSleepID separates legacy sleep IDs reused by different runs during
// a shard cutover. Preserve an already migrated ID on subsequent migrations.
func MigrationSleepID(item QueueItem) string {
	if strings.HasPrefix(item.ID, "migration-sleep-") {
		return item.ID
	}
	scope := ScopeFromQueueItem(item)
	input := "migration-sleep:" + scope.AccountID.String() + ":" + scope.EnvID.String() + ":" + scope.FunctionID.String() + ":" + item.Data.Identifier.RunID.String() + ":" + item.ID
	digest := sha256.Sum256([]byte(input))
	return "migration-sleep-" + hex.EncodeToString(digest[:])
}
