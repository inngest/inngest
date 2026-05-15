package event_trigger_patterns

import "strings"

// GenerateMatchingPatterns returns all matching trigger patterns for the given event name
// including wildcards that follow Inngest's event trigger pattern rules.
//
// Inngest wildcard patterns work as follows:
// - Wildcards can only be used after "/" or "." characters
// - Wildcards match everything after the delimiter in that segment
// - Cannot be used mid-word or in the middle of a pattern
//
// Examples:
//   - "app/user.created" -> ["app/user.created", "app/*", "app/user.*"]
//   - "api/v1/users" -> ["api/v1/users", "api/*", "api/v1/*"]
//   - "user.updated" -> ["user.updated", "user.*"]
//   - "simple" -> ["simple"]
//
// This function generates all possible wildcard patterns that would match
// the given event name, which is used for efficient trigger matching.
func GenerateMatchingPatterns(eventName string) []string {
	patterns := []string{eventName}

	// Handle slash-separated paths (e.g., "api/v1/users" -> ["api/*", "api/v1/*"])
	parts := strings.Split(eventName, "/")
	if len(parts) > 1 {
		for n := range parts[0 : len(parts)-1] {
			prefix := strings.Join(parts[0:n+1], "/")
			patterns = append(patterns, prefix+"/*")
		}
	}

	// Handle dot-separated paths (e.g., "app/user.created" -> ["app/user.*"])
	parts = strings.Split(eventName, ".")
	if len(parts) > 1 {
		for n := range parts[0 : len(parts)-1] {
			prefix := strings.Join(parts[0:n+1], ".")
			patterns = append(patterns, prefix+".*")
		}
	}

	return patterns
}