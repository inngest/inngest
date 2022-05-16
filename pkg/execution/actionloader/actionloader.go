package actionloader

import (
	"context"

	"github.com/inngest/inngestctl/inngest"
)

// ActionLoader loads and returns function definitions for the executor.  These let the executor
// know which drivers to use to run the functions.
//
// It does this because:
//
// A function definition (workflow) specifies which actions it should invoke (as steps).  However,
// it doesn't include the action definition (eg. the driver, docker image, path to the JS file,
// lambda definition, etc.).
type ActionLoader interface {
	// Load returns fully-defined action given a DSN and optional version constraint.  If the
	// versions within the constraint are nil we must return the latest version available.
	Load(ctx context.Context, dsn string, version *inngest.VersionConstraint) (*inngest.ActionVersion, error)
}
