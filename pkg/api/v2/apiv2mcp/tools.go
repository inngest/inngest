package apiv2mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var pathParamPattern = regexp.MustCompile(`\{([^}=]+)(=[^}]*)?}`)

type Execute func(context.Context, apiv2endpoint.Endpoint, *http.Request) (*mcp.CallToolResult, error)

type Options struct {
	BasePath  string
	EnvHeader string
	Execute   Execute
	Exclude   func(apiv2endpoint.Endpoint) bool
}

func Register(server *mcp.Server, opts Options) []apiv2endpoint.Endpoint {
	endpoints := apiv2endpoint.Discover()
	registered := make([]apiv2endpoint.Endpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if opts.Exclude != nil && opts.Exclude(endpoint) {
			continue
		}
		server.AddTool(tool(endpoint, opts.EnvHeader != ""), func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return call(ctx, request, endpoint, opts)
		})
		registered = append(registered, endpoint)
	}
	return registered
}

func Tool(endpoint apiv2endpoint.Endpoint) *mcp.Tool {
	return tool(endpoint, true)
}

func tool(endpoint apiv2endpoint.Endpoint, includeEnv bool) *mcp.Tool {
	description := endpoint.Description
	if description == "" {
		description = endpoint.Summary
	}

	readOnly := endpoint.HTTPMethod == http.MethodGet
	return &mcp.Tool{
		Name:        endpoint.ToolName,
		Title:       endpoint.Summary,
		Description: description,
		InputSchema: inputSchema(endpoint, includeEnv),
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    readOnly,
			DestructiveHint: boolPointer(!readOnly),
			IdempotentHint:  readOnly || endpoint.HTTPMethod == http.MethodPut || endpoint.HTTPMethod == http.MethodDelete,
			OpenWorldHint:   boolPointer(false),
		},
	}
}

func InputSchema(endpoint apiv2endpoint.Endpoint) map[string]any {
	return inputSchema(endpoint, true)
}

func inputSchema(endpoint apiv2endpoint.Endpoint, includeEnv bool) map[string]any {
	properties := map[string]any{}
	if includeEnv {
		properties["env"] = map[string]any{
			"type":        "string",
			"description": "Environment slug. Defaults to the production environment for account-scoped credentials.",
		}
	}
	required := []string{}
	fields := endpoint.Input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		properties[field.JSONName()] = fieldSchema(field, map[protoreflect.FullName]bool{})
		if apiv2endpoint.IsRequired(field) || slices.Contains(endpoint.PathParams, string(field.Name())) {
			required = append(required, field.JSONName())
		}
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func call(ctx context.Context, request *mcp.CallToolRequest, endpoint apiv2endpoint.Endpoint, opts Options) (*mcp.CallToolResult, error) {
	if opts.Execute == nil {
		return nil, fmt.Errorf("REST API v2 executor is not configured")
	}

	args := map[string]any{}
	if request != nil && request.Params != nil && len(request.Params.Arguments) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(request.Params.Arguments))
		decoder.UseNumber()
		if err := decoder.Decode(&args); err != nil {
			return ToolError("invalid tool arguments", map[string]any{"error": err.Error()}), nil
		}
	}

	req, err := Request(ctx, endpoint, args, opts)
	if err != nil {
		return ToolError(err.Error(), map[string]any{"error": err.Error()}), nil
	}
	return opts.Execute(ctx, endpoint, req)
}

func Request(ctx context.Context, endpoint apiv2endpoint.Endpoint, args map[string]any, opts Options) (*http.Request, error) {
	path, err := resolvePath(endpoint, args)
	if err != nil {
		return nil, err
	}

	basePath := strings.TrimRight(opts.BasePath, "/")
	if basePath == "" {
		basePath = "/api/v2"
	}
	path = basePath + path

	query := url.Values{}
	var body any
	if endpoint.Body == "*" {
		body = bodyFields(endpoint, args)
	} else if endpoint.Body != "" {
		field := endpoint.Input.Fields().ByName(protoreflect.Name(endpoint.Body))
		if field == nil {
			return nil, fmt.Errorf("unknown request body field %q", endpoint.Body)
		}
		body = args[field.JSONName()]
	} else {
		addQueryFields(query, endpoint, args)
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var encoded []byte
	if body != nil {
		encoded, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, endpoint.HTTPMethod, path, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if env, ok := args["env"].(string); ok && opts.EnvHeader != "" && strings.TrimSpace(env) != "" {
		req.Header.Set(opts.EnvHeader, env)
	}
	return req, nil
}

func resolvePath(endpoint apiv2endpoint.Endpoint, args map[string]any) (string, error) {
	var firstErr error
	path := pathParamPattern.ReplaceAllStringFunc(endpoint.Path, func(match string) string {
		parts := pathParamPattern.FindStringSubmatch(match)
		if firstErr != nil || len(parts) < 2 {
			return match
		}

		field := endpoint.Input.Fields().ByName(protoreflect.Name(parts[1]))
		if field == nil {
			firstErr = fmt.Errorf("unknown path parameter %q", parts[1])
			return match
		}
		value, ok := args[field.JSONName()]
		if !ok || strings.TrimSpace(fmt.Sprint(value)) == "" {
			firstErr = fmt.Errorf("%s is required", field.JSONName())
			return match
		}
		return url.PathEscape(fmt.Sprint(value))
	})
	return path, firstErr
}

func addQueryFields(query url.Values, endpoint apiv2endpoint.Endpoint, args map[string]any) {
	fields := endpoint.Input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if slices.Contains(endpoint.PathParams, string(field.Name())) {
			continue
		}
		value, ok := args[field.JSONName()]
		if !ok {
			continue
		}
		addQueryValue(query, field.JSONName(), value)
	}
}

func addQueryValue(query url.Values, key string, value any) {
	switch value := value.(type) {
	case []any:
		for _, item := range value {
			query.Add(key, fmt.Sprint(item))
		}
	case []string:
		for _, item := range value {
			query.Add(key, item)
		}
	default:
		query.Set(key, fmt.Sprint(value))
	}
}

func bodyFields(endpoint apiv2endpoint.Endpoint, args map[string]any) map[string]any {
	body := map[string]any{}
	fields := endpoint.Input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if slices.Contains(endpoint.PathParams, string(field.Name())) {
			continue
		}
		if value, ok := args[field.JSONName()]; ok {
			body[field.JSONName()] = value
		}
	}
	return body
}

func fieldSchema(field protoreflect.FieldDescriptor, seen map[protoreflect.FullName]bool) map[string]any {
	schema := scalarSchema(field, seen)
	if field.IsList() && !field.IsMap() {
		schema = map[string]any{"type": "array", "items": schema}
	}
	if description := apiv2endpoint.FieldDescription(field); description != "" {
		schema["description"] = description
	}
	return schema
}

func scalarSchema(field protoreflect.FieldDescriptor, seen map[protoreflect.FullName]bool) map[string]any {
	if field.IsMap() {
		return map[string]any{
			"type":                 "object",
			"additionalProperties": scalarSchema(field.MapValue(), seen),
		}
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		return map[string]any{"type": "boolean"}
	case protoreflect.StringKind:
		return map[string]any{"type": "string"}
	case protoreflect.BytesKind:
		return map[string]any{"type": "string", "contentEncoding": "base64"}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return map[string]any{"type": "integer"}
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return map[string]any{"type": "number"}
	case protoreflect.EnumKind:
		values := field.Enum().Values()
		enums := make([]string, 0, values.Len())
		for i := 0; i < values.Len(); i++ {
			enums = append(enums, string(values.Get(i).Name()))
		}
		return map[string]any{"type": "string", "enum": enums}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return messageSchema(field.Message(), seen)
	default:
		return map[string]any{}
	}
}

func messageSchema(message protoreflect.MessageDescriptor, seen map[protoreflect.FullName]bool) map[string]any {
	switch message.FullName() {
	case "google.protobuf.Timestamp":
		return map[string]any{"type": "string", "format": "date-time"}
	case "google.protobuf.Struct":
		return map[string]any{"type": "object", "additionalProperties": true}
	case "google.protobuf.Value", "google.protobuf.Any":
		return map[string]any{}
	}
	if seen[message.FullName()] {
		return map[string]any{"type": "object"}
	}

	nextSeen := make(map[protoreflect.FullName]bool, len(seen)+1)
	for name, value := range seen {
		nextSeen[name] = value
	}
	nextSeen[message.FullName()] = true

	properties := map[string]any{}
	required := []string{}
	fields := message.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		properties[field.JSONName()] = fieldSchema(field, nextSeen)
		if apiv2endpoint.IsRequired(field) {
			required = append(required, field.JSONName())
		}
	}
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func ToolResult(structured any, text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: text}},
		StructuredContent: structured,
	}
}

func ToolError(message string, structured any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: message}},
		StructuredContent: structured,
		IsError:           true,
	}
}

func boolPointer(value bool) *bool {
	return &value
}
