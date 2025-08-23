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
	"errors"

	pverr "github.com/bufbuild/protovalidate-go/internal/errors"
	"github.com/bufbuild/protovalidate-go/internal/expression"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// celPrograms is an evaluator that executes an expression.ProgramSet.
type celPrograms struct {
	base
	expression.ProgramSet
}

func (c celPrograms) Evaluate(val protoreflect.Value, failFast bool) error {
	err := c.ProgramSet.Eval(val, failFast)
	if err != nil {
		var valErr *pverr.ValidationError
		if errors.As(err, &valErr) {
			for _, violation := range valErr.Violations {
				violation.Proto.Field = c.base.fieldPath()
				violation.Proto.Rule = c.base.rulePath(violation.Proto.GetRule())
				violation.FieldValue = val
				violation.FieldDescriptor = c.base.Descriptor
			}
		}
	}
	return err
}

func (c celPrograms) EvaluateMessage(msg protoreflect.Message, failFast bool) error {
	return c.ProgramSet.Eval(protoreflect.ValueOfMessage(msg), failFast)
}

func (c celPrograms) Tautology() bool {
	return len(c.ProgramSet) == 0
}

var (
	_ evaluator        = celPrograms{}
	_ MessageEvaluator = celPrograms{}
)
