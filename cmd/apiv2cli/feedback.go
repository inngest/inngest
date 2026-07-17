package apiv2cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	localconfig "github.com/inngest/inngest/cmd/internal/config"
	"github.com/inngest/inngest/pkg/api"
	"github.com/urfave/cli/v3"
)

const feedbackSourceCLI = "cli"
const feedbackOperationCLI = "submit-feedback"

// FeedbackCommand returns a top-level `inngest feedback` command that submits
// product feedback to Inngest Cloud (POST /v2/feedback). Defaults to cloud
// since feedback is cloud-only.
func FeedbackCommand() *cli.Command {
	return &cli.Command{
		Name:      "feedback",
		Usage:     "Send product feedback to the Inngest team",
		UsageText: "inngest feedback [message] [flags]",
		Description: strings.Join([]string{
			"Submit product feedback to the Inngest team.",
			"Feedback is delivered to Inngest Cloud only; by default this command targets production.",
			"",
			"Examples:",
			"  inngest feedback \"Pagination examples would help\"",
			"  echo \"Love the new runs API\" | inngest feedback",
			"  inngest feedback --email you@example.com \"Great docs\"",
		}, "\n"),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Category: "Target",
				Name:     "api-host",
				Usage:    "Custom API host or origin; may include /api/v2 or /v2",
			},
			&cli.IntFlag{
				Category: "Target",
				Name:     "api-port",
				Usage:    "Custom API port",
			},
			&cli.StringFlag{
				Category: "Auth",
				Name:     "api-key",
				Usage:    "Optional API key sent as a Bearer token; may attach account context",
				Sources:  cli.EnvVars("INNGEST_API_KEY"),
			},
			&cli.StringFlag{
				Category: "Auth",
				Name:     "signing-key",
				Usage:    "Optional signing key sent as a Bearer token; may attach account context",
				Sources:  cli.EnvVars("INNGEST_SIGNING_KEY"),
			},
			&cli.StringFlag{
				Category: "Auth",
				Name:     "env",
				Usage:    "Environment name sent as X-Inngest-Env",
				Sources:  cli.EnvVars("INNGEST_ENV"),
			},
			&cli.StringFlag{
				Name:  "email",
				Usage: "Optional contact email included with the feedback",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Optional name included with the feedback",
			},
			&cli.StringFlag{
				Name:  "operation",
				Usage: "Optional API operation or CLI command this feedback relates to",
			},
			&cli.DurationFlag{
				Category: "Target",
				Name:     "timeout",
				Usage:    "HTTP request timeout",
				Value:    defaultTimeout,
			},
		},
		Action: runFeedback,
	}
}

func runFeedback(ctx context.Context, cmd *cli.Command) error {
	message, err := feedbackMessage(cmd)
	if err != nil {
		return err
	}

	baseURL, err := resolveFeedbackBaseURL(ctx, cmd)
	if err != nil {
		return err
	}

	body := map[string]any{
		"feedback":  message,
		"source":    feedbackSourceCLI,
		"operation": feedbackOperationCLI,
	}
	if operation := strings.TrimSpace(cmd.String("operation")); operation != "" {
		normalized, err := normalizeFeedbackOperation(operation)
		if err != nil {
			return err
		}
		body["operation"] = normalized
	}
	if email := strings.TrimSpace(cmd.String("email")); email != "" {
		body["email"] = email
	}
	if name := strings.TrimSpace(cmd.String("name")); name != "" {
		body["name"] = name
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}

	u := strings.TrimRight(baseURL, "/") + "/feedback"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	token, err := authToken(cmd)
	if err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("provide --api-key or --signing-key to submit authenticated feedback")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	if env := cmd.String("env"); env != "" {
		req.Header.Set("X-Inngest-Env", env)
	}
	if err := guardPlaintextAuth(req); err != nil {
		return err
	}

	timeout := cmd.Duration("timeout")
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit feedback: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return err
	}
	if int64(len(respBody)) > maxResponseBytes {
		return fmt.Errorf("response body exceeded %d bytes", maxResponseBytes)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	writer := cmd.Root().Writer
	if writer == nil {
		writer = os.Stdout
	}
	_, err = fmt.Fprintln(writer, "Thanks - your feedback was sent.")
	return err
}

// resolveFeedbackBaseURL defaults to Inngest Cloud. Custom --api-host/--api-port
// still work for staging or local monorepo testing.
func resolveFeedbackBaseURL(ctx context.Context, cmd *cli.Command) (string, error) {
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

	return cloudAPIURL, nil
}

func feedbackMessage(cmd *cli.Command) (string, error) {
	parts := cmd.Args().Slice()
	if len(parts) > 0 {
		msg := strings.TrimSpace(strings.Join(parts, " "))
		if msg == "" {
			return "", fmt.Errorf("feedback message cannot be empty")
		}
		return msg, nil
	}

	reader := cmd.Root().Reader
	if reader == nil {
		reader = os.Stdin
	}
	if isInteractiveReader(reader) {
		return "", fmt.Errorf("provide a feedback message as an argument or pipe it on stdin")
	}

	data, err := io.ReadAll(io.LimitReader(reader, 64<<10))
	if err != nil {
		return "", fmt.Errorf("failed to read feedback from stdin: %w", err)
	}
	msg := strings.TrimSpace(string(data))
	if msg == "" {
		return "", fmt.Errorf("feedback message cannot be empty")
	}
	return msg, nil
}

func isInteractiveReader(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func normalizeFeedbackOperation(value string) (string, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", nil
	}
	for _, ep := range discoverEndpoints() {
		if normalized == ep.name {
			return ep.name, nil
		}
	}
	return "", fmt.Errorf("unknown operation %q", value)
}
