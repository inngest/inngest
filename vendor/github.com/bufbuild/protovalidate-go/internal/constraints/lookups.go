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

package constraints

import (
	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	// fieldConstraintsDesc provides a Descriptor for validate.FieldConstraints.
	fieldConstraintsDesc = (*validate.FieldConstraints)(nil).ProtoReflect().Descriptor()

	// fieldConstraintsOneofDesc provides the OneofDescriptor for the type union
	// in FieldConstraints.
	fieldConstraintsOneofDesc = fieldConstraintsDesc.Oneofs().ByName("type")

	// mapFieldConstraintsDesc provides the FieldDescriptor for the map standard
	// constraints.
	mapFieldConstraintsDesc = fieldConstraintsDesc.Fields().ByName("map")

	// repeatedFieldConstraintsDesc provides the FieldDescriptor for the repeated
	// standard constraints.
	repeatedFieldConstraintsDesc = fieldConstraintsDesc.Fields().ByName("repeated")
)

// expectedStandardConstraints maps protocol buffer field kinds to their
// expected field constraints.
var expectedStandardConstraints = map[protoreflect.Kind]protoreflect.FieldDescriptor{
	protoreflect.FloatKind:    fieldConstraintsDesc.Fields().ByName("float"),
	protoreflect.DoubleKind:   fieldConstraintsDesc.Fields().ByName("double"),
	protoreflect.Int32Kind:    fieldConstraintsDesc.Fields().ByName("int32"),
	protoreflect.Int64Kind:    fieldConstraintsDesc.Fields().ByName("int64"),
	protoreflect.Uint32Kind:   fieldConstraintsDesc.Fields().ByName("uint32"),
	protoreflect.Uint64Kind:   fieldConstraintsDesc.Fields().ByName("uint64"),
	protoreflect.Sint32Kind:   fieldConstraintsDesc.Fields().ByName("sint32"),
	protoreflect.Sint64Kind:   fieldConstraintsDesc.Fields().ByName("sint64"),
	protoreflect.Fixed32Kind:  fieldConstraintsDesc.Fields().ByName("fixed32"),
	protoreflect.Fixed64Kind:  fieldConstraintsDesc.Fields().ByName("fixed64"),
	protoreflect.Sfixed32Kind: fieldConstraintsDesc.Fields().ByName("sfixed32"),
	protoreflect.Sfixed64Kind: fieldConstraintsDesc.Fields().ByName("sfixed64"),
	protoreflect.BoolKind:     fieldConstraintsDesc.Fields().ByName("bool"),
	protoreflect.StringKind:   fieldConstraintsDesc.Fields().ByName("string"),
	protoreflect.BytesKind:    fieldConstraintsDesc.Fields().ByName("bytes"),
	protoreflect.EnumKind:     fieldConstraintsDesc.Fields().ByName("enum"),
}

var expectedWKTConstraints = map[protoreflect.FullName]protoreflect.FieldDescriptor{
	"google.protobuf.Any":       fieldConstraintsDesc.Fields().ByName("any"),
	"google.protobuf.Duration":  fieldConstraintsDesc.Fields().ByName("duration"),
	"google.protobuf.Timestamp": fieldConstraintsDesc.Fields().ByName("timestamp"),
}

// ExpectedWrapperConstraints returns the validate.FieldConstraints field that
// is expected for the given wrapper well-known type's full name. If ok is
// false, no standard constraints exist for that type.
func ExpectedWrapperConstraints(fqn protoreflect.FullName) (desc protoreflect.FieldDescriptor, ok bool) {
	switch fqn {
	case "google.protobuf.BoolValue":
		return expectedStandardConstraints[protoreflect.BoolKind], true
	case "google.protobuf.BytesValue":
		return expectedStandardConstraints[protoreflect.BytesKind], true
	case "google.protobuf.DoubleValue":
		return expectedStandardConstraints[protoreflect.DoubleKind], true
	case "google.protobuf.FloatValue":
		return expectedStandardConstraints[protoreflect.FloatKind], true
	case "google.protobuf.Int32Value":
		return expectedStandardConstraints[protoreflect.Int32Kind], true
	case "google.protobuf.Int64Value":
		return expectedStandardConstraints[protoreflect.Int64Kind], true
	case "google.protobuf.StringValue":
		return expectedStandardConstraints[protoreflect.StringKind], true
	case "google.protobuf.UInt32Value":
		return expectedStandardConstraints[protoreflect.Uint32Kind], true
	case "google.protobuf.UInt64Value":
		return expectedStandardConstraints[protoreflect.Uint64Kind], true
	default:
		return nil, false
	}
}
