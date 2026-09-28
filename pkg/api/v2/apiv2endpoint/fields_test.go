package apiv2endpoint

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestIsDeprecated(t *testing.T) {
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    new("deprecation_test.proto"),
		Package: new("test"),
		Syntax:  new("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: new("Request"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:    new("active"),
					Number:  proto.Int32(1),
					Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:    descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
					Options: &descriptorpb.FieldOptions{},
				},
				{
					Name:    new("legacy"),
					Number:  proto.Int32(2),
					Label:   descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:    descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
					Options: &descriptorpb.FieldOptions{Deprecated: new(true)},
				},
			},
		}},
	}, nil)
	require.NoError(t, err)

	fields := file.Messages().ByName("Request").Fields()
	require.False(t, IsDeprecated(fields.ByName("active")))
	require.True(t, IsDeprecated(fields.ByName("legacy")))
}
