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
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// ErrEndpointNotSupported means the server did not advertise an endpoint the
// operation needs.
var ErrEndpointNotSupported = errors.New("endpoint not supported by server")

const pathPrefix = "/v1/{prefix}"

// endpoint identifies a REST catalog operation by method and path template. It
// both gates capability and builds the request path.
type endpoint struct {
	method string
	path   string
}

func (e endpoint) String() string { return e.method + " " + e.path }

// nparams is the number of "{...}" placeholders reqPath expects.
func (e endpoint) nparams() int {
	n := 0
	for s := range strings.SplitSeq(strings.TrimPrefix(e.path, pathPrefix+"/"), "/") {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			n++
		}
	}

	return n
}

// reqPath fills the "{...}" placeholders with params in order, returning the
// path relative to the base URI. It errors on a param count mismatch.
func (e endpoint) reqPath(params ...string) ([]string, error) {
	if n := e.nparams(); len(params) != n {
		return nil, fmt.Errorf("%w: endpoint %s expects %d path parameter(s), got %d",
			ErrRESTError, e, n, len(params))
	}

	segs := strings.Split(strings.TrimPrefix(e.path, pathPrefix+"/"), "/")

	out := make([]string, len(segs))
	p := 0
	for i, s := range segs {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			out[i], p = params[p], p+1
		} else {
			out[i] = s
		}
	}

	return out, nil
}

// endpointFromString parses an endpoint from its "METHOD PATH" wire form.
func endpointFromString(s string) (endpoint, error) {
	fields := strings.Fields(s)
	if len(fields) != 2 {
		return endpoint{}, fmt.Errorf("%w: invalid endpoint %q (expected \"METHOD PATH\")", ErrRESTError, s)
	}

	return endpoint{method: fields[0], path: fields[1]}, nil
}

// Well-known REST catalog endpoints, keyed by advertised method and path template.
var (
	endpointListNamespaces    = endpoint{http.MethodGet, "/v1/{prefix}/namespaces"}
	endpointLoadNamespace     = endpoint{http.MethodGet, "/v1/{prefix}/namespaces/{namespace}"}
	endpointNamespaceExists   = endpoint{http.MethodHead, "/v1/{prefix}/namespaces/{namespace}"}
	endpointCreateNamespace   = endpoint{http.MethodPost, "/v1/{prefix}/namespaces"}
	endpointUpdateNamespace   = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/properties"}
	endpointDeleteNamespace   = endpoint{http.MethodDelete, "/v1/{prefix}/namespaces/{namespace}"}
	endpointCommitTransaction = endpoint{http.MethodPost, "/v1/{prefix}/transactions/commit"}

	endpointListTables       = endpoint{http.MethodGet, "/v1/{prefix}/namespaces/{namespace}/tables"}
	endpointLoadTable        = endpoint{http.MethodGet, "/v1/{prefix}/namespaces/{namespace}/tables/{table}"}
	endpointTableExists      = endpoint{http.MethodHead, "/v1/{prefix}/namespaces/{namespace}/tables/{table}"}
	endpointCreateTable      = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/tables"}
	endpointUpdateTable      = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/tables/{table}"}
	endpointDeleteTable      = endpoint{http.MethodDelete, "/v1/{prefix}/namespaces/{namespace}/tables/{table}"}
	endpointRenameTable      = endpoint{http.MethodPost, "/v1/{prefix}/tables/rename"}
	endpointRegisterTable    = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/register"}
	endpointReportMetrics    = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/tables/{table}/metrics"}
	endpointTableCredentials = endpoint{http.MethodGet, "/v1/{prefix}/namespaces/{namespace}/tables/{table}/credentials"}
	endpointPlanTableScan    = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/tables/{table}/plan"}
	endpointFetchPlanResult  = endpoint{http.MethodGet, "/v1/{prefix}/namespaces/{namespace}/tables/{table}/plan/{plan-id}"}
	endpointCancelPlanning   = endpoint{http.MethodDelete, "/v1/{prefix}/namespaces/{namespace}/tables/{table}/plan/{plan-id}"}
	endpointFetchScanTasks   = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/tables/{table}/tasks"}

	endpointListViews    = endpoint{http.MethodGet, "/v1/{prefix}/namespaces/{namespace}/views"}
	endpointLoadView     = endpoint{http.MethodGet, "/v1/{prefix}/namespaces/{namespace}/views/{view}"}
	endpointViewExists   = endpoint{http.MethodHead, "/v1/{prefix}/namespaces/{namespace}/views/{view}"}
	endpointCreateView   = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/views"}
	endpointUpdateView   = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/views/{view}"}
	endpointDeleteView   = endpoint{http.MethodDelete, "/v1/{prefix}/namespaces/{namespace}/views/{view}"}
	endpointRenameView   = endpoint{http.MethodPost, "/v1/{prefix}/views/rename"}
	endpointRegisterView = endpoint{http.MethodPost, "/v1/{prefix}/namespaces/{namespace}/register-view"}

	// Function (SQL UDF) endpoints. The spec keeps these out of the assumed
	// default endpoint set, so they are only used when the server advertises
	// them.
	endpointListFunctions = endpoint{http.MethodGet, "/v1/{prefix}/namespaces/{namespace}/functions"}
	endpointLoadFunction  = endpoint{http.MethodGet, "/v1/{prefix}/namespaces/{namespace}/functions/{function}"}
)

// defaultEndpoints is the spec default set, assumed when the server advertises
// no endpoints. The HEAD existence endpoints are excluded on purpose: their
// absence is what makes Check*Exists fall back to GET.
var defaultEndpoints = []endpoint{
	endpointListNamespaces, endpointLoadNamespace, endpointCreateNamespace,
	endpointUpdateNamespace, endpointDeleteNamespace,
	endpointListTables, endpointLoadTable, endpointCreateTable,
	endpointUpdateTable, endpointDeleteTable, endpointRenameTable,
	endpointRegisterTable, endpointReportMetrics, endpointCommitTransaction,
}

// viewEndpoints is added to the default set only when view-endpoints-supported
// is set.
var viewEndpoints = []endpoint{
	endpointListViews, endpointLoadView, endpointCreateView,
	endpointUpdateView, endpointDeleteView, endpointRenameView,
}

// allEndpoints lists every endpoint constant, for tests that validate templates.
var allEndpoints = []endpoint{
	endpointListNamespaces, endpointLoadNamespace, endpointNamespaceExists,
	endpointCreateNamespace, endpointUpdateNamespace, endpointDeleteNamespace,
	endpointCommitTransaction,
	endpointListTables, endpointLoadTable, endpointTableExists, endpointCreateTable,
	endpointUpdateTable, endpointDeleteTable, endpointRenameTable, endpointRegisterTable,
	endpointReportMetrics, endpointTableCredentials,
	endpointPlanTableScan, endpointFetchPlanResult, endpointCancelPlanning, endpointFetchScanTasks,
	endpointListViews, endpointLoadView, endpointViewExists, endpointCreateView,
	endpointUpdateView, endpointDeleteView, endpointRenameView, endpointRegisterView,
	endpointListFunctions, endpointLoadFunction,
}

// endpointSet is the set of endpoints a catalog server supports.
type endpointSet map[endpoint]struct{}

func newEndpointSet(eps []endpoint) endpointSet {
	s := endpointSet{}
	for _, e := range eps {
		s[e] = struct{}{}
	}

	return s
}

func (s endpointSet) contains(e endpoint) bool {
	_, ok := s[e]

	return ok
}

// allowed reports whether an endpoint may be called. A nil set (never
// negotiated) permits everything.
func (s endpointSet) allowed(e endpoint) bool {
	return s == nil || s.contains(e)
}

// check returns ErrEndpointNotSupported if the endpoint is not allowed.
func (s endpointSet) check(e endpoint) error {
	if s.allowed(e) {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrEndpointNotSupported, e)
}

// resolveEndpoints builds the effective set from what the server advertised. A
// non-empty list is authoritative: unparseable entries are dropped with a
// warning, so a bad entry narrows capability instead of falling back. An empty
// list falls back to defaultEndpoints, plus viewEndpoints when
// viewEndpointsSupported is set.
func resolveEndpoints(advertised []string, viewEndpointsSupported bool) endpointSet {
	if len(advertised) == 0 {
		eps := defaultEndpoints
		if viewEndpointsSupported {
			eps = append(append([]endpoint{}, defaultEndpoints...), viewEndpoints...)
		}

		return newEndpointSet(eps)
	}

	s := endpointSet{}
	for _, raw := range advertised {
		e, err := endpointFromString(raw)
		if err != nil {
			slog.Warn("dropping unparseable advertised REST endpoint", "endpoint", raw, "error", err)

			continue
		}
		s[e] = struct{}{}
	}

	return s
}
