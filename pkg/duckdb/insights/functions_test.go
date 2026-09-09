package insights

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAllowedFunctionsHaveDescriptionsAndDocsURLs guards every allowlisted
// function against a missing description or a malformed docs link -- both
// are surfaced verbatim by cmd/gen-insights-schema's JSON dump.
func TestAllowedFunctionsHaveDescriptionsAndDocsURLs(t *testing.T) {
	for name, info := range allowedFunctions {
		require.NotEmptyf(t, info.description, "function %q has no description", name)
		require.NotEmptyf(t, info.docsURL, "function %q has no docsURL", name)

		u, err := url.Parse(info.docsURL)
		require.NoErrorf(t, err, "function %q has an unparseable docsURL %q", name, info.docsURL)
		require.Equalf(t, "https", u.Scheme, "function %q docsURL must be https", name)
		require.Truef(t, strings.HasPrefix(u.Host, "duckdb.org"), "function %q docsURL must point at duckdb.org, got %q", name, info.docsURL)
	}
}
