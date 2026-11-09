package kv

import (
	"context"
	"errors"

	"github.com/inngest/inngest/pkg/domain"
)

// Op names one kv call.
type Op string

const (
	OpGet    Op = "get"
	OpSet    Op = "set"
	OpDelete Op = "delete"
)

// Result names how one kv call ended.
type Result string

const (
	ResultOK       Result = "ok"
	ResultNotFound Result = "not_found"
	ResultConflict Result = "conflict"
	ResultError    Result = "error"
)

// Operation describes one completed kv call.
type Operation struct {
	Tenant    domain.Tenant
	Namespace string
	Op        Op
	Result    Result
	// Bytes is the value length read by a successful Get or written by a
	// successful Set.  it is zero for every other call.
	Bytes int64
}

// Hook receives every completed kv call, including calls that ended in not
// found, conflict, or error.  the store calls it on the caller's goroutine
// after the backend returns, so a slow hook slows every kv call.
type Hook interface {
	OnOperation(ctx context.Context, op Operation)
}

// HookFunc adapts a function to Hook.
type HookFunc func(ctx context.Context, op Operation)

func (f HookFunc) OnOperation(ctx context.Context, op Operation) {
	f(ctx, op)
}

// resultOf maps an error from the inner store to a Result.
func resultOf(err error) Result {
	switch {
	case err == nil:
		return ResultOK
	case errors.Is(err, ErrNotFound):
		return ResultNotFound
	case errors.Is(err, ErrConflict):
		return ResultConflict
	default:
		return ResultError
	}
}
