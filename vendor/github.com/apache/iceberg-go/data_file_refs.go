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

package iceberg

import "github.com/apache/iceberg-go/internal"

// DataFileStatsRef returns statistics maps owned by the data file. The token
// restricts this zero-copy accessor to trusted in-module callers; the public
// DataFile getters continue returning defensive copies. The returned maps and
// every byte slice reachable through the bounds maps alias the DataFile and
// must not be mutated or retained beyond the current evaluation. Values are
// the immutable scalar values produced by the manifest decoder; callers must
// not introduce mutable values through this borrowed view.
//
// This view intentionally contains only the maps consumed by metric
// evaluators. ColumnSizes and DistinctValueCounts are excluded because the
// evaluator does not read them, and cloning those maps would add work to the
// hot path without changing its result. Partition has a separate borrowed
// accessor because partition data is read during scan planning.
func (d *dataFile) DataFileStatsRef(_ internal.DataFileRef) (
	valueCounts map[int]int64,
	nullCounts map[int]int64,
	nanCounts map[int]int64,
	lowerBounds map[int][]byte,
	upperBounds map[int][]byte,
) {
	d.initColumnStatsData()

	return d.valCntMap, d.nullCntMap, d.nanCntMap, d.lowerBoundMap, d.upperBoundMap
}

func (d *dataFile) DataFileCollectionsRef(_ internal.DataFileRef) (
	map[int]int64, []byte, []int64, []int,
) {
	d.initColumnStatsData()

	var keyMetadata []byte
	if d.Key != nil {
		keyMetadata = *d.Key
	}
	var splitOffsets []int64
	if d.Splits != nil {
		splitOffsets = *d.Splits
	}
	var equalityFieldIDs []int
	if d.EqualityIDs != nil {
		equalityFieldIDs = *d.EqualityIDs
	}

	return d.colSizeMap, keyMetadata, splitOffsets, equalityFieldIDs
}

// DataFilePartitionRef returns the data file's partition map without copying.
// The token restricts this zero-copy accessor to trusted in-module callers;
// the public Partition getter continues returning a defensive copy. The
// returned map and all mutable values reachable through it alias the DataFile
// and must not be mutated or retained beyond the current planning operation.
func (d *dataFile) DataFilePartitionRef(_ internal.DataFileRef) map[int]any {
	d.initPartitionData()

	return d.fieldIDToPartitionData
}

// DataFileReferencedDataFileRef returns the optional referenced data file
// path without allocating. The token restricts this borrowed accessor to
// trusted in-module callers; the public getter continues returning a copy.
func (d *dataFile) DataFileReferencedDataFileRef(_ internal.DataFileRef) *string {
	return d.ReferencedDataFileField
}

// DataFilePointersRef returns pointer-valued metadata without allocating. The
// token restricts this borrowed accessor to trusted in-module callers; the
// public getters continue returning defensive copies.
func (d *dataFile) DataFilePointersRef(_ internal.DataFileRef) (
	*int, *int64, *string, *int64, *int64,
) {
	return d.SortOrder, d.FirstRowIDField, d.ReferencedDataFileField,
		d.ContentOffsetField, d.ContentSizeInBytesField
}
