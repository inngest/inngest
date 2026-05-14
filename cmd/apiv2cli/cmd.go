package apiv2cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/pkg/api"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/urfave/cli/v3"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	defaultDevServerOrigin = "http://localhost:8288"
	defaultDevServerURL    = defaultDevServerOrigin + "/api/v2"
	cloudAPIURL            = "https://api.inngest.com/v2"
	defaultTimeout         = 30 * time.Second
	maxResponseBytes       = 25 << 20
)

var pathParamPattern = regexp.MustCompile(`\{([^}=]+)(=[^}]*)?}`)

type endpoint struct {
	name       string
	method     string
	path       string
	body       string
	input      protoreflect.MessageDescriptor
	pathParams []string
}

func Command() *cli.Command {
	return &cli.Command{
		Name:      "api",
		Usage:     "Call Inngest REST API v2 endpoints",
		UsageText: "inngest alpha api [target/auth flags] <endpoint> [endpoint flags]",
		Description: strings.Join([]string{
			"By default, the command uses a running local dev server, then falls back to Inngest Cloud.",
			"Set --api-cloud to force use of Inngest Cloud Production, or --api-host/--api-port to target a self-hosted server.",
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
			Usage:     fmt.Sprintf("%s %s", ep.method, ep.path),
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
			Name:     "api-cloud",
			Usage:    "Target Inngest Cloud Production API, ignoring local dev server and target host flags",
		},
		&cli.StringFlag{
			Category: "Target",
			Name:     "config",
			Usage:    "Path to an Inngest configuration file",
		},
		&cli.StringFlag{
			Category: "Target",
			Name:     "api-host",
			Usage:    "Selt-hosted API host or origin. Takes precedence over --host.",
		},
		&cli.IntFlag{
			Category:    "Target",
			Name:        "api-port",
			DefaultText: "same as --port",
			Usage:       "Self-hosted API port. Falls back to --port when unset.",
		},
		&cli.StringFlag{
			Category: "Target",
			Name:     "host",
			Usage:    "Inngest server host",
		},
		&cli.IntFlag{
			Category: "Target",
			Name:     "port",
			Aliases:  []string{"p"},
			Value:    api.DefaultAPIPort,
			Usage:    "Inngest server port",
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
			Usage:    "HTTP request timeout",
		},
		&cli.BoolFlag{
			Category: "Output",
			Name:     "raw",
			Usage:    "Print the response body without JSON formatting",
		},
	}
}

func endpointFlags(ep endpoint) []cli.Flag {
	var flags []cli.Flag
	if ep.body != "" {
		flags = append(flags,
			&cli.StringFlag{
				Category: "Body",
				Name:     "body",
				Usage:    "Raw JSON request body. Endpoint field flags override matching keys.",
			},
			&cli.StringFlag{
				Category: "Body",
				Name:     "body-file",
				Usage:    "Path to a JSON request body file, or '-' for stdin.",
			},
		)
	}

	fields := ep.input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())
		flagName := kebab(name)
		category := "Query"
		if slices.Contains(ep.pathParams, name) {
			category = "Path"
		} else if ep.body != "" {
			category = "Body"
		}

		flags = append(flags, flagForField(category, flagName, field))
	}

	return flags
}

func flagForField(category, name string, field protoreflect.FieldDescriptor) cli.Flag {
	usage := string(field.JSONName())
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
	service := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if service == nil {
		return nil
	}

	endpoints := []endpoint{}
	methods := service.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		if strings.HasPrefix(string(method.Name()), "_") {
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
			name:       kebab(string(method.Name())),
			method:     httpMethod,
			path:       path,
			body:       httpRule.Body,
			input:      method.Input(),
			pathParams: pathParams(path),
		})
	}

	return endpoints
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
	req, err := buildRequest(ctx, cmd, ep)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: cmd.Duration("timeout")}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return err
	}
	if int64(len(body)) > maxResponseBytes {
		return fmt.Errorf("response body exceeded %d bytes", maxResponseBytes)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	writer := cmd.Root().Writer
	if writer == nil {
		writer = os.Stdout
	}

	if cmd.Bool("raw") {
		_, err = writer.Write(append(body, '\n'))
		return err
	}

	formatted, err := prettyJSON(body)
	if err != nil {
		_, err = writer.Write(append(body, '\n'))
		return err
	}

	_, err = writer.Write(append(formatted, '\n'))
	return err
}

func buildRequest(ctx context.Context, cmd *cli.Command, ep endpoint) (*http.Request, error) {
	baseURL, err := resolveBaseURL(ctx, cmd)
	if err != nil {
		return nil, err
	}

	path, err := resolvePath(cmd, ep)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimRight(u.Path, "/") + path

	body, err := requestBody(cmd, ep)
	if err != nil {
		return nil, err
	}

	query, err := queryParams(cmd, ep)
	if err != nil {
		return nil, err
	}
	u.RawQuery = query.Encode()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, ep.method, u.String(), reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if token, err := authToken(cmd); err != nil {
		return nil, err
	} else if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	if env := cmd.String("env"); env != "" {
		req.Header.Set("X-Inngest-Env", env)
	}

	if err := guardPlaintextAuth(req); err != nil {
		return nil, err
	}

	return req, nil
}

// don't ship credentials to a non-local host over http
func guardPlaintextAuth(req *http.Request) error {
	if req.Header.Get("Authorization") == "" {
		return nil
	}
	if req.URL.Scheme != "http" {
		return nil
	}
	if isLocalHost(req.URL.Hostname()) {
		return nil
	}
	return fmt.Errorf("refusing to send credentials over plaintext HTTP to %s; use an https:// target", req.URL.Host)
}

func resolveBaseURL(ctx context.Context, cmd *cli.Command) (string, error) {
	if err := localconfig.InitDevConfig(ctx, cmd); err != nil {
		return "", err
	}

	if localconfig.GetBoolValue(cmd, "api-cloud", false) {
		return cloudAPIURL, nil
	}

	port := localconfig.GetIntValue(cmd, "port", api.DefaultAPIPort)
	apiPort := localconfig.GetIntValue(cmd, "api-port", port)
	if apiHost := localconfig.GetValue(cmd, "api-host", ""); apiHost != "" {
		return normalizeAPIHostTarget(apiHost, apiPort)
	}

	if host := localconfig.GetValue(cmd, "host", ""); host != "" {
		return normalizeServerHostTarget(host, apiPort)
	}

	if apiPort != api.DefaultAPIPort {
		return normalizeServerHostTarget("localhost", apiPort)
	}

	if localDevServerAvailable(ctx) {
		return defaultDevServerURL, nil
	}

	fmt.Fprintf(errWriter(cmd), "No local dev server detected; targeting Inngest Cloud at %s\n", cloudAPIURL)
	return cloudAPIURL, nil
}

func errWriter(cmd *cli.Command) io.Writer {
	if w := cmd.Root().ErrWriter; w != nil {
		return w
	}
	return os.Stderr
}

func normalizeAPIHostTarget(rawHost string, port int) (string, error) {
	if looksLikeURL(rawHost) {
		return normalizeAPIURL(rawHost)
	}

	host := rawHost
	hasPort := true
	if parsedHost, _, err := net.SplitHostPort(rawHost); err == nil {
		host = parsedHost
	} else {
		hasPort = false
	}

	scheme := "http"
	if !isLocalHost(host) {
		scheme = "https"
	} else if !hasPort {
		rawHost = net.JoinHostPort(rawHost, strconv.Itoa(port))
	}

	return normalizeAPIURL(fmt.Sprintf("%s://%s", scheme, rawHost))
}

func normalizeServerHostTarget(rawHost string, port int) (string, error) {
	if looksLikeURL(rawHost) {
		return normalizeAPIURL(rawHost)
	}

	if host, parsedPort, err := net.SplitHostPort(rawHost); err == nil {
		if isUnspecifiedHost(host) {
			rawHost = net.JoinHostPort("localhost", parsedPort)
		}
	} else {
		if isUnspecifiedHost(rawHost) {
			rawHost = "localhost"
		}
		rawHost = net.JoinHostPort(rawHost, strconv.Itoa(port))
	}

	return normalizeAPIURL(fmt.Sprintf("http://%s", rawHost))
}

func normalizeAPIURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("api host must include scheme and host")
	}

	switch strings.TrimRight(parsed.Path, "/") {
	case "":
		if isCloudHost(parsed.Hostname()) {
			parsed.Path = "/v2"
		} else {
			parsed.Path = "/api/v2"
		}
	case "/api":
		parsed.Path = "/api/v2"
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}

// checks if there is a running dev server.
// only used if no explicit configs are provided.
func localDevServerAvailable(ctx context.Context) bool {
	reqCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, defaultDevServerURL+"/health", nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 300 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}

func resolvePath(cmd *cli.Command, ep endpoint) (string, error) {
	var firstErr error
	path := pathParamPattern.ReplaceAllStringFunc(ep.path, func(match string) string {
		if firstErr != nil {
			return match
		}

		parts := pathParamPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			firstErr = fmt.Errorf("invalid path parameter %q", match)
			return match
		}

		name := parts[1]
		flagName := kebab(name)
		if !cmd.IsSet(flagName) || cmd.String(flagName) == "" {
			firstErr = fmt.Errorf("missing required --%s", flagName)
			return match
		}

		return url.PathEscape(cmd.String(flagName))
	})

	if firstErr != nil {
		return "", firstErr
	}
	return path, nil
}

func queryParams(cmd *cli.Command, ep endpoint) (url.Values, error) {
	values := url.Values{}
	if ep.body != "" {
		return values, nil
	}

	fields := ep.input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())
		if slices.Contains(ep.pathParams, name) {
			continue
		}

		flagName := kebab(name)
		if !cmd.IsSet(flagName) {
			continue
		}

		value, err := fieldValue(cmd, field, flagName)
		if err != nil {
			return nil, err
		}

		addQueryValue(values, field.JSONName(), value)
	}

	return values, nil
}

func requestBody(cmd *cli.Command, ep endpoint) (map[string]any, error) {
	if ep.body == "" {
		return nil, nil
	}

	body, err := rawBody(cmd)
	if err != nil {
		return nil, err
	}

	fields := ep.input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())
		if slices.Contains(ep.pathParams, name) {
			continue
		}

		flagName := kebab(name)
		if !cmd.IsSet(flagName) {
			continue
		}

		value, err := fieldValue(cmd, field, flagName)
		if err != nil {
			return nil, err
		}
		body[field.JSONName()] = value
	}

	if err := validateBody(cmd, ep, body); err != nil {
		return nil, err
	}

	return body, nil
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

func validateBody(cmd *cli.Command, ep endpoint, body map[string]any) error {
	fields := ep.input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())
		if slices.Contains(ep.pathParams, name) || !isRequiredField(field) {
			continue
		}

		if _, ok := body[field.JSONName()]; ok {
			continue
		}
		if _, ok := body[name]; ok {
			continue
		}
		if cmd.IsSet(kebab(name)) {
			continue
		}

		return fmt.Errorf("missing required --%s or body field %q", kebab(name), field.JSONName())
	}
	return nil
}

// isRequiredField reports whether the field carries a
// `(google.api.field_behavior) = REQUIRED` annotation.
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

func fieldValue(cmd *cli.Command, field protoreflect.FieldDescriptor, flagName string) (any, error) {
	if field.IsList() {
		return cmd.StringSlice(flagName), nil
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		return cmd.Bool(flagName), nil
	case protoreflect.StringKind:
		return cmd.String(flagName), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return parseInt(cmd.String(flagName), 32)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return parseInt(cmd.String(flagName), 64)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return parseUint(cmd.String(flagName), 32)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return parseUint(cmd.String(flagName), 64)
	case protoreflect.FloatKind:
		return parseFloat(cmd.String(flagName), 32)
	case protoreflect.DoubleKind:
		return parseFloat(cmd.String(flagName), 64)
	case protoreflect.EnumKind:
		return cmd.String(flagName), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		var value any
		if err := json.Unmarshal([]byte(cmd.String(flagName)), &value); err != nil {
			return nil, fmt.Errorf("--%s must be valid JSON: %w", flagName, err)
		}
		return value, nil
	case protoreflect.BytesKind:
		return base64.StdEncoding.EncodeToString([]byte(cmd.String(flagName))), nil
	default:
		return nil, fmt.Errorf("unsupported field type for --%s", flagName)
	}
}

func authToken(cmd *cli.Command) (string, error) {
	if apiKey := cmd.String("api-key"); apiKey != "" {
		return apiKey, nil
	}
	return cmd.String("signing-key"), nil
}

func addQueryValue(values url.Values, key string, value any) {
	switch v := value.(type) {
	case []string:
		for _, item := range v {
			values.Add(key, item)
		}
	default:
		values.Set(key, fmt.Sprint(v))
	}
}

func pathParams(path string) []string {
	matches := pathParamPattern.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

func parseInt(value string, bitSize int) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, bitSize)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func parseUint(value string, bitSize int) (uint64, error) {
	parsed, err := strconv.ParseUint(value, 10, bitSize)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func parseFloat(value string, bitSize int) (float64, error) {
	parsed, err := strconv.ParseFloat(value, bitSize)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func prettyJSON(body []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	return json.MarshalIndent(value, "", "  ")
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

func isCloudHost(host string) bool {
	return host == "api.inngest.com"
}

func isLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "0.0.0.0"
}

func isUnspecifiedHost(host string) bool {
	parsed := net.ParseIP(host)
	return parsed != nil && parsed.IsUnspecified()
}
