package apiv2base

import (
	"context"
	"net/http"
	"net/url"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// AuthzRoute is one HTTP route and the grant an API key must hold to call it.
// Exempt routes carry an empty Grant, so a matched-but-public request stays
// distinguishable from one that matched nothing.
type AuthzRoute struct {
	HTTPMethod string
	// PathTemplate is the proto template, e.g. "/apps/{app_id}/syncs".
	PathTemplate string
	// Grant is the required grant label, e.g. "api:app:write". Empty when the
	// route is exempt.
	Grant string
	Exempt bool
}

// BuildAuthzRoutes returns every HTTP route the v2 service exposes, including
// additional_bindings, with the grant each one requires.
//
// Order is proto declaration order and must stay that way. Templates overlap,
// so which route wins depends on registration order, and grpc-gateway registers
// in declaration order. Sorting these would silently change precedence.
func BuildAuthzRoutes() []AuthzRoute {
	var out []AuthzRoute
	forEachMethod(func(method protoreflect.MethodDescriptor) {
		opts := authzOptions(method)
		grant, protected := GrantForMethod(method)
		exempt := opts != nil && opts.GetExempt()

		for _, b := range httpBindings(method) {
			r := AuthzRoute{HTTPMethod: b.Method, PathTemplate: b.Path, Exempt: exempt}
			if protected {
				r.Grant = grant
			}
			out = append(out, r)
		}
	})
	return out
}

// AuthzMatcher reports which v2 route a method and path resolve to, and what
// that route requires.
//
// It asks grpc-gateway's own router. Compiling the templates into regexps or a
// chi router would be a second implementation of path-template semantics, and
// when the two disagreed the failure would be a silently wrong grant.
// runtime.ServeMux.HandlePath is the only public door to the same httprule
// compiler the gateway uses.
type AuthzMatcher struct {
	mux *runtime.ServeMux
}

type matchedRouteKey struct{}

// NewAuthzMatcher builds a matcher over the given routes, normally
// BuildAuthzRoutes().
func NewAuthzMatcher(routes []AuthzRoute) (*AuthzMatcher, error) {
	mux := runtime.NewServeMux()
	for _, r := range routes {
		route := r
		if err := mux.HandlePath(route.HTTPMethod, route.PathTemplate,
			func(_ http.ResponseWriter, req *http.Request, _ map[string]string) {
				if slot, ok := req.Context().Value(matchedRouteKey{}).(*AuthzRoute); ok {
					*slot = route
				}
			},
		); err != nil {
			return nil, err
		}
	}
	return &AuthzMatcher{mux: mux}, nil
}

// Match reports the route a method and path resolve to. Callers must fail
// closed when the second return is false: a miss means this matcher could not
// tell what the gateway will do, not that the gateway will refuse the request.
//
// The request handed to the matcher is synthetic. Passing the real one would be
// unsafe, because ServeMux.ServeHTTP calls r.ParseForm() when
// X-HTTP-Method-Override is set on a form-encoded POST, consuming the body the
// gateway still needs, and it rewrites r.Method.
//
// It is assembled field by field rather than with httptest.NewRequest, which
// would re-parse the path. Callers pass r.URL.Path, which net/http has already
// percent-decoded once, so a second parse decodes it again:
//
//	wire /runs/x%252Fy/cancel -> gateway /runs/x%2Fy/cancel (3 segments, matches)
//	                          -> re-parsed /runs/x/y/cancel (4 segments, misses)
//
// Every such divergence adds segments, so it can only turn a matched protected
// route into a miss, the direction that skips authorization. Re-parsing also
// panics on a decoded path holding a space or a bare "%".
//
// ServeMux reads only Method and URL.Path, so those are all this needs to
// carry. RawPath is consulted solely under a non-default unescaping mode, which
// neither mux sets.
func (m *AuthzMatcher) Match(httpMethod, path string) (AuthzRoute, bool) {
	if m == nil || m.mux == nil {
		return AuthzRoute{}, false
	}

	var matched AuthzRoute
	ctx := context.WithValue(context.Background(), matchedRouteKey{}, &matched)
	req := (&http.Request{
		Method: httpMethod,
		URL:    &url.URL{Path: path},
		// Non-nil so the mux's header reads and its POST->GET fallback check see
		// an absent header rather than a nil map.
		Header: http.Header{},
	}).WithContext(ctx)

	m.mux.ServeHTTP(discardWriter{}, req)

	if matched.PathTemplate == "" {
		return AuthzRoute{}, false
	}
	return matched, true
}

// discardWriter swallows whatever the matcher mux writes. On a miss it renders
// a 404 through its routing error handler, which this reads as the absence of a
// match instead.
type discardWriter struct{ header http.Header }

func (d discardWriter) Header() http.Header {
	if d.header == nil {
		return http.Header{}
	}
	return d.header
}
func (d discardWriter) Write(b []byte) (int, error) { return len(b), nil }
func (d discardWriter) WriteHeader(int)             {}

type authzRouteCtxKey struct{}

// WithAuthzRoute stores the route a request resolved to, so a downstream
// authorization middleware can read the required grant without re-deriving it.
//
// This is the seam between the two repos: this side decides what a route
// requires, and the caller supplying AuthzMiddleware decides whether to enforce
// it.
func WithAuthzRoute(ctx context.Context, route AuthzRoute) context.Context {
	return context.WithValue(ctx, authzRouteCtxKey{}, route)
}

// AuthzRouteFromContext returns the route stored by WithAuthzRoute. The second
// return is false when the request resolved to no known v2 route.
func AuthzRouteFromContext(ctx context.Context) (AuthzRoute, bool) {
	route, ok := ctx.Value(authzRouteCtxKey{}).(AuthzRoute)
	return route, ok
}

// IsExemptPath reports whether a path belongs to a route annotated
// `exempt: true`.
//
// Matching ignores the HTTP method because the callers are path-level
// authentication gates. A method-aware check would turn POST /health from the
// gateway's 404 into a 401 for no benefit.
//
// Every exempt template today is static, so comparison is exact.
// TestExemptRoutesHaveNoPathParameters keeps that assumption honest.
func IsExemptPath(path string) bool {
	for _, r := range BuildAuthzRoutes() {
		if r.Exempt && r.PathTemplate == path {
			return true
		}
	}
	return false
}
