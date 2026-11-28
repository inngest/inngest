package redis_state

import (
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	osqueue "github.com/inngest/inngest/pkg/execution/queue"
	"github.com/stretchr/testify/require"
)

func TestHasPendingMigrationItems(t *testing.T) {
	for _, tc := range []string{"empty", "ready", "backlog", "normalizing", "read error", "wrong partition"} {
		t.Run(tc, func(t *testing.T) {
			r, rc := initRedis(t)
			t.Cleanup(rc.Close)
			shard := shardFromClient("source", rc)
			scope := osqueue.Scope{AccountID: uuid.New(), EnvID: uuid.New(), FunctionID: uuid.New()}
			partition := scope.FunctionID.String()
			kg := shard.Client().KeyGenerator()
			var key string
			switch tc {
			case "ready":
				key = kg.PartitionQueueSet(enums.PartitionTypeDefault, partition, "")
			case "backlog":
				key = kg.ShadowPartitionSet(partition)
			case "normalizing":
				key = kg.PartitionNormalizeSet(partition)
			case "read error":
				r.SetError("ERR read unavailable")
			case "wrong partition":
				partition = uuid.NewString()
			}
			if key != "" {
				_, err := r.ZAdd(key, 0, "pending")
				require.NoError(t, err)
			}
			pending, err := shard.HasPendingMigrationItems(t.Context(), scope, partition)
			if tc == "read error" || tc == "wrong partition" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc != "empty", pending)
		})
	}
}
