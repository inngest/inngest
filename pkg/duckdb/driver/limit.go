package driver

import (
	"context"

	"github.com/inngest/inngest/pkg/duckdb/driver/internal/result"
)

// ErrResultTooLarge is returned (under errors.Is) by a query run with
// WithMaxResultBytes whose result exceeded that limit. The subprocess is
// left healthy; only that statement fails.
var ErrResultTooLarge = result.ErrTooLarge

// WithMaxResultBytes returns a ctx under which any query this driver runs
// fails with ErrResultTooLarge once its result's encoded size exceeds n
// bytes, rather than buffering the whole result in memory first — every
// transport materializes a result in full before database/sql sees a row,
// so a caller-side check after the fact would be too late. The size counted
// is the transport's wire encoding, an approximation of the decoded rows'
// memory. n <= 0 means no limit.
func WithMaxResultBytes(ctx context.Context, n int64) context.Context {
	return result.WithMaxBytes(ctx, n)
}
