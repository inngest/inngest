package queue

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

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
