package apiv2endpoint

import (
	"strings"

	openapiv2 "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func FieldDescription(field protoreflect.FieldDescriptor) string {
	description := string(field.JSONName())
	if schema := openAPISchema(field); schema != nil && schema.GetDescription() != "" {
		description = schema.GetDescription()
	}
	if values := FieldEnum(field); len(values) > 0 {
		description = strings.TrimSuffix(description, ".") + ". Accepted values: " + strings.Join(values, ", ") + "."
	}
	return description
}

func FieldEnum(field protoreflect.FieldDescriptor) []string {
	schema := openAPISchema(field)
	if schema == nil {
		return nil
	}
	return append([]string(nil), schema.GetEnum()...)
}

func openAPISchema(field protoreflect.FieldDescriptor) *openapiv2.JSONSchema {
	opts := field.Options()
	if !proto.HasExtension(opts, openapiv2.E_Openapiv2Field) {
		return nil
	}
	schema, _ := proto.GetExtension(opts, openapiv2.E_Openapiv2Field).(*openapiv2.JSONSchema)
	return schema
}

func IsRequired(field protoreflect.FieldDescriptor) bool {
	opts := field.Options()
	if !proto.HasExtension(opts, annotations.E_FieldBehavior) {
		return false
	}
	behaviors, ok := proto.GetExtension(opts, annotations.E_FieldBehavior).([]annotations.FieldBehavior)
	if !ok {
		return false
	}
	for _, behavior := range behaviors {
		if behavior == annotations.FieldBehavior_REQUIRED {
			return true
		}
	}
	return false
}

func IsDeprecated(field protoreflect.FieldDescriptor) bool {
	opts, ok := field.Options().(*descriptorpb.FieldOptions)
	return ok && opts.GetDeprecated()
}
