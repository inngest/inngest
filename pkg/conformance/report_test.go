package conformance

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewReportRollsUpCompatibility(t *testing.T) {
	t.Parallel()

	plan := RunPlan{
		Transport: TransportServe,
		Cases: []Case{
			{ID: "basic-invoke", SuiteID: "core", Features: []string{"registration", "invocation"}},
			{ID: "retry-basic", SuiteID: "core", Features: []string{"retry"}},
		},
		Features: []Feature{
			{ID: "invocation", Label: "Invocation"},
			{ID: "registration", Label: "Registration"},
			{ID: "retry", Label: "Retry"},
		},
	}

	report := NewReport(plan, []CaseResult{
		{CaseID: "basic-invoke", SuiteID: "core", Status: CaseStatusPassed},
		{CaseID: "retry-basic", SuiteID: "core", Status: CaseStatusNotImplemented, ReasonCode: ReasonCodeNotImplemented},
	})

	require.Equal(t, CompatibilityPartial, report.Compatibility)
	require.Len(t, report.Features, 3)
}

func TestReportRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	report := Report{
		SchemaVersion: ReportSchemaVersion,
		Transport:     TransportServe,
		Compatibility: CompatibilityFull,
		Cases: []CaseResult{
			{CaseID: "basic-invoke", SuiteID: "core", Status: CaseStatusPassed},
		},
	}

	require.NoError(t, WriteReport(path, report))

	loaded, err := LoadReport(path)
	require.NoError(t, err)
	require.Equal(t, report, loaded)
}
