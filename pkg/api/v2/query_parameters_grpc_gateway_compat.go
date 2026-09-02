package apiv2

import (
	"regexp"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// grpc-gateway v2.28 compatibility.
//
// DefaultQueryParser silently ignores unknown fields, and its map-key and field
// path normalization helpers are not exported. This code mirrors only that
// normalization so strict validation accepts the same query syntax:
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/runtime/query.go#L25-L64
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/runtime/query.go#L73-L103
// Unknown fields are deliberately ignored here:
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/runtime/query.go#L105-L126
//
// Keep value parsing delegated to DefaultQueryParser.

// grpcGatewayQueryMapKeyRegexp is an exact copy of grpc-gateway's unexported
// valuesKeyRegexp:
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/runtime/query.go#L25
var grpcGatewayQueryMapKeyRegexp = regexp.MustCompile(`^(.*)\[(.*)\]$`)

func grpcGatewayQueryFieldPath(msg protoreflect.Message, key string) ([]string, bool) {
	if match := grpcGatewayQueryMapKeyRegexp.FindStringSubmatch(key); len(match) == 3 {
		key = match[1]
	}
	fieldPath := strings.Split(key, ".")
	normalized := make([]string, 0, len(fieldPath))
	for i, fieldName := range fieldPath {
		fields := msg.Descriptor().Fields()
		field := fields.ByTextName(fieldName)
		if field == nil {
			field = fields.ByJSONName(fieldName)
		}
		if field == nil {
			return fieldPath, false
		}

		normalized = append(normalized, string(field.Name()))
		if i == len(fieldPath)-1 {
			return normalized, true
		}
		if field.Message() == nil || field.Cardinality() == protoreflect.Repeated {
			return fieldPath, false
		}
		msg = msg.Get(field).Message()
	}

	return normalized, true
}
