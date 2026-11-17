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
	"context"
	"log"
	"strconv"
	"time"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/metrics"
)

// safeReport forwards a report to rep, recovering from a panic in a misbehaving
// third-party reporter so it can never fail a commit that has already durably
// succeeded. The Reporter contract requires a reporter never affect the observed
// operation; compositeReporter isolates each inner reporter this way, but a bare
// non-composite reporter gets no such protection at the emit site.
func safeReport(ctx context.Context, rep metrics.Reporter, report metrics.MetricsReport) {
	defer func() {
		if v := recover(); v != nil {
			log.Printf("Warning: metrics reporter %T panicked; recovered: %v", rep, v)
		}
	}()
	rep.Report(ctx, report)
}

// commitAddedSnapshot reports whether the commit produced a new snapshot.
// Metadata-only commits (property or schema changes) carry no addSnapshotUpdate
// and must not emit a commit report: they create no snapshot, so the branch head
// is unchanged and reporting it would attribute a prior snapshot's metrics to
// this commit. This mirrors Java, whose CommitReport is emitted only from the
// snapshot-producing path (SnapshotProducer.commit).
func commitAddedSnapshot(updates []Update) bool {
	for _, u := range updates {
		if _, ok := u.(*addSnapshotUpdate); ok {
			return true
		}
	}

	return false
}

// summaryCounter reads a snapshot-summary property and returns it as a
// CounterResult, or nil if the key is absent or unparseable. This mirrors
// Java's CommitMetricsResult.counterFrom exactly: a metric is omitted rather
// than reported as a zero unless the snapshot summary carries a parseable value
// for it. As a result iceberg-go and Java emit the same present/absent metric
// set for an equivalent commit.
func summaryCounter(props iceberg.Properties, key string, unit metrics.Unit) *metrics.CounterResult {
	v, ok := props[key]
	if !ok {
		return nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}

	return metrics.NewCounterResult(unit, n)
}

// buildCommitReport assembles a CommitReport from the committed snapshot's
// summary, mirroring Java's CommitMetricsResult.from(commitMetrics,
// snapshotSummary): attempts and total-duration come from the commit itself,
// and every other metric is read from the snapshot summary under its spec key
// and emitted under Java's commit-report field name so dashboards line up
// across implementations. Metrics whose summary key iceberg-go does not yet
// populate (DVs) are absent and therefore omitted — exactly as Java's
// counterFrom omits summary keys it does not find.
func buildCommitReport(tableName string, snap *Snapshot, attempts int64, dur time.Duration) metrics.CommitReport {
	var (
		snapshotID int64
		seqNum     int64
		operation  string
		props      iceberg.Properties
	)
	if snap != nil {
		snapshotID = snap.SnapshotID
		seqNum = snap.SequenceNumber
		if snap.Summary != nil {
			operation = string(snap.Summary.Operation)
			props = snap.Summary.Properties
		}
	}

	count := func(key string) *metrics.CounterResult { return summaryCounter(props, key, metrics.UnitCount) }
	bytesOf := func(key string) *metrics.CounterResult { return summaryCounter(props, key, metrics.UnitBytes) }

	return metrics.CommitReport{
		TableName:      tableName,
		SnapshotID:     snapshotID,
		SequenceNumber: seqNum,
		Operation:      operation,
		Metadata:       iceberg.EnvironmentContext(),
		Metrics: metrics.CommitMetricsResult{
			TotalDuration: metrics.NewNanosTimerResult(1, dur.Nanoseconds()),
			Attempts:      metrics.NewCounterResult(metrics.UnitCount, attempts),

			AddedDataFiles:   count(addedDataFilesKey),
			RemovedDataFiles: count(deletedDataFilesKey),
			TotalDataFiles:   count(totalDataFilesKey),

			AddedDeleteFiles:   count(addedDeleteFilesKey),
			RemovedDeleteFiles: count(removedDeleteFilesKey),
			TotalDeleteFiles:   count(totalDeleteFilesKey),

			AddedEqualityDeleteFiles:   count(addedEqDeleteFilesKey),
			RemovedEqualityDeleteFiles: count(removedEqDeleteFilesKey),

			AddedPositionalDeleteFiles:   count(addedPosDeleteFilesKey),
			RemovedPositionalDeleteFiles: count(removedPosDeleteFilesKey),

			AddedRecords:   count(addedRecordsKey),
			RemovedRecords: count(deletedRecordsKey),
			TotalRecords:   count(totalRecordsKey),

			AddedFilesSizeBytes:   bytesOf(addedFileSizeKey),
			RemovedFilesSizeBytes: bytesOf(removedFileSizeKey),
			TotalFilesSizeBytes:   bytesOf(totalFileSizeKey),

			AddedPositionalDeletes:   count(addedPosDeletesKey),
			RemovedPositionalDeletes: count(removedPosDeletesKey),
			TotalPositionalDeletes:   count(totalPosDeletesKey),

			AddedEqualityDeletes:   count(addedEqDeletesKey),
			RemovedEqualityDeletes: count(removedEqDeletesKey),
			TotalEqualityDeletes:   count(totalEqDeletesKey),

			ManifestsCreated:  count(manifestsCreatedKey),
			ManifestsReplaced: count(manifestsReplacedKey),
			ManifestsKept:     count(manifestsKeptKey),
			// entries-processed in the summary maps to manifest-entries-processed.
			ManifestEntriesProcessed: count(entriesProcessedKey),
		},
	}
}
