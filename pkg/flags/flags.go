// This package is intended to help ensure a consistent, clean way to
// access feature flags safely
package flags

import (
	"context"

	"github.com/google/uuid"
)

func NewBoolFlag(fn func(ctx context.Context, acctID uuid.UUID) bool) BoolFlag {
	return BoolFlag{fn: fn}
}

type BoolFlag struct {
	fn func(ctx context.Context, acctID uuid.UUID) bool
}

func (b BoolFlag) Enabled(ctx context.Context, acctID uuid.UUID) bool {
	if b.fn == nil {
		return false
	}
	return b.fn(ctx, acctID)
}
