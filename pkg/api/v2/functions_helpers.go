package apiv2

// extractFunctionIdFromFullyQualifiedId extracts the function ID from a fully qualified function ID
// (e.g. "function-123" from "app-123-function-456")
func (s *Service) cleanFunctionId(functionID string, appID string) string {
	// If the functionID is in the format of "{appID}-{functionID}" (e.g., "app-123-function-456"),
	// remove the "{appID}-" prefix if present.
	cleaned := functionID
	prefix := appID + "-"
	if len(appID) > 0 && len(functionID) > len(prefix) && functionID[:len(prefix)] == prefix {
		cleaned = functionID[len(prefix):]
	}
	return cleaned
}
