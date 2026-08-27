package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

var defaultScopes = []string{
	"accounts:read:*",
	"apps:read:*",
	"apps:write:*",
	"environments:read:*",
	"environments:write:*",
	"events:write:*",
	"experiments:read:*",
	"functions:read:*",
	"functions:write:*",
	"insights:read:*",
	"runs:read:*",
	"runs:write:*",
	"sandboxes:read:*",
	"sandboxes:write:*",
	"sessions:read:*",
	"webhooks:read:*",
	"webhooks:write:*",
}

const (
	ClientID      = "inngest-cli"
	cloudIssuer   = "https://api.inngest.com"
	refreshLeeway = 30 * time.Second
	httpTimeout   = 30 * time.Second
)

type Manager struct {
	store      *Store
	httpClient *http.Client
	now        func() time.Time
}

func NewManager() (*Manager, error) {
	store, err := NewStore()
	if err != nil {
		return nil, err
	}
	return &Manager{
		store:      store,
		httpClient: &http.Client{Timeout: httpTimeout},
		now:        time.Now,
	}, nil
}

func (m *Manager) Store() *Store {
	return m.store
}

func (m *Manager) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, m.httpClient)
}

func OAuthConfig(issuer string) *oauth2.Config {
	issuer = strings.TrimRight(issuer, "/")
	return &oauth2.Config{
		ClientID: ClientID,
		Scopes:   DefaultScopes(),
		Endpoint: oauth2.Endpoint{
			DeviceAuthURL: issuer + "/oauth/device/code",
			TokenURL:      issuer + "/oauth/token",
			AuthStyle:     oauth2.AuthStyleInParams,
		},
	}
}

func Issuer() (string, error) {
	raw := strings.TrimSpace(os.Getenv("INNGEST_API_HOST"))
	if raw == "" {
		return cloudIssuer, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("INNGEST_API_HOST must include a scheme and host")
	}
	if err := validateOAuthURL(parsed); err != nil {
		return "", fmt.Errorf("invalid INNGEST_API_HOST: %w", err)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	path := strings.TrimRight(parsed.Path, "/")
	for _, suffix := range []string{"/api/v2", "/v2", "/api"} {
		if strings.HasSuffix(path, suffix) {
			path = strings.TrimSuffix(path, suffix)
			break
		}
	}
	parsed.Path = path
	return strings.TrimRight(parsed.String(), "/"), nil
}

func Resource(issuer string) string {
	return strings.TrimRight(issuer, "/") + "/v2"
}

func DefaultScopes() []string {
	return append([]string(nil), defaultScopes...)
}

func (m *Manager) AccessToken(ctx context.Context, target string) (string, *Metadata, error) {
	metadata, err := m.store.Metadata()
	if err != nil {
		return "", nil, err
	}
	if canonicalResource(target) != canonicalResource(metadata.Resource) {
		return "", nil, errors.New("OAuth session is for a different API host; run `inngest login --force`")
	}
	if err := validateOAuthURLString(metadata.Issuer); err != nil {
		return "", metadata, fmt.Errorf("stored OAuth issuer is invalid: %w", err)
	}
	if err := validateOAuthURLString(metadata.Resource); err != nil {
		return "", metadata, fmt.Errorf("stored OAuth resource is invalid: %w", err)
	}
	_, credential, err := m.store.Load()
	if err != nil {
		return "", metadata, err
	}
	if !metadata.SessionExpiresAt.After(m.now()) {
		return "", metadata, errors.New("OAuth session has expired; run `inngest login`")
	}
	if credential.Expiry.After(m.now().Add(refreshLeeway)) {
		return credential.AccessToken, metadata, nil
	}
	token := &oauth2.Token{
		AccessToken:  credential.AccessToken,
		TokenType:    credential.TokenType,
		RefreshToken: credential.RefreshToken,
		Expiry:       m.now(),
	}
	refreshed, err := OAuthConfig(metadata.Issuer).TokenSource(m.Context(ctx), token).Token()
	if err != nil {
		return "", metadata, fmt.Errorf("refresh OAuth session: %w", err)
	}
	returnedResource := stringExtra(refreshed, "resource")
	if returnedResource != "" && canonicalResource(returnedResource) != canonicalResource(metadata.Resource) {
		return "", metadata, errors.New("authorization server returned a token for a different resource")
	}
	credential = &Credential{
		AccessToken:  refreshed.AccessToken,
		RefreshToken: refreshed.RefreshToken,
		TokenType:    refreshed.TokenType,
		Expiry:       refreshed.Expiry,
	}
	updateMetadataFromToken(metadata, refreshed)
	if err := m.store.Save(*metadata, *credential, metadata.Storage == storageFile); err != nil {
		return "", metadata, err
	}
	return credential.AccessToken, metadata, nil
}

func (m *Manager) Validate(ctx context.Context, metadata *Metadata, accessToken string) error {
	if err := validateOAuthURLString(metadata.Resource); err != nil {
		return fmt.Errorf("stored OAuth resource is invalid: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(metadata.Resource, "/")+"/account", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("OAuth session is no longer valid; run `inngest login`")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("validate OAuth session: %s", resp.Status)
	}
	return nil
}

func (m *Manager) Revoke(ctx context.Context, metadata *Metadata, credential *Credential) error {
	if err := validateOAuthURLString(metadata.Issuer); err != nil {
		return fmt.Errorf("stored OAuth issuer is invalid: %w", err)
	}
	values := url.Values{
		"client_id":       {metadata.ClientID},
		"token":           {credential.RefreshToken},
		"token_type_hint": {"refresh_token"},
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(metadata.Issuer, "/")+"/oauth/revoke",
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("revoke OAuth session: %s", resp.Status)
	}
	return nil
}

func MetadataFromToken(issuer, resource string, token *oauth2.Token) (*Metadata, *Credential, error) {
	if err := validateOAuthURLString(issuer); err != nil {
		return nil, nil, fmt.Errorf("OAuth issuer is invalid: %w", err)
	}
	returnedResource := stringExtra(token, "resource")
	if canonicalResource(returnedResource) != canonicalResource(resource) {
		return nil, nil, errors.New("authorization server returned a token for a different resource")
	}
	metadata := &Metadata{
		Issuer:   issuer,
		Resource: resource,
		ClientID: ClientID,
		Scopes:   strings.Fields(stringExtra(token, "scope")),
	}
	updateMetadataFromToken(metadata, token)
	if metadata.Resource == "" || metadata.SessionID == "" || metadata.AccountID == "" || metadata.SessionExpiresAt.IsZero() {
		return nil, nil, errors.New("authorization server returned incomplete session metadata")
	}
	credential := &Credential{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}
	return metadata, credential, nil
}

func updateMetadataFromToken(metadata *Metadata, token *oauth2.Token) {
	if value := stringExtra(token, "session_id"); value != "" {
		metadata.SessionID = value
	}
	if value := stringExtra(token, "session_expires_at"); value != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
			metadata.SessionExpiresAt = parsed
		}
	}
	if value := stringExtra(token, "account_id"); value != "" {
		metadata.AccountID = value
	}
	if value := stringExtra(token, "account_name"); value != "" {
		metadata.AccountName = value
	}
	if value := stringExtra(token, "resource_boundary_mode"); value != "" {
		metadata.ResourceBoundaryMode = value
	}
	if value := stringExtra(token, "workspace_id"); value != "" {
		metadata.WorkspaceID = &value
	}
	if value := stringExtra(token, "workspace_name"); value != "" {
		metadata.WorkspaceName = value
	}
	if value := stringExtra(token, "scope"); value != "" {
		metadata.Scopes = strings.Fields(value)
	}
}

func stringExtra(token *oauth2.Token, key string) string {
	value := token.Extra(key)
	switch value := value.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func canonicalResource(raw string) string {
	parsed, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil {
		return raw
	}
	if parsed.Path == "/api/v2" {
		parsed.Path = "/v2"
	}
	return parsed.String()
}
