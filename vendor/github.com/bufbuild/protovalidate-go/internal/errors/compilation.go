// Copyright 2023-2024 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package errors

import (
	"fmt"
)

// A CompilationError is returned if a CEL expression cannot be compiled &
// type-checked or if invalid standard constraints are applied.
type CompilationError struct {
	cause error
}

// NewCompilationError creates a new CompilationError with the provided cause.
func NewCompilationError(cause error) *CompilationError {
	return &CompilationError{cause: cause}
}

// NewCompilationErrorf creates a new CompilationError, constructing a causal error from
// the provided format and args.
func NewCompilationErrorf(format string, args ...any) *CompilationError {
	return NewCompilationError(fmt.Errorf(format, args...))
}

func (err *CompilationError) Error() string {
	return fmt.Sprintf("compilation error: %v", err.cause)
}

func (err *CompilationError) Unwrap() error { return err.cause }
