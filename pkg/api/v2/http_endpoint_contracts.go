package apiv2

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var pathParameterRegexp = regexp.MustCompile(`\{[^}]+}`)
var endpointContracts = discoverEndpointContracts()

type endpointContract struct {
	method       string
	pattern      *regexp.Regexp
	acceptsQuery bool
	acceptsBody  bool
}

// grpc-gateway's generated endpoint handlers only call PopulateQueryParameters
// when the request protobuf has fields not bound to the path or body. They only
// decode req.Body when the google.api.http rule declares a body mapping.
// Consequently, unexpected query parameters on queryless endpoints and bodies
// on bodyless endpoints are silently ignored.
//
// This middleware has access to the raw HTTP request, but runs before
// grpc-gateway matches it to a protobuf binding. grpc-gateway does not expose
// that matched binding's request descriptor, path parameters, or body mapping,
// so endpointContracts unfortunately has to reconstruct that metadata from the
// protobuf HTTP rules and match the route again before dispatching to the
// generated handler.
//
// Generated body decoding:
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/protoc-gen-grpc-gateway/internal/gengateway/template.go#L389-L420
// Generated query parsing:
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/protoc-gen-grpc-gateway/internal/gengateway/template.go#L518-L525
func enforceEndpointContracts(base *apiv2base.Base) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contract := matchEndpointContract(r.Method, r.URL.Path)
			if contract == nil {
				next.ServeHTTP(w, r)
				return
			}
			if r.Method != http.MethodGet && !contract.acceptsBody && hasRequestBody(r) {
				base.WriteHTTPError(w, http.StatusBadRequest, apiv2base.ErrorInvalidRequest,
					"Request body is not supported for this endpoint")
				return
			}

			query := r.URL.Query()
			if !contract.acceptsQuery && len(query) > 0 {
				parameters := make([]string, 0, len(query))
				for parameter := range query {
					parameters = append(parameters, parameter)
				}
				sort.Strings(parameters)
				base.WriteHTTPError(w, http.StatusBadRequest, apiv2base.ErrorInvalidRequest,
					fmt.Sprintf("Unexpected query parameter %q", parameters[0]))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func hasRequestBody(r *http.Request) bool {
	return r.Body != nil && r.Body != http.NoBody && r.ContentLength != 0
}

// grpc-gateway keeps the matched binding and its compiled runtime.Pattern
// inside the generated mux registration. Because outer middleware cannot read
// that match, find it again in endpointContracts:
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/protoc-gen-grpc-gateway/internal/gengateway/template.go#L776-L821
func matchEndpointContract(method, path string) *endpointContract {
	for i := range endpointContracts {
		contract := &endpointContracts[i]
		if contract.method == method && contract.pattern.MatchString(path) {
			return contract
		}
	}
	return nil
}

// grpc-gateway's descriptor loader expands the primary HTTP binding and every
// additional binding into separate generated handlers. Rebuild the same list
// from the protobuf descriptors:
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/internal/descriptor/services.go#L182-L200
func discoverEndpointContracts() []endpointContract {
	service := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if service == nil {
		return nil
	}

	var contracts []endpointContract
	methods := service.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		rule := apiv2endpoint.HTTPRule(method)
		if rule == nil {
			continue
		}
		contracts = appendEndpointContract(contracts, method.Input(), rule)
		for _, binding := range rule.AdditionalBindings {
			contracts = appendEndpointContract(contracts, method.Input(), binding)
		}
	}
	return contracts
}

func appendEndpointContract(contracts []endpointContract, input protoreflect.MessageDescriptor, rule *annotations.HttpRule) []endpointContract {
	httpMethod, path := apiv2endpoint.MethodAndPath(rule)
	return append(contracts, endpointContract{
		method:       httpMethod,
		pattern:      compileHTTPPath(path),
		acceptsQuery: hasQueryFields(input, rule),
		acceptsBody:  rule.Body != "",
	})
}

// This mirrors grpc-gateway's HasQueryParam generator helper. A request has
// query parameters when at least one top-level field is not consumed by the
// path or body:
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/protoc-gen-grpc-gateway/internal/gengateway/template.go#L74-L103
func hasQueryFields(input protoreflect.MessageDescriptor, rule *annotations.HttpRule) bool {
	if rule.Body == "*" {
		return false
	}

	bound := make(map[string]struct{})
	_, path := apiv2endpoint.MethodAndPath(rule)
	for _, parameter := range apiv2endpoint.PathParams(path) {
		bound[strings.Split(parameter, ".")[0]] = struct{}{}
	}
	if rule.Body != "" {
		bound[strings.Split(rule.Body, ".")[0]] = struct{}{}
	}

	fields := input.Fields()
	for i := 0; i < fields.Len(); i++ {
		if _, ok := bound[string(fields.Get(i).Name())]; !ok {
			return true
		}
	}
	return false
}

// grpc-gateway compiles HTTP templates into an internal runtime.Pattern, which
// is not available to this outer middleware. Build a regexp matcher for the
// path-template forms currently used by API v2 instead:
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/internal/httprule/compile.go#L45-L121
// https://github.com/grpc-ecosystem/grpc-gateway/blob/v2.28.0/protoc-gen-grpc-gateway/internal/gengateway/template.go#L936-L942
func compileHTTPPath(path string) *regexp.Regexp {
	var pattern strings.Builder
	pattern.WriteString("^")
	last := 0
	for _, match := range pathParameterRegexp.FindAllStringIndex(path, -1) {
		pattern.WriteString(regexp.QuoteMeta(path[last:match[0]]))
		parameter := path[match[0]:match[1]]
		if strings.Contains(parameter, "=**") {
			pattern.WriteString(".+")
		} else {
			pattern.WriteString("[^/]+")
		}
		last = match[1]
	}
	pattern.WriteString(regexp.QuoteMeta(path[last:]))
	pattern.WriteString("$")
	return regexp.MustCompile(pattern.String())
}
