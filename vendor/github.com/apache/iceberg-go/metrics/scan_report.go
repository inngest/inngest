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

import "encoding/json"

// alwaysTrueFilter is the structured Expression emitted for ScanReport.filter
// when no filter is set. Iceberg serializes the always-true expression as the
// bare JSON boolean true (see AlwaysTrue.MarshalJSON in the root package, which
// mirrors Java's ExpressionParser); a {"type":...} object is used only for
// predicates and and/or/not nodes. Emitting anything else here would be
// rejected as invalid Expression JSON by a spec-compliant catalog.
var alwaysTrueFilter = json.RawMessage(`true`)

// ScanMetricsResult is the serializable set of scan-planning metrics. Field
// names and units match Java's ScanMetricsResult so the wire format is
// identical across implementations. Unset metrics are omitted.
type ScanMetricsResult struct {
	TotalPlanningDuration      *TimerResult   `json:"total-planning-duration,omitempty"`
	ResultDataFiles            *CounterResult `json:"result-data-files,omitempty"`
	ResultDeleteFiles          *CounterResult `json:"result-delete-files,omitempty"`
	TotalDataManifests         *CounterResult `json:"total-data-manifests,omitempty"`
	TotalDeleteManifests       *CounterResult `json:"total-delete-manifests,omitempty"`
	ScannedDataManifests       *CounterResult `json:"scanned-data-manifests,omitempty"`
	ScannedDeleteManifests     *CounterResult `json:"scanned-delete-manifests,omitempty"`
	SkippedDataManifests       *CounterResult `json:"skipped-data-manifests,omitempty"`
	SkippedDeleteManifests     *CounterResult `json:"skipped-delete-manifests,omitempty"`
	SkippedDataFiles           *CounterResult `json:"skipped-data-files,omitempty"`
	SkippedDeleteFiles         *CounterResult `json:"skipped-delete-files,omitempty"`
	TotalFileSizeInBytes       *CounterResult `json:"total-file-size-in-bytes,omitempty"`
	TotalDeleteFileSizeInBytes *CounterResult `json:"total-delete-file-size-in-bytes,omitempty"`
	IndexedDeleteFiles         *CounterResult `json:"indexed-delete-files,omitempty"`
	EqualityDeleteFiles        *CounterResult `json:"equality-delete-files,omitempty"`
	PositionalDeleteFiles      *CounterResult `json:"positional-delete-files,omitempty"`
	DVs                        *CounterResult `json:"dvs,omitempty"`
}

// ScanReport is emitted after scan planning completes. It maps to the spec's
// ScanReport schema. Filter holds the scan filter as Expression JSON (a bare
// boolean for always-true/false, or a {"type":...} object for predicates and
// and/or/not nodes); when unset it marshals as always-true (see
// [alwaysTrueFilter]).
type ScanReport struct {
	TableName           string            `json:"table-name"`
	SnapshotID          int64             `json:"snapshot-id"`
	SchemaID            int               `json:"schema-id"`
	ProjectedFieldIDs   []int             `json:"projected-field-ids"`
	ProjectedFieldNames []string          `json:"projected-field-names"`
	Filter              json.RawMessage   `json:"filter"`
	Metrics             ScanMetricsResult `json:"metrics"`
	Metadata            map[string]string `json:"metadata,omitempty"`
}

// MarshalJSON defaults an unset Filter to the "always true" expression so the
// spec-required filter field is always present.
func (s ScanReport) MarshalJSON() ([]byte, error) {
	type alias ScanReport
	a := alias(s)
	if len(a.Filter) == 0 {
		a.Filter = alwaysTrueFilter
	}
	if a.ProjectedFieldIDs == nil {
		a.ProjectedFieldIDs = []int{}
	}
	if a.ProjectedFieldNames == nil {
		a.ProjectedFieldNames = []string{}
	}

	return json.Marshal(a)
}
