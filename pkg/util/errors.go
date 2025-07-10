package util

import (
	"errors"
)

type ErrSet struct {
	errs []error
}

func NewErrSet() *ErrSet {
	return &ErrSet{}
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
