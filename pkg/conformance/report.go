package conformance

const ReportSchemaVersion = "v1alpha1"

type Report struct {
	SchemaVersion string          `json:"schema_version"`
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
