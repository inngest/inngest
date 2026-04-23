package manager

import (
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/db"
)

// New is the permanent constructor name for the cqrs.Manager implementation.
// Phase 1 only establishes the package boundary; runtime call sites move in
// phase 3.
func New(adapter db.Adapter) cqrs.Manager {
	_ = adapter
	return nil
}
