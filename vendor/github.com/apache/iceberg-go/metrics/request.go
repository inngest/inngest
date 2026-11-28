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

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Report-type discriminator values, matching Java's ReportType wire form.
const (
	reportTypeScan   = "scan-report"
	reportTypeCommit = "commit-report"
)

// reportTypeField is the discriminator key in the request body.
const reportTypeField = "report-type"

// ReportMetricsRequest is the body POSTed to the catalog metrics endpoint
// (POST /v1/{prefix}/namespaces/{namespace}/tables/{table}/metrics). It maps to
// the spec's ReportMetricsRequest: a ScanReport or CommitReport flattened
// alongside a "report-type" discriminator (the report's fields are siblings of
// report-type, not nested).
type ReportMetricsRequest struct {
	Report MetricsReport
}

// NewReportMetricsRequest wraps a report for transport.
func NewReportMetricsRequest(report MetricsReport) ReportMetricsRequest {
	return ReportMetricsRequest{Report: report}
}

// MarshalJSON writes report-type plus the report's own fields at the top level.
func (r ReportMetricsRequest) MarshalJSON() ([]byte, error) {
	// Guard nil reports up front. This catches both an untyped nil and a typed
	// nil pointer (e.g. (*ScanReport)(nil)), which would otherwise match the
	// type switch below and marshal to a bare "null" — silently producing a
	// report with no required fields instead of an error. Mirrors isNilReport
	// used by the built-in reporters.
	if isNilReport(r.Report) {
		return nil, errors.New("metrics: cannot marshal nil report")
	}

	var reportType string
	switch r.Report.(type) {
	case ScanReport, *ScanReport:
		reportType = reportTypeScan
	case CommitReport, *CommitReport:
		reportType = reportTypeCommit
	default:
		return nil, fmt.Errorf("metrics: cannot marshal report of type %T", r.Report)
	}

	body, err := json.Marshal(r.Report)
	if err != nil {
		return nil, err
	}

	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	fields[reportTypeField], _ = json.Marshal(reportType)

	return json.Marshal(fields)
}

// UnmarshalJSON reads the report-type discriminator and decodes the matching
// concrete report.
func (r *ReportMetricsRequest) UnmarshalJSON(data []byte) error {
	var disc struct {
		ReportType string `json:"report-type"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}

	switch disc.ReportType {
	case reportTypeScan:
		var sr ScanReport
		if err := json.Unmarshal(data, &sr); err != nil {
			return err
		}
		r.Report = sr
	case reportTypeCommit:
		var cr CommitReport
		if err := json.Unmarshal(data, &cr); err != nil {
			return err
		}
		r.Report = cr
	case "":
		return fmt.Errorf("metrics: missing %q in metrics request", reportTypeField)
	default:
		return fmt.Errorf("metrics: unknown report-type %q", disc.ReportType)
	}

	return nil
}
