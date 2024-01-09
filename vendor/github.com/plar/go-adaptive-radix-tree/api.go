package art

import "errors"

// A constant exposing all node types.
const (
	Leaf    Kind = 0
	Node4   Kind = 1
	Node16  Kind = 2
	Node48  Kind = 3
	Node256 Kind = 4
)

// Traverse Options.
const (
	// Iterate only over leaf nodes.
	TraverseLeaf = 1

	// Iterate only over non-leaf nodes.
	TraverseNode = 2

	// Iterate over all nodes in the tree.
	TraverseAll = TraverseLeaf | TraverseNode
)

// These errors can be returned when iteration over the tree.
var (
	ErrConcurrentModification = errors.New("Concurrent modification has been detected")
	ErrNoMoreNodes            = errors.New("There are no more nodes in the tree")
)

// Kind is a node type.
type Kind int

// Key Type.
// Key can be a set of any characters include unicode chars with null bytes.
type Key []byte

// Value type.
type Value interface{}

// Callback function type for tree traversal.
// if the callback function returns false then iteration is terminated.
type Callback func(node Node) (cont bool)

// Node interface.
type Node interface {
	// Kind returns node type.
	Kind() Kind

	// Key returns leaf's key.
	// This method is only valid for leaf node,
	// if its called on non-leaf node then returns nil.
	Key() Key

	// Value returns leaf's value.
	// This method is only valid for leaf node,
	// if its called on non-leaf node then returns nil.
	Value() Value
}

// Iterator iterates over nodes in key order.
type Iterator interface {
	// Returns true if the iteration has more nodes when traversing the tree.
	HasNext() bool

	// Returns the next element in the tree and advances the iterator position.
	// Returns ErrNoMoreNodes error if there are no more nodes in the tree.
	// Check if there is a next node with HasNext method.
	// Returns ErrConcurrentModification error if the tree has been structurally
	// modified after the iterator was created.
	Next() (Node, error)
}

// Tree is an Adaptive Radix Tree interface.
type Tree interface {
	// Insert a new key into the tree.
	// If the key already in the tree then return oldValue, true and nil, false otherwise.
	Insert(key Key, value Value) (oldValue Value, updated bool)

	// Delete removes a key from the tree and key's value, true is returned.
	// If the key does not exists then nothing is done and nil, false is returned.
	Delete(key Key) (value Value, deleted bool)

	// Search returns the value of the specific key.
	// If the key exists then return value, true and nil, false otherwise.
	Search(key Key) (value Value, found bool)

	// ForEach executes a provided callback once per leaf node by default.
	// The callback iteration is terminated if the callback function returns false.
	// Pass TraverseXXX as an options to execute a provided callback
	// once per NodeXXX type in the tree.
	ForEach(callback Callback, options ...int)

	// ForEachPrefix executes a provided callback once per leaf node that
	// leaf's key starts with the given keyPrefix.
	// The callback iteration is terminated if the callback function returns false.
	ForEachPrefix(keyPrefix Key, callback Callback)

	// Iterator returns an iterator for preorder traversal over leaf nodes by default.
	// Pass TraverseXXX as an options to return an iterator for preorder traversal over all NodeXXX types.
	Iterator(options ...int) Iterator
	//IteratorPrefix(key Key) Iterator

	// Minimum returns the minimum valued leaf, true if leaf is found and nil, false otherwise.
	Minimum() (min Value, found bool)

	// Maximum returns the maximum valued leaf, true if leaf is found and nil, false otherwise.
	Maximum() (max Value, found bool)

	// Returns size of the tree
	Size() int
}

// New creates a new adaptive radix tree
func New() Tree {
	return newTree()
}
