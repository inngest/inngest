package ordered

// Pair represents a key-value pair in the ordered map.
type Pair[K comparable, V any] struct {
	Key   K
	Value V

	next, prev *Pair[K, V] // Pointers to the next and previous pairs in the linked list.
}

// Map represents an ordered map.
type Map[K comparable, V any] struct {
	root *Pair[K, V] // Sentinel node for the circular doubly linked list.
	zero V           // Zero value for the value type.

	pairs map[K]*Pair[K, V] // Map from keys to pairs.
}

// NewMap creates a new ordered map with optional initial data.
func NewMap[K comparable, V any](initialData ...Pair[K, V]) *Map[K, V] {
	m := &Map[K, V]{}
	m.Clear()
	for _, pair := range initialData {
		m.Store(pair.Key, pair.Value)
	}
	return m
}

// Clear removes all pairs from the map.
func (m *Map[K, V]) Clear() {
	if m.root != nil {
		m.root.next, m.root.prev = nil, nil // avoid memory leaks
	}
	for _, pair := range m.pairs {
		pair.next, pair.prev = nil, nil // avoid memory leaks
	}
	m.root = &Pair[K, V]{}
	m.root.next, m.root.prev = m.root, m.root
	m.pairs = make(map[K]*Pair[K, V])
}

// Len returns the number of pairs in the map.
func (m *Map[K, V]) Len() int {
	return len(m.pairs)
}

// Load returns the value associated with the key, and a boolean indicating if the key was found.
func (m *Map[K, V]) Load(key K) (V, bool) {
	if pair, present := m.pairs[key]; present {
		return pair.Value, true
	}
	return m.zero, false
}

// Value returns the value associated with the key, or the zero value if the key is not found.
func (m *Map[K, V]) Value(key K) V {
	if pair, present := m.pairs[key]; present {
		return pair.Value
	}
	return m.zero
}

// Store adds or updates a key-value pair in the map.
func (m *Map[K, V]) Store(key K, value V) {
	if pair, present := m.pairs[key]; present {
		pair.Value = value
		return
	}

	pair := &Pair[K, V]{Key: key, Value: value}
	pair.prev = m.root.prev
	m.root.prev.next = pair
	m.root.prev = pair
	pair.next = m.root
	m.pairs[key] = pair
}

// Delete removes a key-value pair from the map.
func (m *Map[K, V]) Delete(key K) {
	if pair, present := m.pairs[key]; present {
		pair.prev.next = pair.next
		pair.next.prev = pair.prev
		pair.next, pair.prev = nil, nil // avoid memory leaks
		delete(m.pairs, key)
	}
}

// Range calls the given function for each key-value pair in the map in order.
func (m *Map[K, V]) Range(yield func(key K, value V) bool) {
	for pair := m.root.next; pair != m.root; pair = pair.next {
		if !yield(pair.Key, pair.Value) {
			break
		}
	}
}

// Keys returns a slice of all keys in the map in order.
func (m *Map[K, V]) Keys() []K {
	keys := make([]K, 0, len(m.pairs))
	m.Range(func(key K, _ V) bool {
		keys = append(keys, key)
		return true
	})
	return keys
}

// Values returns a slice of all values in the map in order.
func (m *Map[K, V]) Values() []V {
	values := make([]V, 0, len(m.pairs))
	m.Range(func(_ K, value V) bool {
		values = append(values, value)
		return true
	})
	return values
}

// Pairs returns a slice of all key-value pairs in the map in order.
func (m *Map[K, V]) Pairs() []Pair[K, V] {
	pairs := make([]Pair[K, V], 0, len(m.pairs))
	m.Range(func(key K, value V) bool {
		pairs = append(pairs, Pair[K, V]{Key: key, Value: value})
		return true
	})
	return pairs
}
