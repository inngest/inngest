package apiv2endpoint

import (
	openapiv2 "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func FieldDescription(field protoreflect.FieldDescriptor) string {
	if schema := openAPISchema(field); schema != nil && schema.GetDescription() != "" {
		return schema.GetDescription()
	}
	return string(field.JSONName())
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
