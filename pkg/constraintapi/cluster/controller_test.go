package cluster

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func liveSet(nodes ...string) func(string) bool {
	return func(node string) bool {
		for _, n := range nodes {
			if n == node {
				return true
			}
		}
		return false
	}
}

func lsnTable(lsn map[string]map[int]uint64) func(string, int) uint64 {
	return func(node string, shard int) uint64 {
		return lsn[node][shard]
	}
}

func TestPlanLivePrimaryNeedsNothing(t *testing.T) {
	tbl := newTable(t, 3)
	require.Empty(t, Plan(tbl, liveSet("a", "b", "c"), lsnTable(nil)))
}

func TestPlanElectsHighestLSN(t *testing.T) {
	// a is primary of shard 0 only.
	tbl := newTable(t, 3)
	cmds := Plan(tbl, liveSet("b", "c"), lsnTable(map[string]map[int]uint64{
		"a": {0: 99},
		"b": {0: 10},
		"c": {0: 11},
	}))
	require.Equal(t, []Command{
		ElectPrimary{
			Shard: 0,
			Epoch: 1,
			Node:  "c",
		},
	}, cmds)
}

func TestPlanTieGoesToEarliestInSync(t *testing.T) {
	tbl := newTable(t, 3)
	// shard 2 has the in-sync order c, a, b.
	cmds := Plan(tbl, liveSet("a", "b"), lsnTable(map[string]map[int]uint64{
		"a": {2: 7},
		"b": {2: 7},
	}))
	require.Equal(t, []Command{
		ElectPrimary{
			Shard: 2,
			Epoch: 1,
			Node:  "a",
		},
	}, cmds)

	// with no report at all the LSN is 0 for every member.
	cmds = Plan(tbl, liveSet("a", "b"), lsnTable(nil))
	require.Equal(t, "a", cmds[0].(ElectPrimary).Node)
}

func TestPlanSkipsDeadAndOutOfSyncMembers(t *testing.T) {
	tbl := newTable(t, 1)
	// c is ahead and live, and it is not in sync.
	require.NoError(t, tbl.Apply(ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "c",
	}))
	lsn := lsnTable(map[string]map[int]uint64{
		"b": {0: 5},
		"c": {0: 50},
	})
	require.Equal(t, []Command{
		ElectPrimary{
			Shard: 0,
			Epoch: 2,
			Node:  "b",
		},
	}, Plan(tbl, liveSet("b", "c"), lsn))

	// no live in-sync member means no command.
	require.Empty(t, Plan(tbl, liveSet("c"), lsn))
	require.Empty(t, Plan(tbl, liveSet(), lsn))
}

func TestPlanEmptySeat(t *testing.T) {
	tbl := newTable(t, 3)
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		BootID: "a-boot-2",
	}))
	// every node is live.  only shard 0 lost its primary.
	cmds := Plan(tbl, liveSet("a", "b", "c"), lsnTable(map[string]map[int]uint64{
		"b": {0: 3},
		"c": {0: 2},
	}))
	require.Equal(t, []Command{
		ElectPrimary{
			Shard: 0,
			Epoch: 2,
			Node:  "b",
		},
	}, cmds)
}

func TestPlanCommandsApplyAndCoverEveryShard(t *testing.T) {
	tbl := newTable(t, 6)
	cmds := Plan(tbl, liveSet("b", "c"), lsnTable(nil))
	// a is primary of shards 0 and 3.  commands come in shard order.
	require.Len(t, cmds, 2)
	require.Equal(t, 0, cmds[0].(ElectPrimary).Shard)
	require.Equal(t, 3, cmds[1].(ElectPrimary).Shard)
	for _, cmd := range cmds {
		require.NoError(t, tbl.Apply(cmd))
	}
	require.Empty(t, Plan(tbl, liveSet("b", "c"), lsnTable(nil)))
}
