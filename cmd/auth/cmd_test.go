package authcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cliauth "github.com/inngest/inngest/cmd/internal/auth"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestJSONStatusIsOneLineAndFailsWhenLoggedOut(t *testing.T) {
	t.Setenv("INNGEST_CONFIG_DIR", t.TempDir())
	output := bytes.Buffer{}
	command := &cli.Command{
		Name:   "inngest",
		Writer: &output,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json"},
		},
		Commands: []*cli.Command{AuthCommand()},
	}

	err := command.Run(context.Background(), []string{"inngest", "--json", "auth", "status"})
	var reported *ReportedError
	require.True(t, errors.As(err, &reported))

	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n"))
	require.Len(t, lines, 1)
	result := map[string]any{}
	require.NoError(t, json.Unmarshal(lines[0], &result))
	require.Equal(t, "auth_status", result["type"])
	require.Equal(t, false, result["authenticated"])
}

func TestLogoutRemovesLocalCredentialsWhenRevocationFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	t.Setenv("INNGEST_CONFIG_DIR", t.TempDir())
	manager, err := cliauth.NewManager()
	require.NoError(t, err)
	metadata := cliauth.Metadata{
		Issuer:           server.URL,
		Resource:         server.URL + "/v2",
		ClientID:         cliauth.ClientID,
		SessionID:        "session-id",
		SessionExpiresAt: time.Now().Add(time.Hour),
		AccountID:        "account-id",
	}
	credential := cliauth.Credential{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	require.NoError(t, manager.Store().Save(metadata, credential, true))
	output := bytes.Buffer{}
	command := &cli.Command{
		Name:     "inngest",
		Writer:   &output,
		Commands: []*cli.Command{LogoutCommand()},
	}

	err = command.Run(context.Background(), []string{"inngest", "logout"})

	require.NoError(t, err)
	require.Contains(t, output.String(), "Logged out locally")
	_, err = manager.Store().Metadata()
	require.ErrorIs(t, err, cliauth.ErrNotLoggedIn)
}
