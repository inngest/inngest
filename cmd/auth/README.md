# CLI authentication

`inngest login` starts OAuth Device Authorization and opens the Dashboard approval page. The user chooses an account's permissions, environment access, and session duration. The CLI receives credentials after approval.

```sh
inngest login
inngest auth status
inngest logout
```

Use `--no-browser` to open the displayed verification URL manually. `--force` replaces an existing login and attempts to revoke the previous session after saving the new one. `--json` emits newline-delimited status events without access or refresh tokens.

Credentials are stored in the operating system's credential store. In environments without a supported credential store, `--insecure-storage` explicitly opts into a plaintext file readable only by the current user. Session metadata lives in `~/.config/inngest`; `INNGEST_CONFIG_DIR` overrides that directory.

API commands automatically use the stored login for the matching API host and refresh short-lived access tokens when necessary. Refreshes are serialized across CLI processes so concurrent commands do not reuse a refresh token.

Credential precedence for API commands is:

1. `--api-key` or `INNGEST_API_KEY`.
2. An explicit `--signing-key`.
3. A stored login for the target API resource.
4. `INNGEST_SIGNING_KEY`.

Use `--env` or `INNGEST_ENV` to select an environment for environment-specific operations when the session allows all environments. A session bound to one environment retains that boundary.

`INNGEST_API_HOST` selects a custom login server. Credentials are never reused for a different API resource. Remote servers require HTTPS; HTTP is supported for loopback addresses during local development.

Logout revokes the stored login session and removes its local credentials. It does not revoke API keys supplied through flags or environment variables. If the server cannot be reached, local credentials are still removed and the CLI reports that the remote session may remain active until expiration.
