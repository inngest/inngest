package apiv2

import "strings"

// PublicFunctionID keeps function lookup and response IDs on the same rule.
func PublicFunctionID(appID, storedFunctionID, configuredFunctionID string) string {
	if configuredFunctionID != "" && configuredFunctionID != storedFunctionID {
		return configuredFunctionID
	}

	functionID := configuredFunctionID
	if functionID == "" {
		functionID = storedFunctionID
	}
	if appID != "" {
		return strings.TrimPrefix(functionID, appID+"-")
	}
	return functionID
}
