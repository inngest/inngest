package cluster

// Report is what a node sends to the coordinator every heartbeat interval.
type Report struct {
	Node   string
	BootID string
	// LSN is the last log position this node has, by shard.
	LSN map[int]uint64
}

// Plan returns an ElectPrimary for every shard whose primary is empty or not
// live.  it picks the live in-sync member with the highest reported LSN, and
// the earliest in InSync on a tie.  a shard with no live in-sync member gets
// no command.
func Plan(t *Table, live func(node string) bool, lastLSN func(node string, shard int) uint64) []Command {
	var cmds []Command
	for i := range t.Shards {
		row := &t.Shards[i]
		if row.Primary != "" && live(row.Primary) {
			continue
		}
		best := ""
		var bestLSN uint64
		for _, n := range row.InSync {
			if n == row.Primary || !live(n) {
				continue
			}
			if lsn := lastLSN(n, i); best == "" || lsn > bestLSN {
				best = n
				bestLSN = lsn
			}
		}
		if best == "" {
			continue
		}
		cmds = append(cmds, ElectPrimary{
			Shard: i,
			Epoch: row.Epoch,
			Node:  best,
		})
	}
	return cmds
}
