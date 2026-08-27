package apiv2oauth

import (
	"testing"

	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/stretchr/testify/require"
)

func TestScopesForEndpoints(t *testing.T) {
	endpoints := []apiv2endpoint.Endpoint{
		{AuthzPermission: "apps:read:get"},
		{AuthzPermission: "apps:read:list"},
		{AuthzPermission: "functions:write:invoke"},
		{AuthzPermission: "event_keys:read:list"},
		{AuthzPermission: ""},
	}

	require.Equal(t, []string{
		"apps:read:*",
		"functions:write:*",
	}, scopesForEndpoints(endpoints, cliExcludedResources))
}

func TestCLIScopesExcludeNonDelegatableKeys(t *testing.T) {
	scopes := CLIScopes()

	require.Contains(t, scopes, "apps:read:*")
	require.Contains(t, scopes, "apps:write:*")
	require.NotContains(t, scopes, "event_keys:read:*")
	require.NotContains(t, scopes, "signing_keys:read:*")
}
