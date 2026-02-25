// Copyright 2024 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package crstrings

import (
	"fmt"
	"slices"
	"strings"
)

// JoinStringers concatenates the string representations of the given
// fmt.Stringer implementations.
func JoinStringers[T fmt.Stringer](delim string, args ...T) string {
	switch len(args) {
	case 0:
		return ""
	case 1:
		return args[0].String()
	}
	elems := make([]string, len(args))
	for i := range args {
		elems[i] = args[i].String()
	}
	return strings.Join(elems, delim)
}

// MapAndJoin converts each argument to a string using the given function and
// joins the strings with the given delimiter.
func MapAndJoin[T any](fn func(T) string, delim string, args ...T) string {
	switch len(args) {
	case 0:
		return ""
	case 1:
		return fn(args[0])
	}
	elems := make([]string, len(args))
	for i := range args {
		elems[i] = fn(args[i])
	}
	return strings.Join(elems, delim)
}

// If returns the given value if the flag is true, otherwise an empty string.
func If(flag bool, trueValue string) string {
	return IfElse(flag, trueValue, "")
}

// IfElse returns the value that matches the value of the flag.
func IfElse(flag bool, trueValue, falseValue string) string {
	if flag {
		return trueValue
	}
	return falseValue
}

// WithSep prints the strings a and b with the given separator in-between,
// unless one of the strings is empty (in which case the other string is
// returned).
func WithSep(a string, separator string, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return strings.Join([]string{a, b}, separator)
}

// FilterEmpty removes empty strings from the given slice.
func FilterEmpty(elems []string) []string {
	return slices.DeleteFunc(elems, func(s string) bool {
		return s == ""
	})
}

// Lines breaks up the given string into lines.
func Lines(s string) []string {
	// Remove any trailing newline (to avoid getting an extraneous empty line at
	// the end).
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		// In this case, Split returns a slice with a single empty string (which is
		// not what we want).
		return nil
	}
	return strings.Split(s, "\n")
}
