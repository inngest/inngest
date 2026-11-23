package cluster

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// newTable returns a table with the nodes a, b and c registered and the
// given number of shards at RF 3.
func newTable(t *testing.T, shards int) *Table {
	t.Helper()
	tbl := &Table{}
	for _, n := range []string{"a", "b", "c"} {
		require.NoError(t, tbl.Apply(RegisterNode{
			ID:     n,
			Addr:   n + "-addr",
			BootID: n + "-boot-1",
		}))
	}
	require.NoError(t, tbl.Apply(Init{
		Shards: shards,
		RF:     3,
		Nodes:  []string{"a", "b", "c"},
	}))
	return tbl
}

// requireRejected checks that the command is refused and that the table is
// the same as before, byte for byte.
func requireRejected(t *testing.T, tbl *Table, cmd Command) {
	t.Helper()
	before := tbl.Clone()
	err := tbl.Apply(cmd)
	require.ErrorIs(t, err, ErrRejected)
	require.Equal(t, before, tbl)
}

func TestTableInit(t *testing.T) {
	tbl := &Table{}
	require.NoError(t, tbl.Apply(Init{
		Shards: 4,
		RF:     2,
		Nodes:  []string{"a", "b", "c"},
	}))

	require.Equal(t, uint64(1), tbl.Version)
	require.Equal(t, 2, tbl.RF)
	require.Len(t, tbl.Shards, 4)
	require.Equal(t, ShardRow{
		Nonce:    1,
		Epoch:    1,
		Term:     1,
		Primary:  "a",
		Replicas: []string{"a", "b"},
		InSync:   []string{"a", "b"},
	}, tbl.Shards[0])
	require.Equal(t, []string{"c", "a"}, tbl.Shards[2].Replicas)
	require.Equal(t, "c", tbl.Shards[2].Primary)
	require.Equal(t, []string{"a", "b"}, tbl.Shards[3].Replicas)

	nonces := map[uint32]bool{}
	for _, row := range tbl.Shards {
		require.NotZero(t, row.Nonce)
		require.False(t, nonces[row.Nonce], "nonce %d issued twice", row.Nonce)
		nonces[row.Nonce] = true
	}
	require.Equal(t, uint32(5), tbl.NextNonce)
}

func TestTableInitRejected(t *testing.T) {
	for name, cmd := range map[string]Init{
		"no shards": {
			Shards: 0,
			RF:     1,
			Nodes:  []string{"a"},
		},
		"RF zero": {
			Shards: 1,
			RF:     0,
			Nodes:  []string{"a"},
		},
		"RF above node count": {
			Shards: 1,
			RF:     3,
			Nodes:  []string{"a", "b"},
		},
		"node listed twice": {
			Shards: 1,
			RF:     2,
			Nodes:  []string{"a", "a"},
		},
		"empty node name": {
			Shards: 1,
			RF:     1,
			Nodes:  []string{""},
		},
	} {
		t.Run(name, func(t *testing.T) {
			requireRejected(t, &Table{}, cmd)
		})
	}

	t.Run("already has shards", func(t *testing.T) {
		requireRejected(t, newTable(t, 2), Init{
			Shards: 2,
			RF:     1,
			Nodes:  []string{"a"},
		})
	})
}

func TestTableRegisterNode(t *testing.T) {
	tbl := newTable(t, 3)
	version := tbl.Version

	// an unknown node is added and no row changes.
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "d",
		Addr:   "d-addr",
		BootID: "d-boot-1",
	}))
	require.Equal(t, NodeInfo{
		Addr:   "d-addr",
		BootID: "d-boot-1",
	}, tbl.Nodes["d"])
	require.Equal(t, version+1, tbl.Version)

	// the same boot ID only updates the address.
	rows := tbl.Clone().Shards
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		Addr:   "a-addr-2",
		BootID: "a-boot-1",
	}))
	require.Equal(t, "a-addr-2", tbl.Nodes["a"].Addr)
	require.Equal(t, rows, tbl.Shards)
	require.Equal(t, version+2, tbl.Version)

	requireRejected(t, tbl, RegisterNode{
		ID:     "",
		BootID: "x",
	})
	requireRejected(t, tbl, RegisterNode{
		ID:     "a",
		BootID: "",
	})
}

func TestTableRegisterNodeBeforeInitKeepsSeats(t *testing.T) {
	tbl := &Table{}
	require.NoError(t, tbl.Apply(Init{
		Shards: 1,
		RF:     2,
		Nodes:  []string{"a", "b"},
	}))
	// the first registration of a node that Init named is not a restart.
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		Addr:   "a",
		BootID: "boot-1",
	}))
	require.Equal(t, "a", tbl.Shards[0].Primary)
	require.Equal(t, []string{"a", "b"}, tbl.Shards[0].InSync)
	require.Equal(t, uint64(1), tbl.Shards[0].Epoch)
}

func TestTableRestartRule(t *testing.T) {
	// shard 0 has primary a, shard 1 has primary b, shard 2 has primary c.
	tbl := newTable(t, 3)
	version := tbl.Version
	nextNonce := tbl.NextNonce

	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		Addr:   "a-addr",
		BootID: "a-boot-2",
	}))
	require.Equal(t, version+1, tbl.Version)
	require.Equal(t, "a-boot-2", tbl.Nodes["a"].BootID)

	// a was primary of shard 0.  the seat is empty and the term moved.
	require.Equal(t, ShardRow{
		Nonce:    1,
		Epoch:    2,
		Term:     2,
		Primary:  "",
		Replicas: []string{"a", "b", "c"},
		InSync:   []string{"b", "c"},
	}, tbl.Shards[0])
	// a was a follower of shard 1.  only the in-sync set and the epoch
	// changed.
	require.Equal(t, ShardRow{
		Nonce:    2,
		Epoch:    2,
		Term:     1,
		Primary:  "b",
		Replicas: []string{"b", "c", "a"},
		InSync:   []string{"b", "c"},
	}, tbl.Shards[1])
	require.Equal(t, nextNonce, tbl.NextNonce)

	// a row that does not list the node in sync is not changed.
	before := tbl.Clone().Shards
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		Addr:   "a-addr",
		BootID: "a-boot-3",
	}))
	require.Equal(t, before, tbl.Shards)
}

func TestTableRestartRuleEmptyInSync(t *testing.T) {
	tbl := newTable(t, 1)
	// b and c restart.  a is alone in the in-sync set.
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "b",
		BootID: "b-boot-2",
	}))
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "c",
		BootID: "c-boot-2",
	}))
	require.Equal(t, []string{"a"}, tbl.Shards[0].InSync)
	require.Equal(t, uint64(3), tbl.Shards[0].Epoch)
	require.Equal(t, uint64(1), tbl.Shards[0].Term)

	// a restarts.  no in-sync copy is left, so a starts the shard empty.
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		BootID: "a-boot-2",
	}))
	require.Equal(t, ShardRow{
		Nonce:    2,
		Epoch:    4,
		Term:     2,
		Primary:  "a",
		Replicas: []string{"a", "b", "c"},
		InSync:   []string{"a"},
	}, tbl.Shards[0])
	require.Equal(t, uint32(3), tbl.NextNonce)
}

func TestTableRestartRuleEmptyInSyncWithNoPrimary(t *testing.T) {
	tbl := &Table{}
	require.NoError(t, tbl.Apply(Init{
		Shards: 1,
		RF:     2,
		Nodes:  []string{"a", "b"},
	}))
	for _, n := range []string{"a", "b"} {
		require.NoError(t, tbl.Apply(RegisterNode{
			ID:     n,
			BootID: "boot-1",
		}))
	}
	// the primary restarts, then the last in-sync member restarts before
	// an election.
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		BootID: "boot-2",
	}))
	require.Equal(t, "", tbl.Shards[0].Primary)
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "b",
		BootID: "boot-2",
	}))
	require.Equal(t, ShardRow{
		Nonce:    2,
		Epoch:    3,
		Term:     3,
		Primary:  "b",
		Replicas: []string{"a", "b"},
		InSync:   []string{"b"},
	}, tbl.Shards[0])
}

func TestTableShrinkInSync(t *testing.T) {
	tbl := newTable(t, 2)
	version := tbl.Version

	require.NoError(t, tbl.Apply(ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "b",
	}))
	require.Equal(t, ShardRow{
		Nonce:    1,
		Epoch:    2,
		Term:     1,
		Primary:  "a",
		Replicas: []string{"a", "b", "c"},
		InSync:   []string{"a", "c"},
	}, tbl.Shards[0])
	require.Equal(t, uint64(1), tbl.Shards[1].Epoch)
	require.Equal(t, version+1, tbl.Version)

	// the epoch the first command read is stale now.
	requireRejected(t, tbl, ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "c",
	})
	requireRejected(t, tbl, ShrinkInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "a",
	})
	requireRejected(t, tbl, ShrinkInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "b",
	})
	requireRejected(t, tbl, ShrinkInSync{
		Shard: 2,
		Epoch: 1,
		Node:  "b",
	})
	requireRejected(t, tbl, ShrinkInSync{
		Shard: -1,
		Epoch: 1,
		Node:  "b",
	})
}

func TestTableShrinkInSyncKeepsLastMember(t *testing.T) {
	tbl := newTable(t, 1)
	// the primary restarts, then c leaves.  b has the only full copy.
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		BootID: "a-boot-2",
	}))
	require.NoError(t, tbl.Apply(ShrinkInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "c",
	}))
	require.Equal(t, []string{"b"}, tbl.Shards[0].InSync)
	requireRejected(t, tbl, ShrinkInSync{
		Shard: 0,
		Epoch: 3,
		Node:  "b",
	})
}

func TestTableExpandInSync(t *testing.T) {
	tbl := newTable(t, 1)
	require.NoError(t, tbl.Apply(ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "b",
	}))
	version := tbl.Version

	requireRejected(t, tbl, ExpandInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "b",
	})
	requireRejected(t, tbl, ExpandInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "d",
	})
	requireRejected(t, tbl, ExpandInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "c",
	})
	requireRejected(t, tbl, ExpandInSync{
		Shard: 1,
		Epoch: 2,
		Node:  "b",
	})

	require.NoError(t, tbl.Apply(ExpandInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "b",
	}))
	require.Equal(t, ShardRow{
		Nonce:    1,
		Epoch:    3,
		Term:     1,
		Primary:  "a",
		Replicas: []string{"a", "b", "c"},
		InSync:   []string{"a", "c", "b"},
	}, tbl.Shards[0])
	require.Equal(t, version+1, tbl.Version)
}

func TestTableElectPrimary(t *testing.T) {
	tbl := newTable(t, 1)
	version := tbl.Version

	requireRejected(t, tbl, ElectPrimary{
		Shard: 0,
		Epoch: 2,
		Node:  "b",
	})
	requireRejected(t, tbl, ElectPrimary{
		Shard: 0,
		Epoch: 1,
		Node:  "a",
	})
	requireRejected(t, tbl, ElectPrimary{
		Shard: 0,
		Epoch: 1,
		Node:  "d",
	})
	requireRejected(t, tbl, ElectPrimary{
		Shard: 3,
		Epoch: 1,
		Node:  "b",
	})

	require.NoError(t, tbl.Apply(ElectPrimary{
		Shard: 0,
		Epoch: 1,
		Node:  "b",
	}))
	// the old primary stays in the in-sync set.
	require.Equal(t, ShardRow{
		Nonce:    1,
		Epoch:    2,
		Term:     2,
		Primary:  "b",
		Replicas: []string{"a", "b", "c"},
		InSync:   []string{"a", "b", "c"},
	}, tbl.Shards[0])
	require.Equal(t, version+1, tbl.Version)

	// a node outside the in-sync set cannot be elected.
	require.NoError(t, tbl.Apply(ShrinkInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "c",
	}))
	requireRejected(t, tbl, ElectPrimary{
		Shard: 0,
		Epoch: 3,
		Node:  "c",
	})
}

func TestTableElectPrimaryIntoEmptySeat(t *testing.T) {
	tbl := newTable(t, 1)
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		BootID: "a-boot-2",
	}))
	require.Equal(t, "", tbl.Shards[0].Primary)
	require.NoError(t, tbl.Apply(ElectPrimary{
		Shard: 0,
		Epoch: 2,
		Node:  "c",
	}))
	require.Equal(t, "c", tbl.Shards[0].Primary)
	require.Equal(t, uint64(3), tbl.Shards[0].Term)
	require.Equal(t, uint64(3), tbl.Shards[0].Epoch)
}

func TestTableMovePrimary(t *testing.T) {
	tbl := newTable(t, 1)
	version := tbl.Version

	requireRejected(t, tbl, MovePrimary{
		Shard: 0,
		Epoch: 2,
		Node:  "b",
	})
	requireRejected(t, tbl, MovePrimary{
		Shard: 0,
		Epoch: 1,
		Node:  "a",
	})
	requireRejected(t, tbl, MovePrimary{
		Shard: 0,
		Epoch: 1,
		Node:  "d",
	})

	require.NoError(t, tbl.Apply(MovePrimary{
		Shard: 0,
		Epoch: 1,
		Node:  "c",
	}))
	require.Equal(t, ShardRow{
		Nonce:    1,
		Epoch:    2,
		Term:     2,
		Primary:  "c",
		Replicas: []string{"a", "b", "c"},
		InSync:   []string{"b", "c"},
	}, tbl.Shards[0])
	require.Equal(t, version+1, tbl.Version)

	// the old primary is out of sync, so it cannot take the seat back.
	requireRejected(t, tbl, MovePrimary{
		Shard: 0,
		Epoch: 2,
		Node:  "a",
	})
}

func TestTableConflictingCommandsOnOneEpoch(t *testing.T) {
	// the primary adds c back while the controller elects b.  both read
	// epoch 2.  the one that applies first wins and the other is refused.
	tbl := newTable(t, 1)
	require.NoError(t, tbl.Apply(ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "c",
	}))
	expand := ExpandInSync{
		Shard: 0,
		Epoch: 2,
		Node:  "c",
	}
	elect := ElectPrimary{
		Shard: 0,
		Epoch: 2,
		Node:  "b",
	}

	first := tbl.Clone()
	require.NoError(t, first.Apply(expand))
	requireRejected(t, first, elect)

	second := tbl.Clone()
	require.NoError(t, second.Apply(elect))
	requireRejected(t, second, expand)
}

func TestTableUnknownCommandRejected(t *testing.T) {
	// Apply takes commands by value.
	requireRejected(t, newTable(t, 1), &ElectPrimary{
		Shard: 0,
		Epoch: 1,
		Node:  "b",
	})
}

func TestTableJSONRoundTrip(t *testing.T) {
	tbl := newTable(t, 5)
	require.NoError(t, tbl.Apply(RegisterNode{
		ID:     "a",
		BootID: "a-boot-2",
	}))
	require.NoError(t, tbl.Apply(ElectPrimary{
		Shard: 0,
		Epoch: 2,
		Node:  "b",
	}))

	for _, in := range []*Table{{}, tbl} {
		b, err := json.Marshal(in)
		require.NoError(t, err)
		var out Table
		require.NoError(t, json.Unmarshal(b, &out))
		require.Equal(t, in, &out)

		// the encoding is the same every time, so two copies of one
		// table give equal snapshots.
		again, err := json.Marshal(in.Clone())
		require.NoError(t, err)
		require.Equal(t, string(b), string(again))
	}
}

func TestCommandJSONRoundTrip(t *testing.T) {
	cmds := []Command{
		Init{
			Shards: 256,
			RF:     3,
			Nodes:  []string{"a", "b", "c"},
		},
		RegisterNode{
			ID:     "a",
			Addr:   "10.0.0.1:7000",
			BootID: "boot",
		},
		ShrinkInSync{
			Shard: 1,
			Epoch: 2,
			Node:  "b",
		},
		ExpandInSync{
			Shard: 3,
			Epoch: 4,
			Node:  "c",
		},
		ElectPrimary{
			Shard: 5,
			Epoch: 6,
			Node:  "a",
		},
		MovePrimary{
			Shard: 7,
			Epoch: 8,
			Node:  "b",
		},
	}
	for _, cmd := range cmds {
		b, err := EncodeCommand(cmd)
		require.NoError(t, err)
		out, err := DecodeCommand(b)
		require.NoError(t, err)
		require.Equal(t, cmd, out)
	}

	_, err := DecodeCommand([]byte(`{"type":"nope","cmd":{}}`))
	require.Error(t, err)
	_, err = DecodeCommand([]byte(`not json`))
	require.Error(t, err)
}

func TestTableApplyIsDeterministic(t *testing.T) {
	// two tables that apply the same commands, one of them through the JSON
	// form, are equal.
	cmds := []Command{
		RegisterNode{
			ID:     "a",
			BootID: "1",
		},
		RegisterNode{
			ID:     "b",
			BootID: "1",
		},
		Init{
			Shards: 8,
			RF:     2,
			Nodes:  []string{"a", "b"},
		},
		RegisterNode{
			ID:     "a",
			BootID: "2",
		},
		ElectPrimary{
			Shard: 0,
			Epoch: 2,
			Node:  "b",
		},
		ExpandInSync{
			Shard: 0,
			Epoch: 3,
			Node:  "a",
		},
		MovePrimary{
			Shard: 0,
			Epoch: 4,
			Node:  "a",
		},
		// this one is rejected on both tables.
		ShrinkInSync{
			Shard: 0,
			Epoch: 1,
			Node:  "b",
		},
	}
	direct, decoded := &Table{}, &Table{}
	for _, cmd := range cmds {
		b, err := EncodeCommand(cmd)
		require.NoError(t, err)
		again, err := DecodeCommand(b)
		require.NoError(t, err)
		errDirect := direct.Apply(cmd)
		errDecoded := decoded.Apply(again)
		require.Equal(t, errDirect == nil, errDecoded == nil)
	}
	require.Equal(t, direct, decoded)
	require.Equal(t, uint64(7), direct.Version)
}

func TestTableClone(t *testing.T) {
	tbl := newTable(t, 2)
	c := tbl.Clone()
	require.Equal(t, tbl, c)

	require.NoError(t, c.Apply(ShrinkInSync{
		Shard: 0,
		Epoch: 1,
		Node:  "b",
	}))
	require.NoError(t, c.Apply(RegisterNode{
		ID:     "d",
		BootID: "d-boot-1",
	}))
	c.Shards[1].Replicas[0] = "x"

	require.Equal(t, newTable(t, 2), tbl)
}

func TestTableShardOf(t *testing.T) {
	require.Equal(t, -1, (&Table{}).ShardOf(uuid.New()))

	tbl := newTable(t, 16)
	seen := map[int]bool{}
	for range 1000 {
		id := uuid.New()
		s := tbl.ShardOf(id)
		require.GreaterOrEqual(t, s, 0)
		require.Less(t, s, 16)
		require.Equal(t, s, tbl.ShardOf(id))
		seen[s] = true
	}
	require.Len(t, seen, 16)

	// the client and every node compute this mapping on their own.  a
	// change of it sends requests to the wrong shard.
	big := newTable(t, 256)
	require.Equal(t, 199, big.ShardOf(uuid.MustParse("00000000-0000-0000-0000-000000000001")))
	require.Equal(t, 3, big.ShardOf(uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")))
}
