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
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
)

// A ValidationError is returned if one or more rule violations were
// detected.
type ValidationError struct {
	Violations []*Violation
}

func (err *ValidationError) Error() string {
	bldr := &strings.Builder{}
	bldr.WriteString("validation error:")
	for _, violation := range err.Violations {
		bldr.WriteString("\n - ")
		if fieldPath := FieldPathString(violation.Proto.GetField()); fieldPath != "" {
			bldr.WriteString(fieldPath)
			bldr.WriteString(": ")
		}
		_, _ = fmt.Fprintf(bldr, "%s [%s]",
			violation.Proto.GetMessage(),
			violation.Proto.GetRuleId())
	}
	return bldr.String()
}

// ToProto converts this error into its proto.Message form.
func (err *ValidationError) ToProto() *validate.Violations {
	violations := &validate.Violations{
		Violations: make([]*validate.Violation, len(err.Violations)),
	}
	for i, violation := range err.Violations {
		violations.Violations[i] = violation.Proto
	}
	return violations
}
