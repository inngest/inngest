package conformance

import "slices"

type Transport string

const (
	TransportServe   Transport = "serve"
	TransportConnect Transport = "connect"
)

func ValidTransports() []Transport {
	return []Transport{TransportServe, TransportConnect}
}

func IsValidTransport(value Transport) bool {
	return slices.Contains(ValidTransports(), value)
}

type Compatibility string

const (
	CompatibilityFull         Compatibility = "full"
	CompatibilityPartial      Compatibility = "partial"
	CompatibilityIncompatible Compatibility = "incompatible"
	CompatibilityUnknown      Compatibility = "unknown"
)

type CaseStatus string

const (
	CaseStatusPassed         CaseStatus = "passed"
	CaseStatusFailed         CaseStatus = "failed"
	CaseStatusNotImplemented CaseStatus = "not_implemented"
	CaseStatusNotEvaluable   CaseStatus = "not_evaluable"
	CaseStatusSkipped        CaseStatus = "skipped"
)

type ReasonCode string

const (
	ReasonCodeNotImplemented     ReasonCode = "not_implemented"
	ReasonCodeTransportSetup     ReasonCode = "transport_setup_failed"
	ReasonCodeBehaviorMismatch   ReasonCode = "behavior_mismatch"
	ReasonCodeSecurityViolation  ReasonCode = "security_invariant_violated"
	ReasonCodePrerequisiteFailed ReasonCode = "prerequisite_failed"
)

type ReportFormat string

const (
	ReportFormatPretty   ReportFormat = "pretty"
	ReportFormatJSON     ReportFormat = "json"
	ReportFormatJUnit    ReportFormat = "junit"
	ReportFormatMarkdown ReportFormat = "markdown"
)

func ValidReportFormats() []ReportFormat {
	return []ReportFormat{
		ReportFormatPretty,
		ReportFormatJSON,
		ReportFormatJUnit,
		ReportFormatMarkdown,
	}
}

func IsValidReportFormat(value ReportFormat) bool {
	return slices.Contains(ValidReportFormats(), value)
}

type GoldenMode string

const (
	GoldenModeSemantic GoldenMode = "semantic"
)
