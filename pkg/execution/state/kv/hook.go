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
	OpExtend Op = "extend"

	OpPush   Op = "push"
	OpPeek   Op = "peek"
	OpRemove Op = "remove"
	OpClear  Op = "clear"
)

// Result names how one kv call ended.
type Result string

const (
	ResultOK       Result = "ok"
	ResultNotFound Result = "not_found"
	ResultConflict Result = "conflict"
	// ResultFull means a Push found the list full and wrote nothing.
	ResultFull Result = "full"
	// ResultDuplicate means a Push found an item with the same ID in the
	// list and wrote nothing.
	ResultDuplicate Result = "duplicate"
	ResultError     Result = "error"
)

// Operation describes one completed kv call.
type Operation struct {
	Tenant    domain.Tenant
	Namespace string
	Op        Op
	Result    Result
	// Bytes is the value length read by a successful Get or Peek, or written
	// by a successful Set or by a Push that added its item.  it is zero for
	// every other call.
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
	case errors.Is(err, ErrFull):
		return ResultFull
	default:
		return ResultError
	}
}
