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

package evaluator

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protovalidate-go/internal/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

//nolint:gochecknoglobals
var (
	requiredRuleDescriptor = (&validate.FieldConstraints{}).ProtoReflect().Descriptor().Fields().ByName("required")
	requiredRulePath       = &validate.FieldPath{
		Elements: []*validate.FieldPathElement{
			errors.FieldPathElement(requiredRuleDescriptor),
		},
	}
)

// field performs validation on a single message field, defined by its
// descriptor.
type field struct {
	// Value is the evaluator to apply to the field's value
	Value value
	// Required indicates that the field must have a set value.
	Required bool
	// IgnoreEmpty indicates if a field should skip validation on its zero value.
	// This field is generally true for nullable fields or fields with the
	// ignore_empty constraint explicitly set.
	IgnoreEmpty bool
	// IgnoreDefault indicates if a field should skip validation on its zero value,
	// including for fields which have field presence and are set to the zero value.
	IgnoreDefault bool
	// Zero is the default or zero-value for this value's type
	Zero protoreflect.Value
}

func (f field) Evaluate(val protoreflect.Value, failFast bool) error {
	return f.EvaluateMessage(val.Message(), failFast)
}

func (f field) EvaluateMessage(msg protoreflect.Message, failFast bool) (err error) {
	if f.Required && !msg.Has(f.Value.Descriptor) {
		return &errors.ValidationError{Violations: []*errors.Violation{{
			Proto: &validate.Violation{
				Field:        errors.FieldPath(f.Value.Descriptor),
				Rule:         prefixRulePath(f.Value.NestedRule, requiredRulePath),
				ConstraintId: proto.String("required"),
				Message:      proto.String("value is required"),
			},
			FieldValue:      protoreflect.Value{},
			FieldDescriptor: f.Value.Descriptor,
			RuleValue:       protoreflect.ValueOfBool(true),
			RuleDescriptor:  requiredRuleDescriptor,
		}}}
	}

	if f.IgnoreEmpty && !msg.Has(f.Value.Descriptor) {
		return nil
	}

	val := msg.Get(f.Value.Descriptor)
	if f.IgnoreDefault && val.Equal(f.Zero) {
		return nil
	}
	return f.Value.Evaluate(val, failFast)
}

func (f field) Tautology() bool {
	return !f.Required && f.Value.Tautology()
}

var _ MessageEvaluator = field{}
