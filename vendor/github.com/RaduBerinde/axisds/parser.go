// Copyright 2025 Radu Berinde.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package axisds

import (
	"fmt"
	"regexp"
	"runtime/debug"
)

// Parser is an interface for parsing intervals.
type Parser[B Boundary] interface {
	// ParseBoundary is used to parse a "bare" boundary. Used for Endpoint[B].
	ParseBoundary(str string) (b B, err error)

	// ParseInterval parses an interval of the form `boundary1, boundary2`
	// from the input and returns any remaining fields in the string.
	ParseInterval(input string) (start, end B, remaining string, err error)
}

// MakeBasicParser creates a Parser[B] that uses Sscanf with `%v` for the
// boundaries.
func MakeBasicParser[B Boundary]() Parser[B] {
	return basicParser[B]{}
}

// MakeEndpointParser creates a Parser[Endpoint[B]].
func MakeEndpointParser[B Boundary](p Parser[B]) Parser[Endpoint[B]] {
	return &endpointParser[B]{p: p}
}

// MustParseInterval parses a string into an interval; panics on errors.
func MustParseInterval[B Boundary](p Parser[B], input string) (start, end B) {
	start, end, rem := MustParseIntervalPrefix(p, input)
	if rem != "" {
		panic(fmt.Sprintf("extra fields in input: %q", rem))
	}
	return start, end
}

// MustParseIntervalPrefix parses a string into an interval and an optional
// remainder string, panics on errors.
func MustParseIntervalPrefix[B Boundary](
	p Parser[B], input string,
) (start, end B, remaining string) {
	start, end, remaining, err := p.ParseInterval(input)
	if err != nil {
		panic(err)
	}
	return start, end, remaining
}

type basicParser[B Boundary] struct{}

var _ Parser[int] = basicParser[int]{}

func (p basicParser[B]) ParseBoundary(str string) (b B, err error) {
	_, err = fmt.Sscanf(str, "%v", &b)
	if err != nil {
		return b, fmt.Errorf("malformed boundary %q: %v\n%s", str, err, string(debug.Stack()))
	}
	return b, nil
}

func (p basicParser[B]) ParseInterval(input string) (start, end B, remaining string, err error) {
	re := regexp.MustCompile(`^\[([^,]+), ([^)]+)\) *(.*)$`)
	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return start, end, "", fmt.Errorf("malformed interval %q", input)
	}
	start, err = p.ParseBoundary(matches[1])
	if err == nil {
		end, err = p.ParseBoundary(matches[2])
	}
	if err != nil {
		return start, end, "", err
	}
	return start, end, matches[3], nil
}

type endpointParser[B Boundary] struct {
	p Parser[B]
}

func (p endpointParser[B]) ParseBoundary(str string) (e Endpoint[B], err error) {
	return e, fmt.Errorf("not implemented")
}

func (p endpointParser[B]) ParseInterval(
	input string,
) (start, end Endpoint[B], remaining string, err error) {
	re := regexp.MustCompile(`^([(\[])([^,]+), ([^)]+)([)\]]) *(.*)$`)
	matches := re.FindStringSubmatch(input)
	if matches == nil {
		return start, end, "", fmt.Errorf("malformed interval %q", input)
	}
	var b1, b2 B
	b1, err = p.p.ParseBoundary(matches[2])
	if err == nil {
		b2, err = p.p.ParseBoundary(matches[3])
	}
	if err != nil {
		return start, end, "", err
	}
	typ1 := InclusiveIf(matches[1] == "[")
	typ2 := InclusiveIf(matches[4] == "]")
	return MakeStartEndpoint(b1, typ1), MakeEndEndpoint(b2, typ2), matches[5], nil
}
