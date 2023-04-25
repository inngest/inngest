package function

import (
	"time"

	"github.com/inngest/inngest/pkg/inngest"
)

// FunctionVersion represents a given version of a function stored or used by the Inngest system
//
// By default, a FunctionVersion is a draft and will have both valid from and to fields unset.
// When a given FunctionVersion is live (after a deploy), the valid from timestamp is set.
// When a given FunctionVersion is no longer live (after a new version has been deployed),
// the valid to timestamp will be set, recording the entire time window which the version was live.
type FunctionVersion struct {
	FunctionID string
	Version    uint

	// Function config is loaded as a config string then parsed into a Function struct
	Config   string
	Function inngest.Function

	ValidFrom *time.Time
	ValidTo   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
