// Copyright 2014-2022 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package btreemap implements an ordered key-value map using an in-memory
// B-Tree of arbitrary degree.
//
// The internal B-Tree code is based on github.com/google/btree.
package btreemap

import "iter"

// New creates a new map backed by a B-Tree with the given degree.
//
// New(2), for example, will create a 2-3-4 tree, where each node contains 1 to 3
// items and 2 to 4 children).
//
// The passed-in CmpFunc determines how objects of type T are ordered. For
// ordered basic types, use cmp.Compare.
func New[K any, V any](degree int, cmp CmpFunc[K]) *BTreeMap[K, V] {
	return NewWithFreeList(degree, cmp, NewFreeList[K, V](DefaultFreeListSize))
}

// NewWithFreeList creates a new map that uses the given node free list.
func NewWithFreeList[K any, V any](degree int, cmp CmpFunc[K], f *FreeList[K, V]) *BTreeMap[K, V] {
	if degree <= 1 {
		panic("bad degree")
	}
	return &BTreeMap[K, V]{
		degree: degree,
		cow:    &copyOnWriteContext[K, V]{freelist: f, cmp: cmp},
	}
}

// BTreeMap implements an ordered key-value map using an in-memory B-Tree of
// arbitrary degree. It allows easy insertion, removal, and iteration.
//
// Write operations are not safe for concurrent mutation by multiple goroutines,
// but Read operations are.
type BTreeMap[K any, V any] struct {
	degree int
	length int
	root   *node[K, V]
	cow    *copyOnWriteContext[K, V]
}

// CmpFunc returns:
// - 0 if the two keys are equal;
// - a negative number if a < b;
// - a positive number if a > b.
type CmpFunc[K any] func(a, b K) int

// Clone clones the btree, lazily.  Clone should not be called concurrently,
// but the original tree (t) and the new tree (t2) can be used concurrently
// once the Clone call completes.
//
// The internal tree structure of b is marked read-only and shared between t and
// t2. Writes to both t and t2 use copy-on-write logic, creating new nodes
// whenever one of b's original nodes would have been modified. Read operations
// should have no performance degradation. Write operations for both t and t2
// will initially experience minor slow-downs caused by additional allocs and
// copies due to the aforementioned copy-on-write logic, but should converge to
// the original performance characteristics of the original tree.
func (t *BTreeMap[K, V]) Clone() (t2 *BTreeMap[K, V]) {
	// Create two entirely new copy-on-write contexts.
	// This operation effectively creates three trees:
	//   the original, shared nodes (old b.cow)
	//   the new b.cow nodes
	//   the new out.cow nodes
	cow1, cow2 := *t.cow, *t.cow
	out := *t
	t.cow = &cow1
	out.cow = &cow2
	return &out
}

// ReplaceOrInsert adds the given item to the tree.  If an item in the tree
// already equals the given one, it is removed from the tree and returned,
// and the second return value is true.  Otherwise, (zeroValue, false)
//
// nil cannot be added to the tree (will panic).
func (t *BTreeMap[K, V]) ReplaceOrInsert(key K, value V) (_ K, _ V, replaced bool) {
	if t.root == nil {
		t.root = t.cow.newNode()
		t.root.items = append(t.root.items, kv[K, V]{k: key, v: value})
		t.length++
		return
	} else {
		t.root = t.root.mutableFor(t.cow)
		if len(t.root.items) >= t.maxItems() {
			item2, second := t.root.split(t.maxItems() / 2)
			oldroot := t.root
			t.root = t.cow.newNode()
			t.root.items = append(t.root.items, item2)
			t.root.children = append(t.root.children, oldroot, second)
		}
	}
	out, outb := t.root.insert(kv[K, V]{k: key, v: value}, t.maxItems())
	if !outb {
		t.length++
	}
	return out.k, out.v, outb
}

// Delete removes an item equal to the passed in item from the tree, returning
// it.  If no such item exists, returns (zeroValue, false).
func (t *BTreeMap[K, V]) Delete(key K) (K, V, bool) {
	return t.deleteItem(key, removeItem)
}

// DeleteMin removes the smallest item in the tree and returns it.
// If no such item exists, returns (zeroValue, false).
func (t *BTreeMap[K, V]) DeleteMin() (K, V, bool) {
	var zero K
	return t.deleteItem(zero, removeMin)
}

// DeleteMax removes the largest item in the tree and returns it.
// If no such item exists, returns (zeroValue, false).
func (t *BTreeMap[K, V]) DeleteMax() (K, V, bool) {
	var zero K
	return t.deleteItem(zero, removeMax)
}

func (t *BTreeMap[K, V]) deleteItem(key K, typ toRemove) (_ K, _ V, _ bool) {
	if t.root == nil || len(t.root.items) == 0 {
		return
	}
	t.root = t.root.mutableFor(t.cow)
	out, outb := t.root.remove(key, t.minItems(), typ)
	if len(t.root.items) == 0 && len(t.root.children) > 0 {
		oldroot := t.root
		t.root = t.root.children[0]
		t.cow.freeNode(oldroot)
	}
	if outb {
		t.length--
	}
	return out.k, out.v, outb
}

// LowerBound defines an (optional) lower bound for iteration.
type LowerBound[K any] bound[K]

// Min returns a LowerBound that does not limit the lower bound of the iteration.
func Min[K any]() LowerBound[K] { return LowerBound[K]{kind: boundKindNone} }

// GE returns an inclusive lower bound.
func GE[K any](key K) LowerBound[K] { return LowerBound[K]{key: key, kind: boundKindInclusive} }

// GT returns an exclusive lower bound.
func GT[K any](key K) LowerBound[K] { return LowerBound[K]{key: key, kind: boundKindExclusive} }

// UpperBound defines an (optional) upper bound for iteration.
type UpperBound[K any] bound[K]

// Max returns an UpperBound that does not limit the upper bound of the iteration.
func Max[K any]() UpperBound[K] { return UpperBound[K]{kind: boundKindNone} }

// LE returns an inclusive upper bound.
func LE[K any](key K) UpperBound[K] { return UpperBound[K]{key: key, kind: boundKindInclusive} }

// LT returns an exclusive upper bound.
func LT[K any](key K) UpperBound[K] { return UpperBound[K]{key: key, kind: boundKindExclusive} }

// AscendFunc calls yield() for all elements between the start and stop bounds,
// in ascending order.
func (t *BTreeMap[K, V]) AscendFunc(
	start LowerBound[K], stop UpperBound[K], yield func(key K, value V) bool,
) {
	if t.root != nil {
		t.root.ascend(start, stop, false, yield)
	}
}

// Ascend returns an iterator which yields all elements between the start and
// stop bounds, in ascending order.
func (t *BTreeMap[K, V]) Ascend(start LowerBound[K], stop UpperBound[K]) iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		if t.root != nil {
			t.root.ascend(start, stop, false, yield)
		}
	}
}

// DescendFunc calls yield() for all elements between the start and stop bounds,
// in ascending order.
func (t *BTreeMap[K, V]) DescendFunc(
	start UpperBound[K], stop LowerBound[K], yield func(key K, value V) bool,
) {
	if t.root != nil {
		t.root.descend(start, stop, false, yield)
	}
}

// Descend returns an iterator which yields all elements between the start and
// stop bounds, in ascending order.
func (t *BTreeMap[K, V]) Descend(start UpperBound[K], stop LowerBound[K]) iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		if t.root != nil {
			t.root.descend(start, stop, false, yield)
		}
	}
}

// Get looks for the key in the tree, returning (key, value, true) if found, or
// (0, 0, false) otherwise.
func (t *BTreeMap[K, V]) Get(key K) (_ K, _ V, _ bool) {
	if t.root == nil {
		return
	}
	return t.root.get(key)
}

// Min returns the smallest key and associated value in the tree, or
// (0, 0, false) if the tree is empty.
func (t *BTreeMap[K, V]) Min() (K, V, bool) {
	return min(t.root)
}

// Max returns the largest key and associated value in the tree, or
// (0, 0, false) if the tree is empty.
func (t *BTreeMap[K, V]) Max() (K, V, bool) {
	return max(t.root)
}

// Has returns true if the given key is in the tree.
func (t *BTreeMap[K, V]) Has(key K) bool {
	_, _, ok := t.Get(key)
	return ok
}

// Len returns the number of items currently in the tree.
func (t *BTreeMap[K, V]) Len() int {
	return t.length
}

// Clear removes all items from the btree.  If addNodesToFreelist is true,
// t's nodes are added to its freelist as part of this call, until the freelist
// is full.  Otherwise, the root node is simply dereferenced and the subtree
// left to Go's normal GC processes.
//
// This can be much faster
// than calling Delete on all elements, because that requires finding/removing
// each element in the tree and updating the tree accordingly.  It also is
// somewhat faster than creating a new tree to replace the old one, because
// nodes from the old tree are reclaimed into the freelist for use by the new
// one, instead of being lost to the garbage collector.
//
// This call takes:
//
//	O(1): when addNodesToFreelist is false, this is a single operation.
//	O(1): when the freelist is already full, it breaks out immediately
//	O(freelist size):  when the freelist is empty and the nodes are all owned
//	    by this tree, nodes are added to the freelist until full.
//	O(tree size):  when all nodes are owned by another tree, all nodes are
//	    iterated over looking for nodes to add to the freelist, and due to
//	    ownership, none are.
func (t *BTreeMap[K, V]) Clear(addNodesToFreelist bool) {
	if t.root != nil && addNodesToFreelist {
		t.root.reset(t.cow)
	}
	t.root, t.length = nil, 0
}

// maxItems returns the max number of items to allow per node.
func (t *BTreeMap[K, V]) maxItems() int {
	return t.degree*2 - 1
}

// minItems returns the min number of items to allow per node (ignored for the
// root node).
func (t *BTreeMap[K, V]) minItems() int {
	return t.degree - 1
}

// copyOnWriteContext pointers determine node ownership... a tree with a write
// context equivalent to a node's write context is allowed to modify that node.
// A tree whose write context does not match a node's is not allowed to modify
// it, and must create a new, writable copy (IE: it's a Clone).
//
// When doing any write operation, we maintain the invariant that the current
// node's context is equal to the context of the tree that requested the write.
// We do this by, before we descend into any node, creating a copy with the
// correct context if the contexts don't match.
//
// Since the node we're currently visiting on any write has the requesting
// tree's context, that node is modifiable in place.  Children of that node may
// not share context, but before we descend into them, we'll make a mutable
// copy.
type copyOnWriteContext[K any, V any] struct {
	freelist *FreeList[K, V]
	cmp      CmpFunc[K]
}

type bound[K any] struct {
	kind boundKind
	key  K
}

type boundKind uint8

const (
	boundKindNone boundKind = iota
	boundKindInclusive
	boundKindExclusive
)

func (c *copyOnWriteContext[K, V]) newNode() *node[K, V] {
	n := c.freelist.newNode()
	n.cow = c
	return n
}

type freeType int

const (
	ftFreelistFull freeType = iota // node was freed (available for GC, not stored in freelist)
	ftStored                       // node was stored in the freelist for later use
	ftNotOwned                     // node was ignored by COW, since it's owned by another one
)

// freeNode frees a node within a given COW context, if it's owned by that
// context.  It returns what happened to the node (see freeType const
// documentation).
func (c *copyOnWriteContext[K, V]) freeNode(n *node[K, V]) freeType {
	if n.cow == c {
		// clear to allow GC
		n.items.truncate(0)
		n.children.truncate(0)
		n.cow = nil
		if c.freelist.freeNode(n) {
			return ftStored
		} else {
			return ftFreelistFull
		}
	} else {
		return ftNotOwned
	}
}
