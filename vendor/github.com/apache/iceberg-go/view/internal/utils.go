// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package internal

func MapSlice[T any, V any](s []T, fn func(T) V) []V {
	mapped := make([]V, len(s))
	for i, e := range s {
		mapped[i] = fn(e)
	}

	return mapped
}

type Set[T comparable] map[T]struct{}

func ToSet[T comparable](s []T) Set[T] {
	set := Set[T]{}
	for _, v := range s {
		set[v] = struct{}{}
	}

	return set
}

func (s *Set[T]) Contains(e T) bool {
	_, ok := (*s)[e]

	return ok
}

func (s *Set[T]) IsSubset(other Set[T]) bool {
	for k := range *s {
		if !other.Contains(k) {
			return false
		}
	}

	return true
}
