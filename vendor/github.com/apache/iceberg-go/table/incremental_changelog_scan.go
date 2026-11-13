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

// IncrementalChangelogScan plans data-file changes between snapshots. It
// emits insert and delete tasks for data-manifest entries and skips replace
// snapshots. PlanFiles returns an error if any in-range snapshot's manifest
// list references a delete manifest, including ones carried forward from
// earlier snapshots.
type IncrementalChangelogScan struct {
	scan           *Scan
	fromSnapshotID *int64
	fromInclusive  bool
	toSnapshotID   *int64
}

type plannedChangelogTask struct {
	task ChangelogScanTask
	file FileScanTask
}

// NewIncrementalChangelogScan creates an incremental changelog planner.
// Projection and row limits are not applied to returned tasks. Auto planning
// falls back to local planning, while remote planning is not supported. Use
// ChangelogScanTask.ScanTask with Scan.ReadTasks to read the returned files.
// Row filters are attached to each task as residuals without partition-specific
// simplification, matching the existing incremental append scan behavior.
func (t Table) NewIncrementalChangelogScan(opts ...ScanOption) *IncrementalChangelogScan {
	return &IncrementalChangelogScan{scan: t.Scan(opts...)}
}

// FromSnapshotInclusive includes changes committed by the starting snapshot.
// The snapshot is validated when files are planned.
func (s *IncrementalChangelogScan) FromSnapshotInclusive(snapshotID int64) *IncrementalChangelogScan {
	out := *s
	out.fromSnapshotID = &snapshotID
	out.fromInclusive = true

	return &out
}

// FromSnapshotExclusive starts after the given snapshot. The starting
// snapshot must be a parent ancestor of the ending snapshot when planning.
// The snapshot is validated when files are planned.
func (s *IncrementalChangelogScan) FromSnapshotExclusive(snapshotID int64) *IncrementalChangelogScan {
	out := *s
	out.fromSnapshotID = &snapshotID
	out.fromInclusive = false

	return &out
}

// ToSnapshot sets the inclusive ending snapshot.
// The snapshot is validated when files are planned.
func (s *IncrementalChangelogScan) ToSnapshot(snapshotID int64) *IncrementalChangelogScan {
	out := *s
	out.toSnapshotID = &snapshotID

	return &out
}

// PlanFiles returns one task for each added or deleted data-file entry. A
// cancelled context returns its cancellation error before planning starts.
// Tasks are ordered by change ordinal, then by DELETE before INSERT within an
// ordinal, and finally by data-file path. When an ending snapshot is
// available, it emits a ScanReport through the configured reporter on
// successful planning. Changelog reports count every returned task in
// ResultDataFiles and TotalFileSizeInBytes, so a file inserted and deleted
// within the range is counted twice.
func (s *IncrementalChangelogScan) PlanFiles(ctx context.Context) ([]ChangelogScanTask, error) {
	if s == nil || s.scan == nil {
		return nil, fmt.Errorf("%w: incremental changelog scan is not initialized", ErrInvalidOperation)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	switch s.scan.planningMode {
	case ScanPlanningLocal, ScanPlanningAuto:
	case ScanPlanningRemote:
		return nil, fmt.Errorf("%w: incremental changelog scans do not support remote planning", ErrInvalidOperation)
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
			return nil, fmt.Errorf("%w: no ending snapshot found for incremental changelog scan from %d",
				iceberg.ErrInvalidArgument, *s.fromSnapshotID)
		}

		return nil, nil
	}

	planningScan := *s.scan
	planningScan.identifier = slices.Clone(s.scan.identifier)
	planningScan.selectedFields = slices.Clone(s.scan.selectedFields)
	planningScan.options = maps.Clone(s.scan.options)
	if s.toSnapshotID != nil {
		planningScan.snapshotID = &toSnapshot.SnapshotID
		planningScan.asOfTimestamp = nil
	}
	schema, err := planningScan.effectiveSchema()
	if err != nil {
		return nil, err
	}
	residual, err := bindTaskFilter(schema, planningScan.rowFilter, planningScan.caseSensitive)
	if err != nil {
		return nil, fmt.Errorf("bind incremental changelog scan residual: %w", err)
	}
	var acc scanMetricsAccumulator
	finish := func(plannedTasks []plannedChangelogTask) []ChangelogScanTask {
		acc.resultDataFiles = int64(len(plannedTasks))
		tasks := make([]ChangelogScanTask, len(plannedTasks))
		fileTasks := make([]FileScanTask, len(plannedTasks))
		for i, planned := range plannedTasks {
			tasks[i] = planned.task
			fileTasks[i] = planned.file
			acc.totalFileSize += planned.file.File.FileSizeBytes()
		}
		acc.applyResultDeleteMetrics(fileTasks)
		planningDuration := time.Since(start)

		if rep := planningScan.Reporter(); !metrics.IsNop(rep) {
			projected, _ := planningScan.Projection()
			safeReport(ctx, rep, planningScan.buildScanReport(&acc, schema, projected, planningDuration))
		}

		return tasks
	}

	snapshotRange, err := incrementalSnapshotsBetween(
		s.scan.metadata, s.fromSnapshotID, s.fromInclusive, toSnapshot.SnapshotID)
	if err != nil {
		return nil, err
	}
	snapshots, err := changelogSnapshots(snapshotRange)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return finish(nil), nil
	}

	changelogSnapshotIDs := make(map[int64]struct{}, len(snapshots))
	snapshotOrdinals := make(map[int64]int, len(snapshots))
	for ordinal, snapshot := range snapshots {
		changelogSnapshotIDs[snapshot.SnapshotID] = struct{}{}
		snapshotOrdinals[snapshot.SnapshotID] = ordinal
	}

	if s.scan.ioF == nil {
		return nil, fmt.Errorf("%w: table file IO is not configured", ErrInvalidOperation)
	}
	fs, err := s.scan.ioF(ctx)
	if err != nil {
		return nil, err
	}

	manifestsByPath := make(map[string]iceberg.ManifestFile)
	for _, snapshot := range snapshots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		manifests, err := snapshot.Manifests(fs)
		if err != nil {
			return nil, err
		}
		for _, manifest := range manifests {
			if manifest.ManifestContent() == iceberg.ManifestContentDeletes {
				return nil, fmt.Errorf("%w: incremental changelog scan range references a delete manifest originating in snapshot %d",
					ErrInvalidOperation, manifest.SnapshotID())
			}
			if manifest.ManifestContent() != iceberg.ManifestContentData {
				continue
			}
			if _, ok := changelogSnapshotIDs[manifest.SnapshotID()]; !ok {
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

	// Changelog metrics intentionally count only manifests that contain added or
	// deleted data files; no-change manifests are removed before the scan metric
	// accumulator sees them.
	manifestList = slices.DeleteFunc(manifestList, func(manifest iceberg.ManifestFile) bool {
		return !manifestHasChangelogEntries(manifest)
	})
	partitionFilters := planningScan.partitionFiltersForSchema(schema)
	manifestList, err = planningScan.filterManifestsWithSchemaOptions(
		manifestList, schema, &acc, partitionFilters,
		/* includeDeleted= */ true)
	if err != nil {
		return nil, err
	}
	if len(manifestList) == 0 {
		return finish(nil), nil
	}
	entries, err := planningScan.collectManifestEntriesWithSchemaOptions(
		ctx, manifestList, schema,
		partitionFilters,
		/* discardDeleted= */ false,
		/* discardExisting= */ true,
	)
	if err != nil {
		return nil, err
	}

	plannedTasks := make([]plannedChangelogTask, 0, len(entries.dataEntries))
	for _, entry := range entries.dataEntries {
		ordinal, ok := snapshotOrdinals[entry.SnapshotID()]
		if !ok {
			continue
		}

		task, err := newChangelogScanTask(entry, ordinal, residual)
		if err != nil {
			return nil, fmt.Errorf("incremental changelog scan snapshot %d: %w", entry.SnapshotID(), err)
		}
		plannedTasks = append(plannedTasks, plannedChangelogTask{
			task: task,
			file: task.ScanTask(),
		})
	}
	slices.SortFunc(plannedTasks, func(left, right plannedChangelogTask) int {
		if ordinal := cmp.Compare(left.task.ChangeOrdinal(), right.task.ChangeOrdinal()); ordinal != 0 {
			return ordinal
		}
		if operation := cmp.Compare(changelogOperationOrder(left.task.Operation()), changelogOperationOrder(right.task.Operation())); operation != 0 {
			return operation
		}

		return cmp.Compare(left.file.File.FilePath(), right.file.File.FilePath())
	})

	return finish(plannedTasks), nil
}

func manifestHasChangelogEntries(manifest iceberg.ManifestFile) bool {
	// V1 manifest lists use -1 for unknown counts, so only zero means the
	// manifest is known not to contain added or deleted entries.
	return manifest.AddedDataFiles() != 0 || manifest.DeletedDataFiles() != 0
}

func changelogOperationOrder(operation ChangelogOperation) int {
	switch operation {
	// Deletes must be replayed before inserts within one change ordinal. Keep
	// this explicit instead of relying on the string values' lexical order.
	case ChangelogOpDelete:
		return 0
	case ChangelogOpInsert:
		return 1
	case ChangelogOpUpdateBefore:
		return 2
	case ChangelogOpUpdateAfter:
		return 3
	default:
		return 4
	}
}

func changelogOperation(status iceberg.ManifestEntryStatus) (ChangelogOperation, error) {
	switch status {
	case iceberg.EntryStatusADDED:
		return ChangelogOpInsert, nil
	case iceberg.EntryStatusDELETED:
		return ChangelogOpDelete, nil
	default:
		return "", fmt.Errorf("%w: unknown manifest entry status %d", ErrInvalidMetadata, status)
	}
}

func newChangelogScanTask(entry iceberg.ManifestEntry, ordinal int, residual iceberg.BooleanExpression) (ChangelogScanTask, error) {
	operation, err := changelogOperation(entry.Status())
	if err != nil {
		return nil, err
	}

	file := entry.DataFile()
	configureFileScanTask := func(task *FileScanTask) {
		task.Start = 0
		task.Length = file.FileSizeBytes()
		task.Residual = residual
		task.FirstRowID = file.FirstRowID()
		if sequenceNumber := entry.SequenceNum(); sequenceNumber >= 0 {
			task.DataSequenceNumber = &sequenceNumber
		}
	}

	switch operation {
	case ChangelogOpInsert:
		task, err := NewAddedRowsScanTask(file, nil, ordinal, entry.SnapshotID())
		if err != nil {
			return nil, err
		}
		configureFileScanTask(&task.FileScanTask)

		return task, nil
	case ChangelogOpDelete:
		task, err := NewDeletedDataFileScanTask(file, nil, ordinal, entry.SnapshotID())
		if err != nil {
			return nil, err
		}
		configureFileScanTask(&task.FileScanTask)

		return task, nil
	default:
		return nil, fmt.Errorf("%w: unsupported changelog operation %q", ErrInvalidOperation, operation)
	}
}

func (s *IncrementalChangelogScan) toSnapshot() (*Snapshot, error) {
	if s.toSnapshotID != nil {
		snapshot := s.scan.metadata.SnapshotByID(*s.toSnapshotID)
		if snapshot == nil {
			return nil, fmt.Errorf("%w: ending snapshot not found: %d", iceberg.ErrInvalidArgument, *s.toSnapshotID)
		}

		return snapshot, nil
	}

	return s.scan.ResolveSnapshot()
}

// changelogSnapshots retains every snapshot with a non-empty operation except
// replace snapshots. Keeping unknown future operations matches Java's
// incremental changelog scan behavior; only a missing operation is rejected.
func changelogSnapshots(snapshots []Snapshot) ([]Snapshot, error) {
	result := make([]Snapshot, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Summary == nil || snapshot.Summary.Operation == "" {
			return nil, fmt.Errorf("%w: cannot determine operation for snapshot %d",
				ErrMissingOperation, snapshot.SnapshotID)
		}

		if snapshot.Summary.Operation == OpReplace {
			continue
		}
		result = append(result, snapshot)
	}

	return result, nil
}
