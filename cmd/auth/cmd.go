package authcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	cliauth "github.com/inngest/inngest/cmd/internal/auth"
	"github.com/urfave/cli/v3"
	"golang.org/x/oauth2"
)

func LoginCommand() *cli.Command {
	return &cli.Command{
		Name:  "login",
		Usage: "Log in to Inngest Cloud",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-browser",
				Usage: "Do not open the verification page",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Start a new login when already logged in",
			},
			&cli.BoolFlag{
				Name:  "insecure-storage",
				Usage: "Store credentials in a user-only file when no OS credential store is available",
			},
		},
		Action: jsonAction(login),
	}
}

func LogoutCommand() *cli.Command {
	return &cli.Command{
		Name:   "logout",
		Usage:  "Log out of Inngest Cloud",
		Action: jsonAction(logout),
	}
}

func AuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Inspect CLI authentication",
		Commands: []*cli.Command{
			{
				Name:   "status",
				Usage:  "Show the current login",
				Action: jsonAction(status),
			},
		},
	}
}

func login(ctx context.Context, cmd *cli.Command) error {
	manager, err := cliauth.NewManager()
	if err != nil {
		return err
	}
	issuer, err := cliauth.Issuer()
	if err != nil {
		return err
	}
	resource := cliauth.Resource(issuer)
	previousMetadata, previousCredential, loadErr := manager.Store().Load()
	if !cmd.Bool("force") {
		if loadErr == nil {
			accessToken, metadata, tokenErr := manager.AccessToken(ctx, resource)
			if tokenErr == nil && manager.Validate(ctx, metadata, accessToken) == nil {
				return writeStatus(cmd, metadata, true, "already_authenticated")
			}
		} else if !errors.Is(loadErr, cliauth.ErrNotLoggedIn) {
			return loadErr
		}
	}

	oauthConfig := cliauth.OAuthConfig(issuer)
	ctx = manager.Context(ctx)
	device, err := oauthConfig.DeviceAuth(ctx, oauth2.SetAuthURLParam("resource", resource))
	if err != nil {
		return fmt.Errorf("start device login: %w", err)
	}
	if err := writeVerification(cmd, device); err != nil {
		return err
	}
	if !cmd.Bool("no-browser") {
		verificationURL := device.VerificationURIComplete
		if verificationURL == "" {
			verificationURL = device.VerificationURI
		}
		_ = cliauth.OpenBrowser(verificationURL)
	}
	token, err := oauthConfig.DeviceAccessToken(ctx, device, oauth2.SetAuthURLParam("resource", resource))
	if err != nil {
		return fmt.Errorf("complete device login: %w", err)
	}
	metadata, credential, err := cliauth.MetadataFromToken(issuer, resource, token)
	if err != nil {
		return err
	}
	if cmd.Bool("insecure-storage") && !cmd.Bool("json") {
		_, _ = fmt.Fprintln(writer(cmd), "Warning: storing credentials in a plaintext user-only file. Use this only on a trusted machine or ephemeral environment.")
	}
	// save the new login before revoking the old one
	if err := manager.Store().Save(*metadata, *credential, cmd.Bool("insecure-storage")); err != nil {
		// do not leave an unusable server session behind
		_ = manager.Revoke(ctx, metadata, credential)
		if !cmd.Bool("insecure-storage") {
			return fmt.Errorf("%w; retry with --insecure-storage only on a trusted machine", err)
		}
		return err
	}
	if loadErr == nil && previousMetadata.SessionID != metadata.SessionID {
		if err := manager.Revoke(ctx, previousMetadata, previousCredential); err != nil {
			if cmd.Bool("json") {
				if writeErr := writeJSONLine(cmd, map[string]any{
					"type":    "warning",
					"message": "The previous session could not be revoked.",
				}); writeErr != nil {
					return writeErr
				}
			} else {
				_, _ = fmt.Fprintln(writer(cmd), "Warning: the previous session could not be revoked. You can revoke it in the Dashboard.")
			}
		}
	}
	return writeStatus(cmd, metadata, true, "authenticated")
}

func logout(ctx context.Context, cmd *cli.Command) error {
	manager, err := cliauth.NewManager()
	if err != nil {
		return err
	}
	metadata, credential, err := manager.Store().Load()
	if errors.Is(err, cliauth.ErrNotLoggedIn) {
		if cmd.Bool("json") {
			return writeJSONLine(cmd, map[string]any{"type": "logout", "revoked": false})
		}
		_, err = fmt.Fprintln(writer(cmd), "Not logged in.")
		return err
	}
	if err != nil {
		return err
	}
	revokeErr := manager.Revoke(ctx, metadata, credential)
	// local logout must work when the server is unavailable
	if err := manager.Store().Delete(metadata); err != nil {
		return errors.Join(revokeErr, err)
	}
	if cmd.Bool("json") {
		return writeJSONLine(cmd, map[string]any{
			"type":                      "logout",
			"revoked":                   revokeErr == nil,
			"local_credentials_removed": true,
		})
	}
	if revokeErr != nil {
		_, err = fmt.Fprintln(writer(cmd), "Logged out locally. The remote session could not be revoked; revoke it in the Dashboard.")
		return err
	}
	_, err = fmt.Fprintln(writer(cmd), "Logged out.")
	return err
}

func status(ctx context.Context, cmd *cli.Command) error {
	manager, err := cliauth.NewManager()
	if err != nil {
		return err
	}
	metadata, _, err := manager.Store().Load()
	if errors.Is(err, cliauth.ErrNotLoggedIn) {
		if cmd.Bool("json") {
			if err := writeJSONLine(cmd, map[string]any{"type": "auth_status", "authenticated": false}); err != nil {
				return err
			}
			return &ReportedError{}
		}
		return cliauth.ErrNotLoggedIn
	}
	if err != nil {
		return err
	}
	accessToken, metadata, err := manager.AccessToken(ctx, metadata.Resource)
	if err != nil {
		return err
	}
	if err := manager.Validate(ctx, metadata, accessToken); err != nil {
		return err
	}
	return writeStatus(cmd, metadata, true, "auth_status")
}

func writeVerification(cmd *cli.Command, device *oauth2.DeviceAuthResponse) error {
	if cmd.Bool("json") {
		return writeJSONLine(cmd, map[string]any{
			"type":                      "verification",
			"user_code":                 device.UserCode,
			"verification_uri":          device.VerificationURI,
			"verification_uri_complete": device.VerificationURIComplete,
			"expires_at":                device.Expiry.Format(time.RFC3339),
			"interval_seconds":          device.Interval,
		})
	}
	_, err := fmt.Fprintf(
		writer(cmd),
		"Authorize at %s with code %s.\n",
		device.VerificationURI,
		device.UserCode,
	)
	return err
}

func writeStatus(cmd *cli.Command, metadata *cliauth.Metadata, authenticated bool, eventType string) error {
	if cmd.Bool("json") {
		return writeJSONLine(cmd, map[string]any{
			"type":                   eventType,
			"authenticated":          authenticated,
			"account_id":             metadata.AccountID,
			"account_name":           metadata.AccountName,
			"resource":               metadata.Resource,
			"resource_boundary_mode": metadata.ResourceBoundaryMode,
			"workspace_id":           metadata.WorkspaceID,
			"workspace_name":         metadata.WorkspaceName,
			"session_expires_at":     metadata.SessionExpiresAt.Format(time.RFC3339),
		})
	}
	boundary := "all environments"
	if metadata.ResourceBoundaryMode == "single_env" {
		boundary = metadata.WorkspaceName
		if strings.TrimSpace(boundary) == "" && metadata.WorkspaceID != nil {
			boundary = *metadata.WorkspaceID
		}
	}
	account := strings.TrimSpace(metadata.AccountName)
	if account == "" {
		account = metadata.AccountID
	}
	_, err := fmt.Fprintf(
		writer(cmd),
		"Logged in to %s with access to %s. Session expires %s.\n",
		account,
		boundary,
		metadata.SessionExpiresAt.Local().Format(time.RFC1123),
	)
	return err
}

func writeJSONLine(cmd *cli.Command, value any) error {
	return json.NewEncoder(writer(cmd)).Encode(value)
}

func writer(cmd *cli.Command) io.Writer {
	if cmd.Root().Writer != nil {
		return cmd.Root().Writer
	}
	return os.Stdout
}

// stops the cli from printing an error twice
type ReportedError struct{}

func (*ReportedError) Error() string {
	return ""
}

func jsonAction(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		err := action(ctx, cmd)
		if err == nil || !cmd.Bool("json") {
			return err
		}
		var reported *ReportedError
		if errors.As(err, &reported) {
			return err
		}
		if writeErr := writeJSONLine(cmd, map[string]any{"type": "error", "message": err.Error()}); writeErr != nil {
			return writeErr
		}
		return &ReportedError{}
	}
}
