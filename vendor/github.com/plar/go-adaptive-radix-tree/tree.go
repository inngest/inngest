package art

type tree struct {
	// version field is updated by each tree modification
	version int

	root *artNode
	size int
}

// make sure that tree implements all methods from the Tree interface
var _ Tree = &tree{}

func (t *tree) Insert(key Key, value Value) (Value, bool) {
	oldValue, updated := t.recursiveInsert(&t.root, key, value, 0)
	if !updated {
		t.version++
		t.size++
	}

	return oldValue, updated
}

func (t *tree) Delete(key Key) (Value, bool) {
	value, deleted := t.recursiveDelete(&t.root, key, 0)
	if deleted {
		t.version++
		t.size--
		return value, true
	}

	return nil, false
}

func (t *tree) Search(key Key) (Value, bool) {
	current := t.root
	depth := uint32(0)
	for current != nil {
		if current.isLeaf() {
			leaf := current.leaf()
			if leaf.match(key) {
				return leaf.value, true
			}

			return nil, false
		}

		curNode := current.node()

		if curNode.prefixLen > 0 {
			prefixLen := current.match(key, depth)
			if prefixLen != min(curNode.prefixLen, MaxPrefixLen) {
				return nil, false
			}
			depth += curNode.prefixLen
		}

		next := current.findChild(key.charAt(int(depth)), key.valid(int(depth)))
		if *next != nil {
			current = *next
		} else {
			current = nil
		}
		depth++
	}

	return nil, false
}

func (t *tree) Minimum() (value Value, found bool) {
	if t == nil || t.root == nil {
		return nil, false
	}

	leaf := t.root.minimum()

	return leaf.value, true
}

func (t *tree) Maximum() (value Value, found bool) {
	if t == nil || t.root == nil {
		return nil, false
	}

	leaf := t.root.maximum()

	return leaf.value, true
}

func (t *tree) Size() int {
	if t == nil || t.root == nil {
		return 0
	}

	return t.size
}

func (t *tree) recursiveInsert(curNode **artNode, key Key, value Value, depth uint32) (Value, bool) {
	current := *curNode
	if current == nil {
		replaceRef(curNode, factory.newLeaf(key, value))
		return nil, false
	}

	if current.isLeaf() {
		leaf := current.leaf()

		// update exists value
		if leaf.match(key) {
			oldValue := leaf.value
			leaf.value = value
			return oldValue, true
		}
		// new value, split the leaf into new node4
		newLeaf := factory.newLeaf(key, value)
		leaf2 := newLeaf.leaf()
		leafsLCP := t.longestCommonPrefix(leaf, leaf2, depth)

		newNode := factory.newNode4()
		newNode.setPrefix(key[depth:], leafsLCP)
		depth += leafsLCP

		newNode.addChild(leaf.key.charAt(int(depth)), leaf.key.valid(int(depth)), current)
		newNode.addChild(leaf2.key.charAt(int(depth)), leaf2.key.valid(int(depth)), newLeaf)
		replaceRef(curNode, newNode)

		return nil, false
	}

	node := current.node()
	if node.prefixLen > 0 {
		prefixMismatchIdx := current.matchDeep(key, depth)
		if prefixMismatchIdx >= node.prefixLen {
			depth += node.prefixLen
			goto NEXT_NODE
		}

		newNode := factory.newNode4()
		node4 := newNode.node()
		node4.prefixLen = prefixMismatchIdx
		for i := 0; i < int(min(prefixMismatchIdx, MaxPrefixLen)); i++ {
			node4.prefix[i] = node.prefix[i]
		}

		if node.prefixLen <= MaxPrefixLen {
			node.prefixLen -= (prefixMismatchIdx + 1)
			newNode.addChild(node.prefix[prefixMismatchIdx], true, current)

			for i, limit := uint32(0), min(node.prefixLen, MaxPrefixLen); i < limit; i++ {
				node.prefix[i] = node.prefix[prefixMismatchIdx+i+1]
			}

		} else {
			node.prefixLen -= (prefixMismatchIdx + 1)
			leaf := current.minimum()
			newNode.addChild(leaf.key.charAt(int(depth+prefixMismatchIdx)), leaf.key.valid(int(depth+prefixMismatchIdx)), current)

			for i, limit := uint32(0), min(node.prefixLen, MaxPrefixLen); i < limit; i++ {
				node.prefix[i] = leaf.key[depth+prefixMismatchIdx+i+1]
			}
		}

		// Insert the new leaf
		newNode.addChild(key.charAt(int(depth+prefixMismatchIdx)), key.valid(int(depth+prefixMismatchIdx)), factory.newLeaf(key, value))
		replaceRef(curNode, newNode)

		return nil, false
	}

NEXT_NODE:

	// Find a child to recursive to
	next := current.findChild(key.charAt(int(depth)), key.valid(int(depth)))
	if *next != nil {
		return t.recursiveInsert(next, key, value, depth+1)
	}

	// No Child, artNode goes with us
	current.addChild(key.charAt(int(depth)), key.valid(int(depth)), factory.newLeaf(key, value))

	return nil, false
}

func (t *tree) recursiveDelete(curNode **artNode, key Key, depth uint32) (Value, bool) {
	if t == nil || *curNode == nil || len(key) == 0 {
		return nil, false
	}

	current := *curNode
	if current.isLeaf() {
		leaf := current.leaf()
		if leaf.match(key) {
			replaceRef(curNode, nil)
			return leaf.value, true
		}

		return nil, false
	}

	node := current.node()
	if node.prefixLen > 0 {
		prefixLen := current.match(key, depth)
		if prefixLen != min(node.prefixLen, MaxPrefixLen) {
			return nil, false
		}

		depth += node.prefixLen
	}

	next := current.findChild(key.charAt(int(depth)), key.valid(int(depth)))
	if *next == nil {
		return nil, false
	}

	if (*next).isLeaf() {
		leaf := (*next).leaf()
		if leaf.match(key) {
			current.deleteChild(key.charAt(int(depth)), key.valid(int(depth)))
			return leaf.value, true
		}

		return nil, false
	}

	return t.recursiveDelete(next, key, depth+1)
}

func (t *tree) longestCommonPrefix(l1 *leaf, l2 *leaf, depth uint32) uint32 {
	l1key, l2key := l1.key, l2.key
	idx, limit := depth, min(uint32(len(l1key)), uint32(len(l2key)))
	for ; idx < limit; idx++ {
		if l1key[idx] != l2key[idx] {
			break
		}
	}

	return idx - depth
}
