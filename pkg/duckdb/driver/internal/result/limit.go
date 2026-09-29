package result

import (
	"context"
	"errors"
	"fmt"
)

// ErrTooLarge marks a result abandoned because it exceeded the byte budget
// its caller attached with WithMaxBytes. It is always returned wrapped
// together with ErrStatementFailed: the subprocess is healthy, so the
// driver must neither restart nor retry.
var ErrTooLarge = errors.New("duckdb: result exceeds the size limit")

type maxBytesKey struct{}

// WithMaxBytes returns a ctx under which a transport abandons any result
// whose encoded size exceeds n bytes instead of buffering it in full. The
// size is the transport's own wire encoding (quack's response bodies,
// jsonlines' output lines), so it approximates, rather than equals, the
// decoded rows' memory. n <= 0 means no limit.
func WithMaxBytes(ctx context.Context, n int64) context.Context {
	return context.WithValue(ctx, maxBytesKey{}, n)
}

// MaxBytes returns ctx's WithMaxBytes budget, or 0 (no limit).
func MaxBytes(ctx context.Context) int64 {
	n, _ := ctx.Value(maxBytesKey{}).(int64)
	return n
}

// TooLargeError is the error a transport returns once a result exceeds max.
func TooLargeError(max int64) error {
	return fmt.Errorf("%w: %w (limit %d bytes)", ErrStatementFailed, ErrTooLarge, max)
}
