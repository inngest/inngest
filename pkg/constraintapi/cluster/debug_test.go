package cluster

import (
	"fmt"
	"sort"
	"strings"
)

// debugString describes one replica for a failure message.
func (r *replica) debugString() string {
	r.mu.Lock()
	mode, term, fol, rep, transferring := r.mode, r.term, r.fol, r.rep, r.transferring
	row := r.row
	r.mu.Unlock()
	s := fmt.Sprintf("mode=%d term=%d lsn=%d open=%v transferring=%v row{epoch=%d term=%d primary=%s insync=%v}",
		mode, term, r.lastLSN.Load(), r.open.Load(), transferring, row.Epoch, row.Term, row.Primary, row.InSync)
	if fol != nil {
		s += fmt.Sprintf(" following=%s@%d", fol.primary, fol.term)
	}
	if rep != nil {
		rep.mu.Lock()
		var conns []string
		for n, fc := range rep.conns {
			conns = append(conns, fmt.Sprintf("%s{next=%d ready=%v seen=%v transfer=%v}", n, fc.next, fc.ready, fc.seenInSync, fc.transfer))
		}
		sort.Strings(conns)
		s += fmt.Sprintf(" rep{term=%d epoch=%d insync=%v last=%d pending=%d conns=%v acks=%v broken=%v expanding=%d}",
			rep.term, rep.row.Epoch, rep.row.InSync, rep.lastLSN, len(rep.pending), conns, rep.acks, rep.broken, len(rep.expanding))
		rep.mu.Unlock()
	}
	return s
}

// debugString describes every replica of every running node.
func (c *testCluster) debugString() string {
	var b strings.Builder
	for _, name := range c.names {
		n := c.node(name)
		if n == nil {
			fmt.Fprintf(&b, "%s: not running\n", name)
			continue
		}
		n.mu.Lock()
		shards := make([]int, 0, len(n.replicas))
		for i := range n.replicas {
			shards = append(shards, i)
		}
		n.mu.Unlock()
		sort.Ints(shards)
		for _, i := range shards {
			if r := n.replica(i); r != nil {
				fmt.Fprintf(&b, "%s shard %d: %s\n", name, i, r.debugString())
			}
		}
	}
	return b.String()
}
