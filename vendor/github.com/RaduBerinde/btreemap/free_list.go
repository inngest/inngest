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

import "sync"

const DefaultFreeListSize = 32

// FreeList represents a free list of btree nodes. By default each
// BTreeMap has its own FreeList, but multiple BTrees can share the same
// FreeList, in particular when they're created with Clone.
// Two Btrees using the same freelist are safe for concurrent write access.
type FreeList[K, V any] struct {
	mu       sync.Mutex
	freelist []*node[K, V]
}

// NewFreeList creates a new free list.
// size is the maximum size of the returned free list.
func NewFreeList[K any, V any](size int) *FreeList[K, V] {
	return &FreeList[K, V]{freelist: make([]*node[K, V], 0, size)}
}

func (f *FreeList[K, V]) newNode() (n *node[K, V]) {
	f.mu.Lock()
	index := len(f.freelist) - 1
	if index < 0 {
		f.mu.Unlock()
		return new(node[K, V])
	}
	n = f.freelist[index]
	f.freelist[index] = nil
	f.freelist = f.freelist[:index]
	f.mu.Unlock()
	return
}

func (f *FreeList[K, V]) freeNode(n *node[K, V]) (out bool) {
	f.mu.Lock()
	if len(f.freelist) < cap(f.freelist) {
		f.freelist = append(f.freelist, n)
		out = true
	}
	f.mu.Unlock()
	return
}
