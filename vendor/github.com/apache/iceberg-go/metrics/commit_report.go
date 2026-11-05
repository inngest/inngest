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

package metrics

// CommitMetricsResult is the serializable set of commit metrics. Field names
// and units match Java's CommitMetricsResult / CommitMetrics so the wire
// format is identical across implementations. Unset metrics are omitted.
//
// The counter values are derived from the snapshot summary; note that several
// of Java's commit-report metric names differ from iceberg-go's snapshot
// summary keys (e.g. added-files-size-bytes vs the summary's added-files-size),
// and the commit instrumentation is responsible for emitting them under these
// Java names.
type CommitMetricsResult struct {
	TotalDuration *TimerResult   `json:"total-duration,omitempty"`
	Attempts      *CounterResult `json:"attempts,omitempty"`

	AddedDataFiles   *CounterResult `json:"added-data-files,omitempty"`
	RemovedDataFiles *CounterResult `json:"removed-data-files,omitempty"`
	TotalDataFiles   *CounterResult `json:"total-data-files,omitempty"`

	AddedDeleteFiles   *CounterResult `json:"added-delete-files,omitempty"`
	RemovedDeleteFiles *CounterResult `json:"removed-delete-files,omitempty"`
	TotalDeleteFiles   *CounterResult `json:"total-delete-files,omitempty"`

	AddedEqualityDeleteFiles   *CounterResult `json:"added-equality-delete-files,omitempty"`
	RemovedEqualityDeleteFiles *CounterResult `json:"removed-equality-delete-files,omitempty"`

	AddedPositionalDeleteFiles   *CounterResult `json:"added-positional-delete-files,omitempty"`
	RemovedPositionalDeleteFiles *CounterResult `json:"removed-positional-delete-files,omitempty"`

	AddedDVs   *CounterResult `json:"added-dvs,omitempty"`
	RemovedDVs *CounterResult `json:"removed-dvs,omitempty"`

	AddedRecords   *CounterResult `json:"added-records,omitempty"`
	RemovedRecords *CounterResult `json:"removed-records,omitempty"`
	TotalRecords   *CounterResult `json:"total-records,omitempty"`

	AddedFilesSizeBytes   *CounterResult `json:"added-files-size-bytes,omitempty"`
	RemovedFilesSizeBytes *CounterResult `json:"removed-files-size-bytes,omitempty"`
	TotalFilesSizeBytes   *CounterResult `json:"total-files-size-bytes,omitempty"`

	AddedPositionalDeletes   *CounterResult `json:"added-positional-deletes,omitempty"`
	RemovedPositionalDeletes *CounterResult `json:"removed-positional-deletes,omitempty"`
	TotalPositionalDeletes   *CounterResult `json:"total-positional-deletes,omitempty"`

	AddedEqualityDeletes   *CounterResult `json:"added-equality-deletes,omitempty"`
	RemovedEqualityDeletes *CounterResult `json:"removed-equality-deletes,omitempty"`
	TotalEqualityDeletes   *CounterResult `json:"total-equality-deletes,omitempty"`

	ManifestsCreated         *CounterResult `json:"manifests-created,omitempty"`
	ManifestsReplaced        *CounterResult `json:"manifests-replaced,omitempty"`
	ManifestsKept            *CounterResult `json:"manifests-kept,omitempty"`
	ManifestEntriesProcessed *CounterResult `json:"manifest-entries-processed,omitempty"`
}

// CommitReport is emitted after a commit completes. It maps to the spec's
// CommitReport schema.
type CommitReport struct {
	TableName      string              `json:"table-name"`
	SnapshotID     int64               `json:"snapshot-id"`
	SequenceNumber int64               `json:"sequence-number"`
	Operation      string              `json:"operation"`
	Metrics        CommitMetricsResult `json:"metrics"`
	Metadata       map[string]string   `json:"metadata,omitempty"`
}
