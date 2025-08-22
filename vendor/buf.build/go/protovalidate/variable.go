// Copyright 2023-2025 Buf Technologies, Inc.
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

package protovalidate

import (
	"sync"

	"github.com/google/cel-go/interpreter"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// variable implements interpreter.Activation, providing a lightweight named
// variable to cel.Program executions.
type variable struct {
	// Next is the parent activation
	Next interpreter.Activation
	// Name is the variable's name
	Name string
	// Val is the value for this variable
	Val any
}

func (v *variable) ResolveName(name string) (any, bool) {
	switch {
	case name == v.Name:
		return v.Val, true
	case v.Next != nil:
		return v.Next.ResolveName(name)
	default:
		return nil, false
	}
}

func (v *variable) Parent() interpreter.Activation { return nil }

type variablePool sync.Pool

func (p *variablePool) Put(v *variable) {
	(*sync.Pool)(p).Put(v)
}

func (p *variablePool) Get() *variable {
	v := (*sync.Pool)(p).Get().(*variable) //nolint:errcheck,forcetypeassert
	v.Next = nil
	return v
}

// now implements interpreter.Activation, providing a lazily produced timestamp
// for accessing the variable `now` that's constant within an evaluation.
type now struct {
	// TS is the already resolved timestamp. If unset, the field is populated with
	// the output of nowFn.
	TS    *timestamppb.Timestamp
	nowFn func() *timestamppb.Timestamp
}

func (n *now) ResolveName(name string) (any, bool) {
	if name != "now" {
		return nil, false
	} else if n.TS == nil {
		n.TS = n.nowFn()
	}
	return n.TS, true
}

func (n *now) Parent() interpreter.Activation { return nil }

type nowPool sync.Pool

func (p *nowPool) Put(v *now) {
	(*sync.Pool)(p).Put(v)
}

func (p *nowPool) Get(nowFn func() *timestamppb.Timestamp) *now {
	n := (*sync.Pool)(p).Get().(*now) //nolint:errcheck,forcetypeassert
	n.nowFn = nowFn
	n.TS = nil
	return n
}
