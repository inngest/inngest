package apiv2endpoint

import (
	"net/http"
	"regexp"
	"strings"
	"unicode"

	openapiv2 "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2/options"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var pathParamPattern = regexp.MustCompile(`\{([^}=]+)(=[^}]*)?}`)

var hiddenMethods = map[string]struct{}{
	"CreatePartnerAccount": {},
	"FetchPartnerAccounts": {},
}

var commandNames = map[string]string{
	"ListRuns": "get-function-runs",
}

var toolNames = map[string]string{
	"FetchAccountEnvs": "list_envs",
	"GetFunctions":     "list_functions",
	"GetFunctionRun":   "get_run",
	"GetFunctionTrace": "get_run_trace",
}

type Endpoint struct {
	MethodName          string
	CommandName         string
	CommandNameExplicit bool
	ToolName            string
	HTTPMethod          string
	Path                string
	Summary             string
	Description         string
	Body                string
	Input               protoreflect.MessageDescriptor
	PathParams          []string
	ServerStreaming     bool
}

func Discover() []Endpoint {
	service := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if service == nil {
		return nil
	}

	endpoints := []Endpoint{}
	methods := service.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		methodName := string(method.Name())
		if strings.HasPrefix(methodName, "_") || Hidden(methodName) {
			continue
		}

		rule := HTTPRule(method)
		if rule == nil {
			continue
		}

		httpMethod, path := MethodAndPath(rule)
		if path == "" {
			continue
		}

		summary, description := MethodHelp(method)
		endpoints = append(endpoints, Endpoint{
			MethodName:          methodName,
			CommandName:         CommandName(methodName),
			CommandNameExplicit: HasCommandNameOverride(methodName),
			ToolName:            ToolName(methodName),
			HTTPMethod:          httpMethod,
			Path:                path,
			Summary:             summary,
			Description:         description,
			Body:                rule.Body,
			Input:               method.Input(),
			PathParams:          PathParams(path),
			ServerStreaming:     method.IsStreamingServer(),
		})
	}

	return endpoints
}

func Hidden(methodName string) bool {
	_, ok := hiddenMethods[methodName]
	return ok
}

func CommandName(methodName string) string {
	if name, ok := commandNames[methodName]; ok {
		return name
	}

	name := kebab(methodName)
	for _, prefix := range []string{"fetch-", "list-"} {
		if strings.HasPrefix(name, prefix) {
			return "get-" + strings.TrimPrefix(name, prefix)
		}
	}
	return name
}

func HasCommandNameOverride(methodName string) bool {
	_, ok := commandNames[methodName]
	return ok
}

func ToolName(methodName string) string {
	if name, ok := toolNames[methodName]; ok {
		return name
	}
	return snake(methodName)
}

func HTTPRule(method protoreflect.MethodDescriptor) *annotations.HttpRule {
	opts := method.Options()
	if !proto.HasExtension(opts, annotations.E_Http) {
		return nil
	}
	rule, _ := proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule)
	return rule
}

func MethodAndPath(rule *annotations.HttpRule) (string, string) {
	switch pattern := rule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		return http.MethodGet, pattern.Get
	case *annotations.HttpRule_Post:
		return http.MethodPost, pattern.Post
	case *annotations.HttpRule_Put:
		return http.MethodPut, pattern.Put
	case *annotations.HttpRule_Delete:
		return http.MethodDelete, pattern.Delete
	case *annotations.HttpRule_Patch:
		return http.MethodPatch, pattern.Patch
	default:
		return http.MethodPost, ""
	}
}

func MethodHelp(method protoreflect.MethodDescriptor) (string, string) {
	opts := method.Options()
	if !proto.HasExtension(opts, openapiv2.E_Openapiv2Operation) {
		return "", ""
	}

	op, ok := proto.GetExtension(opts, openapiv2.E_Openapiv2Operation).(*openapiv2.Operation)
	if !ok {
		return "", ""
	}
	return op.GetSummary(), op.GetDescription()
}

func PathParams(path string) []string {
	matches := pathParamPattern.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

func kebab(value string) string {
	return separated(value, '-')
}

func snake(value string) string {
	return separated(value, '_')
}

func separated(value string, separator rune) string {
	var result strings.Builder
	for i, r := range value {
		switch {
		case r == '_':
			result.WriteRune(separator)
		case unicode.IsUpper(r):
			if i > 0 {
				result.WriteRune(separator)
			}
			result.WriteRune(unicode.ToLower(r))
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}
