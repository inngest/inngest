package conformance

import (
	"encoding/json"
	"fmt"
	"os"
)

const ReportSchemaVersion = "v1alpha1"

// Report is the portable artifact written by the conformance runner.
//
// The shape is still intentionally compact in Phase 2. It contains enough data
// to drive terminal rendering, JSON artifacts, and later diff/report commands
// without forcing verbose per-transport traces into the base file.
type Report struct {
	SchemaVersion string          `json:"schema_version"`
	Transport     Transport       `json:"transport,omitempty"`
	Compatibility Compatibility   `json:"compatibility"`
	Features      []FeatureResult `json:"features,omitempty"`
	Cases         []CaseResult    `json:"cases,omitempty"`
}

type FeatureResult struct {
	FeatureID      string        `json:"feature_id"`
	Compatibility  Compatibility `json:"compatibility"`
	ReasonCode     ReasonCode    `json:"reason_code,omitempty"`
	Reason         string        `json:"reason,omitempty"`
	BackingCaseIDs []string      `json:"backing_case_ids,omitempty"`
}

type CaseResult struct {
	CaseID      string     `json:"case_id"`
	SuiteID     string     `json:"suite_id"`
	Status      CaseStatus `json:"status"`
	ReasonCode  ReasonCode `json:"reason_code,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	ArtifactRef string     `json:"artifact_ref,omitempty"`
}

// NewReport creates a report from the resolved plan and case outcomes.
func NewReport(plan RunPlan, caseResults []CaseResult) Report {
	features := BuildFeatureResults(plan, caseResults)

	return Report{
		SchemaVersion: ReportSchemaVersion,
		Transport:     plan.Transport,
		Compatibility: OverallCompatibility(features),
		Features:      features,
		Cases:         caseResults,
	}
}

// WriteReport persists the base report artifact as stable JSON.
func WriteReport(path string, report Report) error {
	byt, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	if err := os.WriteFile(path, byt, 0o600); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}

// LoadReport reads a previously written JSON report artifact.
func LoadReport(path string) (Report, error) {
	byt, err := os.ReadFile(path)
	if err != nil {
		return Report{}, fmt.Errorf("read report: %w", err)
	}

	var report Report
	if err := json.Unmarshal(byt, &report); err != nil {
		return Report{}, fmt.Errorf("decode report: %w", err)
	}

	return report, nil
}
