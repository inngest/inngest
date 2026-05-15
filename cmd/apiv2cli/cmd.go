package apiv2cli

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	apiv2proto "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/urfave/cli/v3"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	cloudGRPCTarget = "api.inngest.com:443"
	defaultTimeout  = 30 * time.Second
)

var hiddenEndpointMethods = map[string]struct{}{
	"CreatePartnerAccount": {},
	"FetchPartnerAccounts": {},
}

type endpoint struct {
	name       string
	methodName string
	fullMethod string
	httpMethod string
	path       string
	input      protoreflect.MessageDescriptor
	output     protoreflect.MessageDescriptor
}

func Command() *cli.Command {
	return &cli.Command{
		Name:      "api",
		Usage:     "Call Inngest API v2 endpoints over gRPC",
		UsageText: "inngest alpha api [target/auth flags] <endpoint> [endpoint flags]",
		Description: strings.Join([]string{
			"By default, the command targets the local dev server's gRPC endpoint.",
			"Set --prod to target Inngest Cloud Production, or --api-host/--api-port to target a custom server.",
		}, "\n"),
		Flags:    commonFlags(),
		Commands: endpointCommands(),
	}
}

func endpointCommands() []*cli.Command {
	endpoints := discoverEndpoints()
	cmds := make([]*cli.Command, 0, len(endpoints))

	for _, ep := range endpoints {
		cmds = append(cmds, &cli.Command{
			Name:      ep.name,
			Usage:     fmt.Sprintf("%s %s", ep.httpMethod, ep.path),
			UsageText: fmt.Sprintf("inngest alpha api [target/auth flags] %s [endpoint flags]", ep.name),
			Flags:     endpointFlags(ep),
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return callEndpoint(ctx, cmd, ep)
			},
		})
	}

	return cmds
}

func commonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Category: "Target",
			Name:     "prod",
			Usage:    "Target Inngest Cloud Production unless --api-host or --api-port is set",
		},
		&cli.StringFlag{
			Category: "Target",
			Name:     "config",
			Usage:    "Path to an Inngest configuration file",
		},
		&cli.StringFlag{
			Category: "Target",
			Name:     "api-host",
			Usage:    "Custom host (or host:port) for the gRPC endpoint",
		},
		&cli.IntFlag{
			Category:    "Target",
			Name:        "api-port",
			DefaultText: "default API v2 gRPC port",
			Usage:       "Custom API v2 gRPC port",
		},
		&cli.BoolFlag{
			Category: "Target",
			Name:     "insecure",
			Usage:    "Force plaintext gRPC (default for localhost, TLS otherwise)",
		},
		&cli.StringFlag{
			Category: "Auth",
			Name:     "api-key",
			Usage:    "API key sent as a Bearer token",
			Sources:  cli.EnvVars("INNGEST_API_KEY"),
		},
		&cli.StringFlag{
			Category: "Auth",
			Name:     "signing-key",
			Usage:    "Signing key sent as a Bearer token",
			Sources:  cli.EnvVars("INNGEST_SIGNING_KEY"),
		},
		&cli.StringFlag{
			Category: "Auth",
			Name:     "env",
			Usage:    "Environment name sent as X-Inngest-Env",
			Sources:  cli.EnvVars("INNGEST_ENV"),
		},
		&cli.DurationFlag{
			Category: "Target",
			Name:     "timeout",
			Value:    defaultTimeout,
			Usage:    "RPC call timeout",
		},
		&cli.BoolFlag{
			Category: "Output",
			Name:     "raw",
			Usage:    "Print the response as compact JSON (no pretty-printing)",
		},
	}
}

func endpointFlags(ep endpoint) []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Category: "Body",
			Name:     "body",
			Usage:    "Raw JSON request body. Field flags override matching keys.",
		},
		&cli.StringFlag{
			Category: "Body",
			Name:     "body-file",
			Usage:    "Path to a JSON request body file, or '-' for stdin.",
		},
	}

	fields := ep.input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		flags = append(flags, flagForField("Field", kebab(string(field.Name())), field))
	}

	return flags
}

func flagForField(category, name string, field protoreflect.FieldDescriptor) cli.Flag {
	usage := string(field.JSONName())
	if isRequiredField(field) {
		usage += " (required)"
	}
	if field.IsList() {
		return &cli.StringSliceFlag{
			Category: category,
			Name:     name,
			Usage:    usage,
		}
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		return &cli.BoolFlag{Category: category, Name: name, Usage: usage}
	default:
		return &cli.StringFlag{Category: category, Name: name, Usage: usage}
	}
}

func discoverEndpoints() []endpoint {
	service := apiv2proto.File_api_v2_service_proto.Services().ByName("V2")
	if service == nil {
		return nil
	}
	serviceFullName := string(service.FullName())

	endpoints := []endpoint{}
	methods := service.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		methodName := string(method.Name())
		if strings.HasPrefix(methodName, "_") {
			continue
		}
		if _, hidden := hiddenEndpointMethods[methodName]; hidden {
			continue
		}

		httpRule := httpRule(method)
		if httpRule == nil {
			continue
		}

		httpMethod, path := methodAndPath(httpRule)
		if path == "" {
			continue
		}

		endpoints = append(endpoints, endpoint{
			name:       endpointCommandName(methodName),
			methodName: methodName,
			fullMethod: fmt.Sprintf("/%s/%s", serviceFullName, methodName),
			httpMethod: httpMethod,
			path:       path,
			input:      method.Input(),
			output:     method.Output(),
		})
	}

	return endpoints
}

func endpointCommandName(methodName string) string {
	name := kebab(methodName)
	for _, prefix := range []string{"fetch-", "list-"} {
		if strings.HasPrefix(name, prefix) {
			return "get-" + strings.TrimPrefix(name, prefix)
		}
	}
	return name
}

func httpRule(method protoreflect.MethodDescriptor) *annotations.HttpRule {
	opts := method.Options()
	if !proto.HasExtension(opts, annotations.E_Http) {
		return nil
	}
	return proto.GetExtension(opts, annotations.E_Http).(*annotations.HttpRule)
}

func methodAndPath(rule *annotations.HttpRule) (string, string) {
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

func callEndpoint(ctx context.Context, cmd *cli.Command, ep endpoint) error {
	target, useTLS, err := resolveTarget(ctx, cmd)
	if err != nil {
		return err
	}

	token, err := authToken(cmd)
	if err != nil {
		return err
	}

	if token != "" && !useTLS && !targetIsLocal(target) {
		return fmt.Errorf("refusing to send credentials over plaintext gRPC to %s; pass --insecure only for local targets", target)
	}

	req := dynamicpb.NewMessage(ep.input)
	if err := populateRequest(cmd, ep, req); err != nil {
		return err
	}

	var creds credentials.TransportCredentials = insecure.NewCredentials()
	if useTLS {
		creds = credentials.NewTLS(&tls.Config{})
	}

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(creds))
	if err != nil {
		return err
	}
	defer conn.Close()

	md := metadata.MD{}
	if token != "" {
		md.Set("authorization", "Bearer "+token)
	}
	if env := cmd.String("env"); env != "" {
		md.Set("x-inngest-env", env)
	}

	timeout := cmd.Duration("timeout")
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	callCtx, cancel := context.WithTimeout(metadata.NewOutgoingContext(ctx, md), timeout)
	defer cancel()

	resp := dynamicpb.NewMessage(ep.output)
	if err := conn.Invoke(callCtx, ep.fullMethod, req, resp); err != nil {
		if targetIsLocal(target) && isConnRefused(err) {
			return fmt.Errorf("local dev server is not available at %s; start it with `inngest dev` or use --prod to target Inngest Cloud: %w", target, err)
		}
		return formatGRPCError(err)
	}

	return writeResponse(cmd, resp)
}

func populateRequest(cmd *cli.Command, ep endpoint, msg *dynamicpb.Message) error {
	body, err := rawBody(cmd)
	if err != nil {
		return err
	}

	fields := ep.input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		flagName := kebab(string(field.Name()))
		if !cmd.IsSet(flagName) {
			continue
		}
		value, err := flagJSONValue(cmd, field, flagName)
		if err != nil {
			return err
		}
		body[field.JSONName()] = value
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}

	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(encoded, msg); err != nil {
		return fmt.Errorf("invalid request payload: %w", err)
	}
	return nil
}

func rawBody(cmd *cli.Command) (map[string]any, error) {
	if cmd.IsSet("body") && cmd.IsSet("body-file") {
		return nil, errors.New("--body and --body-file cannot both be set")
	}

	var data []byte
	switch {
	case cmd.IsSet("body"):
		data = []byte(cmd.String("body"))
	case cmd.IsSet("body-file"):
		byt, err := readBodyFile(cmd, cmd.String("body-file"))
		if err != nil {
			return nil, err
		}
		data = byt
	default:
		return map[string]any{}, nil
	}

	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("invalid JSON body: %w", err)
	}
	return body, nil
}

func readBodyFile(cmd *cli.Command, path string) ([]byte, error) {
	if path == "-" {
		reader := cmd.Root().Reader
		if reader == nil {
			reader = os.Stdin
		}
		return io.ReadAll(reader)
	}
	return os.ReadFile(path)
}

// flagJSONValue returns a JSON-serialisable value (string, bool, number, list)
// that protojson can decode into the typed proto field. We let protojson
// validate kinds rather than reimplementing it here.
func flagJSONValue(cmd *cli.Command, field protoreflect.FieldDescriptor, flagName string) (any, error) {
	if field.IsList() {
		return cmd.StringSlice(flagName), nil
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		return cmd.Bool(flagName), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		var value any
		if err := json.Unmarshal([]byte(cmd.String(flagName)), &value); err != nil {
			return nil, fmt.Errorf("--%s must be valid JSON: %w", flagName, err)
		}
		return value, nil
	default:
		return cmd.String(flagName), nil
	}
}

func isRequiredField(field protoreflect.FieldDescriptor) bool {
	opts := field.Options()
	if !proto.HasExtension(opts, annotations.E_FieldBehavior) {
		return false
	}
	behaviors, ok := proto.GetExtension(opts, annotations.E_FieldBehavior).([]annotations.FieldBehavior)
	if !ok {
		return false
	}
	return slices.Contains(behaviors, annotations.FieldBehavior_REQUIRED)
}

func authToken(cmd *cli.Command) (string, error) {
	if apiKey := cmd.String("api-key"); apiKey != "" {
		return apiKey, nil
	}
	return cmd.String("signing-key"), nil
}

func resolveTarget(ctx context.Context, cmd *cli.Command) (string, bool, error) {
	if err := localconfig.InitDevConfig(ctx, cmd); err != nil {
		return "", false, err
	}

	insecureFlag := localconfig.GetBoolValue(cmd, "insecure", false)
	apiPort := localconfig.GetIntValue(cmd, "api-port", 0)
	apiHost := localconfig.GetValue(cmd, "api-host", "")

	if apiHost != "" {
		target, useTLS, err := buildTarget(apiHost, apiPort, insecureFlag)
		return target, useTLS, err
	}

	if apiPort != 0 {
		return net.JoinHostPort("localhost", strconv.Itoa(apiPort)), false, nil
	}

	if localconfig.GetBoolValue(cmd, "prod", false) {
		return cloudGRPCTarget, !insecureFlag, nil
	}

	return net.JoinHostPort("localhost", strconv.Itoa(apiv2.DefaultGRPCPort)), false, nil
}

func buildTarget(rawHost string, port int, insecureFlag bool) (string, bool, error) {
	host := rawHost
	if looksLikeURL(rawHost) {
		parsed, err := url.Parse(rawHost)
		if err != nil {
			return "", false, err
		}
		host = parsed.Host
		if host == "" {
			return "", false, fmt.Errorf("api host must include a host name")
		}
	}

	if parsedHost, parsedPort, err := net.SplitHostPort(host); err == nil {
		if port == 0 {
			p, perr := strconv.Atoi(parsedPort)
			if perr != nil {
				return "", false, perr
			}
			port = p
		}
		host = parsedHost
	}

	if port == 0 {
		if isLocalHost(host) {
			port = apiv2.DefaultGRPCPort
		} else if insecureFlag {
			port = apiv2.DefaultGRPCPort
		} else {
			port = 443
		}
	}

	target := net.JoinHostPort(host, strconv.Itoa(port))
	useTLS := !insecureFlag && !isLocalHost(host)
	return target, useTLS, nil
}

func writeResponse(cmd *cli.Command, msg *dynamicpb.Message) error {
	writer := cmd.Root().Writer
	if writer == nil {
		writer = os.Stdout
	}

	marshal := protojson.MarshalOptions{
		Multiline:       !cmd.Bool("raw"),
		Indent:          "  ",
		EmitUnpopulated: false,
	}
	body, err := marshal.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = writer.Write(append(body, '\n'))
	return err
}

func formatGRPCError(err error) error {
	if s, ok := status.FromError(err); ok {
		return fmt.Errorf("%s: %s", s.Code(), s.Message())
	}
	return err
}

func isConnRefused(err error) bool {
	return err != nil && strings.Contains(err.Error(), "connection refused")
}

func targetIsLocal(target string) bool {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	return isLocalHost(host)
}

func kebab(value string) string {
	var b strings.Builder
	for i, r := range value {
		switch {
		case r == '_':
			b.WriteRune('-')
		case unicode.IsUpper(r):
			if i > 0 {
				b.WriteRune('-')
			}
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func looksLikeURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func isLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0"
}
