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
	"github.com/inngest/inngest/pkg/api/tel"
	"github.com/inngest/inngest/pkg/api/v2/apiv2endpoint"
	"github.com/urfave/cli/v3"
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
	summary    string
	help       string
	body       string
	input      protoreflect.MessageDescriptor
	pathParams []string
	streaming  bool
}

func Command() *cli.Command {
	return &cli.Command{
		Name:      "api",
		Usage:     "Call Inngest REST API v2 endpoints (beta)",
		UsageText: "inngest api [target/auth flags] <endpoint> [endpoint flags]",
		Description: strings.Join([]string{
			"Beta: this command is under active development and may change.",
			"By default, the command targets the local dev server.",
			"Set --prod to target Inngest Cloud Production, or --api-host/--api-port to target a custom API server.",
			"Authentication: https://api-docs.inngest.com/authentication",
		}, "\n"),
		Flags:    commonFlags(),
		Commands: endpointCommands(),
	}
}

func MovedCommand() *cli.Command {
	return &cli.Command{
		Name:        "api",
		Usage:       "Moved to inngest api",
		UsageText:   "inngest alpha api",
		Description: "The alpha api command has moved. Use `inngest api` instead.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_, err := fmt.Fprintln(cmd.Root().Writer, "The alpha api command has moved. Use `inngest api` instead.")
			return err
		},
	}
}

func endpointCommands() []*cli.Command {
	endpoints := discoverEndpoints()
	cmds := make([]*cli.Command, 0, len(endpoints))

	for _, ep := range endpoints {
		cmds = append(cmds, &cli.Command{
			Name:        ep.name,
			Usage:       endpointUsage(ep),
			UsageText:   endpointUsageText(ep),
			Description: endpointDescription(ep),
			Flags:       endpointFlags(ep),
			Arguments:   endpointArguments(ep),
			Action: func(ctx context.Context, cmd *cli.Command) error {
				tel.SendCommandExecutedEvent(ctx, "inngest api", commandTelemetryContext(cmd, ep))
				return callEndpoint(ctx, cmd, ep)
			},
		})
	}

	return cmds
}

func commandTelemetryContext(cmd *cli.Command, ep endpoint) map[string]any {
	flags := []string{}
	for _, flag := range append(commonFlags(), endpointFlags(ep)...) {
		name := flag.Names()[0]
		if cmd.IsSet(name) {
			flags = append(flags, name)
		}
	}
	slices.Sort(flags)

	return map[string]any{
		"endpoint": ep.name,
		"flags":    flags,
	}
}

func commonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Category: "Target",
			Name:     "prod",
			Usage:    "Target Inngest Cloud Production API unless --api-host or --api-port is set",
		},
		&cli.StringFlag{
			Category: "Target",
			Name:     "config",
			Usage:    "Path to an Inngest configuration file",
		},
		&cli.StringFlag{
			Category: "Target",
			Name:     "api-host",
			Usage:    "Custom API host or origin; may include /api/v2 or /v2",
		},
		&cli.IntFlag{
			Category:    "Target",
			Name:        "api-port",
			DefaultText: "default local dev server port",
			Usage:       "Custom API port",
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

func endpointUsageText(ep endpoint) string {
	var positional strings.Builder
	for _, name := range ep.pathParams {
		fmt.Fprintf(&positional, " [<%s>]", kebab(name))
	}
	return fmt.Sprintf("inngest api [target/auth flags] %s%s [endpoint flags]", ep.name, positional.String())
}

func endpointArguments(ep endpoint) []cli.Argument {
	if len(ep.pathParams) == 0 {
		return nil
	}
	args := make([]cli.Argument, 0, len(ep.pathParams))
	for _, name := range ep.pathParams {
		flagName := kebab(name)
		args = append(args, &cli.StringArg{
			Name:      flagName,
			UsageText: fmt.Sprintf("[<%s>]", flagName),
		})
	}
	return args
}

func endpointFlags(ep endpoint) []cli.Flag {
	var flags []cli.Flag
	rawBody := isRawHTTPBodyEndpoint(ep)
	if rawBody {
		flags = append(flags,
			&cli.StringFlag{
				Category: "Body",
				Name:     "body",
				Usage:    "Raw request body.",
			},
			&cli.StringFlag{
				Category: "Body",
				Name:     "body-file",
				Usage:    "Path to a raw request body file, or '-' for stdin.",
			},
		)
	} else if ep.body != "" {
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
		if rawBody && name == ep.body {
			continue
		}
		flagName := kebab(name)
		category := "Query"
		if slices.Contains(ep.pathParams, name) {
			category = "Path"
		} else if ep.body != "" && !rawBody {
			category = "Body"
		}

		if ep.name == "rerun" && flagName == "from-step" {
			flags = append(flags, &cli.StringFlag{
				Category: category,
				Name:     flagName,
				Usage:    "Step name to rerun from.",
			})
			continue
		}
		flags = append(flags, flagForField(category, flagName, field))
	}
	if ep.name == "rerun" {
		flags = append(flags, &cli.StringFlag{
			Category: "Body",
			Name:     "input",
			Usage:    "Optional replacement step input as a JSON array.",
		})
	}

	return flags
}

func flagForField(category, name string, field protoreflect.FieldDescriptor) cli.Flag {
	usage := fieldUsage(field)
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

func fieldUsage(field protoreflect.FieldDescriptor) string {
	usage := apiv2endpoint.FieldDescription(field)

	if isRequiredField(field) {
		usage += " (required)"
	}

	return usage
}

func discoverEndpoints() []endpoint {
	shared := canonicalCommandEndpoints(apiv2endpoint.Discover())
	endpoints := make([]endpoint, 0, len(shared))
	for _, discovered := range shared {
		endpoints = append(endpoints, endpoint{
			name:       discovered.CommandName,
			method:     discovered.HTTPMethod,
			path:       discovered.Path,
			summary:    discovered.Summary,
			help:       discovered.Description,
			body:       discovered.Body,
			input:      discovered.Input,
			pathParams: discovered.PathParams,
			streaming:  discovered.ServerStreaming,
		})
	}

	return endpoints
}

func canonicalCommandEndpoints(endpoints []apiv2endpoint.Endpoint) []apiv2endpoint.Endpoint {
	explicitOwners := map[string]string{}
	for _, endpoint := range endpoints {
		if endpoint.CommandNameExplicit {
			explicitOwners[endpoint.CommandName] = endpoint.MethodName
		}
	}

	result := make([]apiv2endpoint.Endpoint, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if owner, ok := explicitOwners[endpoint.CommandName]; ok && owner != endpoint.MethodName {
			continue
		}
		result = append(result, endpoint)
	}
	return result
}

func endpointUsage(ep endpoint) string {
	if ep.summary != "" {
		return ep.summary
	}
	return fmt.Sprintf("%s %s", ep.method, ep.path)
}

func endpointDescription(ep endpoint) string {
	lines := []string{}
	if ep.help != "" {
		lines = append(lines, ep.help, "")
	}

	lines = append(lines,
		fmt.Sprintf("Endpoint: %s %s", ep.method, ep.path),
		"",
		"Target, auth, and output flags are inherited from `inngest api`:",
		"  --prod                  Target Inngest Cloud Production",
		"  --api-host, --api-port  Target a custom API server; host may include /api/v2 or /v2",
		"  --api-key               API key, or INNGEST_API_KEY",
		"  --signing-key           Signing key, or INNGEST_SIGNING_KEY",
		"  --env                   Environment name, or INNGEST_ENV",
		"  --raw                   Print the response body without formatting",
		"",
		"Authentication: https://api-docs.inngest.com/authentication",
	)

	return strings.Join(lines, "\n")
}

func endpointCommandName(methodName string) string {
	return apiv2endpoint.CommandName(methodName)
}

func callEndpoint(ctx context.Context, cmd *cli.Command, ep endpoint) error {
	if extras := cmd.Args().Slice(); len(extras) > 0 {
		return fmt.Errorf("unexpected positional argument(s): %s", strings.Join(extras, " "))
	}

	req, err := buildRequest(ctx, cmd, ep)
	if err != nil {
		return err
	}

	timeout := cmd.Duration("timeout")
	if ep.streaming && !cmd.IsSet("timeout") {
		timeout = 0
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		if req.URL.Scheme+"://"+req.URL.Host == defaultDevServerOrigin {
			return fmt.Errorf("local dev server is not available at %s; start it with `inngest dev` or use --prod to target Inngest Cloud: %w", defaultDevServerOrigin, err)
		}
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
		if err != nil {
			return err
		}
		if int64(len(body)) > maxResponseBytes {
			return fmt.Errorf("response body exceeded %d bytes", maxResponseBytes)
		}
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	writer := cmd.Root().Writer
	if writer == nil {
		writer = os.Stdout
	}
	if ep.streaming {
		_, err = io.Copy(writer, resp.Body)
		return err
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return err
	}
	if int64(len(body)) > maxResponseBytes {
		return fmt.Errorf("response body exceeded %d bytes", maxResponseBytes)
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

	query, err := queryParams(cmd, ep)
	if err != nil {
		return nil, err
	}
	u.RawQuery = query.Encode()

	var reader io.Reader
	rawBody := isRawHTTPBodyEndpoint(ep)
	if rawBody {
		reader, err = rawHTTPRequestBody(cmd)
		if err != nil {
			return nil, err
		}
	} else if body, err := requestBody(cmd, ep); err != nil {
		return nil, err
	} else if body != nil {
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
	if rawBody {
		req.Header.Set("Content-Type", "application/octet-stream")
	} else if reader != nil {
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

	apiPort := localconfig.GetIntValue(cmd, "api-port", 0)
	if apiHost := localconfig.GetValue(cmd, "api-host", ""); apiHost != "" {
		if apiPort == 0 && !looksLikeURL(apiHost) {
			apiPort = api.DefaultAPIPort
		}
		return normalizeAPIHostTarget(apiHost, apiPort)
	}

	if apiPort != 0 {
		return normalizeAPIHostTarget("localhost", apiPort)
	}

	if localconfig.GetBoolValue(cmd, "prod", false) {
		return cloudAPIURL, nil
	}

	return defaultDevServerURL, nil
}

func normalizeAPIHostTarget(rawHost string, port int) (string, error) {
	if looksLikeURL(rawHost) {
		return normalizeAPIURLWithPort(rawHost, port)
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

func normalizeAPIURLWithPort(rawURL string, port int) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("api host must include scheme and host")
	}
	if port != 0 && parsed.Port() == "" {
		parsed.Host = net.JoinHostPort(parsed.Hostname(), strconv.Itoa(port))
	}

	return normalizeAPIURL(parsed.String())
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
		value, ok := pathParamValue(cmd, ep, name)
		if !ok {
			firstErr = fmt.Errorf("missing required --%s or positional argument <%s>", flagName, flagName)
			return match
		}

		return url.PathEscape(value)
	})

	if firstErr != nil {
		return "", firstErr
	}
	return path, nil
}

func pathParamValue(cmd *cli.Command, _ endpoint, name string) (string, bool) {
	flagName := kebab(name)
	if cmd.IsSet(flagName) && cmd.String(flagName) != "" {
		return cmd.String(flagName), true
	}
	if value := cmd.StringArg(flagName); value != "" {
		return value, true
	}
	return "", false
}

func queryParams(cmd *cli.Command, ep endpoint) (url.Values, error) {
	values := url.Values{}
	if ep.body != "" && !isRawHTTPBodyEndpoint(ep) {
		return values, nil
	}

	fields := ep.input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())
		if slices.Contains(ep.pathParams, name) || name == ep.body {
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

func isRawHTTPBodyEndpoint(ep endpoint) bool {
	if ep.body == "" || ep.body == "*" {
		return false
	}
	field := ep.input.Fields().ByName(protoreflect.Name(ep.body))
	return field != nil && field.Message() != nil && field.Message().FullName() == "google.api.HttpBody"
}

func rawHTTPRequestBody(cmd *cli.Command) (io.Reader, error) {
	if cmd.IsSet("body") && cmd.IsSet("body-file") {
		return nil, errors.New("--body and --body-file cannot both be set")
	}
	if cmd.IsSet("body") {
		return strings.NewReader(cmd.String("body")), nil
	}
	if !cmd.IsSet("body-file") {
		return nil, errors.New("missing --body or --body-file")
	}
	path := cmd.String("body-file")
	if path == "-" {
		reader := cmd.Root().Reader
		if reader == nil {
			reader = os.Stdin
		}
		return reader, nil
	}
	return os.Open(path)
}

func requestBody(cmd *cli.Command, ep endpoint) (map[string]any, error) {
	if ep.body == "" {
		return nil, nil
	}

	body, err := rawBody(cmd)
	if err != nil {
		return nil, err
	}
	if ep.name == "rerun" {
		if err := addRerunFromStepBody(cmd, body); err != nil {
			return nil, err
		}
	}

	fields := ep.input.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())
		if slices.Contains(ep.pathParams, name) {
			continue
		}

		flagName := kebab(name)
		if ep.name == "rerun" && flagName == "from-step" {
			continue
		}
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

func addRerunFromStepBody(cmd *cli.Command, body map[string]any) error {
	if !cmd.IsSet("from-step") {
		if cmd.IsSet("input") {
			return errors.New("--input requires --from-step")
		}
		return nil
	}

	fromStep := map[string]any{"stepId": cmd.String("from-step")}

	if cmd.IsSet("input") {
		var input []any
		if err := json.Unmarshal([]byte(cmd.String("input")), &input); err != nil {
			return fmt.Errorf("--input must be a valid JSON array: %w", err)
		}
		if input == nil {
			return errors.New("--input must be a valid JSON array")
		}
		fromStep["input"] = input
	}
	body["fromStep"] = fromStep
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

func isRequiredField(field protoreflect.FieldDescriptor) bool {
	return apiv2endpoint.IsRequired(field)
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
		if field.Message().FullName() == "google.protobuf.Timestamp" {
			value, err := parseTimestamp(cmd.String(flagName))
			if err != nil {
				return nil, fmt.Errorf("--%s must be an RFC 3339 timestamp: %w", flagName, err)
			}
			return value, nil
		}

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

func parseTimestamp(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, `"`) {
		if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
			return "", err
		}
	}

	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		return "", err
	}
	return value, nil
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
