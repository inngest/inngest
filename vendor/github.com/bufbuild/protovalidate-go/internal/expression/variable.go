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

package expression

import (
	"sync"

	"github.com/google/cel-go/interpreter"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Variable implements interpreter.Activation, providing a lightweight named
// variable to cel.Program executions.
type Variable struct {
	// Next is the parent activation
	Next interpreter.Activation
	// Name is the variable's name
	Name string
	// Val is the value for this variable
	Val any
}

func (v *Variable) ResolveName(name string) (any, bool) {
	switch {
	case name == v.Name:
		return v.Val, true
	case v.Next != nil:
		return v.Next.ResolveName(name)
	default:
		return nil, false
	}
}

func (v *Variable) Parent() interpreter.Activation { return nil }

type VariablePool sync.Pool

func (p *VariablePool) Put(v *Variable) {
	(*sync.Pool)(p).Put(v)
}

func (p *VariablePool) Get() *Variable {
	v := (*sync.Pool)(p).Get().(*Variable) //nolint:errcheck,forcetypeassert
	v.Next = nil
	return v
}

// Now implements interpreter.Activation, providing a lazily produced timestamp
// for accessing the variable `now` that's constant within an evaluation.
type Now struct {
	// TS is the already resolved timestamp. If unset, the field is populated with
	// timestamppb.Now.
	TS *timestamppb.Timestamp
}

func (n *Now) ResolveName(name string) (any, bool) {
	if name != "now" {
		return nil, false
	} else if n.TS == nil {
		n.TS = timestamppb.Now()
	}
	return n.TS, true
}

func (n *Now) Parent() interpreter.Activation { return nil }

type NowPool sync.Pool

func (p *NowPool) Put(v *Now) {
	(*sync.Pool)(p).Put(v)
}

func (p *NowPool) Get() *Now {
	n := (*sync.Pool)(p).Get().(*Now) //nolint:errcheck,forcetypeassert
	n.TS = nil
	return n
}
