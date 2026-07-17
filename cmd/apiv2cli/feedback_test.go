package apiv2cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestFeedbackMessageFromArgs(t *testing.T) {
	cmd := &cli.Command{
		Name: "feedback",
		Action: func(ctx context.Context, c *cli.Command) error {
			msg, err := feedbackMessage(c)
			require.NoError(t, err)
			require.Equal(t, "hello world", msg)
			return nil
		},
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"feedback", "hello", "world"}))
}

func TestFeedbackMessageFromStdin(t *testing.T) {
	cmd := &cli.Command{
		Name:   "feedback",
		Reader: strings.NewReader("  from stdin  \n"),
		Action: func(ctx context.Context, c *cli.Command) error {
			msg, err := feedbackMessage(c)
			require.NoError(t, err)
			require.Equal(t, "from stdin", msg)
			return nil
		},
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"feedback"}))
}

func TestFeedbackMessageEmpty(t *testing.T) {
	cmd := &cli.Command{
		Name:   "feedback",
		Reader: strings.NewReader("   \n"),
		Action: func(ctx context.Context, c *cli.Command) error {
			_, err := feedbackMessage(c)
			require.Error(t, err)
			return nil
		},
	}
	require.NoError(t, cmd.Run(context.Background(), []string{"feedback"}))
}

func TestRunFeedbackPostsToCloud(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		defer r.Body.Close()
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"accepted":true},"metadata":{}}`))
	}))
	t.Cleanup(server.Close)

	var out bytes.Buffer
	cmd := FeedbackCommand()
	cmd.Writer = &out
	cmd.Reader = io.NopCloser(strings.NewReader(""))

	err := cmd.Run(context.Background(), []string{
		"feedback",
		"--api-host", server.URL + "/v2",
		"--api-key", "test-key",
		"--email", "dev@example.com",
		"--name", "Dev",
		"please add more examples",
	})
	require.NoError(t, err)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/v2/feedback", gotPath)
	require.Equal(t, "Bearer test-key", gotAuth)
	require.Equal(t, "please add more examples", gotBody["feedback"])
	require.Equal(t, "cli", gotBody["source"])
	require.Equal(t, "dev@example.com", gotBody["email"])
	require.Equal(t, "Dev", gotBody["name"])
	require.Contains(t, out.String(), "Thanks")
}

func TestRunFeedbackRequiresAuth(t *testing.T) {
	cmd := FeedbackCommand()
	cmd.Reader = io.NopCloser(strings.NewReader(""))

	err := cmd.Run(context.Background(), []string{
		"feedback",
		"--api-host", "http://localhost:1",
		"please add more examples",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "provide --api-key or --signing-key")
}

func TestResolveFeedbackBaseURLUsesDefaultPortForBareLocalHost(t *testing.T) {
	cmd := FeedbackCommand()
	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		baseURL, err := resolveFeedbackBaseURL(ctx, c)
		require.NoError(t, err)
		require.Equal(t, "http://localhost:8288/api/v2", baseURL)
		return nil
	}

	require.NoError(t, cmd.Run(context.Background(), []string{"feedback", "--api-host", "localhost", "hello"}))
}
