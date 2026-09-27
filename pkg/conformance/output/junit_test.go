package output

import (
	"encoding/xml"
	"testing"

	"github.com/inngest/inngest/pkg/conformance"
	"github.com/stretchr/testify/require"
)

// TestRenderJUnit_MapsStatuses verifies the conformance-to-JUnit status mapping
// that CI pipelines rely on: failures and errors should be countable separately
// from skipped (not-implemented) cases.
func TestRenderJUnit_MapsStatuses(t *testing.T) {
	t.Parallel()

	report := conformance.Report{
		SchemaVersion: conformance.ReportSchemaVersion,
		Transport:     conformance.TransportServe,
		Compatibility: conformance.CompatibilityPartial,
		Cases: []conformance.CaseResult{
			{
				CaseID:  "basic-invoke",
				SuiteID: "core",
				Status:  conformance.CaseStatusPassed,
			},
			{
				CaseID:     "retry-basic",
				SuiteID:    "core",
				Status:     conformance.CaseStatusNotImplemented,
				ReasonCode: conformance.ReasonCodeNotImplemented,
				Reason:     "no Phase 2 serve executor is defined for this case",
			},
			{
				CaseID:     "steps-serial",
				SuiteID:    "core",
				Status:     conformance.CaseStatusFailed,
				ReasonCode: conformance.ReasonCodeBehaviorMismatch,
				Reason:     "unexpected generator response",
			},
			{
				CaseID:     "malformed-payload",
				SuiteID:    "negative",
				Status:     conformance.CaseStatusNotEvaluable,
				ReasonCode: conformance.ReasonCodeTransportSetup,
				Reason:     "sdk.url is required",
			},
		},
	}

	byt, err := RenderJUnit(report)
	require.NoError(t, err)
	require.Contains(t, string(byt), `<?xml version="1.0" encoding="UTF-8"?>`)

	// Round-trip through xml.Unmarshal to ensure the output is well-formed XML.
	var parsed junitTestsuites
	require.NoError(t, xml.Unmarshal(byt, &parsed))

	require.Equal(t, "inngest-conformance", parsed.Name)
	require.Equal(t, 4, parsed.Tests)
	require.Equal(t, 1, parsed.Failures)
	require.Equal(t, 1, parsed.Errors)
	require.Equal(t, 1, parsed.Skipped)
	require.Len(t, parsed.Suites, 2)

	// Locate the "core" suite and inspect individual testcases.
	var coreSuite *junitTestsuite
	for i := range parsed.Suites {
		if parsed.Suites[i].Name == "core" {
			coreSuite = &parsed.Suites[i]
			break
		}
	}
	require.NotNil(t, coreSuite)
	require.Equal(t, 3, coreSuite.Tests)
	require.Equal(t, 1, coreSuite.Failures)
	require.Equal(t, 1, coreSuite.Skipped)

	byName := map[string]junitTestcase{}
	for _, tc := range coreSuite.Cases {
		byName[tc.Name] = tc
	}

	require.Nil(t, byName["basic-invoke"].Failure)
	require.Nil(t, byName["basic-invoke"].Skipped)

	require.NotNil(t, byName["retry-basic"].Skipped)
	require.Contains(t, byName["retry-basic"].Skipped.Message, "no Phase 2 serve executor")

	require.NotNil(t, byName["steps-serial"].Failure)
	require.Equal(t, "behavior_mismatch", byName["steps-serial"].Failure.Type)
	require.Equal(t, "unexpected generator response", byName["steps-serial"].Failure.Body)
}

// TestRenderJUnit_EmptyReport ensures an empty report still produces valid XML.
// Useful when a suite selection resolves to zero cases (edge case / future guard).
func TestRenderJUnit_EmptyReport(t *testing.T) {
	t.Parallel()

	byt, err := RenderJUnit(conformance.Report{
		SchemaVersion: conformance.ReportSchemaVersion,
	})
	require.NoError(t, err)

	var parsed junitTestsuites
	require.NoError(t, xml.Unmarshal(byt, &parsed))
	require.Equal(t, 0, parsed.Tests)
	require.Empty(t, parsed.Suites)
}

// TestRenderJUnit_DeterministicOrdering verifies that repeated renders produce
// identical output. CI diff tools and golden snapshots depend on this property.
func TestRenderJUnit_DeterministicOrdering(t *testing.T) {
	t.Parallel()

	report := conformance.Report{
		SchemaVersion: conformance.ReportSchemaVersion,
		Cases: []conformance.CaseResult{
			{CaseID: "z-case", SuiteID: "core", Status: conformance.CaseStatusPassed},
			{CaseID: "a-case", SuiteID: "core", Status: conformance.CaseStatusPassed},
			{CaseID: "b-case", SuiteID: "alpha", Status: conformance.CaseStatusPassed},
		},
	}

	first, err := RenderJUnit(report)
	require.NoError(t, err)

	second, err := RenderJUnit(report)
	require.NoError(t, err)

	require.Equal(t, string(first), string(second))
}
