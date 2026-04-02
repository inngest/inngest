package ociauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"cuelabs.dev/go/oci/ociregistry"
)

// TODO decide on a good value for this.
const oauthClientID = "cuelabs-ociauth"

var ErrNoAuth = fmt.Errorf("no authorization token available to add to request")

// stdTransport implements [http.RoundTripper] by acquiring authorization tokens
// using the flows implemented
// by the usual docker clients. Note that this is _not_ documented as
// part of any official OCI spec.
//
// See https://distribution.github.io/distribution/spec/auth/token/ for an overview.
type stdTransport struct {
	config     Config
	transport  http.RoundTripper
	mu         sync.Mutex
	registries map[string]*registry
}

type StdTransportParams struct {
	// Config represents the underlying configuration file information.
	// It is consulted for authorization information on the hosts
	// to which the HTTP requests are made.
	Config Config

	// HTTPClient is used to make the underlying HTTP requests.
	// If it's nil, [http.DefaultTransport] will be used.
	Transport http.RoundTripper
}

// NewStdTransport returns an [http.RoundTripper] implementation that
// acquires authorization tokens using the flows implemented by the
// usual docker clients. Note that this is _not_ documented as part of
// any official OCI spec.
//
// See https://distribution.github.io/distribution/spec/auth/token/ for an overview.
//
// The RoundTrip method acquires authorization before invoking the
// request. request. It may invoke the request more than once, and can
// use [http.Request.GetBody] to reset the request body if it gets
// consumed.
//
// It ensures that the authorization token used will have at least the
// capability to execute operations in the required scope associated
// with the request context (see [ContextWithRequestInfo]). Any other
// auth scope inside the context (see [ContextWithScope]) may also be
// taken into account when acquiring new tokens.
func NewStdTransport(p StdTransportParams) http.RoundTripper {
	if p.Config == nil {
		p.Config = emptyConfig{}
	}
	if p.Transport == nil {
		p.Transport = http.DefaultTransport
	}
	return &stdTransport{
		config:     p.Config,
		transport:  p.Transport,
		registries: make(map[string]*registry),
	}
}

// registry holds currently known auth information for a registry.
type registry struct {
	host      string
	transport http.RoundTripper
	config    Config
	initOnce  sync.Once
	initErr   error

	// mu guards the fields that follow it.
	mu sync.Mutex

	// wwwAuthenticate holds the Www-Authenticate header from
	// the most recent 401 response. If there was a 401 response
	// that didn't hold such a header, this will still be non-nil
	// but hold a zero authHeader.
	wwwAuthenticate *authHeader

	accessTokens []*scopedToken
	refreshToken string
	basic        *userPass
}

type scopedToken struct {
	// scope holds the scope that the token is good for.
	scope Scope
	// token holds the actual access token.
	token string
	// expires holds when the token expires.
	expires time.Time
}

type userPass struct {
	username string
	password string
}

var forever = time.Date(99999, time.January, 1, 0, 0, 0, 0, time.UTC)

// RoundTrip implements [http.RoundTripper.RoundTrip].
func (a *stdTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// From the [http.RoundTripper] docs:
	//	RoundTrip should not modify the request, except for
	//	consuming and closing the Request's Body.
	req = req.Clone(req.Context())

	// From the [http.RoundTripper] docs:
	//	RoundTrip must always close the body, including on errors, [...]
	needBodyClose := true
	defer func() {
		if needBodyClose && req.Body != nil {
			req.Body.Close()
		}
	}()

	a.mu.Lock()
	r := a.registries[req.URL.Host]
	if r == nil {
		r = &registry{
			host:      req.URL.Host,
			config:    a.config,
			transport: a.transport,
		}
		a.registries[r.host] = r
	}
	a.mu.Unlock()
	if err := r.init(); err != nil {
		return nil, err
	}

	ctx := req.Context()
	requiredScope := RequestInfoFromContext(ctx).RequiredScope
	wantScope := ScopeFromContext(ctx)

	if err := r.setAuthorization(ctx, req, requiredScope, wantScope); err != nil {
		return nil, err
	}
	resp, err := r.transport.RoundTrip(req)

	// The underlying transport should now have closed the request body
	// so we don't have to.
	needBodyClose = false
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	challenge := challengeFromResponse(resp)
	if challenge == nil {
		return resp, nil
	}
	authAdded, tokenAcquired, err := r.setAuthorizationFromChallenge(ctx, req, challenge, requiredScope, wantScope)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	if !authAdded {
		// Couldn't acquire any more authorization than we had initially.
		return resp, nil
	}
	resp.Body.Close()
	// rewind request body if needed and possible.
	if req.GetBody != nil {
		req.Body, err = req.GetBody()
		if err != nil {
			return nil, err
		}
	}
	resp, err = r.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized || !tokenAcquired {
		return resp, nil
	}
	// The server has responded with Unauthorized (401) even though we've just
	// provided a token that it gave us. Treat it as Forbidden (403) instead.
	// TODO include the original body/error as part of the message or message detail?
	resp.Body.Close()
	data, err := json.Marshal(&ociregistry.WireErrors{
		Errors: []ociregistry.WireError{{
			Code_:   ociregistry.ErrDenied.Code(),
			Message: "unauthorized response with freshly acquired auth token",
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot marshal response body: %v", err)
	}
	resp.Header.Set("Content-Type", "application/json")
	resp.ContentLength = int64(len(data))
	resp.Body = io.NopCloser(bytes.NewReader(data))
	resp.StatusCode = http.StatusForbidden
	resp.Status = http.StatusText(resp.StatusCode)
	return resp, nil
}

// setAuthorization sets up authorization on the given request using any
// auth information currently available.
func (r *registry) setAuthorization(ctx context.Context, req *http.Request, requiredScope, wantScope Scope) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Remove tokens that have expired or will expire soon so that
	// the caller doesn't start using a token only for it to expire while it's
	// making the request.
	r.deleteExpiredTokens(time.Now().UTC().Add(time.Second))

	if accessToken := r.accessTokenForScope(requiredScope); accessToken != nil {
		// We have a potentially valid access token. Use it.
		req.Header.Set("Authorization", "Bearer "+accessToken.token)
		return nil
	}
	if r.wwwAuthenticate == nil {
		// We haven't seen a 401 response yet. Avoid putting any
		// basic authorization in the request, because that can mean that
		// the server sends a 401 response without a Www-Authenticate
		// header.
		return nil
	}
	if r.refreshToken != "" && r.wwwAuthenticate.scheme == "bearer" {
		// We've got a refresh token that we can use to try to
		// acquire an access token and we've seen a Www-Authenticate response
		// that tells us how we can use it.

		// TODO we're holding the lock (r.mu) here, which is precluding
		// acquiring several tokens concurrently. We should relax the lock
		// to allow that.

		accessToken, err := r.acquireAccessToken(ctx, requiredScope, wantScope)
		if err != nil {
			// Avoid using %w to wrap the error because we don't want the
			// caller of RoundTrip (usually ociclient) to assume that the
			// error applies to the target server rather than the token server.
			return fmt.Errorf("cannot acquire access token: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		return nil
	}
	if r.wwwAuthenticate.scheme != "bearer" && r.basic != nil {
		req.SetBasicAuth(r.basic.username, r.basic.password)
		return nil
	}
	return nil
}

func (r *registry) setAuthorizationFromChallenge(ctx context.Context, req *http.Request, challenge *authHeader, requiredScope, wantScope Scope) (authAdded, tokenAcquired bool, _ error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wwwAuthenticate = challenge

	switch {
	case r.wwwAuthenticate.scheme == "bearer":
		scope := ParseScope(r.wwwAuthenticate.params["scope"])
		accessToken, err := r.acquireAccessToken(ctx, scope, wantScope.Union(requiredScope))
		if err != nil {
			return false, false, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		return true, true, nil
	case r.basic != nil:
		req.SetBasicAuth(r.basic.username, r.basic.password)
		return true, false, nil
	}
	return false, false, nil
}

// init initializes the registry instance by acquiring auth information from
// the Config, if available. As this might be slow (invoking EntryForRegistry
// can end up invoking slow external commands), we ensure that it's only
// done once.
// TODO it's possible that this could take a very long time, during which
// the outer context is cancelled, but we'll ignore that. We probably shouldn't.
func (r *registry) init() error {
	inner := func() error {
		info, err := r.config.EntryForRegistry(r.host)
		if err != nil {
			return fmt.Errorf("cannot acquire auth info for registry %q: %v", r.host, err)
		}
		r.refreshToken = info.RefreshToken
		if info.AccessToken != "" {
			r.accessTokens = append(r.accessTokens, &scopedToken{
				scope:   UnlimitedScope(),
				token:   info.AccessToken,
				expires: forever,
			})
		}
		if info.Username != "" && info.Password != "" {
			r.basic = &userPass{
				username: info.Username,
				password: info.Password,
			}
		}
		return nil
	}
	r.initOnce.Do(func() {
		r.initErr = inner()
	})
	return r.initErr
}

// acquireAccessToken tries to acquire an access token for authorizing a request.
// The requiredScopeStr parameter indicates the scope that's definitely
// required. This is a string because apparently some servers are picky
// about getting exactly the same scope in the auth request that was
// returned in the challenge. The wantScope parameter indicates
// what scope might be required in the future.
//
// This method assumes that there has been a previous 401 response with
// a Www-Authenticate: Bearer... header.
func (r *registry) acquireAccessToken(ctx context.Context, requiredScope, wantScope Scope) (string, error) {
	scope := requiredScope.Union(wantScope)
	tok, err := r.acquireToken(ctx, scope)
	if err != nil {
		var herr ociregistry.HTTPError
		if !errors.As(err, &herr) || herr.StatusCode() != http.StatusUnauthorized {
			return "", err
		}
		// The documentation says this:
		//
		//	If the client only has a subset of the requested
		// 	access it _must not be considered an error_ as it is
		//	not the responsibility of the token server to
		//	indicate authorization errors as part of this
		//	workflow.
		//
		// However it's apparently not uncommon for servers to reject
		// such requests anyway, so if we've got an unauthorized error
		// and wantScope goes beyond requiredScope, it may be because
		// the server is rejecting the request.
		scope = requiredScope
		tok, err = r.acquireToken(ctx, scope)
		if err != nil {
			return "", err
		}
		// TODO mark the registry as picky about tokens so we don't
		// attempt twice every time?
	}
	if tok.RefreshToken != "" {
		r.refreshToken = tok.RefreshToken
	}
	accessToken := tok.Token
	if accessToken == "" {
		accessToken = tok.AccessToken
	}
	if accessToken == "" {
		return "", fmt.Errorf("no access token found in auth server response")
	}
	var expires time.Time
	now := time.Now().UTC()
	if tok.ExpiresIn == 0 {
		expires = now.Add(60 * time.Second) // TODO link to where this is mentioned
	} else {
		expires = now.Add(time.Duration(tok.ExpiresIn) * time.Second)
	}
	r.accessTokens = append(r.accessTokens, &scopedToken{
		scope:   scope,
		token:   accessToken,
		expires: expires,
	})
	// TODO persist the access token to save round trips when doing
	// the authorization flow in a newly run executable.
	return accessToken, nil
}

func (r *registry) acquireToken(ctx context.Context, scope Scope) (*wireToken, error) {
	realm := r.wwwAuthenticate.params["realm"]
	if realm == "" {
		return nil, fmt.Errorf("malformed Www-Authenticate header (missing realm)")
	}
	if r.refreshToken != "" {
		v := url.Values{}
		v.Set("scope", scope.String())
		if service := r.wwwAuthenticate.params["service"]; service != "" {
			v.Set("service", service)
		}
		v.Set("client_id", oauthClientID)
		v.Set("grant_type", "refresh_token")
		v.Set("refresh_token", r.refreshToken)
		req, err := http.NewRequestWithContext(ctx, "POST", realm, strings.NewReader(v.Encode()))
		if err != nil {
			return nil, fmt.Errorf("cannot form HTTP request to %q: %v", realm, err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		tok, err := r.doTokenRequest(req)
		if err == nil {
			return tok, nil
		}
		var herr ociregistry.HTTPError
		if !errors.As(err, &herr) || herr.StatusCode() != http.StatusNotFound {
			return tok, err
		}
		// The request to the endpoint returned 404 from the POST request,
		// Note: Not all token servers implement oauth2, so fall
		// back to using a GET with basic auth.
		// See the Token documentation for the HTTP GET method supported by all token servers.
		// TODO where in that documentation is this documented?
	}
	u, err := url.Parse(realm)
	if err != nil {
		return nil, fmt.Errorf("malformed Www-Authenticate header (malformed realm %q): %v", realm, err)
	}
	v := u.Query()
	// TODO where is it documented that we should send multiple scope
	// attributes rather than a single space-separated attribute as
	// the POST method does?
	v["scope"] = strings.Split(scope.String(), " ")
	if service := r.wwwAuthenticate.params["service"]; service != "" {
		// TODO the containerregistry code sets this even if it's empty.
		// Is that better?
		v.Set("service", service)
	}
	u.RawQuery = v.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}
	// TODO if there's an unlimited-scope access token, the original code
	// will use it as Bearer authorization at this point. If
	// that's valid, why are we even acquiring another token?
	if r.basic != nil {
		req.SetBasicAuth(r.basic.username, r.basic.password)
	}
	return r.doTokenRequest(req)
}

// wireToken describes the JSON encoding used in the response to a token
// acquisition method. The comments are taken from the [token docs]
// and made available here for ease of reference.
//
// [token docs]: https://distribution.github.io/distribution/spec/auth/token/#token-response-fields
type wireToken struct {
	// Token holds an opaque Bearer token that clients should supply
	// to subsequent requests in the Authorization header.
	// AccessToken is provided for compatibility with OAuth 2.0: it's equivalent to Token.
	// At least one of these fields must be specified, but both may also appear (for compatibility with older clients).
	// When both are specified, they should be equivalent; if they differ the client's choice is undefined.
	Token       string `json:"token"`
	AccessToken string `json:"access_token,omitempty"`

	// Refresh token optionally holds a token which can be used to
	// get additional access tokens for the same subject with different scopes.
	// This token should be kept secure by the client and only sent
	// to the authorization server which issues bearer tokens. This
	// field will only be set when `offline_token=true` is provided
	// in the request.
	RefreshToken string `json:"refresh_token"`

	// ExpiresIn holds the duration in seconds since the token was
	// issued that it will remain valid. When omitted, this defaults
	// to 60 seconds. For compatibility with older clients, a token
	// should never be returned with less than 60 seconds to live.
	ExpiresIn int `json:"expires_in"`
}

func (r *registry) doTokenRequest(req *http.Request) (*wireToken, error) {
	client := &http.Client{
		Transport: r.transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, bodyErr := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, ociregistry.NewHTTPError(nil, resp.StatusCode, resp, data)
	}
	if bodyErr != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}
	var tok wireToken
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("malformed JSON token in response: %v", err)
	}
	return &tok, nil
}

// deleteExpiredTokens removes all tokens from r that expire after the given
// time.
// TODO ask the store to remove expired tokens?
func (r *registry) deleteExpiredTokens(now time.Time) {
	r.accessTokens = slices.DeleteFunc(r.accessTokens, func(tok *scopedToken) bool {
		return now.After(tok.expires)
	})
}

func (r *registry) accessTokenForScope(scope Scope) *scopedToken {
	for _, tok := range r.accessTokens {
		if tok.scope.Contains(scope) {
			// TODO prefer tokens with less scope?
			return tok
		}
	}
	return nil
}

type emptyConfig struct{}

func (emptyConfig) EntryForRegistry(host string) (ConfigEntry, error) {
	return ConfigEntry{}, nil
}
