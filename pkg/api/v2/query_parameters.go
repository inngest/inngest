package apiv2

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"google.golang.org/protobuf/proto"
)

// grpc-gateway v2.28 stores the query parser in package-global state. Install
// one parser during package initialization, rather than mutating it whenever a
// handler is constructed.
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/runtime/mux.go#L139-L145
func init() {
	_ = runtime.NewServeMux(runtime.SetQueryParameterParser(v2QueryParameterParser{
		defaultParser: &runtime.DefaultQueryParser{},
	}))
}

type v2QueryParameterParser struct {
	defaultParser runtime.QueryParameterParser
}

func (p v2QueryParameterParser) Parse(msg proto.Message, values url.Values, filter *utilities.DoubleArray) error {
	// Scope the stricter behavior to API v2 messages. Other grpc-gateway
	// services in the process retain the default behavior.
	if msg.ProtoReflect().Descriptor().ParentFile().Package() != "api.v2" {
		return p.defaultParser.Parse(msg, values, filter)
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fieldPath, found := grpcGatewayQueryFieldPath(msg.ProtoReflect(), key)
		if filter.HasCommonPrefix(fieldPath) {
			continue
		}
		if !found {
			return unexpectedRequestFieldError{location: "query parameter", field: key}
		}
	}

	return p.defaultParser.Parse(msg, values, filter)
}

type unexpectedRequestFieldError struct {
	location string
	field    string
}

func (e unexpectedRequestFieldError) Error() string {
	response := apiv2base.ErrorResponse{Errors: []apiv2base.ErrorItem{{
		Code:    apiv2base.ErrorInvalidRequest,
		Message: fmt.Sprintf("Unexpected %s %q", e.location, e.field),
	}}}
	data, _ := json.Marshal(response)
	return string(data)
}
