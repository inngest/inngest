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

	"github.com/apache/iceberg-go"
)

// ChangelogOperation is the kind of change a changelog scan task produces.
type ChangelogOperation string

const (
	ChangelogOpInsert       ChangelogOperation = "INSERT"
	ChangelogOpDelete       ChangelogOperation = "DELETE"
	ChangelogOpUpdateBefore ChangelogOperation = "UPDATE_BEFORE"
	ChangelogOpUpdateAfter  ChangelogOperation = "UPDATE_AFTER"
)

// ChangelogScanTask is a unit of work that produces changelog rows.
type ChangelogScanTask interface {
	// Implementations are intentionally limited to the changelog task types
	// provided by this package.
	isChangelogScanTask()
	// ScanTask returns the underlying file scan task for reading the data file.
	ScanTask() FileScanTask
	Operation() ChangelogOperation
	ChangeOrdinal() int
	CommitSnapshotID() int64
}

var (
	_ ChangelogScanTask = AddedRowsScanTask{}
	_ ChangelogScanTask = DeletedDataFileScanTask{}
	_ ChangelogScanTask = DeletedRowsScanTask{}
)

// classifiedDeletes holds delete files split the same way FileScanTask does,
// without a second FileScanTask whose range and lineage fields would be zero.
type classifiedDeletes struct {
	pos, eq, dv []iceberg.DataFile
}

func (d classifiedDeletes) files() []iceberg.DataFile {
	out := make([]iceberg.DataFile, 0, len(d.pos)+len(d.eq)+len(d.dv))
	out = append(out, d.pos...)
	out = append(out, d.eq...)
	out = append(out, d.dv...)

	return out
}

// AddedRowsScanTask is a changelog insert produced by adding a data file.
// Matching delete files committed in the same snapshot, or from squashed
// snapshots, are applied while reading so deleted rows are not emitted as
// inserts.
type AddedRowsScanTask struct {
	FileScanTask
	changeOrdinal    int
	commitSnapshotID int64
}

// NewAddedRowsScanTask constructs an insert task for dataFile. deletes are
// delete files that apply while reading the added file. Position deletes,
// equality deletes, and deletion vectors are stored on the matching
// FileScanTask fields.
func NewAddedRowsScanTask(dataFile iceberg.DataFile, deletes []iceberg.DataFile, changeOrdinal int, commitSnapshotID int64) (AddedRowsScanTask, error) {
	task, err := fileScanTaskWithDeletes(dataFile, deletes)
	if err != nil {
		return AddedRowsScanTask{}, err
	}

	return AddedRowsScanTask{
		FileScanTask:     task,
		changeOrdinal:    changeOrdinal,
		commitSnapshotID: commitSnapshotID,
	}, nil
}

func (t AddedRowsScanTask) Operation() ChangelogOperation { return ChangelogOpInsert }
func (t AddedRowsScanTask) ChangeOrdinal() int            { return t.changeOrdinal }
func (t AddedRowsScanTask) CommitSnapshotID() int64       { return t.commitSnapshotID }
func (t AddedRowsScanTask) ScanTask() FileScanTask        { return t.FileScanTask }
func (AddedRowsScanTask) isChangelogScanTask()            {}

// Deletes returns every delete file applied while reading the added data
// file: position deletes, then equality deletes, then deletion vectors.
func (t AddedRowsScanTask) Deletes() []iceberg.DataFile {
	return allDeleteFiles(t.FileScanTask)
}

// DeletedDataFileScanTask is a changelog delete produced by removing a data
// file. ExistingDeletes are delete files that were already present and must
// be applied so only rows that were live when the file was removed appear as
// deletes.
type DeletedDataFileScanTask struct {
	FileScanTask
	changeOrdinal    int
	commitSnapshotID int64
}

// NewDeletedDataFileScanTask constructs a delete task for a removed data file.
func NewDeletedDataFileScanTask(dataFile iceberg.DataFile, existingDeletes []iceberg.DataFile, changeOrdinal int, commitSnapshotID int64) (DeletedDataFileScanTask, error) {
	task, err := fileScanTaskWithDeletes(dataFile, existingDeletes)
	if err != nil {
		return DeletedDataFileScanTask{}, err
	}

	return DeletedDataFileScanTask{
		FileScanTask:     task,
		changeOrdinal:    changeOrdinal,
		commitSnapshotID: commitSnapshotID,
	}, nil
}

func (t DeletedDataFileScanTask) Operation() ChangelogOperation { return ChangelogOpDelete }
func (t DeletedDataFileScanTask) ChangeOrdinal() int            { return t.changeOrdinal }
func (t DeletedDataFileScanTask) CommitSnapshotID() int64       { return t.commitSnapshotID }
func (t DeletedDataFileScanTask) ScanTask() FileScanTask        { return t.FileScanTask }
func (DeletedDataFileScanTask) isChangelogScanTask()            {}

// ExistingDeletes returns delete files that applied before the data file was
// removed.
func (t DeletedDataFileScanTask) ExistingDeletes() []iceberg.DataFile {
	return allDeleteFiles(t.FileScanTask)
}

// DeletedRowsScanTask is a changelog delete produced by adding delete files
// against a data file that remains in the table. AddedDeletes remove rows
// that should appear in the changelog. ExistingDeletes already applied and
// those rows must not be emitted again.
type DeletedRowsScanTask struct {
	FileScanTask
	addedDeletes     classifiedDeletes
	changeOrdinal    int
	commitSnapshotID int64
}

// NewDeletedRowsScanTask constructs a row-level delete task. existingDeletes
// are stored on the embedded FileScanTask so later readers can reuse the
// normal scan delete path for the live-row baseline.
func NewDeletedRowsScanTask(dataFile iceberg.DataFile, addedDeletes, existingDeletes []iceberg.DataFile, changeOrdinal int, commitSnapshotID int64) (DeletedRowsScanTask, error) {
	existing, err := fileScanTaskWithDeletes(dataFile, existingDeletes)
	if err != nil {
		return DeletedRowsScanTask{}, err
	}

	added, err := classifyDeleteFiles(addedDeletes)
	if err != nil {
		return DeletedRowsScanTask{}, err
	}

	return DeletedRowsScanTask{
		FileScanTask:     existing,
		addedDeletes:     added,
		changeOrdinal:    changeOrdinal,
		commitSnapshotID: commitSnapshotID,
	}, nil
}

func (t DeletedRowsScanTask) Operation() ChangelogOperation { return ChangelogOpDelete }
func (t DeletedRowsScanTask) ChangeOrdinal() int            { return t.changeOrdinal }
func (t DeletedRowsScanTask) CommitSnapshotID() int64       { return t.commitSnapshotID }
func (t DeletedRowsScanTask) ScanTask() FileScanTask        { return t.FileScanTask }
func (DeletedRowsScanTask) isChangelogScanTask()            {}

// AddedDeletes returns delete files whose removals should appear in the
// changelog.
func (t DeletedRowsScanTask) AddedDeletes() []iceberg.DataFile {
	return t.addedDeletes.files()
}

// ExistingDeletes returns delete files that already applied before this
// snapshot's added deletes.
func (t DeletedRowsScanTask) ExistingDeletes() []iceberg.DataFile {
	return allDeleteFiles(t.FileScanTask)
}

func fileScanTaskWithDeletes(dataFile iceberg.DataFile, deletes []iceberg.DataFile) (FileScanTask, error) {
	classified, err := classifyDeleteFiles(deletes)
	if err != nil {
		return FileScanTask{}, err
	}

	return FileScanTask{
		File:                dataFile,
		DeleteFiles:         classified.pos,
		EqualityDeleteFiles: classified.eq,
		DeletionVectorFiles: classified.dv,
	}, nil
}

func classifyDeleteFiles(files []iceberg.DataFile) (classifiedDeletes, error) {
	var out classifiedDeletes
	for _, f := range files {
		kind, err := classifyDataFile(f)
		if err != nil {
			return classifiedDeletes{}, err
		}

		switch kind {
		case dataFileKindPosDeletes:
			out.pos = append(out.pos, f)
		case dataFileKindEqDeletes:
			out.eq = append(out.eq, f)
		case dataFileKindDeletionVector:
			out.dv = append(out.dv, f)
		default:
			return classifiedDeletes{}, fmt.Errorf("%w: expected delete file, got content type %s",
				ErrInvalidMetadata, f.ContentType())
		}
	}

	return out, nil
}

func allDeleteFiles(task FileScanTask) []iceberg.DataFile {
	return classifiedDeletes{
		pos: task.DeleteFiles,
		eq:  task.EqualityDeleteFiles,
		dv:  task.DeletionVectorFiles,
	}.files()
}
