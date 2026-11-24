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
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/metrics"
)

// scanMetricsAccumulator gathers scan-planning counts. Every field is written
// from a single goroutine — the manifest counts in
// fetchPartitionSpecFilteredManifestsWithSchema and the result/delete counts in
// planFilesLocal after its concurrent manifest/task barrier — so plain
// integers are race-free; no field is touched from the concurrent workers.
//
// The scanned/skipped manifest counts measure partition-spec pruning (the
// filter applied while listing manifests). The per-entry skipped-data-files /
// skipped-delete-files counts (which would be incremented inside the concurrent
// manifest-entry loop, and would use atomics) and indexed-delete-files are left
// for a follow-up and omitted rather than reported as zero.
type scanMetricsAccumulator struct {
	totalDataManifests     int64
	totalDeleteManifests   int64
	scannedDataManifests   int64
	scannedDeleteManifests int64
	skippedDataManifests   int64
	skippedDeleteManifests int64

	resultDataFiles       int64
	resultDeleteFiles     int64
	totalFileSize         int64
	totalDeleteFileSize   int64
	positionalDeleteFiles int64
	equalityDeleteFiles   int64
	dvs                   int64
}

func counterCount(v int64) *metrics.CounterResult {
	return metrics.NewCounterResult(metrics.UnitCount, v)
}

func counterBytes(v int64) *metrics.CounterResult {
	return metrics.NewCounterResult(metrics.UnitBytes, v)
}

// applyResultDeleteMetrics derives the result-scoped delete-file metrics from
// the planned tasks. Following Java's ScanMetricsUtil.fileTask, every delete
// file is counted once per data-file task it applies to — a delete covering N
// data files is counted N times — with no cross-task dedup, so these
// Java-parity field names stay directly comparable with what Java consumers
// report. Sizes accumulate the same way.
//
// A deletion vector's size is its per-blob content_size_in_bytes (Java's
// ScanTaskUtil.contentSizeInBytes), not the size of the Puffin file that holds
// it: DVWriter.Flush packs one DV blob per data file into a single Puffin file,
// so the Puffin length would over-count.
func (acc *scanMetricsAccumulator) applyResultDeleteMetrics(tasks []FileScanTask) {
	for _, t := range tasks {
		acc.addResultDeleteMetrics(t)
	}
}

func (acc *scanMetricsAccumulator) addResultDeleteMetrics(t FileScanTask) {
	for _, df := range t.DeleteFiles {
		acc.positionalDeleteFiles++
		acc.totalDeleteFileSize += df.FileSizeBytes()
	}
	for _, df := range t.EqualityDeleteFiles {
		acc.equalityDeleteFiles++
		acc.totalDeleteFileSize += df.FileSizeBytes()
	}
	for _, df := range t.DeletionVectorFiles {
		acc.dvs++
		_, _, _, _, csb := iceberginternal.BorrowedDataFilePointers(df)
		if csb != nil {
			acc.totalDeleteFileSize += *csb
		}
	}

	acc.resultDeleteFiles = acc.positionalDeleteFiles + acc.equalityDeleteFiles + acc.dvs
}

// buildScanReport assembles the ScanReport for a completed planning operation.
// schema is the schema the scan planned against and projected is the projected
// (column-selected) schema, both threaded in from the planner so the report
// describes exactly what was used rather than re-resolving it here. projected may
// be nil when the projection could not be resolved, in which case the report
// simply carries no projected fields. Only the metrics that are actually measured
// are populated; unmeasured ones (e.g. skipped-data-files, indexed-delete-files)
// are left unset and omitted.
func (scan *Scan) buildScanReport(acc *scanMetricsAccumulator, schema, projected *iceberg.Schema, planning time.Duration) metrics.ScanReport {
	ids, names := projectedFields(projected)
	envCtx := iceberg.EnvironmentContext()
	metadata := make(map[string]string, len(scan.options)+len(envCtx))
	maps.Copy(metadata, scan.options)
	// Match Java's precedence: process-wide context overrides scan options.
	maps.Copy(metadata, envCtx)

	var snapshotID int64
	if snap := scan.Snapshot(); snap != nil {
		snapshotID = snap.SnapshotID
	}

	// Serialize the scan's row filter as Expression JSON, first sanitizing it so
	// predicate literals (which can be user data) never reach a reporter or REST
	// metrics sink. On any error we leave Filter unset so ScanReport.MarshalJSON
	// falls back to always-true rather than failing the scan or, worse, emitting
	// unsanitized literals.
	var filter json.RawMessage
	if scan.rowFilter != nil {
		if sanitized, err := iceberg.SanitizeExpression(scan.rowFilter); err == nil {
			if raw, err := json.Marshal(sanitized); err == nil {
				filter = raw
			}
		}
	}

	return metrics.ScanReport{
		TableName:           strings.Join(scan.identifier, "."),
		SnapshotID:          snapshotID,
		SchemaID:            schema.ID,
		ProjectedFieldIDs:   ids,
		ProjectedFieldNames: names,
		Filter:              filter,
		Metadata:            metadata,
		Metrics: metrics.ScanMetricsResult{
			TotalPlanningDuration:      metrics.NewNanosTimerResult(1, planning.Nanoseconds()),
			ResultDataFiles:            counterCount(acc.resultDataFiles),
			ResultDeleteFiles:          counterCount(acc.resultDeleteFiles),
			TotalDataManifests:         counterCount(acc.totalDataManifests),
			TotalDeleteManifests:       counterCount(acc.totalDeleteManifests),
			ScannedDataManifests:       counterCount(acc.scannedDataManifests),
			ScannedDeleteManifests:     counterCount(acc.scannedDeleteManifests),
			SkippedDataManifests:       counterCount(acc.skippedDataManifests),
			SkippedDeleteManifests:     counterCount(acc.skippedDeleteManifests),
			TotalFileSizeInBytes:       counterBytes(acc.totalFileSize),
			TotalDeleteFileSizeInBytes: counterBytes(acc.totalDeleteFileSize),
			EqualityDeleteFiles:        counterCount(acc.equalityDeleteFiles),
			PositionalDeleteFiles:      counterCount(acc.positionalDeleteFiles),
			DVs:                        counterCount(acc.dvs),
		},
	}
}

// projectedFields resolves the field ids and dotted column names the scan reads,
// matching Java's TypeUtil.getProjectedIds + Schema.findColumnName: nested fields
// are reported by id and full dotted path, not just top-level columns. It walks
// the projected schema threaded in from the planner directly rather than
// re-resolving scan.Projection(), so the report reflects exactly what planning
// used and does not silently drop the projected fields if Projection() fails for
// a reason planning never hit (e.g. the row-lineage version check). A nil schema
// (projection unresolved) yields no projected fields. Ids are returned sorted for
// deterministic output; an id not resolvable to a name in the schema (e.g. a
// reserved row-lineage column) is skipped so ids and names stay aligned.
func projectedFields(schema *iceberg.Schema) ([]int, []string) {
	if schema == nil {
		return nil, nil
	}

	idToField, err := iceberg.IndexByID(schema)
	if err != nil {
		return nil, nil
	}

	ids := make([]int, 0, len(idToField))
	for id := range idToField {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	resolvedIDs := make([]int, 0, len(ids))
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		name, ok := schema.FindColumnName(id)
		if !ok {
			continue
		}
		resolvedIDs = append(resolvedIDs, id)
		names = append(names, name)
	}

	return resolvedIDs, names
}
