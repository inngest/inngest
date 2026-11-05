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

package table

import (
	"fmt"

	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/iceberg-go"
)

// resolveSortKeys converts an Iceberg SortOrder into Arrow compute sort keys
// against the given file schema. Each sort field's source column is used as
// the sort key directly: order-preserving transforms (identity, truncate,
// year/month/day/hour) sort the same as their source values.
//
// Returns nil keys for an unsorted SortOrder. Returns an error when a sort
// field's source id is not a top-level column of fileSchema.
//
// Limitations:
//
//   - Per-batch, not per-file: keys are applied to each batch independently,
//     not merged into a globally sorted run across batches. Files therefore
//     record no sort_order_id, which asserts the whole file is sorted.
//   - Bucket transform falls back to source-column order, since the hash
//     bucket is not order-preserving.
//   - Multi-argument transform sort fields use only the first source column.
//   - Nested-field sort sources are not supported: a source id that is not a
//     top-level column of fileSchema is rejected with an error.
func resolveSortKeys(order SortOrder, fileSchema *iceberg.Schema) ([]compute.SortKey, error) {
	if order.IsUnsorted() {
		return nil, nil
	}

	fields := fileSchema.Fields()
	keys := make([]compute.SortKey, 0, order.Len())
	for _, field := range order.fields {
		idx, ok := topLevelFieldIndex(fields, field.SourceID())
		if !ok {
			return nil, fmt.Errorf("sort order %d: source id %d is not a top-level column in schema",
				order.OrderID(), field.SourceID())
		}

		key := compute.SortKey{ColumnIndex: idx, Order: compute.SortOrderAscending, NullPlacement: compute.SortNullsAtEnd}
		if field.Direction == SortDESC {
			key.Order = compute.SortOrderDescending
		}
		if field.NullOrder == NullsFirst {
			key.NullPlacement = compute.SortNullsAtStart
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func topLevelFieldIndex(fields []iceberg.NestedField, sourceID int) (int, bool) {
	for i, f := range fields {
		if f.ID == sourceID {
			return i, true
		}
	}

	return 0, false
}
