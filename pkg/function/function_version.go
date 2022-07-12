package function

import (
	"context"
	"time"

	"github.com/inngest/inngest-cli/inngest"
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
	Function   Function
	ValidFrom  time.Time
	ValidTo    time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// ActionVersions provide the action configuration for each step of the function with the exact version
// of a given Action used
func (fv *FunctionVersion) Actions(ctx context.Context) ([]inngest.ActionVersion, []inngest.Edge, error) {
	avs, edges, err := fv.Function.Actions(ctx)
	if err != nil {
		return nil, nil, err
	}
	// ActionVersions for a given function are currently defaulted to use the major version of 1 and the
	// minor version of the function version. This will be changed in the future to enable sharing of
	// action versions across functions
	for i := range avs {
		av := &avs[i]
		av.Version = &inngest.VersionInfo{
			Major: 1,
			Minor: fv.Version,
		}
	}
	return avs, edges, nil
}
