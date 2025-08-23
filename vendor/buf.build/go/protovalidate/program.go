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
	"fmt"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

//nolint:gochecknoglobals // amortized, eliminates allocations for all CEL programs
var globalVarPool = &variablePool{New: func() any { return &variable{} }}

//nolint:gochecknoglobals // amortized, eliminates allocations for all CEL programs
var globalNowPool = &nowPool{New: func() any { return &now{} }}

// programSet is a list of compiledProgram expressions that are evaluated
// together with the same input value. All expressions in a programSet may refer
// to a `this` variable.
type programSet []compiledProgram

// Eval applies the contained expressions to the provided `this` val, returning
// either *errors.ValidationError if the input is invalid or errors.RuntimeError
// if there is a type or range error. If failFast is true, execution stops at
// the first failed expression.
func (s programSet) Eval(val protoreflect.Value, cfg *validationConfig) error {
	binding := s.bindThis(val.Interface())
	defer globalVarPool.Put(binding)

	var violations []*Violation
	for _, expr := range s {
		violation, err := expr.eval(binding, cfg)
		if err != nil {
			return err
		}
		if violation != nil {
			violations = append(violations, violation)
			if cfg.failFast {
				break
			}
		}
	}

	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}

	return nil
}

func (s programSet) bindThis(val any) *variable {
	binding := globalVarPool.Get()
	binding.Name = "this"

	switch value := val.(type) {
	case protoreflect.Message:
		binding.Val = value.Interface()
	case protoreflect.Map:
		// TODO: expensive to create this copy, but getting this into a ref.Val with
		//  traits.Mapper is not terribly feasible from this type.
		bindingVal := make(map[any]any, value.Len())
		value.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			// Cel operates on 64-bit integers, so if our map type is 32-bit, we
			// need to widen to a 64-bit type in the binding due to our usage of
			// a map[any]any.
			switch key.Interface().(type) {
			case int32:
				bindingVal[key.Int()] = value.Interface()
			case uint32:
				bindingVal[key.Uint()] = value.Interface()
			default:
				bindingVal[key.Interface()] = value.Interface()
			}
			return true
		})
		binding.Val = bindingVal
	default:
		binding.Val = value
	}

	return binding
}

// compiledProgram is a parsed and type-checked cel.Program along with the
// source Expression.
type compiledProgram struct {
	Program    cel.Program
	Source     *validate.Rule
	Path       []*validate.FieldPathElement
	Value      protoreflect.Value
	Descriptor protoreflect.FieldDescriptor
}

//nolint:nilnil // non-existence of violations is intentional
func (expr compiledProgram) eval(bindings *variable, cfg *validationConfig) (*Violation, error) {
	now := globalNowPool.Get(cfg.nowFn)
	defer globalNowPool.Put(now)
	bindings.Next = now

	value, _, err := expr.Program.Eval(bindings)
	if err != nil {
		return nil, &RuntimeError{cause: fmt.Errorf(
			"error evaluating %s: %w", expr.Source.GetId(), err)}
	}
	switch val := value.Value().(type) {
	case string:
		if val == "" {
			return nil, nil
		}
		return &Violation{
			Proto: &validate.Violation{
				Rule:    expr.rulePath(),
				RuleId:  proto.String(expr.Source.GetId()),
				Message: proto.String(val),
			},
			RuleValue:      expr.Value,
			RuleDescriptor: expr.Descriptor,
		}, nil
	case bool:
		if val {
			return nil, nil
		}
		return &Violation{
			Proto: &validate.Violation{
				Rule:    expr.rulePath(),
				RuleId:  proto.String(expr.Source.GetId()),
				Message: proto.String(expr.Source.GetMessage()),
			},
			RuleValue:      expr.Value,
			RuleDescriptor: expr.Descriptor,
		}, nil
	default:
		return nil, &RuntimeError{cause: fmt.Errorf(
			"resolved to an unexpected type %T", val)}
	}
}

func (expr compiledProgram) rulePath() *validate.FieldPath {
	if len(expr.Path) > 0 {
		return &validate.FieldPath{Elements: expr.Path}
	}
	return nil
}
