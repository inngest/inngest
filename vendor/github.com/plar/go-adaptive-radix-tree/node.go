package art

import (
	"bytes"
	"math/bits"
	"unsafe"
)

type prefix [MaxPrefixLen]byte

// ART node stores all available nodes, leaf and node type
type artNode struct {
	ref  unsafe.Pointer
	kind Kind
}

// a key with the null suffix will be stored as zeroChild
type node struct {
	prefixLen   uint32
	prefix      prefix
	numChildren uint16
	zeroChild   *artNode
}

// Node with 4 children
type node4 struct {
	node
	children [node4Max]*artNode
	keys     [node4Max]byte
	present  [node4Max]byte
}

// Node with 16 children
type node16 struct {
	node
	children [node16Max]*artNode
	keys     [node16Max]byte
	present  uint16 // need 16 bits for keys
}

// Node with 48 children
const (
	n48s = 6  // 2^n48s == n48m
	n48m = 64 // it should be sizeof(node48.present[0])
)

type node48 struct {
	node
	children [node48Max]*artNode
	keys     [node256Max]byte
	present  [4]uint64 // need 256 bits for keys
}

// Node with 256 children
type node256 struct {
	node
	children [node256Max]*artNode
}

// Leaf node with variable key length
type leaf struct {
	key   Key
	value interface{}
}

// String returns string representation of the Kind value
func (k Kind) String() string {
	return []string{"Leaf", "Node4", "Node16", "Node48", "Node256"}[k]
}

func (k Key) charAt(pos int) byte {
	if pos < 0 || pos >= len(k) {
		return 0
	}
	return k[pos]
}

func (k Key) valid(pos int) bool {
	return pos >= 0 && pos < len(k)
}

// Node interface implementation
func (an *artNode) node() *node {
	return (*node)(an.ref)
}

func (an *artNode) Kind() Kind {
	return an.kind
}

func (an *artNode) Key() Key {
	if an.isLeaf() {
		return an.leaf().key
	}

	return nil
}

func (an *artNode) Value() Value {
	if an.isLeaf() {
		return an.leaf().value
	}

	return nil
}

func (an *artNode) isLeaf() bool {
	return an.kind == Leaf
}

func (an *artNode) setPrefix(key Key, prefixLen uint32) *artNode {
	node := an.node()
	node.prefixLen = prefixLen
	for i := uint32(0); i < min(prefixLen, MaxPrefixLen); i++ {
		node.prefix[i] = key[i]
	}

	return an
}

func (an *artNode) matchDeep(key Key, depth uint32) uint32 /* mismatch index*/ {
	mismatchIdx := an.match(key, depth)
	if mismatchIdx < MaxPrefixLen {
		return mismatchIdx
	}

	leaf := an.minimum()
	limit := min(uint32(len(leaf.key)), uint32(len(key))) - depth
	for ; mismatchIdx < limit; mismatchIdx++ {
		if leaf.key[mismatchIdx+depth] != key[mismatchIdx+depth] {
			break
		}
	}

	return mismatchIdx
}

// Find the minimum leaf under a artNode
func (an *artNode) minimum() *leaf {
	switch an.kind {
	case Leaf:
		return an.leaf()

	case Node4:
		node := an.node4()
		if node.zeroChild != nil {
			return node.zeroChild.minimum()
		} else if node.children[0] != nil {
			return node.children[0].minimum()
		}

	case Node16:
		node := an.node16()
		if node.zeroChild != nil {
			return node.zeroChild.minimum()
		} else if node.children[0] != nil {
			return node.children[0].minimum()
		}

	case Node48:
		node := an.node48()
		if node.zeroChild != nil {
			return node.zeroChild.minimum()
		}

		idx := uint8(0)
		for node.present[idx>>n48s]&(1<<uint8(idx%n48m)) == 0 {
			idx++
		}
		if node.children[node.keys[idx]] != nil {
			return node.children[node.keys[idx]].minimum()
		}

	case Node256:
		node := an.node256()
		if node.zeroChild != nil {
			return node.zeroChild.minimum()
		} else if len(node.children) > 0 {
			idx := 0
			for ; node.children[idx] == nil; idx++ {
				// find 1st non empty
			}
			return node.children[idx].minimum()
		}
	}

	return nil // that should never happen in normal case
}

func (an *artNode) maximum() *leaf {
	switch an.kind {
	case Leaf:
		return an.leaf()

	case Node4:
		node := an.node4()
		return node.children[node.numChildren-1].maximum()

	case Node16:
		node := an.node16()
		return node.children[node.numChildren-1].maximum()

	case Node48:
		idx := uint8(node256Max - 1)
		node := an.node48()
		for node.present[idx>>n48s]&(1<<uint8(idx%n48m)) == 0 {
			idx--
		}
		return node.children[node.keys[idx]].maximum()

	case Node256:
		idx := node256Max - 1
		node := an.node256()
		for node.children[idx] == nil {
			idx--
		}
		return node.children[idx].maximum()
	}

	return nil // that should never happen in normal case
}

func (an *artNode) index(c byte) int {
	switch an.kind {
	case Node4:
		node := an.node4()
		for idx := 0; idx < int(node.numChildren); idx++ {
			if node.keys[idx] == c {
				return idx
			}
		}

	case Node16:
		node := an.node16()
		bitfield := uint(0)
		for i := uint(0); i < node16Max; i++ {
			if node.keys[i] == c {
				bitfield |= (1 << i)
			}
		}
		mask := (1 << node.numChildren) - 1
		bitfield &= uint(mask)
		if bitfield != 0 {
			return bits.TrailingZeros(bitfield)
		}

	case Node48:
		node := an.node48()
		if s := node.present[c>>n48s] & (1 << (c % n48m)); s > 0 {
			if idx := int(node.keys[c]); idx >= 0 {
				return idx
			}
		}

	case Node256:
		return int(c)
	}

	return -1 // not found
}

var nodeNotFound *artNode

func (an *artNode) findChild(c byte, valid bool) **artNode {
	node := an.node()

	if !valid {
		return &node.zeroChild
	}

	idx := an.index(c)
	if idx != -1 {
		switch an.kind {
		case Node4:
			return &an.node4().children[idx]

		case Node16:
			return &an.node16().children[idx]

		case Node48:
			return &an.node48().children[idx]

		case Node256:
			return &an.node256().children[idx]
		}
	}

	return &nodeNotFound
}

func (an *artNode) node4() *node4 {
	return (*node4)(an.ref)
}

func (an *artNode) node16() *node16 {
	return (*node16)(an.ref)
}

func (an *artNode) node48() *node48 {
	return (*node48)(an.ref)
}

func (an *artNode) node256() *node256 {
	return (*node256)(an.ref)
}

func (an *artNode) leaf() *leaf {
	return (*leaf)(an.ref)
}

func (an *artNode) _addChild4(c byte, valid bool, child *artNode) bool {
	node := an.node4()

	// grow to node16
	if node.numChildren >= node4Max {
		newNode := an.grow()
		newNode.addChild(c, valid, child)
		replaceNode(an, newNode)
		return true
	}

	// zero byte in the key
	if !valid {
		node.zeroChild = child
		return false
	}

	// just add a new child
	i := uint16(0)
	for ; i < node.numChildren; i++ {
		if c < node.keys[i] {
			break
		}
	}

	limit := node.numChildren - i
	for j := limit; limit > 0 && j > 0; j-- {
		node.keys[i+j] = node.keys[i+j-1]
		node.present[i+j] = node.present[i+j-1]
		node.children[i+j] = node.children[i+j-1]
	}
	node.keys[i] = c
	node.present[i] = 1
	node.children[i] = child
	node.numChildren++
	return false
}

func (an *artNode) _addChild16(c byte, valid bool, child *artNode) bool {
	node := an.node16()

	if node.numChildren >= node16Max {
		newNode := an.grow()
		newNode.addChild(c, valid, child)
		replaceNode(an, newNode)
		return true
	}

	if !valid {
		node.zeroChild = child
		return false
	}

	idx := node.numChildren
	bitfield := uint(0)
	for i := uint(0); i < node16Max; i++ {
		if node.keys[i] > c {
			bitfield |= (1 << i)
		}
	}
	mask := (1 << node.numChildren) - 1
	bitfield &= uint(mask)
	if bitfield != 0 {
		idx = uint16(bits.TrailingZeros(bitfield))
	}

	for i := node.numChildren; i > uint16(idx); i-- {
		node.keys[i] = node.keys[i-1]
		node.present = (node.present & ^(1 << i)) | ((node.present & (1 << (i - 1))) << 1)
		node.children[i] = node.children[i-1]
	}

	node.keys[idx] = c
	node.present |= (1 << uint16(idx))
	node.children[idx] = child
	node.numChildren++
	return false
}

func (an *artNode) _addChild48(c byte, valid bool, child *artNode) bool {
	node := an.node48()
	if node.numChildren >= node48Max {
		newNode := an.grow()
		newNode.addChild(c, valid, child)
		replaceNode(an, newNode)
		return true
	}

	if !valid {
		node.zeroChild = child
		return false
	}

	index := byte(0)
	for node.children[index] != nil {
		index++
	}

	node.keys[c] = index
	node.present[c>>n48s] |= (1 << (c % n48m))
	node.children[index] = child
	node.numChildren++
	return false
}

func (an *artNode) _addChild256(c byte, valid bool, child *artNode) bool {
	node := an.node256()
	if !valid {
		node.zeroChild = child
	} else {
		node.numChildren++
		node.children[c] = child
	}

	return false
}

func (an *artNode) addChild(c byte, valid bool, child *artNode) bool {
	switch an.kind {
	case Node4:
		return an._addChild4(c, valid, child)

	case Node16:
		return an._addChild16(c, valid, child)

	case Node48:
		return an._addChild48(c, valid, child)

	case Node256:
		return an._addChild256(c, valid, child)
	}

	return false
}

func (an *artNode) _deleteChild4(c byte, valid bool) uint16 {
	node := an.node4()
	if !valid {
		node.zeroChild = nil
	} else if idx := an.index(c); idx >= 0 {
		node.numChildren--

		node.keys[idx] = 0
		node.present[idx] = 0
		node.children[idx] = nil

		for i := uint16(idx); i <= node.numChildren && i+1 < node4Max; i++ {
			node.keys[i] = node.keys[i+1]
			node.present[i] = node.present[i+1]
			node.children[i] = node.children[i+1]
		}

		node.keys[node.numChildren] = 0
		node.present[node.numChildren] = 0
		node.children[node.numChildren] = nil
	}

	// we have to return the number of children for the current node(node4) as
	// `node.numChildren` plus one if null node is not nil.
	// `Shrink` method can be invoked after this method,
	// `Shrink` can convert this node into a leaf node type.
	// For all higher nodes(16/48/256) we simply copy null node to a smaller node
	// see deleteChild() and shrink() methods for implementation details
	numChildren := node.numChildren
	if node.zeroChild != nil {
		numChildren++
	}

	return numChildren
}

func (an *artNode) _deleteChild16(c byte, valid bool) uint16 {
	node := an.node16()
	if !valid {
		node.zeroChild = nil
	} else if idx := an.index(c); idx >= 0 {
		node.numChildren--
		node.keys[idx] = 0
		node.present &= ^(1 << uint16(idx))
		node.children[idx] = nil

		for i := uint16(idx); i <= node.numChildren && i+1 < node16Max; i++ {
			node.keys[i] = node.keys[i+1]
			node.present = (node.present & ^(1 << i)) | ((node.present & (1 << (i + 1))) >> 1)
			node.children[i] = node.children[i+1]
		}

		node.keys[node.numChildren] = 0
		node.present &= ^(1 << node.numChildren)
		node.children[node.numChildren] = nil
	}

	return node.numChildren
}

func (an *artNode) _deleteChild48(c byte, valid bool) uint16 {
	node := an.node48()
	if !valid {
		node.zeroChild = nil
	} else if idx := an.index(c); idx >= 0 && node.children[idx] != nil {
		node.children[idx] = nil
		node.keys[c] = 0
		node.present[c>>n48s] &= ^(1 << (c % n48m))
		node.numChildren--
	}

	return node.numChildren
}

func (an *artNode) _deleteChild256(c byte, valid bool) uint16 {
	node := an.node256()
	if !valid {
		node.zeroChild = nil
		return node.numChildren
	} else if idx := an.index(c); node.children[idx] != nil {
		node.children[idx] = nil
		node.numChildren--
	}

	return node.numChildren
}

func (an *artNode) deleteChild(c byte, valid bool) bool {
	var (
		numChildren uint16
		minChildren uint16
	)

	deleted := false
	switch an.kind {
	case Node4:
		numChildren = an._deleteChild4(c, valid)
		minChildren = node4Min
		deleted = true

	case Node16:
		numChildren = an._deleteChild16(c, valid)
		minChildren = node16Min
		deleted = true

	case Node48:
		numChildren = an._deleteChild48(c, valid)
		minChildren = node48Min
		deleted = true

	case Node256:
		numChildren = an._deleteChild256(c, valid)
		minChildren = node256Min
		deleted = true
	}

	if deleted && numChildren < minChildren {
		newNode := an.shrink()
		replaceNode(an, newNode)
		return true
	}

	return false
}

func (an *artNode) copyMeta(src *artNode) *artNode {
	if src == nil {
		return an
	}

	d := an.node()
	s := src.node()

	d.numChildren = s.numChildren
	d.prefixLen = s.prefixLen

	for i, limit := uint32(0), min(s.prefixLen, MaxPrefixLen); i < limit; i++ {
		d.prefix[i] = s.prefix[i]
	}

	return an
}

func (an *artNode) grow() *artNode {
	switch an.kind {
	case Node4:
		node := factory.newNode16().copyMeta(an)

		d := node.node16()
		s := an.node4()
		d.zeroChild = s.zeroChild

		for i := uint16(0); i < s.numChildren; i++ {
			if s.present[i] != 0 {
				d.keys[i] = s.keys[i]
				d.present |= (1 << i)
				d.children[i] = s.children[i]
			}
		}

		return node

	case Node16:
		node := factory.newNode48().copyMeta(an)

		d := node.node48()
		s := an.node16()
		d.zeroChild = s.zeroChild

		var numChildren byte
		for i := uint16(0); i < s.numChildren; i++ {
			if s.present&(1<<i) != 0 {
				ch := s.keys[i]
				d.keys[ch] = numChildren
				d.present[ch>>n48s] |= (1 << (ch % n48m))
				d.children[numChildren] = s.children[i]
				numChildren++
			}
		}

		return node

	case Node48:
		node := factory.newNode256().copyMeta(an)

		d := node.node256()
		s := an.node48()
		d.zeroChild = s.zeroChild

		for i := uint16(0); i < node256Max; i++ {
			if s.present[i>>n48s]&(1<<(i%n48m)) != 0 {
				d.children[i] = s.children[s.keys[i]]
			}
		}

		return node
	}

	return nil
}

func (an *artNode) shrink() *artNode {
	switch an.kind {
	case Node4:
		node4 := an.node4()
		child := node4.children[0]
		if child == nil {
			child = node4.zeroChild
		}

		if child.isLeaf() {
			return child
		}

		curPrefixLen := node4.prefixLen
		if curPrefixLen < MaxPrefixLen {
			node4.prefix[curPrefixLen] = node4.keys[0]
			curPrefixLen++
		}

		childNode := child.node()
		if curPrefixLen < MaxPrefixLen {
			childPrefixLen := min(childNode.prefixLen, MaxPrefixLen-curPrefixLen)
			for i := uint32(0); i < childPrefixLen; i++ {
				node4.prefix[curPrefixLen+i] = childNode.prefix[i]
			}
			curPrefixLen += childPrefixLen
		}

		for i := uint32(0); i < min(curPrefixLen, MaxPrefixLen); i++ {
			childNode.prefix[i] = node4.prefix[i]
		}
		childNode.prefixLen += node4.prefixLen + 1

		return child

	case Node16:
		node16 := an.node16()

		newNode := factory.newNode4().copyMeta(an)
		node4 := newNode.node4()
		node4.numChildren = 0
		for i := uint16(0); i < node4Max; i++ {
			node4.keys[i] = node16.keys[i]
			if node16.present&(1<<i) != 0 {
				node4.present[i] = 1
			}
			node4.children[i] = node16.children[i]
			node4.numChildren++
		}

		node4.zeroChild = node16.zeroChild

		return newNode

	case Node48:
		node48 := an.node48()

		newNode := factory.newNode16().copyMeta(an)
		node16 := newNode.node16()
		node16.numChildren = 0
		for i, idx := range node48.keys {
			if node48.present[uint16(i)>>n48s]&(1<<(uint16(i)%n48m)) == 0 {
				continue
			}

			if child := node48.children[idx]; child != nil {
				node16.children[node16.numChildren] = child
				node16.keys[node16.numChildren] = byte(i)
				node16.present |= (1 << node16.numChildren)
				node16.numChildren++
			}
		}

		node16.zeroChild = node48.zeroChild

		return newNode

	case Node256:
		node256 := an.node256()

		newNode := factory.newNode48().copyMeta(an)
		node48 := newNode.node48()
		node48.numChildren = 0
		for i, child := range node256.children {
			if child != nil {
				node48.children[node48.numChildren] = child
				node48.keys[byte(i)] = byte(node48.numChildren)
				node48.present[uint16(i)>>n48s] |= (1 << (uint16(i) % n48m))
				node48.numChildren++
			}
		}

		node48.zeroChild = node256.zeroChild

		return newNode
	}

	return nil
}

// Leaf methods
func (l *leaf) match(key Key) bool {
	if len(key) == 0 && len(l.key) == 0 {
		return true
	}

	if key == nil || len(l.key) != len(key) {
		return false
	}

	return bytes.Compare(l.key[:len(key)], key) == 0
}

func (l *leaf) prefixMatch(key Key) bool {
	if key == nil || len(l.key) < len(key) {
		return false
	}

	return bytes.Compare(l.key[:len(key)], key) == 0
}

// Base node methods
func (an *artNode) match(key Key, depth uint32) uint32 /* 1st mismatch index*/ {
	idx := uint32(0)
	if len(key)-int(depth) < 0 {
		return idx
	}

	node := an.node()

	limit := min(min(node.prefixLen, MaxPrefixLen), uint32(len(key))-depth)
	for ; idx < limit; idx++ {
		if node.prefix[idx] != key[idx+depth] {
			return idx
		}
	}

	return idx
}

// Node helpers
func replaceRef(oldNode **artNode, newNode *artNode) {
	*oldNode = newNode
}

func replaceNode(oldNode *artNode, newNode *artNode) {
	*oldNode = *newNode
}
