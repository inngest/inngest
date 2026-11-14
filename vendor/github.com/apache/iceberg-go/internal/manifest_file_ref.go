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

// ManifestFileRef authorizes zero-copy access to immutable manifest state from
// trusted packages within this module. Data obtained with this token is
// borrowed, must be treated as read-only, and must not be retained beyond the
// current operation. Go's internal-package rule prevents external callers
// from constructing this token.
type ManifestFileRef struct{}

// ManifestFilePartitions is the public partition-summary surface needed by
// the internal borrowed-partition helper. It is parameterized so this package
// does not need to import the root package.
type ManifestFilePartitions[T any] interface {
	Partitions() []T
}

// ManifestPartitionBorrower is implemented by the built-in manifest file to
// expose its partition summaries to trusted in-module callers.
type ManifestPartitionBorrower[T any] interface {
	ManifestFilePartitionRef(ManifestFileRef) []T
}

// BorrowedManifestFilePartitions returns partition summaries without copying
// for the built-in manifest file and falls back to the public getter for
// external implementations. The returned slice and all nested values are
// read-only borrows that must not escape the current operation.
func BorrowedManifestFilePartitions[T any](file ManifestFilePartitions[T]) []T {
	if ref, ok := file.(ManifestPartitionBorrower[T]); ok {
		return ref.ManifestFilePartitionRef(ManifestFileRef{})
	}

	return file.Partitions()
}
