package util

import (
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsGRPCMessageSizeError distinguishes gRPC's send/receive size violations from
// ResourceExhausted errors caused by transient quota or capacity constraints.
func IsGRPCMessageSizeError(err error) bool {
	s, ok := status.FromError(err)
	if !ok || s.Code() != codes.ResourceExhausted {
		return false
	}
	msg := s.Message()
	return strings.Contains(msg, "received message larger than max") ||
		strings.Contains(msg, "received message after decompression larger than max") ||
		strings.Contains(msg, "trying to send message larger than max")
}

type ErrSet struct {
	errs []error
}

func NewErrSet(errs ...error) *ErrSet {
	return &ErrSet{errs: errs}
}

func (e *ErrSet) Add(err error) {
	if err != nil {
		e.errs = append(e.errs, err)
	}
}

func (e *ErrSet) HasErrors() bool {
	return len(e.errs) > 0
}

func (e *ErrSet) Err() error {
	if !e.HasErrors() {
		return nil
	}

	return errors.Join(e.errs...)
}

func (e *ErrSet) Merge(other *ErrSet) *ErrSet {
	if other == nil {
		return e
	}

	if e == nil {
		return other
	}

	new := NewErrSet(append(e.errs, other.errs...)...)

	return new
}
