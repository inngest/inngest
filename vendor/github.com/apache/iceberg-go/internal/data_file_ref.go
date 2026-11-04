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

// DataFileRef authorizes zero-copy access to immutable DataFile state from
// trusted packages within this module. Go's internal-package rule prevents
// external callers from constructing this token.
type DataFileRef struct{}

// DataFileStats is the public statistics surface needed by the internal
// borrowed-statistics helper. It is intentionally smaller than iceberg.DataFile
// so this low-level package does not import the root package.
type DataFileStats interface {
	ValueCounts() map[int]int64
	NullValueCounts() map[int]int64
	NaNValueCounts() map[int]int64
	LowerBoundValues() map[int][]byte
	UpperBoundValues() map[int][]byte
}

// DataFileStatsRef is implemented by the built-in data file to expose its
// immutable statistics maps to trusted in-module callers.
type DataFileStatsRef interface {
	DataFileStatsRef(DataFileRef) (
		valueCounts map[int]int64,
		nullCounts map[int]int64,
		nanCounts map[int]int64,
		lowerBounds map[int][]byte,
		upperBounds map[int][]byte,
	)
}

// BorrowedDataFileStats returns statistics without copying for the built-in
// data file and falls back to the public getters for external implementations.
// The returned values must be treated as read-only and must not escape the
// current operation.
func BorrowedDataFileStats(file DataFileStats) (
	valueCounts map[int]int64,
	nullCounts map[int]int64,
	nanCounts map[int]int64,
	lowerBounds map[int][]byte,
	upperBounds map[int][]byte,
) {
	if ref, ok := file.(DataFileStatsRef); ok {
		return ref.DataFileStatsRef(DataFileRef{})
	}

	return file.ValueCounts(), file.NullValueCounts(), file.NaNValueCounts(),
		file.LowerBoundValues(), file.UpperBoundValues()
}

type DataFileCollections interface {
	ColumnSizes() map[int]int64
	KeyMetadata() []byte
	SplitOffsets() []int64
	EqualityFieldIDs() []int
}

type DataFileCollectionsRef interface {
	DataFileCollectionsRef(DataFileRef) (
		columnSizes map[int]int64,
		keyMetadata []byte,
		splitOffsets []int64,
		equalityFieldIDs []int,
	)
}

func BorrowedDataFileCollections(file DataFileCollections) (
	columnSizes map[int]int64,
	keyMetadata []byte,
	splitOffsets []int64,
	equalityFieldIDs []int,
) {
	if ref, ok := file.(DataFileCollectionsRef); ok {
		return ref.DataFileCollectionsRef(DataFileRef{})
	}

	return file.ColumnSizes(), file.KeyMetadata(), file.SplitOffsets(), file.EqualityFieldIDs()
}

// BorrowedDataFileBounds returns the lower and upper bounds without copying
// for the built-in data file and falls back to the public getters for external
// implementations. The returned maps and byte slices are read-only borrows.
func BorrowedDataFileBounds(file DataFileStats) (lowerBounds, upperBounds map[int][]byte) {
	if ref, ok := file.(DataFileStatsRef); ok {
		_, _, _, lowerBounds, upperBounds = ref.DataFileStatsRef(DataFileRef{})

		return lowerBounds, upperBounds
	}

	return file.LowerBoundValues(), file.UpperBoundValues()
}

// DataFilePartition is the public partition surface needed by the internal
// borrowed-partition helper. It avoids importing the root package here.
type DataFilePartition interface {
	Partition() map[int]any
}

// DataFilePartitionRef is implemented by the built-in data file to expose its
// immutable partition map to trusted in-module callers.
type DataFilePartitionRef interface {
	DataFilePartitionRef(DataFileRef) map[int]any
}

// BorrowedDataFilePartition returns a partition map without copying for the
// built-in data file and falls back to the public getter for external
// implementations. The returned map and its values must be treated as
// read-only and must not escape the current operation.
func BorrowedDataFilePartition(file DataFilePartition) map[int]any {
	if ref, ok := file.(DataFilePartitionRef); ok {
		return ref.DataFilePartitionRef(DataFileRef{})
	}

	return file.Partition()
}

// DataFileReferencedDataFile is the public optional reference surface needed
// by the internal borrowed-reference helper.
type DataFileReferencedDataFile interface {
	ReferencedDataFile() *string
}

// DataFileReferencedDataFileRef is implemented by the built-in data file to
// expose its optional referenced path without allocating a defensive pointer.
type DataFileReferencedDataFileRef interface {
	DataFileReferencedDataFileRef(DataFileRef) *string
}

// BorrowedDataFileReferencedDataFile returns the optional referenced path
// without copying for the built-in data file and falls back to the public
// getter for external implementations. The pointer must only be dereferenced
// during the current operation and must not be retained.
func BorrowedDataFileReferencedDataFile(file DataFileReferencedDataFile) *string {
	if ref, ok := file.(DataFileReferencedDataFileRef); ok {
		return ref.DataFileReferencedDataFileRef(DataFileRef{})
	}

	return file.ReferencedDataFile()
}

// DataFilePointers is the pointer-valued DataFile surface used by internal
// readers. It avoids importing the root package here.
type DataFilePointers interface {
	SortOrderID() *int
	FirstRowID() *int64
	ReferencedDataFile() *string
	ContentOffset() *int64
	ContentSizeInBytes() *int64
}

// DataFilePointersRef is implemented by the built-in data file to expose its
// pointer-valued fields without allocating defensive copies.
type DataFilePointersRef interface {
	DataFilePointersRef(DataFileRef) (
		sortOrderID *int,
		firstRowID *int64,
		referencedDataFile *string,
		contentOffset *int64,
		contentSizeInBytes *int64,
	)
}

// BorrowedDataFilePointers returns pointer-valued metadata without copying for
// the built-in data file and falls back to the public getters for external
// implementations. The returned pointers must only be dereferenced during
// the current operation and must not be retained.
func BorrowedDataFilePointers(file DataFilePointers) (
	sortOrderID *int,
	firstRowID *int64,
	referencedDataFile *string,
	contentOffset *int64,
	contentSizeInBytes *int64,
) {
	if ref, ok := file.(DataFilePointersRef); ok {
		return ref.DataFilePointersRef(DataFileRef{})
	}

	return file.SortOrderID(), file.FirstRowID(), file.ReferencedDataFile(),
		file.ContentOffset(), file.ContentSizeInBytes()
}
