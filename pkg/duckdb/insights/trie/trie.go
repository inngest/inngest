// Package trie provides a generic prefix trie with exact and wildcard
// (longest-prefix) key registration.
//
// Copied from the reference pkg/insights.trie package (the Cloud/ClickHouse
// equivalent of this repo's DuckDB insights feature) — its CELScope
// (field-path -> CEL predicate handler) uses this same trie, and
// pkg/cqrs/duckdbquery's own CEL-to-DuckDB-SQL field scope mirrors that
// pattern.
package trie

type Trie[K ~[]E, E comparable, V any] struct {
	value    *V
	wild     bool
	children map[E]*Trie[K, E, V]
}

func New[K ~[]E, E comparable, V any]() *Trie[K, E, V] {
	return &Trie[K, E, V]{children: make(map[E]*Trie[K, E, V])}
}

func (t *Trie[K, E, V]) add(k K, v V, wild bool) *Trie[K, E, V] {
	node := t
	for _, r := range k {
		child := node.children[r]
		if child == nil {
			child = New[K, E, V]()
			node.children[r] = child
		}
		node = child
	}

	node.wild = wild
	node.value = &v
	return t
}

// AddWild adds a wildcard matching entry to the trie. When multiple paths match, the longest match will be
// taken.
func (t *Trie[K, E, V]) AddWild(k K, v V) *Trie[K, E, V] {
	return t.add(k, v, true)
}

// Add adds an exact matching entry to the trie.
func (t *Trie[K, E, V]) Add(k K, v V) *Trie[K, E, V] {
	return t.add(k, v, false)
}

func (t *Trie[K, E, V]) Has(k K) bool {
	_, has := t.Get(k)
	return has
}

// Get retrieves the matching key from the trie. Failing that, it retrieves the longest matching wildcard
// match.
func (t *Trie[K, E, V]) Get(k K) (V, bool) {
	if t == nil {
		var zero V
		return zero, false
	}

	var bestValue *V
	node := t
	for _, r := range k {
		var ok bool
		node, ok = node.children[r]
		if !ok {
			if bestValue == nil {
				var zero V
				return zero, false
			}

			return *bestValue, true
		}

		if node.wild {
			bestValue = node.value
		}
	}

	if node.value != nil {
		bestValue = node.value
	}

	if bestValue == nil {
		var zero V
		return zero, false
	}

	return *bestValue, true
}

func (t *Trie[K, E, V]) LongestMatch(k K) int {
	if t == nil {
		return 0
	}

	var validLength int
	var length int
	node := t
	for _, r := range k {
		var ok bool
		node, ok = node.children[r]
		if !ok {
			return validLength
		}
		length++

		if node.value != nil {
			validLength = length
		}
	}

	return validLength
}

func (t *Trie[K, E, V]) Remove(k K) {
	if t == nil {
		return
	}

	stack := []*Trie[K, E, V]{t}
	node := t
	for _, r := range k {
		var ok bool
		node, ok = node.children[r]
		if !ok {
			return
		}
		stack = append(stack, node)
	}

	node.value = nil
	node.wild = false

	for i := len(stack) - 1; i >= 0; i-- {
		if len(stack[i].children) == 0 && stack[i].value == nil && i > 0 {
			delete(stack[i-1].children, []E(k)[i-1])
		}
	}
}

// Keys returns all keys that have an associated value (both exact and wildcard entries).
func (t *Trie[K, E, V]) Keys() []K {
	if t == nil {
		return nil
	}
	var result []K
	var walk func(node *Trie[K, E, V], prefix K)
	walk = func(node *Trie[K, E, V], prefix K) {
		if node.value != nil {
			cp := make(K, len(prefix))
			copy(cp, prefix)
			result = append(result, cp)
		}
		for e, child := range node.children {
			walk(child, append(prefix, e))
		}
	}
	walk(t, nil)
	return result
}

func (t *Trie[K, E, V]) Clone() *Trie[K, E, V] {
	if t == nil {
		return nil
	}

	n := New[K, E, V]()
	n.value = t.value
	n.wild = t.wild
	for r, child := range t.children {
		n.children[r] = child.Clone()
	}

	return n
}
