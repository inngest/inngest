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

// oneof performs validation on a oneof union.
type oneof struct {
	// Descriptor is the OneofDescriptor targeted by this evaluator
	Descriptor protoreflect.OneofDescriptor
	// Required indicates that a member of the oneof must be set
	Required bool
}

func (o oneof) Evaluate(val protoreflect.Value, failFast bool) error {
	return o.EvaluateMessage(val.Message(), failFast)
}

func (o oneof) EvaluateMessage(msg protoreflect.Message, _ bool) error {
	if o.Required && msg.WhichOneof(o.Descriptor) == nil {
		return &errors.ValidationError{Violations: []*errors.Violation{{
			Proto: &validate.Violation{
				Field: &validate.FieldPath{
					Elements: []*validate.FieldPathElement{{
						FieldName: proto.String(string(o.Descriptor.Name())),
					}},
				},
				ConstraintId: proto.String("required"),
				Message:      proto.String("exactly one field is required in oneof"),
			},
		}}}
	}
	return nil
}

func (o oneof) Tautology() bool {
	return !o.Required
}

var _ MessageEvaluator = oneof{}
