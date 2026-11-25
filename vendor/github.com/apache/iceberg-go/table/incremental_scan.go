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
	"slices"

	"github.com/apache/iceberg-go"
)

// incrementalSnapshotsBetween returns the snapshots in one ancestry chain
// from oldest to newest, applying the same inclusive/exclusive boundary rules
// to all incremental scan types. An exclusive starting snapshot may have
// expired because AncestorsBetween can validate it by parent ID alone. With no
// explicit starting snapshot, an expired intermediate snapshot makes
// AncestorsOf truncate the chain and the caller receives the remaining
// snapshots with ordinals starting at zero; this preserves the existing
// incremental-scan behavior.
func incrementalSnapshotsBetween(
	metadata Metadata,
	fromSnapshotID *int64,
	fromInclusive bool,
	toSnapshotID int64,
) ([]Snapshot, error) {
	ancestors := AncestorsOf(toSnapshotID, metadata.SnapshotByID)
	if len(ancestors) == 0 {
		return nil, fmt.Errorf("%w: ending snapshot not found: %d", iceberg.ErrInvalidArgument, toSnapshotID)
	}

	if fromSnapshotID == nil {
		slices.Reverse(ancestors)

		return ancestors, nil
	}

	fromID := *fromSnapshotID
	if !fromInclusive {
		if fromID == toSnapshotID {
			return nil, fmt.Errorf("%w: starting snapshot %d must be a parent ancestor of ending snapshot %d for an exclusive scan",
				iceberg.ErrInvalidArgument, fromID, toSnapshotID)
		}
		between, found := AncestorsBetween(toSnapshotID, fromID, metadata.SnapshotByID)
		if !found {
			return nil, fmt.Errorf("%w: starting snapshot %d is not an ancestor of ending snapshot %d", iceberg.ErrInvalidArgument, fromID, toSnapshotID)
		}
		slices.Reverse(between)

		return between, nil
	}

	if metadata.SnapshotByID(fromID) == nil {
		return nil, fmt.Errorf("%w: starting snapshot not found: %d", iceberg.ErrInvalidArgument, fromID)
	}
	if !IsAncestorOf(toSnapshotID, fromID, metadata.SnapshotByID) {
		return nil, fmt.Errorf("%w: starting snapshot %d is not an ancestor of ending snapshot %d", iceberg.ErrInvalidArgument, fromID, toSnapshotID)
	}
	selected := make([]Snapshot, 0, len(ancestors))
	for _, snapshot := range ancestors {
		selected = append(selected, snapshot)
		if snapshot.SnapshotID == fromID {
			break
		}
	}
	slices.Reverse(selected)

	return selected, nil
}
