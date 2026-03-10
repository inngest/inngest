// Copyright 2024 The Cockroach Authors
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

package swiss

import "unsafe"

// Option provides an interface for passing configuration parameters for Map
// initialization.
type Option[K comparable, V any] interface {
	apply(m *Map[K, V])
}

type hashOption[K comparable, V any] struct {
	hash func(key *K, seed uintptr) uintptr
}

func (op hashOption[K, V]) apply(m *Map[K, V]) {
	m.hash = *(*hashFn)(noescape(unsafe.Pointer(&op.hash)))
}

// WithHash is an option to specify the hash function to use for a Map[K,V].
func WithHash[K comparable, V any](hash func(key *K, seed uintptr) uintptr) Option[K, V] {
	return hashOption[K, V]{hash}
}

type maxBucketCapacityOption[K comparable, V any] struct {
	maxBucketCapacity uint32
}

func (op maxBucketCapacityOption[K, V]) apply(m *Map[K, V]) {
	m.maxBucketCapacity = op.maxBucketCapacity
}

// WithMaxBucketCapacity is an option to specify the max bucket size to use
// for a Map[K,V]. Specifying a very large bucket size results in slower
// resize operations but delivers performance more akin to a raw Swiss table.
func WithMaxBucketCapacity[K comparable, V any](v uint32) Option[K, V] {
	return maxBucketCapacityOption[K, V]{v}
}

// Allocator specifies an interface for allocating and releasing memory used
// by a Map. The default allocator utilizes Go's builtin make() and allows the
// GC to reclaim memory.
//
// If the allocator is manually managing memory and requires that slots and
// controls be freed then Map.Close must be called in order to ensure
// FreeSlots and FreeControls are called.
type Allocator[K comparable, V any] interface {
	// Alloc should return a slice equivalent to make([]Group, n).
	Alloc(n int) []Group[K, V]

	// Free can optionally release the memory associated with the supplied
	// slice that is guaranteed to have been allocated by Alloc.
	Free(groups []Group[K, V])
}

type defaultAllocator[K comparable, V any] struct{}

func (defaultAllocator[K, V]) Alloc(n int) []Group[K, V] {
	return make([]Group[K, V], n)
}

func (defaultAllocator[K, V]) Free(_ []Group[K, V]) {
}

type allocatorOption[K comparable, V any] struct {
	allocator Allocator[K, V]
}

func (op allocatorOption[K, V]) apply(m *Map[K, V]) {
	m.allocator = op.allocator
}

// WithAllocator is an option for specifying the Allocator to use for a Map[K,V].
func WithAllocator[K comparable, V any](allocator Allocator[K, V]) Option[K, V] {
	return allocatorOption[K, V]{allocator}
}
