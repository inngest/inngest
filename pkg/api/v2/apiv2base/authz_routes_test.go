package apiv2base

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestMatcher(t *testing.T) *AuthzMatcher {
	t.Helper()
	m, err := NewAuthzMatcher(BuildAuthzRoutes())
	require.NoError(t, err)
	return m
}

func TestBuildAuthzRoutesCoversEveryBinding(t *testing.T) {
	routes := BuildAuthzRoutes()
	require.NotEmpty(t, routes)

	var exempt, protected int
	for _, r := range routes {
		switch {
		case r.Exempt:
			exempt++
			require.Empty(t, r.Grant, "exempt route %s %s must not carry a grant", r.HTTPMethod, r.PathTemplate)
		default:
			protected++
			require.NotEmpty(t, r.Grant, "route %s %s has no grant and is not exempt", r.HTTPMethod, r.PathTemplate)
		}
		require.NotEmpty(t, r.HTTPMethod)
		require.NotEmpty(t, r.PathTemplate)
	}
	// Totals are not pinned, since they change whenever an endpoint is added
	// and the per-route assertions above are what matter.
	require.Equal(t, 2, exempt)
	require.Greater(t, protected, 0)
}

// A path-keyed map cannot serve these: it would store the template and be
// looked up by concrete URL, so anything parameterised never matches.
func TestMatcherResolvesConcreteURLs(t *testing.T) {
	m := newTestMatcher(t)

	cases := []struct {
		method, path, wantGrant string
	}{
		{http.MethodPost, "/apps/abc-123/syncs", "api:app:write"},
		{http.MethodGet, "/runs/01JPVDWE5Q1K8ZH3JW31HNW5QS", "api:run:read"},
		{http.MethodGet, "/apps/my-app/functions/my-fn", "api:function:read"},
		{http.MethodPost, "/apps/my-app/functions/my-fn/invoke", "api:function:write"},
		// Multi-segment wildcard with a literal suffix after it, the case that
		// disqualified chi, whose wildcards must be terminal.
		{http.MethodGet, "/sessions/key-1/sess/nested/id/runs", "api:session:read"},
		{http.MethodGet, "/apps/a/functions/f/experiments/exp/nested", "api:experiment:read"},
		{http.MethodGet, "/runs", "api:run:read"},
		{http.MethodGet, "/account", "api:account:read"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			got, ok := m.Match(tc.method, tc.path)
			require.True(t, ok, "expected a match")
			require.Equal(t, tc.wantGrant, got.Grant)
			require.False(t, got.Exempt)
		})
	}
}

// Same path, different method, different grant, which a path-only map
// cannot express.
func TestMatcherDistinguishesMethods(t *testing.T) {
	m := newTestMatcher(t)

	get, ok := m.Match(http.MethodGet, "/env/webhooks")
	require.True(t, ok)
	require.Equal(t, "api:webhook:read", get.Grant)

	post, ok := m.Match(http.MethodPost, "/env/webhooks")
	require.True(t, ok)
	require.Equal(t, "api:webhook:write", post.Grant)

	getP, ok := m.Match(http.MethodGet, "/partner/accounts")
	require.True(t, ok)
	require.Equal(t, "api:partner:read", getP.Grant)

	postP, ok := m.Match(http.MethodPost, "/partner/accounts")
	require.True(t, ok)
	require.Equal(t, "api:partner:write", postP.Grant)
}

func TestMatcherReportsExemptRoutes(t *testing.T) {
	m := newTestMatcher(t)

	got, ok := m.Match(http.MethodGet, "/health")
	require.True(t, ok, "an exempt route still matches — it is a known endpoint")
	require.True(t, got.Exempt)
	require.Empty(t, got.Grant)
}

// A miss must be distinguishable from an exempt match, so the caller can
// answer 404 rather than turning a typo'd URL into a 403.
func TestMatcherMisses(t *testing.T) {
	m := newTestMatcher(t)

	for _, path := range []string{
		"/nope",
		"/runs/01JPVDWE5Q1K8ZH3JW31HNW5QS/not-a-thing",
		"/apps/abc/not-a-thing",
	} {
		_, ok := m.Match(http.MethodGet, path)
		require.False(t, ok, "%s should not match any v2 route", path)
	}

	_, ok := m.Match(http.MethodDelete, "/runs")
	require.False(t, ok)
}

// additional_bindings must be reachable through the matcher too, or a request
// arriving on the secondary path would be treated as "not a v2 endpoint" and
// skip enforcement entirely.
func TestMatcherCoversAdditionalBindings(t *testing.T) {
	m := newTestMatcher(t)

	primary, ok := m.Match(http.MethodGet, "/experiments")
	require.True(t, ok)
	require.Equal(t, "api:experiment:read", primary.Grant)

	secondary, ok := m.Match(http.MethodGet, "/apps/my-app/functions/my-fn/experiments")
	require.True(t, ok, "additional_binding must match, or enforcement misses this path")
	require.Equal(t, "api:experiment:read", secondary.Grant)
}

// Templates can legitimately overlap, so which route wins depends on
// registration order. That is safe only while overlapping routes agree on the
// grant; otherwise the permission enforced would depend on proto declaration
// order.
//
// This asserts every template is reachable and that whatever route a probe
// lands on requires the same grant as the template it came from.
func TestOverlappingRoutesAgreeOnGrant(t *testing.T) {
	routes := BuildAuthzRoutes()
	m := newTestMatcher(t)

	overlaps := 0
	for _, r := range routes {
		probe := concreteProbe(r.PathTemplate)
		got, ok := m.Match(r.HTTPMethod, probe)
		require.True(t, ok, "probe %s %s (from %s) matched nothing", r.HTTPMethod, probe, r.PathTemplate)

		if got.PathTemplate != r.PathTemplate {
			overlaps++
			t.Logf("overlap: probe %s %s (from %s) resolves to %s",
				r.HTTPMethod, probe, r.PathTemplate, got.PathTemplate)
		}
		require.Equal(t, r.Grant, got.Grant,
			"probe %s %s from template %s resolved to %s, which requires a DIFFERENT grant "+
				"(%s vs %s) — enforcement would depend on registration order",
			r.HTTPMethod, probe, r.PathTemplate, got.PathTemplate, r.Grant, got.Grant)
		require.Equal(t, r.Exempt, got.Exempt,
			"probe %s %s from %s resolved to %s with a different exempt status",
			r.HTTPMethod, probe, r.PathTemplate, got.PathTemplate)
	}
	// Currently exactly one: the experiments pair. If this grows, each new
	// overlap needs the same-grant reasoning above.
	require.LessOrEqual(t, overlaps, 1, "more template overlaps than expected; see the log lines")
}

// Declaration order is load-bearing for precedence, so a sort creeping into
// BuildAuthzRoutes would silently change which route wins.
func TestRoutesAreInDeclarationOrder(t *testing.T) {
	routes := BuildAuthzRoutes()
	require.Equal(t, "GET", routes[0].HTTPMethod)
	require.Equal(t, "/health", routes[0].PathTemplate, "Health is the first rpc declared")
	require.Equal(t, "/_internal/schema-only", routes[1].PathTemplate, "_SchemaOnly is declared second")
}

// concreteProbe turns a path template into a callable URL by substituting each
// parameter with a placeholder segment.
func concreteProbe(template string) string {
	out := ""
	i := 0
	for i < len(template) {
		if template[i] == '{' {
			depth := 1
			j := i + 1
			for j < len(template) && depth > 0 {
				if template[j] == '{' {
					depth++
				}
				if template[j] == '}' {
					depth--
				}
				j++
			}
			// A "=**" parameter spans multiple segments, so give it two.
			if containsWildcard(template[i:j]) {
				out += "probe/probe2"
			} else {
				out += "probe"
			}
			i = j
			continue
		}
		out += string(template[i])
		i++
	}
	return out
}

func containsWildcard(param string) bool {
	for i := 0; i+1 < len(param); i++ {
		if param[i] == '*' && param[i+1] == '*' {
			return true
		}
	}
	return false
}

// The matcher must not touch the caller's request. A form-encoded POST carrying
// X-HTTP-Method-Override makes ServeMux.ServeHTTP call ParseForm and rewrite the
// method, which would eat the body the real gateway still has to read.
func TestMatcherDoesNotConsumeRequestBody(t *testing.T) {
	m := newTestMatcher(t)

	body := "a=1&b=2"
	req := httptest.NewRequest(http.MethodPost, "/apps/abc/syncs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-HTTP-Method-Override", "GET")

	got, ok := m.Match(req.Method, req.URL.Path)
	require.True(t, ok)
	require.Equal(t, "api:app:write", got.Grant)

	require.Equal(t, http.MethodPost, req.Method)
	buf := make([]byte, len(body))
	n, _ := req.Body.Read(buf)
	require.Equal(t, body, string(buf[:n]), "matcher consumed the caller's body")
}

// Callers pass r.URL.Path, which net/http has already percent-decoded. The
// matcher must not decode it again: the gateway splits that exact string on
// "/", so an extra decode invents segments the gateway never saw, turns a
// protected route into a miss, and skips the grant check.
//
// Each case is a path that reaches the gateway looking exactly like this.
func TestMatcherDoesNotDecodeThePathAgain(t *testing.T) {
	m := newTestMatcher(t)

	// Wire /runs/x%252Fy/cancel. The gateway sees three segments and cancels a
	// run; re-parsing turns "%2F" into a separator and matches nothing.
	got, ok := m.Match(http.MethodPost, "/runs/x%2Fy/cancel")
	require.True(t, ok, "a literal %%2F in a path parameter must not split the path")
	require.Equal(t, "api:run:write", got.Grant)
	require.Equal(t, "/runs/{run_id}/cancel", got.PathTemplate)

	// Wire /sessions/a%20b. Re-parsing rejected the space as a malformed
	// request line and panicked before it could match.
	got, ok = m.Match(http.MethodGet, "/sessions/a b")
	require.True(t, ok, "a decoded space must not panic or miss")
	require.Equal(t, "api:session:read", got.Grant)

	// Wire /runs/50%25. Re-parsing panicked on the dangling escape.
	got, ok = m.Match(http.MethodGet, "/runs/50%")
	require.True(t, ok, "a bare %% must not panic or miss")
	require.Equal(t, "api:run:read", got.Grant)
}

// The mux writes response headers on the miss path, and a future version may
// read one back. Header must therefore behave like a real ResponseWriter's and
// hand out the same map every time, rather than a fresh one that drops writes.
func TestDiscardWriterHeaderPersists(t *testing.T) {
	w := &discardWriter{}

	w.Header().Set("Content-Type", "application/json")
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	w.Header().Del("Content-Type")
	require.Empty(t, w.Header().Get("Content-Type"))
}

// A miss is not a decision, so nothing may read it as one. This pins the
// contract the caller relies on to fail closed.
func TestMatcherMissIsNotExempt(t *testing.T) {
	m := newTestMatcher(t)

	route, ok := m.Match(http.MethodGet, "/nope")
	require.False(t, ok)
	require.False(t, route.Exempt,
		"a miss must not look like a declared-public route")
	require.Empty(t, route.Grant)
}

// IsExemptPath compares paths exactly, which is only valid while no exempt
// route is parameterised. If one ever is, this fails and the comparison has to
// become a real match instead.
func TestExemptRoutesHaveNoPathParameters(t *testing.T) {
	for _, r := range BuildAuthzRoutes() {
		if !r.Exempt {
			continue
		}
		require.NotContains(t, r.PathTemplate, "{",
			"exempt route %s is parameterised; IsExemptPath's exact comparison is no longer valid",
			r.PathTemplate)
	}
}

func TestIsExemptPath(t *testing.T) {
	require.True(t, IsExemptPath("/health"))
	require.True(t, IsExemptPath("/_internal/schema-only"))
	require.False(t, IsExemptPath("/runs"))
	require.False(t, IsExemptPath("/apps/abc/syncs"))
	require.False(t, IsExemptPath(""))
}
