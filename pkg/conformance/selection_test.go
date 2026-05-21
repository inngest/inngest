package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryIsValid(t *testing.T) {
	t.Parallel()

	require.NoError(t, DefaultRegistry().Validate())
}

func TestResolveSelectionDefaultsToAllCases(t *testing.T) {
	t.Parallel()

	plan, err := (Selection{}).Resolve(DefaultRegistry())
	require.NoError(t, err)
	require.NotEmpty(t, plan.Cases)
	require.NotEmpty(t, plan.Suites)
	require.NotEmpty(t, plan.Features)
}

func TestResolveSelectionFiltersByTransportAndFeature(t *testing.T) {
	t.Parallel()

	plan, err := (Selection{
		Transport: TransportConnect,
		Features:  []string{"connect-readiness"},
	}).Resolve(DefaultRegistry())
	require.NoError(t, err)
	require.Len(t, plan.Cases, 1)
	require.Equal(t, "connect-ready", plan.Cases[0].ID)
	require.Len(t, plan.Features, 1)
	require.Equal(t, "connect-readiness", plan.Features[0].ID)
}

func TestResolveSelectionRejectsUnknownSuite(t *testing.T) {
	t.Parallel()

	_, err := (Selection{Suites: []string{"missing"}}).Resolve(DefaultRegistry())
	require.ErrorContains(t, err, `unknown suite "missing"`)
}

func TestResolveSelectionRejectsEmptyMatch(t *testing.T) {
	t.Parallel()

	_, err := (Selection{
		Transport: TransportServe,
		Features:  []string{"connect-readiness"},
	}).Resolve(DefaultRegistry())
	require.ErrorContains(t, err, "selection did not match any conformance cases")
}
