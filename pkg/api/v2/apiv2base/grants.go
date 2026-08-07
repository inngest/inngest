package apiv2base

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// A grant name is namespace + resource, with no action segment baked in.
var grantNameRe = regexp.MustCompile(`^api:[a-z]+$`)

// Mirror the AuthzAction enum. The wire format stores these as the action list
// on an API key's scope rule.
const (
	ActionRead  = "read"
	ActionWrite = "write"
)

// Grant is one selectable permission: a resource plus an action.
//
// Name is the resource half only ("api:run"). The label shown to people is
// Name + ":" + Action, which is also what an OpenAPI operation advertises.
type Grant struct {
	Name        string
	Action      string
	Description string
	Category    string
	// Sensitive grants are never handed out by a preset. Declared per resource,
	// so it covers both actions: a resource whose read half is dangerous has no
	// safe half to preselect.
	Sensitive bool
	// Internal grants are enforced but never offered for minting. They are
	// absent from GrantCatalog, so only AllGrants reports one.
	Internal bool
}

// String returns the label, e.g. "api:run:read".
func (g Grant) String() string { return g.Name + ":" + g.Action }

// The catalog is declared as file options on service.proto.
func grantDefinitions() map[string]*apiv2.GrantDefinition {
	out := map[string]*apiv2.GrantDefinition{}
	opts := apiv2.File_api_v2_service_proto.Options()
	if opts == nil {
		return out
	}
	defs, ok := proto.GetExtension(opts, apiv2.E_GrantDefinition).([]*apiv2.GrantDefinition)
	if !ok {
		return out
	}
	for _, d := range defs {
		out[d.GetName()] = d
	}
	return out
}

func actionString(a apiv2.AuthzAction) string {
	switch a {
	case apiv2.AuthzAction_AUTHZ_ACTION_READ:
		return ActionRead
	case apiv2.AuthzAction_AUTHZ_ACTION_WRITE:
		return ActionWrite
	default:
		return ""
	}
}

// GrantCatalog returns the grants an API key may be minted with: every (grant,
// action) pair some endpoint requires, minus the internal ones, sorted by name
// then action.
//
// Every minting surface reads this one list, so excluding a grant withdraws it
// from all of them at once. A pair no endpoint requires is absent too, since a
// toggle that gates nothing would mislead.
//
// Enforcement does not read this. The middleware resolves a route's requirement
// through GrantForMethod and the OpenAPI docs through GrantsByHTTPRoute, so an
// internal grant is still required on its routes; it just cannot be asked for.
func GrantCatalog() []Grant {
	return collectGrants(false)
}

// AllGrants returns every declared (grant, action) pair, internal ones included.
//
// Only the committed catalog snapshot uses it. The snapshot is where a reviewer
// sees what the API declares, so hiding internal grants would make flipping the
// flag invisible in review.
func AllGrants() []Grant {
	return collectGrants(true)
}

func collectGrants(includeInternal bool) []Grant {
	defs := grantDefinitions()

	seen := map[string]Grant{}
	forEachMethod(func(method protoreflect.MethodDescriptor) {
		opts := authzOptions(method)
		if opts == nil || opts.GetExempt() {
			return
		}
		name, action := opts.GetGrant(), actionString(opts.GetAction())
		if name == "" || action == "" {
			return
		}
		def := defs[name]
		if def.GetInternal() && !includeInternal {
			return
		}
		g := Grant{
			Name:        name,
			Action:      action,
			Description: def.GetDescription(),
			Category:    def.GetCategory(),
			Sensitive:   def.GetSensitive(),
			Internal:    def.GetInternal(),
		}
		seen[g.String()] = g
	})

	out := make([]Grant, 0, len(seen))
	for _, g := range seen {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Action < out[j].Action
	})
	return out
}

// GrantForMethod returns the grant label an endpoint requires. The second
// return is false for exempt or unannotated methods.
func GrantForMethod(method protoreflect.MethodDescriptor) (string, bool) {
	opts := authzOptions(method)
	if opts == nil || opts.GetExempt() {
		return "", false
	}
	name, action := opts.GetGrant(), actionString(opts.GetAction())
	if name == "" || action == "" {
		return "", false
	}
	return name + ":" + action, true
}

// HTTPBinding is one HTTP route an rpc answers on. A method can have several
// through additional_bindings, and every binding needs the same grant or a
// request arriving on the secondary route would match no rule.
type HTTPBinding struct {
	Method string
	Path   string
}

func httpBindings(method protoreflect.MethodDescriptor) []HTTPBinding {
	rule := getHTTPRule(method)
	if rule == nil {
		return nil
	}

	var out []HTTPBinding
	add := func(r *annotations.HttpRule) {
		m, p := httpMethodAndPathFromRule(r)
		if p != "" {
			out = append(out, HTTPBinding{Method: m, Path: p})
		}
	}
	add(rule)
	for _, extra := range rule.GetAdditionalBindings() {
		add(extra)
	}
	return out
}

// GrantsByHTTPRoute maps "METHOD /path/template" to the grant that route
// requires, covering every binding of every protected method.
//
// Keys use the proto path template verbatim. The generated OpenAPI document
// does not: protoc-gen-openapiv2 camelCases path parameters and drops the `=**`
// suffix, so a docs consumer has to canonicalize both sides.
func GrantsByHTTPRoute() map[string]string {
	out := map[string]string{}
	forEachMethod(func(method protoreflect.MethodDescriptor) {
		grant, ok := GrantForMethod(method)
		if !ok {
			return
		}
		for _, b := range httpBindings(method) {
			out[b.Method+" "+b.Path] = grant
		}
	})
	return out
}

// CanonicalRoute reduces a route to "METHOD /a/{}/b", making proto templates
// comparable with the generated OpenAPI paths, whose parameter names are
// camelCased and whose `=**` suffixes are stripped.
func CanonicalRoute(httpMethod, path string) string {
	return strings.ToUpper(httpMethod) + " " + pathParamRe.ReplaceAllString(path, "{}")
}

var pathParamRe = regexp.MustCompile(`\{[^}]*\}`)

// Called from tests rather than at startup: a violation is a build-time
// authoring mistake, not a runtime condition.
func ValidateGrantCatalog(allowedCategories []string) error {
	defs := grantDefinitions()
	allowed := map[string]bool{}
	for _, c := range allowedCategories {
		allowed[c] = true
	}

	var errs []string
	referenced := map[string]bool{}

	for name, d := range defs {
		if !grantNameRe.MatchString(name) {
			errs = append(errs, fmt.Sprintf("grant %q: name must match %s", name, grantNameRe))
		}
		if strings.TrimSpace(d.GetDescription()) == "" {
			errs = append(errs, fmt.Sprintf("grant %q: description is required", name))
		}
		if !allowed[d.GetCategory()] {
			errs = append(errs, fmt.Sprintf("grant %q: category %q is not one of the allowed categories", name, d.GetCategory()))
		}
	}

	forEachMethod(func(method protoreflect.MethodDescriptor) {
		name := string(method.Name())
		opts := authzOptions(method)
		if opts == nil {
			errs = append(errs, fmt.Sprintf("rpc %s: missing (authz) annotation — add a grant, or exempt: true with a comment saying why", name))
			return
		}
		if opts.GetExempt() {
			if opts.GetGrant() != "" || opts.GetAction() != apiv2.AuthzAction_AUTHZ_ACTION_UNSPECIFIED {
				errs = append(errs, fmt.Sprintf("rpc %s: exempt cannot be combined with a grant or action", name))
			}
			return
		}
		if opts.GetGrant() == "" {
			errs = append(errs, fmt.Sprintf("rpc %s: (authz) has no grant and is not exempt", name))
			return
		}
		if _, ok := defs[opts.GetGrant()]; !ok {
			errs = append(errs, fmt.Sprintf("rpc %s: grant %q is not declared via grant_definition", name, opts.GetGrant()))
		}
		if actionString(opts.GetAction()) == "" {
			errs = append(errs, fmt.Sprintf("rpc %s: action is unspecified", name))
		}
		referenced[opts.GetGrant()] = true
	})

	for name := range defs {
		if !referenced[name] {
			errs = append(errs, fmt.Sprintf("grant %q is declared but no rpc requires it — the minting UI would offer a toggle that gates nothing", name))
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("grant catalog is invalid:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func forEachMethod(fn func(protoreflect.MethodDescriptor)) {
	serviceDesc := apiv2.File_api_v2_service_proto.Services().ByName("V2")
	if serviceDesc == nil {
		return
	}
	methods := serviceDesc.Methods()
	for i := 0; i < methods.Len(); i++ {
		fn(methods.Get(i))
	}
}
