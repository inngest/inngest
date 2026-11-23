package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
)

// ErrRejected is returned by Apply when a command does not fit the table.
// the table is unchanged.
var ErrRejected = errors.New("cluster: command rejected")

// NodeInfo is what the table knows about one node.
type NodeInfo struct {
	Addr string
	// BootID is drawn at every process start.  a change of it tells the
	// table that the node lost its memory.
	BootID string
}

// ShardRow is the table's record of one shard.
type ShardRow struct {
	// Nonce is in every lease ID of this shard.
	Nonce uint32
	// Epoch increases on every change to this row.
	Epoch uint64
	// Term increases when Primary changes.
	Term uint64
	// Primary is empty when the shard has none.
	Primary  string
	Replicas []string
	// InSync lists the replicas that have every answered write.  only a
	// member of it can become primary.
	InSync []string
}

// Table is the shard table.  it is a pure state machine with no clock and no
// randomness, so every copy that applies the same commands is equal.
type Table struct {
	// Version increases on every applied command.
	Version uint64
	RF      int
	// NextNonce is the next shard nonce.  a nonce is never issued twice.
	NextNonce uint32
	Nodes     map[string]NodeInfo
	// Shards is empty until Init.
	Shards []ShardRow
}

// Command is one change to the table.
type Command interface {
	kind() string
}

// Init creates the shards.  shard i gets the replicas Nodes[(i+j) % len(Nodes)]
// for j below RF, the first as primary, all in sync.
type Init struct {
	Shards int
	RF     int
	Nodes  []string
}

// RegisterNode adds a node or updates its address.  a boot ID that differs
// from the stored one removes the node from every in-sync set and every
// primary seat.
type RegisterNode struct {
	ID     string
	Addr   string
	BootID string
}

// ShrinkInSync removes a node that is not the primary from the in-sync set.
type ShrinkInSync struct {
	Shard int
	Epoch uint64
	Node  string
}

// ExpandInSync adds a replica to the in-sync set.
type ExpandInSync struct {
	Shard int
	Epoch uint64
	Node  string
}

// ElectPrimary makes an in-sync member the primary.
type ElectPrimary struct {
	Shard int
	Epoch uint64
	Node  string
}

// MovePrimary makes an in-sync member the primary and removes the old
// primary from the in-sync set.
type MovePrimary struct {
	Shard int
	Epoch uint64
	Node  string
}

func (Init) kind() string         { return "init" }
func (RegisterNode) kind() string { return "register_node" }
func (ShrinkInSync) kind() string { return "shrink_in_sync" }
func (ExpandInSync) kind() string { return "expand_in_sync" }
func (ElectPrimary) kind() string { return "elect_primary" }
func (MovePrimary) kind() string  { return "move_primary" }

type commandEnvelope struct {
	Type string          `json:"type"`
	Cmd  json.RawMessage `json:"cmd"`
}

// EncodeCommand returns the JSON form of a command with its type tag.
func EncodeCommand(cmd Command) ([]byte, error) {
	body, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}
	return json.Marshal(commandEnvelope{
		Type: cmd.kind(),
		Cmd:  body,
	})
}

// DecodeCommand reads what EncodeCommand wrote.
func DecodeCommand(b []byte) (Command, error) {
	var env commandEnvelope
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	switch env.Type {
	case Init{}.kind():
		return decodeCommand[Init](env.Cmd)
	case RegisterNode{}.kind():
		return decodeCommand[RegisterNode](env.Cmd)
	case ShrinkInSync{}.kind():
		return decodeCommand[ShrinkInSync](env.Cmd)
	case ExpandInSync{}.kind():
		return decodeCommand[ExpandInSync](env.Cmd)
	case ElectPrimary{}.kind():
		return decodeCommand[ElectPrimary](env.Cmd)
	case MovePrimary{}.kind():
		return decodeCommand[MovePrimary](env.Cmd)
	}
	return nil, fmt.Errorf("cluster: unknown command type %q", env.Type)
}

func decodeCommand[C Command](b []byte) (Command, error) {
	var c C
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return c, nil
}

// Clone returns a deep copy.
func (t *Table) Clone() *Table {
	c := *t
	c.Nodes = maps.Clone(t.Nodes)
	c.Shards = slices.Clone(t.Shards)
	for i := range c.Shards {
		c.Shards[i].Replicas = slices.Clone(c.Shards[i].Replicas)
		c.Shards[i].InSync = slices.Clone(c.Shards[i].InSync)
	}
	return &c
}

// ShardOf returns the shard of an account.  it returns -1 before Init.
func (t *Table) ShardOf(accountID uuid.UUID) int {
	if len(t.Shards) == 0 {
		return -1
	}
	return int(xxhash.Sum64(accountID[:]) % uint64(len(t.Shards)))
}

// Apply runs one command.  an error that wraps ErrRejected leaves the table
// unchanged.
func (t *Table) Apply(cmd Command) error {
	var err error
	switch c := cmd.(type) {
	case Init:
		err = t.init(c)
	case RegisterNode:
		err = t.registerNode(c)
	case ShrinkInSync:
		err = t.shrinkInSync(c)
	case ExpandInSync:
		err = t.expandInSync(c)
	case ElectPrimary:
		err = t.electPrimary(c.Shard, c.Epoch, c.Node, false)
	case MovePrimary:
		err = t.electPrimary(c.Shard, c.Epoch, c.Node, true)
	default:
		err = rejectf("unknown command %T", cmd)
	}
	if err != nil {
		return err
	}
	t.Version++
	return nil
}

func rejectf(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrRejected}, a...)...)
}

// nonce issues the next shard nonce.  the first one is 1, so a nonce of 0
// always means that none was set.
func (t *Table) nonce() uint32 {
	if t.NextNonce == 0 {
		t.NextNonce = 1
	}
	n := t.NextNonce
	t.NextNonce++
	return n
}

func (t *Table) init(c Init) error {
	if len(t.Shards) > 0 {
		return rejectf("the table already has shards")
	}
	if c.Shards < 1 {
		return rejectf("shard count %d", c.Shards)
	}
	if c.RF < 1 || c.RF > len(c.Nodes) {
		return rejectf("RF %d with %d nodes", c.RF, len(c.Nodes))
	}
	seen := make(map[string]bool, len(c.Nodes))
	for _, n := range c.Nodes {
		if n == "" || seen[n] {
			return rejectf("node %q is empty or listed twice", n)
		}
		seen[n] = true
	}

	t.RF = c.RF
	t.Shards = make([]ShardRow, c.Shards)
	for i := range t.Shards {
		replicas := make([]string, c.RF)
		for j := range replicas {
			replicas[j] = c.Nodes[(i+j)%len(c.Nodes)]
		}
		t.Shards[i] = ShardRow{
			Nonce:    t.nonce(),
			Epoch:    1,
			Term:     1,
			Primary:  replicas[0],
			Replicas: replicas,
			InSync:   slices.Clone(replicas),
		}
	}
	return nil
}

func (t *Table) registerNode(c RegisterNode) error {
	if c.ID == "" || c.BootID == "" {
		return rejectf("node ID or boot ID is empty")
	}
	old, known := t.Nodes[c.ID]
	if t.Nodes == nil {
		t.Nodes = map[string]NodeInfo{}
	}
	t.Nodes[c.ID] = NodeInfo{
		Addr:   c.Addr,
		BootID: c.BootID,
	}
	if !known || old.BootID == c.BootID {
		return nil
	}

	// the process restarted.  it acknowledged records that it no longer
	// has, so it cannot stay in an in-sync set or stay primary.
	for i := range t.Shards {
		row := &t.Shards[i]
		if !slices.Contains(row.InSync, c.ID) {
			continue
		}
		row.InSync = without(row.InSync, c.ID)
		row.Epoch++
		wasPrimary := row.Primary == c.ID
		if wasPrimary {
			row.Primary = ""
		}
		if len(row.InSync) == 0 {
			// no copy with every answered write is left.  the shard
			// starts empty on this node, and the new nonce makes every
			// older lease ID read as foreign.
			row.Primary = c.ID
			row.InSync = []string{c.ID}
			row.Nonce = t.nonce()
		}
		if wasPrimary || row.Primary == c.ID {
			row.Term++
		}
	}
	return nil
}

func (t *Table) shrinkInSync(c ShrinkInSync) error {
	row, err := t.row(c.Shard, c.Epoch)
	if err != nil {
		return err
	}
	if c.Node == row.Primary {
		return rejectf("node %q is the primary of shard %d", c.Node, c.Shard)
	}
	if !slices.Contains(row.InSync, c.Node) {
		return rejectf("node %q is not in sync on shard %d", c.Node, c.Shard)
	}
	if len(row.InSync) == 1 {
		return rejectf("node %q is the last in-sync member of shard %d", c.Node, c.Shard)
	}
	row.InSync = without(row.InSync, c.Node)
	row.Epoch++
	return nil
}

func (t *Table) expandInSync(c ExpandInSync) error {
	row, err := t.row(c.Shard, c.Epoch)
	if err != nil {
		return err
	}
	if !slices.Contains(row.Replicas, c.Node) {
		return rejectf("node %q is not a replica of shard %d", c.Node, c.Shard)
	}
	if slices.Contains(row.InSync, c.Node) {
		return rejectf("node %q is already in sync on shard %d", c.Node, c.Shard)
	}
	row.InSync = append(row.InSync, c.Node)
	row.Epoch++
	return nil
}

func (t *Table) electPrimary(shard int, epoch uint64, node string, dropOld bool) error {
	row, err := t.row(shard, epoch)
	if err != nil {
		return err
	}
	if !slices.Contains(row.InSync, node) {
		return rejectf("node %q is not in sync on shard %d", node, shard)
	}
	if node == row.Primary {
		return rejectf("node %q is already the primary of shard %d", node, shard)
	}
	if dropOld && row.Primary != "" {
		row.InSync = without(row.InSync, row.Primary)
	}
	row.Primary = node
	row.Term++
	row.Epoch++
	return nil
}

// row returns the shard's row if it is at the epoch the command read.
func (t *Table) row(shard int, epoch uint64) (*ShardRow, error) {
	if shard < 0 || shard >= len(t.Shards) {
		return nil, rejectf("no shard %d", shard)
	}
	row := &t.Shards[shard]
	if row.Epoch != epoch {
		return nil, rejectf("shard %d is at epoch %d, not %d", shard, row.Epoch, epoch)
	}
	return row, nil
}

// without returns a new slice with every element but node, in order.
func without(s []string, node string) []string {
	out := make([]string, 0, len(s))
	for _, n := range s {
		if n != node {
			out = append(out, n)
		}
	}
	return out
}
