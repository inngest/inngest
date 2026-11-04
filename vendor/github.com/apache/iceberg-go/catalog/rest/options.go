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
	"crypto/tls"
	"net/http"
	"net/url"

	"github.com/apache/iceberg-go"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type Option func(*options)

// TransportFactory creates a transport owned by the REST catalog and returns
// the cleanup function to run when the catalog is closed. The TLS config is
// the one configured with WithTLSConfig.
type TransportFactory func(*tls.Config) (http.RoundTripper, func())

func WithCredential(cred string) Option {
	return func(o *options) {
		o.credential = cred
	}
}

func WithOAuthToken(token string) Option {
	return func(o *options) {
		o.oauthToken = token
	}
}

func WithHeaders(headers map[string]string) Option {
	return func(o *options) {
		o.headers = headers
	}
}

func WithAuthManager(authManager AuthManager) Option {
	return func(o *options) {
		o.authManager = authManager
	}
}

func WithTLSConfig(config *tls.Config) Option {
	return func(o *options) {
		o.tlsConfig = config
	}
}

// WithOAuthTLSConfig sets a separate TLS configuration for the HTTP client
// used to communicate with the OAuth2 server. This is useful when the OAuth2
// server (oauth2-server-uri) is a different host than the catalog and requires
// different TLS settings (e.g. a different CA or client certificate).
//
// If not set, the OAuth2 client reuses the catalog's HTTP client (and its TLS
// configuration).
func WithOAuthTLSConfig(config *tls.Config) Option {
	return func(o *options) {
		o.oauthTLSConfig = config
	}
}

func WithWarehouseLocation(loc string) Option {
	return func(o *options) {
		o.warehouseLocation = loc
	}
}

func WithMetadataLocation(loc string) Option {
	return func(o *options) {
		o.metadataLocation = loc
	}
}

func WithSigV4() Option {
	return func(o *options) {
		o.enableSigv4 = true
		o.sigv4Service = "execute-api"
	}
}

func WithSigV4RegionSvc(region, service string) Option {
	return func(o *options) {
		o.enableSigv4 = true
		o.sigv4Region = region

		if service == "" {
			o.sigv4Service = "execute-api"
		} else {
			o.sigv4Service = service
		}
	}
}

// WithAuthURI sets the OAuth2 token endpoint (oauth2-server-uri). When not set,
// the client falls back to {catalog}/v1/oauth/tokens. This is the programmatic
// equivalent of the oauth2-server-uri property (or its rest.authorization-url
// alias).
func WithAuthURI(uri *url.URL) Option {
	return func(o *options) {
		o.authUri = uri
	}
}

func WithPrefix(prefix string) Option {
	return func(o *options) {
		o.prefix = prefix
	}
}

func WithAwsConfig(cfg aws.Config) Option {
	return func(o *options) {
		o.awsConfig = cfg
		o.awsConfigSet = true
	}
}

func WithScope(scope string) Option {
	return func(o *options) {
		o.scope = scope
	}
}

// WithAudience sets the audience parameter sent in OAuth token requests.
// See RFC 8693 for details on token exchange audiences.
func WithAudience(audience string) Option {
	return func(o *options) {
		o.audience = audience
	}
}

// WithResource sets the resource parameter sent in OAuth token requests.
// See RFC 8707 for details on resource indicators.
func WithResource(resource string) Option {
	return func(o *options) {
		o.resource = resource
	}
}

func WithAdditionalProps(props iceberg.Properties) Option {
	return func(o *options) {
		o.additionalProps = props
	}
}

// WithCustomTransport replaces the internally configured http.Transport with the provided http.RoundTripper.
// Certain options such as WithTLSConfig which modify the default http.Transport will no longer work since the entire transport is replaced.
func WithCustomTransport(transport http.RoundTripper) Option {
	return func(o *options) {
		o.transport = transport
	}
}

// WithTransportFactory configures a catalog-owned transport factory. The
// returned cleanup function is called when the catalog is closed. Use
// WithCustomTransport when the caller owns the transport instead.
func WithTransportFactory(factory TransportFactory) Option {
	return func(o *options) {
		o.transportFactory = factory
	}
}

type options struct {
	awsConfig         aws.Config
	awsConfigSet      bool
	tlsConfig         *tls.Config
	oauthToken        string
	credential        string
	authManager       AuthManager
	warehouseLocation string
	metadataLocation  string
	enableSigv4       bool
	sigv4Region       string
	sigv4Service      string
	prefix            string
	authUri           *url.URL
	scope             string
	audience          string
	resource          string
	transport         http.RoundTripper
	transportFactory  TransportFactory
	headers           map[string]string

	oauthTLSConfig *tls.Config

	additionalProps iceberg.Properties
}
