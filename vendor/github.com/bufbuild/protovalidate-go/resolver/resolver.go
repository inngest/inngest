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

package resolver

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protovalidate-go/internal/extensions"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// DefaultResolver resolves protovalidate constraints options from descriptors.
type DefaultResolver struct{}

// ResolveMessageConstraints returns the MessageConstraints option set for the
// MessageDescriptor.
func (r DefaultResolver) ResolveMessageConstraints(desc protoreflect.MessageDescriptor) *validate.MessageConstraints {
	return extensions.Resolve[*validate.MessageConstraints](desc.Options(), validate.E_Message)
}

// ResolveOneofConstraints returns the OneofConstraints option set for the
// OneofDescriptor.
func (r DefaultResolver) ResolveOneofConstraints(desc protoreflect.OneofDescriptor) *validate.OneofConstraints {
	return extensions.Resolve[*validate.OneofConstraints](desc.Options(), validate.E_Oneof)
}

// ResolveFieldConstraints returns the FieldConstraints option set for the
// FieldDescriptor.
func (r DefaultResolver) ResolveFieldConstraints(desc protoreflect.FieldDescriptor) *validate.FieldConstraints {
	return extensions.Resolve[*validate.FieldConstraints](desc.Options(), validate.E_Field)
}

// ResolvePredefinedConstraints returns the PredefinedConstraints option set for
// the FieldDescriptor. Note that this value is only meaningful if it is set on
// a field or extension of a field rule message. This method is provided for
// convenience.
func (r DefaultResolver) ResolvePredefinedConstraints(desc protoreflect.FieldDescriptor) *validate.PredefinedConstraints {
	return extensions.Resolve[*validate.PredefinedConstraints](desc.Options(), validate.E_Predefined)
}
