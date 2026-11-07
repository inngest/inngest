// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package rest

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
)

// AuthManager is an interface for providing custom authorization headers.
type AuthManager interface {
	// AuthHeader returns the key and value for the authorization header.
	AuthHeader() (string, string, error)
}

// ContextAuthManager is an optional interface an AuthManager may implement to
// honor a caller-supplied context (deadline and cancellation) while producing
// the authorization header. sessionTransport prefers AuthHeaderWithContext when
// the manager implements it, so a request's deadline also bounds the auth step
// rather than only the request/response cycle. A manager implementing only
// AuthManager still works, but its auth step cannot be interrupted mid-call —
// bounding that case is the implementer's responsibility.
type ContextAuthManager interface {
	AuthManager
	// AuthHeaderWithContext returns the authorization header, honoring ctx.
	AuthHeaderWithContext(ctx context.Context) (string, string, error)
}

// Oauth2AuthManager is an implementation of the AuthManager interface which
// uses an oauth2.TokenSource to provide bearer tokens. The token source
// handles caching, thread-safe refresh, and expiry management.
type Oauth2AuthManager struct {
	tokenSource oauth2.TokenSource
}

var _ ContextAuthManager = (*Oauth2AuthManager)(nil)

// AuthHeader returns the authorization header with the bearer token.
func (o *Oauth2AuthManager) AuthHeader() (string, string, error) {
	return o.authHeader()
}

// AuthHeaderWithContext returns the authorization header, honoring ctx. A
// cached, unexpired token is returned without any network call, so a bounded
// caller (such as the metrics dispatcher) pays no auth latency in the common
// case. When a refresh is required it goes through the token-refresh HTTP
// client, whose Timeout bounds how long a stalled token endpoint can block; the
// ctx check here additionally short-circuits work for a caller whose deadline
// has already elapsed.
func (o *Oauth2AuthManager) AuthHeaderWithContext(ctx context.Context) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}

	return o.authHeader()
}

func (o *Oauth2AuthManager) authHeader() (string, string, error) {
	tok, err := o.tokenSource.Token()
	if err != nil {
		var re *oauth2.RetrieveError
		if errors.As(err, &re) {
			return "", "", oauthError{
				code: re.ErrorCode,
				desc: re.ErrorDescription,
				uri:  re.ErrorURI,
			}
		}

		return "", "", fmt.Errorf("%w: %s", ErrOAuthError, err)
	}

	return "Authorization", tok.Type() + " " + tok.AccessToken, nil
}

// oauthError wraps OAuth2 error details and implements the error chain
// so that errors.Is(err, ErrOAuthError) returns true.
type oauthError struct {
	code string
	desc string
	uri  string
}

func (e oauthError) Error() string {
	msg := e.code
	if e.desc != "" {
		msg += ": " + e.desc
	}
	if e.uri != "" {
		msg += " (" + e.uri + ")"
	}

	return msg
}

func (e oauthError) Unwrap() error { return ErrOAuthError }
