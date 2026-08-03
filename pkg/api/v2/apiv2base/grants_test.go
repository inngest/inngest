package apiv2base

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// allowedCategories is the set the key-minting UI groups by. Keep in sync with
// the categories in the spec; a new one needs a UI decision, not just a proto
// edit, which is why this list lives in the test rather than being inferred.
var allowedCategories = []string{
	"Accounts, Environments & Keys",
	"Apps, Functions & Runs",
	"Observability & AI Evals",
	"Compute",
}

// This is the guardrail that makes fail-closed enforcement safe to turn on: it
// fails if an rpc is added without either a grant or an explicit exemption, so
// the gap is caught at build time rather than becoming an unprotected endpoint.
func TestGrantCatalogIsValid(t *testing.T) {
	require.NoError(t, ValidateGrantCatalog(allowedCategories))
}

func TestEveryMethodIsAnnotated(t *testing.T) {
	var missing, exempt, protected []string
	forEachMethod(func(m protoreflect.MethodDescriptor) {
		name := string(m.Name())
		opts := authzOptions(m)
		switch {
		case opts == nil:
			missing = append(missing, name)
		case opts.GetExempt():
			exempt = append(exempt, name)
		default:
			protected = append(protected, name)
		}
	})

	// The security-relevant assertions: nothing is unannotated, and the public
	// set is exactly these two. Deliberately not asserting a total count — that
	// only breaks every time an endpoint is added, without protecting anything.
	require.Empty(t, missing, "every rpc needs a grant or exempt: true")
	require.ElementsMatch(t, []string{"Health", "_SchemaOnly"}, exempt,
		"adding a public endpoint is a deliberate decision; update this list with the reason")
	require.NotEmpty(t, protected)
}

func TestGrantCatalogShape(t *testing.T) {
	cat := GrantCatalog()
	require.NotEmpty(t, cat)

	allowed := map[string]bool{}
	for _, c := range allowedCategories {
		allowed[c] = true
	}

	seen := map[string]bool{}
	for _, g := range cat {
		require.NotEmpty(t, g.Description, "grant %s has no description", g)
		require.True(t, allowed[g.Category], "grant %s has category %q", g, g.Category)
		require.Regexp(t, `^api:[a-z]+$`, g.Name)
		require.Contains(t, []string{ActionRead, ActionWrite}, g.Action)
		require.False(t, seen[g.String()], "duplicate catalog entry %s", g)
		seen[g.String()] = true
	}

	// Sorted by name then action, so the committed artifact and the UI order
	// are both stable.
	for i := 1; i < len(cat); i++ {
		prev, cur := cat[i-1], cat[i]
		require.True(t,
			prev.Name < cur.Name || (prev.Name == cur.Name && prev.Action < cur.Action),
			"catalog not sorted at %d: %s then %s", i, prev, cur)
	}

	// A grant's prose must not depend on which endpoint you reached it through.
	byName := map[string]Grant{}
	for _, g := range cat {
		if prev, ok := byName[g.Name]; ok {
			require.Equal(t, prev.Description, g.Description, "grant %s has two descriptions", g.Name)
			require.Equal(t, prev.Category, g.Category, "grant %s has two categories", g.Name)
		}
		byName[g.Name] = g
	}
}

// The app-sync grant is the one the legacy alias in chunk 4 depends on, and
// the partner grants are what Replit's keys exercise. Pin them so a rename is
// a deliberate act rather than a silent break.
func TestLoadBearingGrants(t *testing.T) {
	routes := GrantsByHTTPRoute()
	require.Equal(t, "api:app:write", routes["POST /apps/{app_id}/syncs"])
	require.Equal(t, "api:partner:write", routes["POST /partner/accounts"])
	require.Equal(t, "api:partner:read", routes["GET /partner/accounts"])

	// Same path, different methods, different grants — the bug the old
	// path-keyed map could not express.
	require.Equal(t, "api:webhook:read", routes["GET /env/webhooks"])
	require.Equal(t, "api:webhook:write", routes["POST /env/webhooks"])
}

func TestExemptMethodsHaveNoGrant(t *testing.T) {
	routes := GrantsByHTTPRoute()
	require.NotContains(t, routes, "GET /health")
	require.NotContains(t, routes, "GET /_internal/schema-only")
}

// Templated and **-suffixed paths must survive into the route map verbatim,
// since that is the key space the docs and the enforcement matcher share.
func TestRouteKeysUseProtoTemplates(t *testing.T) {
	routes := GrantsByHTTPRoute()
	require.Equal(t, "api:run:read", routes["GET /runs/{run_id}"])
	require.Equal(t, "api:session:read", routes["GET /sessions/{session_key}/{session_id=**}/runs"])
	require.Equal(t, "api:experiment:read",
		routes["GET /apps/{app_id}/functions/{function_id}/experiments/{experiment_id=**}"])
}

func TestValidateCatalogCatchesBadCategory(t *testing.T) {
	err := ValidateGrantCatalog([]string{"Only This One"})
	require.Error(t, err, "a category not in the allowed set must fail validation")
	require.Contains(t, err.Error(), "is not one of the allowed categories")
}

func TestActionStringRejectsUnspecified(t *testing.T) {
	require.Equal(t, "", actionString(apiv2.AuthzAction_AUTHZ_ACTION_UNSPECIFIED))
	require.Equal(t, ActionRead, actionString(apiv2.AuthzAction_AUTHZ_ACTION_READ))
	require.Equal(t, ActionWrite, actionString(apiv2.AuthzAction_AUTHZ_ACTION_WRITE))
}

// Every route in the map is reachable prose-wise: the label is what OpenAPI
// advertises and what a key stores, so it must always be name + ":" + action.
func TestGrantLabelsAreWellFormed(t *testing.T) {
	for route, grant := range GrantsByHTTPRoute() {
		parts := strings.Split(grant, ":")
		require.Len(t, parts, 3, "route %s has malformed grant %q", route, grant)
		require.Equal(t, "api", parts[0])
		require.Contains(t, []string{ActionRead, ActionWrite}, parts[2])
	}
}

// TestGrantCatalogSnapshot pins the effective grant set to a committed file.
//
// The generated OpenAPI document is gitignored and its docs deploy is manual,
// so without this the aggregate set — what the minting UI offers and what
// "Full Access" resolves to at mint time — would not be visible in review. A
// one-line proto edit can add a toggle to the product; this makes that a diff.
//
// Regenerate with: make grant-catalog
func TestGrantCatalogSnapshot(t *testing.T) {
	type entry struct {
		Grant       string `json:"grant"`
		Name        string `json:"name"`
		Action      string `json:"action"`
		Category    string `json:"category"`
		Description string `json:"description"`
	}
	type snapshot struct {
		Comment string            `json:"_comment"`
		Grants  []entry           `json:"grants"`
		Routes  map[string]string `json:"routesByGrant"`
	}

	cat := GrantCatalog()
	entries := make([]entry, 0, len(cat))
	for _, g := range cat {
		entries = append(entries, entry{
			Grant: g.String(), Name: g.Name, Action: g.Action,
			Category: g.Category, Description: g.Description,
		})
	}
	snap := snapshot{
		Comment: "Generated by `make grant-catalog` from proto/api/v2/service.proto. " +
			"Every grant the v2 API offers, and the route that requires each. " +
			"A change here changes what an API key can be minted with.",
		Grants: entries,
		Routes: GrantsByHTTPRoute(),
	}

	// Encode without HTML escaping — the categories contain "&", and a
	// committed artifact reviewers read should not be full of &.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	require.NoError(t, enc.Encode(snap))
	got := buf.Bytes()

	path := filepath.Join("testdata", "grant_catalog.json")
	want, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, got, 0o644))
		t.Logf("generated %s", path)
		return
	}
	require.NoError(t, err)
	require.Equal(t, string(want), string(got),
		"the grant catalog changed. If intentional, run `make grant-catalog` and "+
			"review the diff — it shows exactly which permissions the API now offers")
}

// additional_bindings are real: ListExperiments answers on two paths, and both
// need the same grant or a request on the secondary route matches no rule.
// This was missed on the first pass — the route map only read primary bindings.
func TestAdditionalBindingsAreCovered(t *testing.T) {
	routes := GrantsByHTTPRoute()
	require.Equal(t, "api:experiment:read", routes["GET /experiments"])
	require.Equal(t, "api:experiment:read",
		routes["GET /apps/{app_id}/functions/{function_id}/experiments"],
		"additional_bindings must be in the route map, or enforcement misses that path")
}

// The generated OpenAPI document does not reproduce proto path templates
// verbatim, so anything matching routes across the two needs canonical form.
func TestCanonicalRouteNormalizesParams(t *testing.T) {
	// Proto template and the camelCased OpenAPI path collapse to one key.
	require.Equal(t,
		CanonicalRoute("GET", "/apps/{app_id}/functions/{function_id}"),
		CanonicalRoute("get", "/apps/{appId}/functions/{functionId}"))

	// So do the `=**` and its stripped OpenAPI counterpart.
	require.Equal(t,
		CanonicalRoute("GET", "/sessions/{session_key}/{session_id=**}/runs"),
		CanonicalRoute("GET", "/sessions/{sessionKey}/{sessionId}/runs"))

	require.Equal(t, "GET /runs/{}", CanonicalRoute("GET", "/runs/{run_id}"))
	require.Equal(t, "GET /health", CanonicalRoute("GET", "/health"))

	// Distinct routes must not collide.
	require.NotEqual(t,
		CanonicalRoute("GET", "/runs/{run_id}"),
		CanonicalRoute("GET", "/sessions/{session_key}"))
}

// Canonicalizing throws away parameter names, so it must not merge two routes
// that need different grants.
func TestCanonicalRoutesAreUnambiguous(t *testing.T) {
	byCanonical := map[string]string{}
	for route, grant := range GrantsByHTTPRoute() {
		method, path, ok := strings.Cut(route, " ")
		require.True(t, ok, "malformed route key %q", route)
		key := CanonicalRoute(method, path)
		if prev, seen := byCanonical[key]; seen {
			require.Equal(t, prev, grant,
				"routes collapsing to %q require different grants (%s vs %s)", key, prev, grant)
		}
		byCanonical[key] = grant
	}
}
