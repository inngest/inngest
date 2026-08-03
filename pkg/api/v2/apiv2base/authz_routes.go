package apiv2base

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// AuthzRoute is one HTTP route and the grant an API key must hold to call it.
// Exempt routes appear with an empty Grant so a matched-but-public request stays
// distinguishable from one that matched nothing at all.
type AuthzRoute struct {
	HTTPMethod string
	// PathTemplate is the proto template, e.g. "/apps/{app_id}/syncs".
	PathTemplate string
	// Grant is the required grant label, e.g. "api:app:write". Empty when the
	// route is exempt.
	Grant string
	// Exempt marks a deliberately public route.
	Exempt bool
}

// BuildAuthzRoutes returns every HTTP route the v2 service exposes, including
// additional_bindings, with the grant each one requires.
//
// Order is proto declaration order and must stay that way. Some templates
// overlap — `{experiment_id=**}` also matches the ListExperiments binding one
// segment shorter — so which route wins depends on registration order.
// grpc-gateway's own registration follows declaration order, so preserving it
// here is what keeps the matcher's answer identical to the routing decision the
// gateway actually made. Sorting these would silently change precedence.
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

// AuthzMatcher answers "which v2 route is this request, and what does it
// require?" for a concrete method and path.
//
// It works by asking grpc-gateway's own router. That matters: the alternative —
// compiling the templates into regexps or into a chi router — would be a second
// implementation of path-template semantics that has to agree with the real
// routing, and when it disagreed the failure would be a silently wrong grant.
// runtime.ServeMux.HandlePath compiles templates with the same httprule compiler
// the gateway uses, and it is the only public door to that compiler.
type AuthzMatcher struct {
	mux *runtime.ServeMux
}

type matchedRouteKey struct{}

// NewAuthzMatcher builds a matcher over the given routes. Callers normally pass
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

// Match reports the route a method and path resolve to. The second return is
// false when nothing matched, which callers should treat as "not a v2 endpoint"
// rather than as a denial — the gateway will 404 it.
//
// The request handed to the matcher is synthetic, carrying only the method and
// path. Passing the real one would be unsafe: ServeMux.ServeHTTP calls
// r.ParseForm() when X-HTTP-Method-Override is set on a form-encoded POST, which
// consumes the body the real gateway still needs, and it rewrites r.Method.
func (m *AuthzMatcher) Match(httpMethod, path string) (AuthzRoute, bool) {
	if m == nil || m.mux == nil {
		return AuthzRoute{}, false
	}

	var matched AuthzRoute
	req := httptest.NewRequest(httpMethod, path, nil)
	req = req.WithContext(context.WithValue(req.Context(), matchedRouteKey{}, &matched))

	m.mux.ServeHTTP(discardWriter{}, req)

	if matched.PathTemplate == "" {
		return AuthzRoute{}, false
	}
	return matched, true
}

// discardWriter swallows whatever the matcher mux writes — on a miss it renders
// a 404 through its routing error handler, which we read from the absence of a
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

// WithAuthzRoute stores the route a request resolved to, so an authorization
// middleware downstream can read the required grant without re-deriving it.
//
// This is the seam between the two repos: the open-source side decides *what* a
// route requires, and the caller supplying AuthzMiddleware decides whether to
// enforce it.
func WithAuthzRoute(ctx context.Context, route AuthzRoute) context.Context {
	return context.WithValue(ctx, authzRouteCtxKey{}, route)
}

// AuthzRouteFromContext returns the route stored by WithAuthzRoute. The second
// return is false when the request did not resolve to a known v2 route.
func AuthzRouteFromContext(ctx context.Context) (AuthzRoute, bool) {
	route, ok := ctx.Value(authzRouteCtxKey{}).(AuthzRoute)
	return route, ok
}
