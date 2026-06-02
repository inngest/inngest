// Copyright 2024 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package crsync

import "sync/atomic"

// TypedAtomicInt64 is a thin wrapper aorund atomic.Int64 that provides type
// safety.
type TypedAtomicInt64[T ~int64] struct {
	v atomic.Int64
}

// Load atomically loads and returns the value stored in x.
func (x *TypedAtomicInt64[T]) Load() T { return T(x.v.Load()) }

// Store atomically stores val into x.
func (x *TypedAtomicInt64[T]) Store(val T) { x.v.Store(int64(val)) }

// Swap atomically stores new into x and returns the previous value.
func (x *TypedAtomicInt64[T]) Swap(new T) (old T) { return T(x.v.Swap(int64(new))) }

// CompareAndSwap executes the compare-and-swap operation for x.
func (x *TypedAtomicInt64[T]) CompareAndSwap(old, new T) (swapped bool) {
	return x.v.CompareAndSwap(int64(old), int64(new))
}

// Add atomically adds delta to x and returns the new value.
func (x *TypedAtomicInt64[T]) Add(delta T) (new T) { return T(x.v.Add(int64(delta))) }
