package authcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

func TestDeviceLoginAndLogout(t *testing.T) {
	t.Setenv("INNGEST_CONFIG_DIR", t.TempDir())
	// Explicit credentials are not part of the login-managed session.
	t.Setenv("INNGEST_API_KEY", "unrelated-api-key")
	var revoked atomic.Bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/device/code":
			require.Equal(t, cliauth.ClientID, r.PostForm.Get("client_id"))
			require.Equal(t, server.URL+"/v2", r.PostForm.Get("resource"))
			require.NotContains(t, r.PostForm.Get("scope"), "event_keys:")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"device_code": "device-secret", "user_code": "ABCD-EFGH",
				"verification_uri": server.URL + "/oauth/device",
				"expires_in":       30, "interval": 1,
			}))
		case "/oauth/token":
			require.Equal(t, "urn:ietf:params:oauth:grant-type:device_code", r.PostForm.Get("grant_type"))
			require.Equal(t, "device-secret", r.PostForm.Get("device_code"))
			require.Equal(t, server.URL+"/v2", r.PostForm.Get("resource"))
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"access_token": "access-secret", "refresh_token": "refresh-secret",
				"token_type": "Bearer", "expires_in": 3600,
				"resource": server.URL + "/v2", "scope": "apps:read:*",
				"session_id": "session-id", "account_id": "account-id",
				"account_name": "Test account", "resource_boundary_mode": "all_envs",
				"session_expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			}))
		case "/oauth/revoke":
			require.Equal(t, cliauth.ClientID, r.PostForm.Get("client_id"))
			require.Equal(t, "refresh-secret", r.PostForm.Get("token"))
			revoked.Store(true)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	t.Setenv("INNGEST_API_HOST", server.URL)
	var output bytes.Buffer
	command := &cli.Command{
		Name: "inngest", Writer: &output,
		Flags:    []cli.Flag{&cli.BoolFlag{Name: "json"}},
		Commands: []*cli.Command{LoginCommand()},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, command.Run(ctx, []string{"inngest", "--json", "login", "--no-browser", "--insecure-storage"}))
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	require.Len(t, lines, 2)
	for i, wantType := range []string{"verification", "authenticated"} {
		var event map[string]any
		require.NoError(t, json.Unmarshal([]byte(lines[i]), &event))
		require.Equal(t, wantType, event["type"])
	}
	for _, secret := range []string{"access-secret", "refresh-secret", "device-secret", "unrelated-api-key"} {
		require.NotContains(t, output.String(), secret)
	}
	manager, err := cliauth.NewManager()
	require.NoError(t, err)
	metadata, credential, err := manager.Store().Load()
	require.NoError(t, err)
	require.Equal(t, "session-id", metadata.SessionID)
	require.Equal(t, "refresh-secret", credential.RefreshToken)
	command = &cli.Command{Name: "inngest", Writer: &output, Commands: []*cli.Command{LogoutCommand()}}
	require.NoError(t, command.Run(ctx, []string{"inngest", "logout"}))
	require.True(t, revoked.Load())
	_, err = manager.Store().Metadata()
	require.ErrorIs(t, err, cliauth.ErrNotLoggedIn)
}

func TestLogoutWaitsForRefresh(t *testing.T) {
	t.Setenv("INNGEST_CONFIG_DIR", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	refreshStarted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	revoked := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		switch r.URL.Path {
		case "/oauth/token":
			close(refreshStarted)
			select {
			case <-releaseRefresh:
			case <-ctx.Done():
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "new-access", "refresh_token": "new-refresh",
				"token_type": "Bearer", "expires_in": 3600,
			})
		case "/oauth/revoke":
			revoked <- r.PostForm.Get("token")
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	manager, err := cliauth.NewManager()
	require.NoError(t, err)
	metadata := cliauth.Metadata{
		Issuer: server.URL, Resource: server.URL + "/v2", ClientID: cliauth.ClientID,
		SessionID: "session", SessionExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, manager.Store().Save(metadata, cliauth.Credential{
		AccessToken: "old-access", RefreshToken: "old-refresh", Expiry: time.Now().Add(-time.Minute),
	}, true))
	refreshDone := make(chan error, 1)
	go func() {
		_, _, err := manager.AccessToken(ctx, metadata.Resource)
		refreshDone <- err
	}()
	select {
	case <-refreshStarted:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var output bytes.Buffer
	command := &cli.Command{Name: "inngest", Writer: &output, Commands: []*cli.Command{LogoutCommand()}}
	logoutDone := make(chan error, 1)
	go func() { logoutDone <- command.Run(ctx, []string{"inngest", "logout"}) }()
	select {
	case <-revoked:
		t.Fatal("logout raced with refresh")
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseRefresh)
	require.NoError(t, <-refreshDone)
	require.NoError(t, <-logoutDone)
	require.Equal(t, "new-refresh", <-revoked)
	_, err = manager.Store().Metadata()
	require.ErrorIs(t, err, cliauth.ErrNotLoggedIn)
}
