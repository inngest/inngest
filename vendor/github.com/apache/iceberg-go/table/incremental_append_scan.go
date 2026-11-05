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
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/metrics"
)

// IncrementalAppendScan plans data files added by append snapshots between a
// starting snapshot and an ending snapshot. It follows one snapshot ancestry
// chain and never returns files inherited from an earlier snapshot. Every
// selected snapshot must identify a recognized operation; planning fails with
// [ErrMissingOperation] or [ErrInvalidOperation] rather than silently omitting
// snapshots whose operation cannot be determined.
type IncrementalAppendScan struct {
	scan           *Scan
	fromSnapshotID *int64
	fromInclusive  bool
	toSnapshotID   *int64
}

// NewIncrementalAppendScan creates an incremental append file planner.
// Planning-related ScanOptions configure snapshot selection, filtering, case
// sensitivity, concurrency, planning mode, and reporting. Projection and row
// limits are not applied to the returned file tasks; callers can read planned
// tasks with a separately configured [Scan.ReadTasks]. Auto planning falls back
// to local planning. Remote planning returns ErrInvalidOperation until
// incremental remote planning is implemented.
func (t Table) NewIncrementalAppendScan(opts ...ScanOption) *IncrementalAppendScan {
	return &IncrementalAppendScan{scan: t.Scan(opts...)}
}

// FromSnapshotInclusive includes files added by the starting snapshot.
// The snapshot is validated when files are planned.
func (s *IncrementalAppendScan) FromSnapshotInclusive(snapshotID int64) *IncrementalAppendScan {
	out := *s
	out.fromSnapshotID = &snapshotID
	out.fromInclusive = true

	return &out
}

// FromSnapshotExclusive starts after the given snapshot. The starting
// snapshot must be a parent ancestor of the ending snapshot when planning.
func (s *IncrementalAppendScan) FromSnapshotExclusive(snapshotID int64) *IncrementalAppendScan {
	out := *s
	out.fromSnapshotID = &snapshotID
	out.fromInclusive = false

	return &out
}

// ToSnapshot sets the inclusive ending snapshot. The snapshot is validated
// when files are planned.
func (s *IncrementalAppendScan) ToSnapshot(snapshotID int64) *IncrementalAppendScan {
	out := *s
	out.toSnapshotID = &snapshotID

	return &out
}

// PlanFiles returns one task per newly added data file and emits a ScanReport
// through the configured reporter on successful planning. Delete files are not
// applied because appended files are not present before the append snapshot.
func (s *IncrementalAppendScan) PlanFiles(ctx context.Context) ([]FileScanTask, error) {
	switch s.scan.planningMode {
	case ScanPlanningLocal, ScanPlanningAuto:
	case ScanPlanningRemote:
		return nil, fmt.Errorf("%w: incremental append scans do not support remote planning", ErrInvalidOperation)
	default:
		return nil, fmt.Errorf("%w: unknown scan planning mode %q", iceberg.ErrInvalidArgument, s.scan.planningMode)
	}
	start := time.Now()

	toSnapshot, err := s.toSnapshot()
	if err != nil {
		return nil, err
	}
	if toSnapshot == nil {
		if s.fromSnapshotID != nil {
			return nil, fmt.Errorf("%w: no ending snapshot found for incremental append scan from %d",
				iceberg.ErrInvalidArgument, *s.fromSnapshotID)
		}

		return nil, nil
	}

	planningScan := *s.scan
	planningScan.identifier = slices.Clone(s.scan.identifier)
	planningScan.selectedFields = slices.Clone(s.scan.selectedFields)
	planningScan.options = maps.Clone(s.scan.options)
	if s.toSnapshotID != nil {
		// An explicit end snapshot is a historical scan and must use that
		// snapshot's schema. An implicit current end remains a live scan so a
		// schema-only metadata update is visible during pruning.
		planningScan.snapshotID = &toSnapshot.SnapshotID
		planningScan.asOfTimestamp = nil
	}
	schema, err := planningScan.effectiveSchema()
	if err != nil {
		return nil, err
	}
	residual, err := bindTaskFilter(schema, planningScan.rowFilter, planningScan.caseSensitive)
	if err != nil {
		return nil, fmt.Errorf("bind incremental scan residual: %w", err)
	}
	var acc scanMetricsAccumulator
	finish := func(tasks []FileScanTask) ([]FileScanTask, error) {
		acc.resultDataFiles = int64(len(tasks))
		for _, task := range tasks {
			acc.totalFileSize += task.File.FileSizeBytes()
		}
		acc.applyResultDeleteMetrics(tasks)
		planningDuration := time.Since(start)

		if rep := planningScan.Reporter(); !metrics.IsNop(rep) {
			projected, _ := planningScan.Projection()
			safeReport(ctx, rep, planningScan.buildScanReport(&acc, schema, projected, planningDuration))
		}

		return tasks, nil
	}

	snapshots, err := s.snapshotsBetween(toSnapshot.SnapshotID)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return finish(nil)
	}
	appendSnapshots := make(map[int64]struct{}, len(snapshots))
	for _, snapshot := range snapshots {
		appendSnapshots[snapshot.SnapshotID] = struct{}{}
	}

	if s.scan.ioF == nil {
		return nil, fmt.Errorf("%w: table file IO is not configured", ErrInvalidOperation)
	}

	// An inherited manifest can occur in every later snapshot's manifest list.
	// Read each manifest path once, just as the Java incremental append scan
	// collects the selected manifests into a set before opening them.
	manifestsByPath := make(map[string]iceberg.ManifestFile)
	manifestFS := sharedSnapshotManifestFSF(s.scan.ioF)
	for _, snapshot := range snapshots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		manifestSet, err := planningScan.manifestSetWithFSF(ctx, snapshot, manifestFS)
		if err != nil {
			return nil, err
		}
		for _, manifest := range manifestSet.dataManifests() {
			if _, ok := appendSnapshots[manifest.SnapshotID()]; !ok {
				continue
			}
			manifestsByPath[manifest.FilePath()] = manifest
		}
	}

	paths := make([]string, 0, len(manifestsByPath))
	for path := range manifestsByPath {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	manifestList := make([]iceberg.ManifestFile, 0, len(paths))
	for _, path := range paths {
		manifestList = append(manifestList, manifestsByPath[path])
	}

	// Use one projection cache for manifest-summary and data-file pruning.
	partitionFilters := planningScan.partitionFiltersForSchema(schema)
	manifestList, err = planningScan.filterManifestsWithSchema(manifestList, schema, &acc, partitionFilters)
	if err != nil {
		return nil, err
	}
	if len(manifestList) == 0 {
		return finish(nil)
	}
	// The IO above is scoped to reading manifest lists. Manifest workers reuse
	// one factory result per concurrent batch, then reacquire through the factory
	// so long-running incremental plans can renew vended credentials between
	// batches.
	entries, err := planningScan.collectManifestEntriesWithSchema(ctx, manifestList, schema, partitionFilters)
	if err != nil {
		return nil, err
	}

	tasks := make([]FileScanTask, 0, len(entries.dataEntries))
	for _, entry := range entries.dataEntries {
		if entry.Status() != iceberg.EntryStatusADDED {
			continue
		}
		if _, ok := appendSnapshots[entry.SnapshotID()]; !ok {
			continue
		}
		file := entry.DataFile()
		task := FileScanTask{File: file, Start: 0, Length: file.FileSizeBytes()}
		task.Residual = residual
		task.FirstRowID = file.FirstRowID()
		if sequenceNumber := entry.SequenceNum(); sequenceNumber >= 0 {
			task.DataSequenceNumber = &sequenceNumber
		}
		tasks = append(tasks, task)
	}
	slices.SortFunc(tasks, func(left, right FileScanTask) int {
		return cmp.Compare(left.File.FilePath(), right.File.FilePath())
	})

	return finish(tasks)
}

func (s *IncrementalAppendScan) toSnapshot() (*Snapshot, error) {
	if s.toSnapshotID != nil {
		snapshot := s.scan.metadata.SnapshotByID(*s.toSnapshotID)
		if snapshot == nil {
			return nil, fmt.Errorf("%w: ending snapshot not found: %d", iceberg.ErrInvalidArgument, *s.toSnapshotID)
		}

		return snapshot, nil
	}

	return s.scan.ResolveSnapshot()
}

func (s *IncrementalAppendScan) snapshotsBetween(toSnapshotID int64) ([]Snapshot, error) {
	snapshots, err := incrementalSnapshotsBetween(
		s.scan.metadata, s.fromSnapshotID, s.fromInclusive, toSnapshotID)
	if err != nil {
		return nil, err
	}

	return appendOnlySnapshots(snapshots)
}

func appendOnlySnapshots(snapshots []Snapshot) ([]Snapshot, error) {
	result := make([]Snapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Summary == nil || snapshot.Summary.Operation == "" {
			return nil, fmt.Errorf("%w: cannot determine operation for snapshot %d",
				ErrMissingOperation, snapshot.SnapshotID)
		}

		switch snapshot.Summary.Operation {
		case OpAppend:
			result = append(result, snapshot)
		case OpReplace, OpOverwrite, OpDelete:
		default:
			return nil, fmt.Errorf("%w: snapshot %d has operation %q",
				ErrInvalidOperation, snapshot.SnapshotID, snapshot.Summary.Operation)
		}
	}

	return result, nil
}
