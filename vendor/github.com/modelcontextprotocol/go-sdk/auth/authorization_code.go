// Copyright 2026 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build mcp_go_client_oauth

package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
)

// ClientSecretAuthConfig is used to configure client authentication using client_secret.
// Authentication method will be selected based on the authorization server's supported methods,
// according to the following preference order:
//  1. client_secret_post
//  2. client_secret_basic
type ClientSecretAuthConfig struct {
	// ClientID is the client ID to be used for client authentication.
	ClientID string
	// ClientSecret is the client secret to be used for client authentication.
	ClientSecret string
}

// ClientIDMetadataDocumentConfig is used to configure the Client ID Metadata Document
// based client registration per
// https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization#client-id-metadata-documents.
// See https://client.dev/ for more information.
type ClientIDMetadataDocumentConfig struct {
	// URL is the client identifier URL as per
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-00#section-3.
	URL string
}

// PreregisteredClientConfig is used to configure a pre-registered client per
// https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization#preregistration.
// Currently only "client_secret_basic" and "client_secret_post" authentication methods are supported.
type PreregisteredClientConfig struct {
	// ClientSecretAuthConfig is the client_secret based configuration to be used for client authentication.
	ClientSecretAuthConfig *ClientSecretAuthConfig
}

// DynamicClientRegistrationConfig is used to configure dynamic client registration per
// https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization#dynamic-client-registration.
type DynamicClientRegistrationConfig struct {
	// Metadata to be used in dynamic client registration request as per
	// https://datatracker.ietf.org/doc/html/rfc7591#section-2.
	Metadata *oauthex.ClientRegistrationMetadata
}

// AuthorizationResult is the result of an authorization flow.
// It is returned by [AuthorizationCodeHandler].AuthorizationCodeFetcher implementations.
type AuthorizationResult struct {
	// Code is the authorization code obtained from the authorization server.
	Code string
	// State string returned by the authorization server.
	State string
}

// AuthorizationArgs is the input to [AuthorizationCodeHandlerConfig].AuthorizationCodeFetcher.
type AuthorizationArgs struct {
	// Authorization URL to be opened in a browser for the user to start the authorization process.
	URL string
}

// AuthorizationCodeHandlerConfig is the configuration for [AuthorizationCodeHandler].
type AuthorizationCodeHandlerConfig struct {
	// Client registration configuration.
	// It is attempted in the following order:
	//  1. Client ID Metadata Document
	//  2. Preregistration
	//  3. Dynamic Client Registration
	// At least one method must be configured.
	ClientIDMetadataDocumentConfig  *ClientIDMetadataDocumentConfig
	PreregisteredClientConfig       *PreregisteredClientConfig
	DynamicClientRegistrationConfig *DynamicClientRegistrationConfig

	// RedirectURL is a required URL to redirect to after authorization.
	// The caller is responsible for handling the redirect out of band.
	//
	// If Dynamic Client Registration is used:
	//  - this field is permitted to be empty, in which case it will be set
	//    to the first redirect URI from
	//    DynamicClientRegistrationConfig.Metadata.RedirectURIs.
	//  - if the field is not empty, it must be one of the redirect URIs in
	//    DynamicClientRegistrationConfig.Metadata.RedirectURIs.
	RedirectURL string

	// AuthorizationCodeFetcher is a required function called to initiate the authorization flow.
	// It is responsible for opening the URL in a browser for the user to start the authorization process.
	// It should return the authorization code and state once the Authorization Server
	// redirects back to the RedirectURL.
	AuthorizationCodeFetcher func(ctx context.Context, args *AuthorizationArgs) (*AuthorizationResult, error)

	// Client is an optional HTTP client to use for HTTP requests.
	// It is used for the following requests:
	//  - Fetching Protected Resource Metadata
	//  - Fetching Authorization Server Metadata
	//  - Registering a client dynamically
	//  - Exchanging an authorization code for an access token
	//  - Refreshing an access token
	// Custom clients can include additional security configurations,
	// such as SSRF protections, see
	// https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices#server-side-request-forgery-ssrf
	// If not provided, http.DefaultClient will be used.
	Client *http.Client
}

// AuthorizationCodeHandler is an implementation of [OAuthHandler] that uses
// the authorization code flow to obtain access tokens.
type AuthorizationCodeHandler struct {
	config *AuthorizationCodeHandlerConfig

	// tokenSource is the token source to use for authorization.
	tokenSource oauth2.TokenSource
}

var _ OAuthHandler = (*AuthorizationCodeHandler)(nil)

func (h *AuthorizationCodeHandler) isOAuthHandler() {}

func (h *AuthorizationCodeHandler) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	return h.tokenSource, nil
}

// NewAuthorizationCodeHandler creates a new AuthorizationCodeHandler.
// It performs validation of the configuration and returns an error if it is invalid.
// The passed config is consumed by the handler and should not be modified after.
func NewAuthorizationCodeHandler(config *AuthorizationCodeHandlerConfig) (*AuthorizationCodeHandler, error) {
	if config == nil {
		return nil, errors.New("config must be provided")
	}
	if config.ClientIDMetadataDocumentConfig == nil &&
		config.PreregisteredClientConfig == nil &&
		config.DynamicClientRegistrationConfig == nil {
		return nil, errors.New("at least one client registration configuration must be provided")
	}
	if config.AuthorizationCodeFetcher == nil {
		return nil, errors.New("AuthorizationCodeFetcher is required")
	}
	if config.ClientIDMetadataDocumentConfig != nil && !isNonRootHTTPSURL(config.ClientIDMetadataDocumentConfig.URL) {
		return nil, fmt.Errorf("client ID metadata document URL must be a non-root HTTPS URL")
	}
	preCfg := config.PreregisteredClientConfig
	if preCfg != nil {
		if preCfg.ClientSecretAuthConfig == nil {
			return nil, errors.New("ClientSecretAuthConfig is required for pre-registered client")
		}
		if preCfg.ClientSecretAuthConfig.ClientID == "" || preCfg.ClientSecretAuthConfig.ClientSecret == "" {
			return nil, fmt.Errorf("pre-registered client ID or secret is empty")
		}
	}
	dCfg := config.DynamicClientRegistrationConfig
	if dCfg != nil {
		if dCfg.Metadata == nil {
			return nil, errors.New("Metadata is required for dynamic client registration")
		}
		if len(dCfg.Metadata.RedirectURIs) == 0 {
			return nil, errors.New("Metadata.RedirectURIs is required for dynamic client registration")
		}
		if config.RedirectURL == "" {
			config.RedirectURL = dCfg.Metadata.RedirectURIs[0]
		} else if !slices.Contains(dCfg.Metadata.RedirectURIs, config.RedirectURL) {
			return nil, fmt.Errorf("RedirectURL %q is not in the list of allowed redirect URIs for dynamic client registration", config.RedirectURL)
		}
	}
	if config.RedirectURL == "" {
		// If the RedirectURL was supposed to be set by the dynamic client registration,
		// it should have been set by now. Otherwise, it is required.
		return nil, errors.New("RedirectURL is required")
	}
	if config.Client == nil {
		config.Client = http.DefaultClient
	}
	return &AuthorizationCodeHandler{config: config}, nil
}

func isNonRootHTTPSURL(u string) bool {
	pu, err := url.Parse(u)
	if err != nil {
		return false
	}
	return pu.Scheme == "https" && pu.Path != ""
}

// Authorize performs the authorization flow.
// It is designed to perform the whole Authorization Code Grant flow.
// On success, [AuthorizationCodeHandler.TokenSource] will return a token source with the fetched token.
func (h *AuthorizationCodeHandler) Authorize(ctx context.Context, req *http.Request, resp *http.Response) error {
	defer resp.Body.Close()

	wwwChallenges, err := oauthex.ParseWWWAuthenticate(resp.Header[http.CanonicalHeaderKey("WWW-Authenticate")])
	if err != nil {
		return fmt.Errorf("failed to parse WWW-Authenticate header: %v", err)
	}

	if resp.StatusCode == http.StatusForbidden && errorFromChallenges(wwwChallenges) != "insufficient_scope" {
		// We only want to perform step-up authorization for insufficient_scope errors.
		// Returning nil, so that the call is retried immediately and the response
		// is handled appropriately by the connection.
		// Step-up authorization is defined at
		// https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization#step-up-authorization-flow
		return nil
	}

	prm, err := h.getProtectedResourceMetadata(ctx, wwwChallenges, req.URL.String())
	if err != nil {
		return err
	}

	asm, err := h.getAuthServerMetadata(ctx, prm)
	if err != nil {
		return err
	}

	resolvedClientConfig, err := h.handleRegistration(ctx, asm)
	if err != nil {
		return err
	}

	scps := scopesFromChallenges(wwwChallenges)
	if len(scps) == 0 && len(prm.ScopesSupported) > 0 {
		scps = prm.ScopesSupported
	}

	cfg := &oauth2.Config{
		ClientID:     resolvedClientConfig.clientID,
		ClientSecret: resolvedClientConfig.clientSecret,

		Endpoint: oauth2.Endpoint{
			AuthURL:   asm.AuthorizationEndpoint,
			TokenURL:  asm.TokenEndpoint,
			AuthStyle: resolvedClientConfig.authStyle,
		},
		RedirectURL: h.config.RedirectURL,
		Scopes:      scps,
	}

	authRes, err := h.getAuthorizationCode(ctx, cfg, req.URL.String())
	if err != nil {
		// Purposefully leaving the error unwrappable so it can be handled by the caller.
		return err
	}

	return h.exchangeAuthorizationCode(ctx, cfg, authRes, prm.Resource)
}

// resourceMetadataURLFromChallenges returns a resource metadata URL from the given "WWW-Authenticate" header challenges,
// or the empty string if there is none.
func resourceMetadataURLFromChallenges(cs []oauthex.Challenge) string {
	for _, c := range cs {
		if u := c.Params["resource_metadata"]; u != "" {
			return u
		}
	}
	return ""
}

// scopesFromChallenges returns the scopes from the given "WWW-Authenticate" header challenges.
// It only looks at challenges with the "Bearer" scheme.
func scopesFromChallenges(cs []oauthex.Challenge) []string {
	for _, c := range cs {
		if c.Scheme == "bearer" && c.Params["scope"] != "" {
			return strings.Fields(c.Params["scope"])
		}
	}
	return nil
}

// errorFromChallenges returns the error from the given "WWW-Authenticate" header challenges.
// It only looks at challenges with the "Bearer" scheme.
func errorFromChallenges(cs []oauthex.Challenge) string {
	for _, c := range cs {
		if c.Scheme == "bearer" && c.Params["error"] != "" {
			return c.Params["error"]
		}
	}
	return ""
}

// getProtectedResourceMetadata returns the protected resource metadata.
// If no metadata was found or the fetched metadata fails security checks,
// it returns an error.
func (h *AuthorizationCodeHandler) getProtectedResourceMetadata(ctx context.Context, wwwChallenges []oauthex.Challenge, mcpServerURL string) (*oauthex.ProtectedResourceMetadata, error) {
	var errs []error
	// Use MCP server URL as the resource URI per
	// https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization#canonical-server-uri.
	for _, url := range protectedResourceMetadataURLs(resourceMetadataURLFromChallenges(wwwChallenges), mcpServerURL) {
		prm, err := oauthex.GetProtectedResourceMetadata(ctx, url.URL, url.Resource, h.config.Client)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if prm == nil {
			errs = append(errs, fmt.Errorf("protected resource metadata is nil"))
			continue
		}
		return prm, nil
	}
	return nil, fmt.Errorf("failed to get protected resource metadata: %v", errors.Join(errs...))
}

type prmURL struct {
	// URL represents a URL where Protected Resource Metadata may be retrieved.
	URL string
	// Resource represents the corresponding resource URL for [URL].
	// It is required to perform validation described in RFC 9728, section 3.3.
	Resource string
}

// protectedResourceMetadataURLs returns a list of URLs to try when looking for
// protected resource metadata as mandated by the MCP specification:
// https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization#protected-resource-metadata-discovery-requirements
func protectedResourceMetadataURLs(metadataURL, resourceURL string) []prmURL {
	var urls []prmURL
	if metadataURL != "" {
		urls = append(urls, prmURL{
			URL:      metadataURL,
			Resource: resourceURL,
		})
	}
	ru, err := url.Parse(resourceURL)
	if err != nil {
		return urls
	}
	mu := *ru
	// "At the path of the server's MCP endpoint".
	mu.Path = "/.well-known/oauth-protected-resource/" + strings.TrimLeft(ru.Path, "/")
	urls = append(urls, prmURL{
		URL:      mu.String(),
		Resource: resourceURL,
	})
	// "At the root".
	mu.Path = "/.well-known/oauth-protected-resource"
	ru.Path = ""
	urls = append(urls, prmURL{
		URL:      mu.String(),
		Resource: ru.String(),
	})
	return urls
}

// getAuthServerMetadata returns the authorization server metadata.
// The provided Protected Resource Metadata must not be nil.
// It returns an error if the metadata request fails with non-4xx HTTP status code
// or the fetched metadata fails security checks.
// If no metadata was found, it returns a minimal set of endpoints
// as a fallback to 2025-03-26 spec.
func (h *AuthorizationCodeHandler) getAuthServerMetadata(ctx context.Context, prm *oauthex.ProtectedResourceMetadata) (*oauthex.AuthServerMeta, error) {
	var authServerURL string
	if len(prm.AuthorizationServers) > 0 {
		// Use the first authorization server, similarly to other SDKs.
		authServerURL = prm.AuthorizationServers[0]
	} else {
		// Fallback to 2025-03-26 spec: MCP server base URL acts as Authorization Server.
		authURL, err := url.Parse(prm.Resource)
		if err != nil {
			return nil, fmt.Errorf("failed to parse resource URL: %v", err)
		}
		authURL.Path = ""
		authServerURL = authURL.String()
	}

	for _, u := range authorizationServerMetadataURLs(authServerURL) {
		asm, err := oauthex.GetAuthServerMeta(ctx, u, authServerURL, h.config.Client)
		if err != nil {
			return nil, fmt.Errorf("failed to get authorization server metadata: %w", err)
		}
		if asm != nil {
			return asm, nil
		}
	}

	// Fallback to 2025-03-26 spec: predefined endpoints.
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/authorization#fallbacks-for-servers-without-metadata-discovery
	asm := &oauthex.AuthServerMeta{
		Issuer:                authServerURL,
		AuthorizationEndpoint: authServerURL + "/authorize",
		TokenEndpoint:         authServerURL + "/token",
		RegistrationEndpoint:  authServerURL + "/register",
	}
	return asm, nil
}

// authorizationServerMetadataURLs returns a list of URLs to try when looking for
// authorization server metadata as mandated by the MCP specification:
// https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization#authorization-server-metadata-discovery.
func authorizationServerMetadataURLs(issuerURL string) []string {
	var urls []string

	baseURL, err := url.Parse(issuerURL)
	if err != nil {
		return nil
	}

	if baseURL.Path == "" {
		// "OAuth 2.0 Authorization Server Metadata".
		baseURL.Path = "/.well-known/oauth-authorization-server"
		urls = append(urls, baseURL.String())
		// "OpenID Connect Discovery 1.0".
		baseURL.Path = "/.well-known/openid-configuration"
		urls = append(urls, baseURL.String())
		return urls
	}

	originalPath := baseURL.Path
	// "OAuth 2.0 Authorization Server Metadata with path insertion".
	baseURL.Path = "/.well-known/oauth-authorization-server/" + strings.TrimLeft(originalPath, "/")
	urls = append(urls, baseURL.String())
	// "OpenID Connect Discovery 1.0 with path insertion".
	baseURL.Path = "/.well-known/openid-configuration/" + strings.TrimLeft(originalPath, "/")
	urls = append(urls, baseURL.String())
	// "OpenID Connect Discovery 1.0 with path appending".
	baseURL.Path = "/" + strings.Trim(originalPath, "/") + "/.well-known/openid-configuration"
	urls = append(urls, baseURL.String())
	return urls
}

type registrationType int

const (
	registrationTypeClientIDMetadataDocument registrationType = iota
	registrationTypePreregistered
	registrationTypeDynamic
)

type resolvedClientConfig struct {
	registrationType registrationType
	clientID         string
	clientSecret     string
	authStyle        oauth2.AuthStyle
}

func selectTokenAuthMethod(supported []string) oauth2.AuthStyle {
	prefOrder := []string{
		// Preferred in OAuth 2.1 draft: https://www.ietf.org/archive/id/draft-ietf-oauth-v2-1-14.html#name-client-secret.
		"client_secret_post",
		"client_secret_basic",
	}
	for _, method := range prefOrder {
		if slices.Contains(supported, method) {
			return authMethodToStyle(method)
		}
	}
	return oauth2.AuthStyleAutoDetect
}

func authMethodToStyle(method string) oauth2.AuthStyle {
	switch method {
	case "client_secret_post":
		return oauth2.AuthStyleInParams
	case "client_secret_basic":
		return oauth2.AuthStyleInHeader
	case "none":
		// "none" is equivalent to "client_secret_post" but without sending client secret.
		return oauth2.AuthStyleInParams
	default:
		// "client_secret_basic" is the default per https://datatracker.ietf.org/doc/html/rfc7591#section-2.
		return oauth2.AuthStyleInHeader
	}
}

// handleRegistration handles client registration.
// The provided authorization server metadata must be non-nil.
// Support for different registration methods is defined as follows:
//   - Client ID Metadata Document: metadata must have
//     `ClientIDMetadataDocumentSupported` set to true.
//   - Pre-registered client: assumed to be supported.
//   - Dynamic client registration: metadata must have
//     `RegistrationEndpoint` set to a non-empty value.
func (h *AuthorizationCodeHandler) handleRegistration(ctx context.Context, asm *oauthex.AuthServerMeta) (*resolvedClientConfig, error) {
	// 1. Attempt to use Client ID Metadata Document (SEP-991).
	cimdCfg := h.config.ClientIDMetadataDocumentConfig
	if cimdCfg != nil && asm.ClientIDMetadataDocumentSupported {
		return &resolvedClientConfig{
			registrationType: registrationTypeClientIDMetadataDocument,
			clientID:         cimdCfg.URL,
		}, nil
	}
	// 2. Attempt to use pre-registered client configuration.
	pCfg := h.config.PreregisteredClientConfig
	if pCfg != nil {
		authStyle := selectTokenAuthMethod(asm.TokenEndpointAuthMethodsSupported)
		return &resolvedClientConfig{
			registrationType: registrationTypePreregistered,
			clientID:         pCfg.ClientSecretAuthConfig.ClientID,
			clientSecret:     pCfg.ClientSecretAuthConfig.ClientSecret,
			authStyle:        authStyle,
		}, nil
	}
	// 3. Attempt to use dynamic client registration.
	dcrCfg := h.config.DynamicClientRegistrationConfig
	if dcrCfg != nil && asm.RegistrationEndpoint != "" {
		regResp, err := oauthex.RegisterClient(ctx, asm.RegistrationEndpoint, dcrCfg.Metadata, h.config.Client)
		if err != nil {
			return nil, fmt.Errorf("failed to register client: %w", err)
		}
		cfg := &resolvedClientConfig{
			registrationType: registrationTypeDynamic,
			clientID:         regResp.ClientID,
			clientSecret:     regResp.ClientSecret,
			authStyle:        authMethodToStyle(regResp.TokenEndpointAuthMethod),
		}
		return cfg, nil
	}
	return nil, fmt.Errorf("no configured client registration methods are supported by the authorization server")
}

type authResult struct {
	*AuthorizationResult
	// usedCodeVerifier is the PKCE code verifier used to obtain the authorization code.
	// It is preserved for the token exchange step.
	usedCodeVerifier string
}

// getAuthorizationCode uses the [AuthorizationCodeHandler.AuthorizationCodeFetcher]
// to obtain an authorization code.
func (h *AuthorizationCodeHandler) getAuthorizationCode(ctx context.Context, cfg *oauth2.Config, resourceURL string) (*authResult, error) {
	codeVerifier := oauth2.GenerateVerifier()
	state := rand.Text()

	authURL := cfg.AuthCodeURL(state,
		oauth2.S256ChallengeOption(codeVerifier),
		oauth2.SetAuthURLParam("resource", resourceURL),
	)

	authRes, err := h.config.AuthorizationCodeFetcher(ctx, &AuthorizationArgs{URL: authURL})
	if err != nil {
		// Purposefully leaving the error unwrappable so it can be handled by the caller.
		return nil, err
	}
	if authRes.State != state {
		return nil, fmt.Errorf("state mismatch")
	}
	return &authResult{
		AuthorizationResult: authRes,
		usedCodeVerifier:    codeVerifier,
	}, nil
}

// exchangeAuthorizationCode exchanges the authorization code for a token
// and stores it in a token source.
func (h *AuthorizationCodeHandler) exchangeAuthorizationCode(ctx context.Context, cfg *oauth2.Config, authResult *authResult, resourceURL string) error {
	opts := []oauth2.AuthCodeOption{
		oauth2.VerifierOption(authResult.usedCodeVerifier),
		oauth2.SetAuthURLParam("resource", resourceURL),
	}
	clientCtx := context.WithValue(ctx, oauth2.HTTPClient, h.config.Client)
	token, err := cfg.Exchange(clientCtx, authResult.Code, opts...)
	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}
	h.tokenSource = cfg.TokenSource(clientCtx, token)
	return nil
}
