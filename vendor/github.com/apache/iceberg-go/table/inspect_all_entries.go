// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package table

import (
	"context"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/iceberg-go"
)

// AllEntries returns every entry from each manifest reachable from any
// snapshot currently tracked by the table. Shared manifests are scanned once,
// and deleted entries remain visible.
func (i InspectTable) AllEntries(ctx context.Context) (array.RecordReader, error) {
	return i.inspectEntries(ctx, "all entries", true, AllEntriesSchema)
}

// AllEntriesSchema returns the schema of the all_entries metadata table.
func AllEntriesSchema(partitionType *iceberg.StructType) *iceberg.Schema {
	return EntriesSchema(partitionType)
}
