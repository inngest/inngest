package queue

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// MigrationSleepID separates legacy sleep IDs reused by different runs during
// a shard cutover. Preserve an already migrated ID on subsequent migrations.
func MigrationSleepID(item QueueItem) string {
	if strings.HasPrefix(item.ID, "migration-sleep-") {
		return item.ID
	}
	scope := ScopeFromQueueItem(item)
	digest := sha256.Sum256([]byte(fmt.Sprintf("migration-sleep:%s:%s:%s:%s:%s", scope.AccountID, scope.EnvID, scope.FunctionID, item.Data.Identifier.RunID, item.ID)))
	return fmt.Sprintf("migration-sleep-%x", digest)
}
