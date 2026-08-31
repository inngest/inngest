package apiv2oauth

import (
	"slices"
	"strings"

	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
)

// event and signing keys are not available through oauth
var cliExcludedResources = map[string]struct{}{
	"event_keys":   {},
	"signing_keys": {},
}

func CLIScopes() []string {
	return scopesForEndpoints(apiv2endpoint.Discover(), cliExcludedResources)
}

func scopesForEndpoints(endpoints []apiv2endpoint.Endpoint, excludedResources map[string]struct{}) []string {
	scopes := map[string]struct{}{}
	for _, endpoint := range endpoints {
		resource, access, ok := scopeParts(endpoint.AuthzPermission)
		if !ok {
			continue
		}
		if _, excluded := excludedResources[resource]; excluded {
			continue
		}
		scopes[resource+":"+access+":*"] = struct{}{}
	}

	result := make([]string, 0, len(scopes))
	for scope := range scopes {
		result = append(result, scope)
	}
	slices.Sort(result)
	return result
}

func scopeParts(permission string) (string, string, bool) {
	parts := strings.Split(permission, ":")
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return "", "", false
	}
	if parts[1] != "read" && parts[1] != "write" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
