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

// grantNameRe constrains a grant to namespace + resource, with no action
// segment accidentally baked into the name.
var grantNameRe = regexp.MustCompile(`^api:[a-z]+$`)

// Actions a grant can be held for. These mirror the AuthzAction enum; the
// wire format stores them as the action list on an API key's scope rule.
const (
	ActionRead  = "read"
	ActionWrite = "write"
)

// Grant is one selectable permission: a resource plus an action, with the
// prose the minting UI and the OpenAPI docs both render.
//
// Name is the resource half only ("api:run"); the label shown to people is
// Name + ":" + Action, which is also what an OpenAPI operation advertises.
type Grant struct {
	Name        string
	Action      string
	Description string
	Category    string
}

// String returns the human-facing label, e.g. "api:run:read".
func (g Grant) String() string { return g.Name + ":" + g.Action }

// grantDefinitions returns the grant catalog declared as file options on
// service.proto, keyed by name.
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

// actionString maps the annotation enum onto the stored action name.
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

// GrantCatalog returns every (grant, action) pair that some endpoint actually
// requires, sorted by name then action. A pair no endpoint requires is
// deliberately absent: offering a toggle that gates nothing would be a lie in
// the minting UI.
func GrantCatalog() []Grant {
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
		g := Grant{
			Name:        name,
			Action:      action,
			Description: def.GetDescription(),
			Category:    def.GetCategory(),
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

// GrantForMethod returns the grant label an endpoint requires, e.g.
// "api:run:read". The second return is false for exempt or unannotated
// methods.
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

// HTTPBinding is one HTTP route an rpc answers on. A method can have several:
// google.api.http supports additional_bindings, and every binding needs the
// same grant or a request arriving on the secondary route would match no rule.
type HTTPBinding struct {
	Method string
	Path   string
}

// httpBindings returns every route a method answers on — the primary binding
// plus any additional_bindings.
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

// GrantsByHTTPRoute maps "METHOD /path/template" to the grant label that route
// requires, covering every binding of every protected method.
//
// Keys use the proto path template verbatim. Note that the generated OpenAPI
// document does NOT use these keys as-is — protoc-gen-openapiv2 camelCases path
// parameters and drops the `=**` suffix — so a docs consumer has to canonicalize
// both sides rather than matching literally.
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

// CanonicalRoute reduces a route to "METHOD /a/{}/b" by replacing each path
// parameter with an empty placeholder. This makes proto templates comparable
// with the generated OpenAPI paths, whose parameter names are camelCased and
// whose `=**` suffixes are stripped.
func CanonicalRoute(httpMethod, path string) string {
	return strings.ToUpper(httpMethod) + " " + pathParamRe.ReplaceAllString(path, "{}")
}

var pathParamRe = regexp.MustCompile(`\{[^}]*\}`)

// ValidateGrantCatalog checks the annotations hold together. It is called from
// tests rather than at startup: a violation is a build-time authoring mistake,
// not a runtime condition.
func ValidateGrantCatalog(allowedCategories []string) error {
	defs := grantDefinitions()
	allowed := map[string]bool{}
	for _, c := range allowedCategories {
		allowed[c] = true
	}

	var errs []string
	referenced := map[string]bool{}

	// Declarations are well-formed.
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

	// Every method is either annotated with a declared grant, or exempt.
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

	// No declared grant is unreachable.
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

// forEachMethod walks the V2 service's methods.
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
