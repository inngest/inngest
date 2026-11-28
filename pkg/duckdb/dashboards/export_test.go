package dashboards

import (
	"testing"
	"time"
)

// SetGetTimeout shrinks Get's deadline for the rest of t.
func SetGetTimeout(t *testing.T, d time.Duration) {
	orig := getTimeout
	getTimeout = d
	t.Cleanup(func() { getTimeout = orig })
}
