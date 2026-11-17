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

// SnapshotLookup returns the snapshot for the given id, or nil if not found.
// It matches the signature of Metadata.SnapshotByID.
type SnapshotLookup func(id int64) *Snapshot

// AncestorsOf walks the parent chain starting from snapshotID and yields
// every snapshot reachable via ParentSnapshotID, including the starting
// snapshot itself. Iteration stops when a snapshot has no parent, the
// parent cannot be resolved via lookup, or a cycle is detected in
// malformed metadata.
//
// Returns snapshots by value in reverse-chronological order (newest first).
// Returns an empty slice if snapshotID cannot be resolved.
//
// The returned slice may be truncated if an intermediate snapshot is
// missing from the lookup (e.g. expired) or if a cycle is encountered.
// Callers that need to distinguish a complete walk from a truncated one
// should use AncestorsOfChecked instead.
func AncestorsOf(snapshotID int64, lookup SnapshotLookup) []Snapshot {
	ancestors, _ := AncestorsOfChecked(snapshotID, lookup)

	return ancestors
}

// AncestorsOfChecked is AncestorsOf with completeness tracking. The
// second return value is true when the walk terminated at a snapshot
// with no parent (a clean root). It is false when the walk was
// truncated by an unresolvable starting snapshot, a missing intermediate
// snapshot, or a cycle in malformed metadata.
//
// Callers performing conflict detection (where a truncated ancestry
// equates to under-counting concurrent snapshots) MUST treat
// complete=false as divergent and refuse the commit, mirroring
// AncestorsBetween's baseFound=false contract. When complete is false
// the returned slice is the partial walk before truncation —
// diagnostic context only, NOT an enumerable ancestry.
//
// Snapshots are returned by value in reverse-chronological order.
// Returns an empty slice and false when snapshotID cannot be resolved.
func AncestorsOfChecked(snapshotID int64, lookup SnapshotLookup) ([]Snapshot, bool) {
	var ancestors []Snapshot
	visited := make(map[int64]struct{})

	id := snapshotID
	for {
		if _, seen := visited[id]; seen {
			return ancestors, false
		}
		visited[id] = struct{}{}

		snap := lookup(id)
		if snap == nil {
			return ancestors, false
		}

		ancestors = append(ancestors, *snap)

		if snap.ParentSnapshotID == nil {
			return ancestors, true
		}
		id = *snap.ParentSnapshotID
	}
}

// AncestorsBetween returns the snapshots from latestID (inclusive) down
// to but not including baseID, by walking the parent chain from
// latestID backward. The second return value (baseFound) is true when
// baseID was actually reached during the walk.
//
// These snapshots are the "concurrent" snapshots that a writer based on
// baseID needs to examine for conflict detection.
//
// When baseFound is false, the walk terminated without reaching baseID
// — either because latestID was unknown, baseID is not in latestID's
// ancestry (diverged branch or expired base), the chain was broken by
// a missing intermediate snapshot, or a cycle was detected in
// malformed metadata. In all of these cases the returned slice is
// diagnostic context only — it is NOT an enumerable "concurrent
// snapshots" list. Callers performing conflict detection MUST treat
// baseFound=false as divergent and refuse the commit.
//
// Returns (nil, true) when latestID == baseID (no concurrent snapshots).
//
// Snapshots are returned by value in reverse-chronological order.
func AncestorsBetween(latestID, baseID int64, lookup SnapshotLookup) ([]Snapshot, bool) {
	if latestID == baseID {
		return nil, true
	}

	var between []Snapshot
	visited := make(map[int64]struct{})

	id := latestID
	for {
		if id == baseID {
			return between, true
		}
		if _, seen := visited[id]; seen {
			// Cycle detected before reaching baseID.
			break
		}
		visited[id] = struct{}{}

		snap := lookup(id)
		if snap == nil {
			break
		}

		between = append(between, *snap)

		if snap.ParentSnapshotID == nil {
			break
		}
		id = *snap.ParentSnapshotID
	}

	return between, false
}

// IsAncestorOf returns true if ancestorID is in the parent chain of
// snapshotID (or equal to snapshotID, provided snapshotID resolves).
//
// Returns false if snapshotID cannot be resolved, if ancestorID is not
// reachable from snapshotID via ParentSnapshotID, or if a cycle
// prevents reaching ancestorID.
//
// Note: every snapshot on the walked chain must be resolvable via
// lookup. This diverges from Java's SnapshotUtil.isAncestorOf which
// walks parent *ids* regardless of snapshot resolvability. The Go
// semantics are stricter (conservative) — broken chains return false
// rather than matching an unreachable ancestor id.
func IsAncestorOf(snapshotID, ancestorID int64, lookup SnapshotLookup) bool {
	visited := make(map[int64]struct{})

	id := snapshotID
	for {
		if _, seen := visited[id]; seen {
			return false
		}

		snap := lookup(id)
		if snap == nil {
			return false
		}

		// Equality check must come BEFORE the visited write so that an
		// ancestor reached in the second hop of a cycle (e.g. 1 ↔ 2,
		// IsAncestorOf(1, 2)) is correctly returned as true before
		// cycle detection fires on the next iteration.
		if id == ancestorID {
			return true
		}

		visited[id] = struct{}{}

		if snap.ParentSnapshotID == nil {
			return false
		}
		id = *snap.ParentSnapshotID
	}
}
