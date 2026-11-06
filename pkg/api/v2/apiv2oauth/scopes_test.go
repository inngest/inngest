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
		{AuthzPermission: "api_keys:write:create"},
		{AuthzPermission: "partners:read:list_accounts"},
		{AuthzPermission: ""},
	}

	require.Equal(t, []string{
		"apps:read:*",
		"functions:write:*",
	}, scopesForEndpoints(endpoints, cliExcludedResources))
}

func TestCLIScopesExcludeNonDelegatableKeys(t *testing.T) {
	scopes := CLIScopes()
	require.Contains(t, scopes, "events:read:*")

	require.Contains(t, scopes, "apps:read:*")
	require.Contains(t, scopes, "apps:write:*")
	require.NotContains(t, scopes, "event_keys:read:*")
	require.NotContains(t, scopes, "signing_keys:read:*")
}

func TestAppSyncScopes(t *testing.T) {
	for _, method := range []string{"ListAppSyncs", "GetAppSync"} {
		t.Run(method, func(t *testing.T) {
			var endpoints []apiv2endpoint.Endpoint
			for _, endpoint := range apiv2endpoint.Discover() {
				if endpoint.MethodName == method {
					endpoints = append(endpoints, endpoint)
				}
			}
			require.Equal(t, []string{"apps:read:*"}, scopesForEndpoints(endpoints, cliExcludedResources))
		})
	}
}
