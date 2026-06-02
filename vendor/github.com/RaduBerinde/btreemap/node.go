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

package btreemap

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// node is an internal node in a tree.
//
// It must at all times maintain the invariant that either
//   - len(children) == 0, len(items) unconstrained
//   - len(children) == len(items) + 1
type node[K any, V any] struct {
	items    items[kv[K, V]]
	children items[*node[K, V]]
	cow      *copyOnWriteContext[K, V]
}

type kv[K any, V any] struct {
	k K
	v V
}

func (n *node[K, V]) mutableFor(cow *copyOnWriteContext[K, V]) *node[K, V] {
	if n.cow == cow {
		return n
	}
	out := cow.newNode()
	if cap(out.items) >= len(n.items) {
		out.items = out.items[:len(n.items)]
	} else {
		out.items = make(items[kv[K, V]], len(n.items), cap(n.items))
	}
	copy(out.items, n.items)
	// Copy children
	if cap(out.children) >= len(n.children) {
		out.children = out.children[:len(n.children)]
	} else {
		out.children = make(items[*node[K, V]], len(n.children), cap(n.children))
	}
	copy(out.children, n.children)
	return out
}

func (n *node[K, V]) mutableChild(i int) *node[K, V] {
	c := n.children[i].mutableFor(n.cow)
	n.children[i] = c
	return c
}

// split splits the given node at the given index.  The current node shrinks,
// and this function returns the item that existed at that index and a new node
// containing all items/children after it.
func (n *node[K, V]) split(i int) (kv[K, V], *node[K, V]) {
	item := n.items[i]
	next := n.cow.newNode()
	next.items = append(next.items, n.items[i+1:]...)
	n.items.truncate(i)
	if len(n.children) > 0 {
		next.children = append(next.children, n.children[i+1:]...)
		n.children.truncate(i + 1)
	}
	return item, next
}

// maybeSplitChild checks if a child should be split, and if so splits it.
// Returns whether or not a split occurred.
func (n *node[K, V]) maybeSplitChild(i, maxItems int) bool {
	if len(n.children[i].items) < maxItems {
		return false
	}
	first := n.mutableChild(i)
	item, second := first.split(maxItems / 2)
	n.items.insertAt(i, item)
	n.children.insertAt(i+1, second)
	return true
}

// insert inserts an item into the subtree rooted at this node, making sure
// no nodes in the subtree exceed maxItems items.  Should an equivalent item be
// be found/replaced by insert, it will be returned.
func (n *node[K, V]) insert(item kv[K, V], maxItems int) (_ kv[K, V], _ bool) {
	i, found := findKV(n.items, item.k, n.cow.cmp)
	if found {
		out := n.items[i]
		n.items[i] = item
		return out, true
	}
	if len(n.children) == 0 {
		n.items.insertAt(i, item)
		return
	}
	if n.maybeSplitChild(i, maxItems) {
		inTree := n.items[i]
		switch c := n.cow.cmp(item.k, inTree.k); {
		case c < 0:
			// no change, we want first split node
		case c > 0:
			i++ // we want second split node
		default:
			out := n.items[i]
			n.items[i] = item
			return out, true
		}
	}
	return n.mutableChild(i).insert(item, maxItems)
}

// get finds the given key in the subtree and returns the key and value.
func (n *node[K, V]) get(key K) (_ K, _ V, _ bool) {
	i, found := findKV(n.items, key, n.cow.cmp)
	if found {
		return n.items[i].k, n.items[i].v, true
	} else if len(n.children) > 0 {
		return n.children[i].get(key)
	}
	return
}

// min returns the first item in the subtree.
func min[K any, V any](n *node[K, V]) (_ K, _ V, ok bool) {
	if n == nil {
		return
	}
	for len(n.children) > 0 {
		n = n.children[0]
	}
	if len(n.items) == 0 {
		return
	}
	return n.items[0].k, n.items[0].v, true
}

// max returns the last item in the subtree.
func max[K any, V any](n *node[K, V]) (_ K, _ V, ok bool) {
	if n == nil {
		return
	}
	for len(n.children) > 0 {
		n = n.children[len(n.children)-1]
	}
	if len(n.items) == 0 {
		return
	}
	out := n.items[len(n.items)-1]
	return out.k, out.v, true
}

// toRemove details what item to remove in a node.remove call.
type toRemove int

const (
	removeItem toRemove = iota // removes the given item
	removeMin                  // removes smallest item in the subtree
	removeMax                  // removes largest item in the subtree
)

// remove removes an item from the subtree rooted at this node.
func (n *node[K, V]) remove(key K, minItems int, typ toRemove) (_ kv[K, V], _ bool) {
	var i int
	var found bool
	switch typ {
	case removeMax:
		if len(n.children) == 0 {
			return n.items.pop(), true
		}
		i = len(n.items)
	case removeMin:
		if len(n.children) == 0 {
			return n.items.removeAt(0), true
		}
		i = 0
	case removeItem:
		i, found = findKV(n.items, key, n.cow.cmp)
		if len(n.children) == 0 {
			if found {
				return n.items.removeAt(i), true
			}
			return
		}
	default:
		panic("invalid type")
	}
	// If we get to here, we have children.
	if len(n.children[i].items) <= minItems {
		return n.growChildAndRemove(i, key, minItems, typ)
	}
	child := n.mutableChild(i)
	// Either we had enough items to begin with, or we've done some
	// merging/stealing, because we've got enough now and we're ready to return
	// stuff.
	if found {
		// The item exists at index 'i', and the child we've selected can give us a
		// predecessor, since if we've gotten here it's got > minItems items in it.
		out := n.items[i]
		// We use our special-case 'remove' call with typ=maxItem to pull the
		// predecessor of item i (the rightmost leaf of our immediate left child)
		// and set it into where we pulled the item from.
		var zero K
		n.items[i], _ = child.remove(zero, minItems, removeMax)
		return out, true
	}
	// Final recursive call.  Once we're here, we know that the item isn't in this
	// node and that the child is big enough to remove from.
	return child.remove(key, minItems, typ)
}

// growChildAndRemove grows child 'i' to make sure it's possible to remove an
// item from it while keeping it at minItems, then calls remove to actually
// remove it.
//
// Most documentation says we have to do two sets of special casing:
//  1. item is in this node
//  2. item is in child
//
// In both cases, we need to handle the two subcases:
//
//	A) node has enough values that it can spare one
//	B) node doesn't have enough values
//
// For the latter, we have to check:
//
//	a) left sibling has node to spare
//	b) right sibling has node to spare
//	c) we must merge
//
// To simplify our code here, we handle cases #1 and #2 the same:
// If a node doesn't have enough items, we make sure it does (using a,b,c).
// We then simply redo our remove call, and the second time (regardless of
// whether we're in case 1 or 2), we'll have enough items and can guarantee
// that we hit case A.
func (n *node[K, V]) growChildAndRemove(i int, key K, minItems int, typ toRemove) (kv[K, V], bool) {
	if i > 0 && len(n.children[i-1].items) > minItems {
		// Steal from left child
		child := n.mutableChild(i)
		stealFrom := n.mutableChild(i - 1)
		stolenItem := stealFrom.items.pop()
		child.items.insertAt(0, n.items[i-1])
		n.items[i-1] = stolenItem
		if len(stealFrom.children) > 0 {
			child.children.insertAt(0, stealFrom.children.pop())
		}
	} else if i < len(n.items) && len(n.children[i+1].items) > minItems {
		// steal from right child
		child := n.mutableChild(i)
		stealFrom := n.mutableChild(i + 1)
		stolenItem := stealFrom.items.removeAt(0)
		child.items = append(child.items, n.items[i])
		n.items[i] = stolenItem
		if len(stealFrom.children) > 0 {
			child.children = append(child.children, stealFrom.children.removeAt(0))
		}
	} else {
		if i >= len(n.items) {
			i--
		}
		child := n.mutableChild(i)
		// merge with right child
		mergeItem := n.items.removeAt(i)
		mergeChild := n.children.removeAt(i + 1)
		child.items = append(child.items, mergeItem)
		child.items = append(child.items, mergeChild.items...)
		child.children = append(child.children, mergeChild.children...)
		n.cow.freeNode(mergeChild)
	}
	return n.remove(key, minItems, typ)
}

// asced provides a simple method for iterating over elements in the tree, in
// ascending order.
func (n *node[K, V]) ascend(
	start LowerBound[K], stop UpperBound[K], hit bool, iter func(K, V) bool,
) (bool, bool) {
	var ok bool
	var index int
	if start.kind != boundKindNone {
		index, _ = findKV(n.items, start.key, n.cow.cmp)
	}
	for i := index; i < len(n.items); i++ {
		if len(n.children) > 0 {
			if hit, ok = n.children[i].ascend(start, stop, hit, iter); !ok {
				return hit, false
			}
		}
		if start.kind == boundKindExclusive && !hit && n.cow.cmp(start.key, n.items[i].k) >= 0 {
			hit = true
			continue
		}
		hit = true
		if stop.kind != boundKindNone {
			c := n.cow.cmp(n.items[i].k, stop.key)
			if c > 0 || (c == 0 && stop.kind == boundKindExclusive) {
				return hit, false
			}
		}
		if !iter(n.items[i].k, n.items[i].v) {
			return hit, false
		}
	}
	if len(n.children) > 0 {
		if hit, ok = n.children[len(n.children)-1].ascend(start, stop, hit, iter); !ok {
			return hit, false
		}
	}
	return hit, true
}

// descend provides a simple method for iterating over elements in the tree, in
// ascending order.
func (n *node[K, V]) descend(
	start UpperBound[K], stop LowerBound[K], hit bool, iter func(K, V) bool,
) (bool, bool) {
	var ok bool
	var index int
	if start.kind != boundKindNone {
		var found bool
		index, found = findKV(n.items, start.key, n.cow.cmp)
		if !found {
			index = index - 1
		}
	} else {
		index = len(n.items) - 1
	}
	for i := index; i >= 0; i-- {
		if start.kind != boundKindNone {
			c := n.cow.cmp(start.key, n.items[i].k)
			if c < 0 || (c == 0 && (start.kind == boundKindExclusive || hit)) {
				continue
			}
		}
		if len(n.children) > 0 {
			if hit, ok = n.children[i+1].descend(start, stop, hit, iter); !ok {
				return hit, false
			}
		}
		if stop.kind != boundKindNone {
			c := n.cow.cmp(n.items[i].k, stop.key)
			if c < 0 || (c == 0 && stop.kind == boundKindExclusive) {
				return hit, false
			}
		}
		hit = true
		if !iter(n.items[i].k, n.items[i].v) {
			return hit, false
		}
	}
	if len(n.children) > 0 {
		if hit, ok = n.children[0].descend(start, stop, hit, iter); !ok {
			return hit, false
		}
	}
	return hit, true
}

// print is used for testing/debugging purposes.
func (n *node[K, V]) print(w io.Writer, level int) {
	fmt.Fprintf(w, "%sNODE:%v\n", strings.Repeat("  ", level), n.items)
	for _, c := range n.children {
		c.print(w, level+1)
	}
}

// reset returns a subtree to the freelist.  It breaks out immediately if the
// freelist is full, since the only benefit of iterating is to fill that
// freelist up.  Returns true if parent reset call should continue.
func (n *node[K, V]) reset(c *copyOnWriteContext[K, V]) bool {
	for _, child := range n.children {
		if !child.reset(c) {
			return false
		}
	}
	return c.freeNode(n) != ftFreelistFull
}

// items stores items in a node.
type items[T any] []T

// insertAt inserts a value into the given index, pushing all subsequent values
// forward.
func (s *items[T]) insertAt(index int, item T) {
	var zero T
	*s = append(*s, zero)
	if index < len(*s) {
		copy((*s)[index+1:], (*s)[index:])
	}
	(*s)[index] = item
}

// removeAt removes a value at a given index, pulling all subsequent values
// back.
func (s *items[T]) removeAt(index int) T {
	item := (*s)[index]
	copy((*s)[index:], (*s)[index+1:])
	var zero T
	(*s)[len(*s)-1] = zero
	*s = (*s)[:len(*s)-1]
	return item
}

// pop removes and returns the last element in the list.
func (s *items[T]) pop() (out T) {
	index := len(*s) - 1
	out = (*s)[index]
	var zero T
	(*s)[index] = zero
	*s = (*s)[:index]
	return
}

// truncate truncates this instance at index so that it contains only the
// first index items. index must be less than or equal to length.
func (s *items[T]) truncate(index int) {
	var toClear items[T]
	*s, toClear = (*s)[:index], (*s)[index:]
	var zero T
	for i := 0; i < len(toClear); i++ {
		toClear[i] = zero
	}
}

// findKV returns the index where the given key should be inserted into this
// list.  'found' is true if the kty already exists in the list at the given
// index.
func findKV[K any, V any](s items[kv[K, V]], key K, cmp CmpFunc[K]) (index int, found bool) {
	i := sort.Search(len(s), func(i int) bool {
		return cmp(key, s[i].k) <= 0
	})
	return i, i < len(s) && cmp(key, s[i].k) == 0
}
