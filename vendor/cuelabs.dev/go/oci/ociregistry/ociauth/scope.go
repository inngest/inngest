package ociauth

import (
	"math/bits"
	"slices"
	"strings"
)

// knownAction represents an action that we know about
// and use a more efficient internal representation for.
type knownAction byte

const (
	unknownAction knownAction = iota
	// Note: ordered by lexical string representation.
	pullAction
	pushAction
	numActions
)

const (
	// Known resource types.
	TypeRepository = "repository"
	TypeRegistry   = "registry"

	// Known action types.
	ActionPull = "pull"
	ActionPush = "push"
)

func (a knownAction) String() string {
	switch a {
	case pullAction:
		return ActionPull
	case pushAction:
		return ActionPush
	default:
		return "unknown"
	}
}

// CatalogScope defines the resource scope used to allow
// listing all the items in a registry.
var CatalogScope = ResourceScope{
	ResourceType: TypeRegistry,
	Resource:     "catalog",
	Action:       "*",
}

// ResourceScope defines a component of an authorization scope
// associated with a single resource and action only.
// See [Scope] for a way of combining multiple ResourceScopes
// into a single value.
type ResourceScope struct {
	// ResourceType holds the type of resource the scope refers to.
	// Known values for this include TypeRegistry and TypeRepository.
	// When a scope does not conform to the standard resourceType:resource:actions
	// syntax, ResourceType will hold the entire scope.
	ResourceType string

	// Resource names the resource the scope pertains to.
	// For resource type TypeRepository, this will be the name of the repository.
	Resource string

	// Action names an action that can be performed on the resource.
	// This is usually ActionPush or ActionPull.
	Action string
}

func (rs1 ResourceScope) Equal(rs2 ResourceScope) bool {
	return rs1.Compare(rs2) == 0
}

// Compare returns -1, 0 or 1 depending on whether
// rs1 compares less than, equal, or greater than, rs2.
//
// In most to least precedence, the fields are compared in the order
// ResourceType, Resource, Action.
func (rs1 ResourceScope) Compare(rs2 ResourceScope) int {
	if c := strings.Compare(rs1.ResourceType, rs2.ResourceType); c != 0 {
		return c
	}
	if c := strings.Compare(rs1.Resource, rs2.Resource); c != 0 {
		return c
	}
	return strings.Compare(rs1.Action, rs2.Action)
}

func (rs ResourceScope) isKnown() bool {
	switch rs.ResourceType {
	case TypeRepository:
		return parseKnownAction(rs.Action) != unknownAction
	case TypeRegistry:
		return rs == CatalogScope
	}
	return false
}

// Scope holds a set of [ResourceScope] values. The zero value
// represents the empty set.
type Scope struct {
	// original holds the original string from which
	// this Scope was parsed. This maintains the string
	// representation unchanged as far as possible.
	original string

	// unlimited holds whether this scope is considered to include all
	// other scopes.
	unlimited bool

	// repositories holds all the repositories that the scope
	// refers to. An empty repository name implies a CatalogScope
	// entry. The elements of this are maintained in sorted order.
	repositories []string

	// actions holds an element for each element in repositories
	// defining the set of allowed actions for that repository
	// as a bitmask of 1<<knownAction bytes.
	// For CatalogScope, this is 1<<pullAction so that
	// the bit count reflects the number of resource scopes.
	actions []byte

	// others holds actions that don't fit into
	// the above categories. These may or may not be repository-scoped:
	// we just store them here verbatim.
	others []ResourceScope
}

// ParseScope parses a scope as defined in the [Docker distribution spec].
//
// For scopes that don't fit that syntax, it returns a Scope with
// the ResourceType field set to the whole string.
//
// [Docker distribution spec]: https://distribution.github.io/distribution/spec/auth/scope/
func ParseScope(s string) Scope {
	fields := strings.Fields(s)
	rscopes := make([]ResourceScope, 0, len(fields))
	for _, f := range fields {
		parts := strings.Split(f, ":")
		if len(parts) != 3 {
			rscopes = append(rscopes, ResourceScope{
				ResourceType: f,
			})
			continue
		}
		for _, action := range strings.Split(parts[2], ",") {
			rscopes = append(rscopes, ResourceScope{
				ResourceType: parts[0],
				Resource:     parts[1],
				Action:       action,
			})
		}
	}
	scope := NewScope(rscopes...)
	scope.original = s
	return scope
}

// NewScope returns a Scope value that holds the set of everything in rss.
func NewScope(rss ...ResourceScope) Scope {
	// TODO it might well be worth special-casing the single element scope case.
	slices.SortFunc(rss, ResourceScope.Compare)
	rss = slices.Compact(rss)
	var s Scope
	for _, rs := range rss {
		if !rs.isKnown() {
			s.others = append(s.others, rs)
			continue
		}
		if rs.ResourceType == TypeRegistry {
			// CatalogScope
			s.repositories = append(s.repositories, "")
			s.actions = append(s.actions, 1<<pullAction)
			continue
		}
		actionMask := byte(1 << parseKnownAction(rs.Action))
		if i := len(s.repositories); i > 0 && s.repositories[i-1] == rs.Resource {
			s.actions[i-1] |= actionMask
		} else {
			s.repositories = append(s.repositories, rs.Resource)
			s.actions = append(s.actions, actionMask)
		}
	}
	slices.SortFunc(s.others, ResourceScope.Compare)
	s.others = slices.Compact(s.others)
	return s
}

// Len returns the number of ResourceScopes in the scope set.
// It panics if the scope is unlimited.
func (s Scope) Len() int {
	if s.IsUnlimited() {
		panic("Len called on unlimited scope")
	}
	n := len(s.others)
	for _, b := range s.actions {
		n += bits.OnesCount8(b)
	}
	return n
}

// UnlimitedScope returns a scope that contains all other
// scopes. This is not representable in the docker scope syntax,
// but it's useful to represent the scope of tokens that can
// be used for arbitrary access.
func UnlimitedScope() Scope {
	return Scope{
		unlimited: true,
	}
}

// IsUnlimited reports whether s is unlimited in scope.
func (s Scope) IsUnlimited() bool {
	return s.unlimited
}

// IsEmpty reports whether the scope holds the empty set.
func (s Scope) IsEmpty() bool {
	return len(s.repositories) == 0 &&
		len(s.others) == 0 &&
		!s.unlimited
}

// Iter returns an iterator over all the individual scopes that are
// part of s. The items will be produced according to [Scope.Compare]
// ordering.
//
// The unlimited scope does not yield any scopes.
func (s Scope) Iter() func(yield func(ResourceScope) bool) {
	return func(yield0 func(ResourceScope) bool) {
		if s.unlimited {
			return
		}
		others := s.others
		yield := func(scope ResourceScope) bool {
			// Yield any scopes from others that are ready to
			// be produced, thus preserving ordering of all
			// values in the iterator.
			for len(others) > 0 && others[0].Compare(scope) < 0 {
				if !yield0(others[0]) {
					return false
				}
				others = others[1:]
			}
			return yield0(scope)
		}
		for i, repo := range s.repositories {
			if repo == "" {
				if !yield(CatalogScope) {
					return
				}
				continue
			}
			acts := s.actions[i]
			for k := knownAction(0); k < numActions; k++ {
				if acts&(1<<k) == 0 {
					continue
				}
				rscope := ResourceScope{
					ResourceType: TypeRepository,
					Resource:     repo,
					Action:       k.String(),
				}
				if !yield(rscope) {
					return
				}
			}
		}
		// Send any scopes in others that haven't already been sent.
		for _, rscope := range others {
			if !yield0(rscope) {
				return
			}
		}
	}
}

// Union returns a scope consisting of all the resource scopes from
// both s1 and s2. If the result is the same as s1, its
// string representation will also be the same as s1.
func (s1 Scope) Union(s2 Scope) Scope {
	if s1.IsUnlimited() || s2.IsUnlimited() {
		return UnlimitedScope()
	}
	// Cheap test that we can return the original unchanged.
	if s2.IsEmpty() || s1.Equal(s2) {
		return s1
	}
	r := Scope{
		repositories: make([]string, 0, len(s1.repositories)+len(s2.repositories)),
		actions:      make([]byte, 0, len(s1.repositories)+len(s2.repositories)),
		others:       make([]ResourceScope, 0, len(s1.others)+len(s2.others)),
	}
	i1, i2 := 0, 0
	for i1 < len(s1.repositories) && i2 < len(s2.repositories) {
		repo1, repo2 := s1.repositories[i1], s2.repositories[i2]

		switch strings.Compare(repo1, repo2) {
		case 0:
			r.repositories = append(r.repositories, repo1)
			r.actions = append(r.actions, s1.actions[i1]|s2.actions[i2])
			i1++
			i2++
		case -1:
			r.repositories = append(r.repositories, s1.repositories[i1])
			r.actions = append(r.actions, s1.actions[i1])
			i1++
		case 1:
			r.repositories = append(r.repositories, s2.repositories[i2])
			r.actions = append(r.actions, s2.actions[i2])
			i2++
		default:
			panic("unreachable")
		}
	}
	switch {
	case i1 < len(s1.repositories):
		r.repositories = append(r.repositories, s1.repositories[i1:]...)
		r.actions = append(r.actions, s1.actions[i1:]...)
	case i2 < len(s2.repositories):
		r.repositories = append(r.repositories, s2.repositories[i2:]...)
		r.actions = append(r.actions, s2.actions[i2:]...)
	}
	i1, i2 = 0, 0
	for i1 < len(s1.others) && i2 < len(s2.others) {
		a1, a2 := s1.others[i1], s2.others[i2]
		switch a1.Compare(a2) {
		case 0:
			r.others = append(r.others, a1)
			i1++
			i2++
		case -1:
			r.others = append(r.others, a1)
			i1++
		case 1:
			r.others = append(r.others, a2)
			i2++
		}
	}
	switch {
	case i1 < len(s1.others):
		r.others = append(r.others, s1.others[i1:]...)
	case i2 < len(s2.others):
		r.others = append(r.others, s2.others[i2:]...)
	}
	if r.Equal(s1) {
		// Maintain the string representation.
		return s1
	}
	return r
}

func (s Scope) Holds(r ResourceScope) bool {
	if s.IsUnlimited() {
		return true
	}
	if r == CatalogScope {
		_, ok := slices.BinarySearch(s.repositories, "")
		return ok
	}
	if r.ResourceType == TypeRepository {
		if action := parseKnownAction(r.Action); action != unknownAction {
			// It's a known action on a repository.
			i, ok := slices.BinarySearch(s.repositories, r.Resource)
			if !ok {
				return false
			}
			return s.actions[i]&(1<<action) != 0
		}
	}
	// We're either searching for an unknown resource type or
	// an unknown action on a repository. In any case,
	// we'll find the result in s.other.
	_, ok := slices.BinarySearchFunc(s.others, r, ResourceScope.Compare)
	return ok
}

// Contains reports whether s1 is a (non-strict) superset of s2.
func (s1 Scope) Contains(s2 Scope) bool {
	if s1.IsUnlimited() {
		return true
	}
	if s2.IsUnlimited() {
		return false
	}
	i1 := 0
outer1:
	for i2, repo2 := range s2.repositories {
		for i1 < len(s1.repositories) {
			switch repo1 := s1.repositories[i1]; strings.Compare(repo1, repo2) {
			case 1:
				// repo2 definitely doesn't exist in s1.
				return false
			case 0:
				if (s1.actions[i1] & s2.actions[i2]) != s2.actions[i2] {
					// s2's actions for this repo aren't in s1.
					return false
				}
				i1++
				continue outer1
			case -1:
				i1++
				// continue looking through s1 for repo2.
			}
		}
		// We ran out of repositories in s1 to look for.
		return false
	}
	i1 = 0
outer2:
	for _, sc2 := range s2.others {
		for i1 < len(s1.others) {
			sc1 := s1.others[i1]
			switch sc1.Compare(sc2) {
			case 1:
				return false
			case 0:
				i1++
				continue outer2
			case -1:
				i1++
			}
		}
		return false
	}
	return true
}

func (s1 Scope) Equal(s2 Scope) bool {
	return s1.IsUnlimited() == s2.IsUnlimited() &&
		slices.Equal(s1.repositories, s2.repositories) &&
		slices.Equal(s1.actions, s2.actions) &&
		slices.Equal(s1.others, s2.others)
}

// Canonical returns s with the same contents
// but with its string form made canonical (the
// default is to mirror exactly the string that it was
// created with).
func (s Scope) Canonical() Scope {
	s.original = ""
	return s
}

// String returns the string representation of the scope, as suitable
// for passing to the token refresh "scopes" attribute.
func (s Scope) String() string {
	if s.IsUnlimited() {
		// There's no official representation of this, but
		// we shouldn't be passing an unlimited scope
		// as a scopes attribute anyway.
		return "*"
	}
	if s.original != "" || s.IsEmpty() {
		return s.original
	}
	var buf strings.Builder
	var prev ResourceScope
	// TODO use range when we can use range-over-func.
	s.Iter()(func(s ResourceScope) bool {
		prev0 := prev
		prev = s
		if s.ResourceType == TypeRepository && prev0.ResourceType == TypeRepository && s.Resource == prev0.Resource {
			buf.WriteByte(',')
			buf.WriteString(s.Action)
			return true
		}
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(s.ResourceType)
		if s.Resource != "" || s.Action != "" {
			buf.WriteByte(':')
			buf.WriteString(s.Resource)
			buf.WriteByte(':')
			buf.WriteString(s.Action)
		}
		return true
	})
	return buf.String()
}

func parseKnownAction(s string) knownAction {
	switch s {
	case ActionPull:
		return pullAction
	case ActionPush:
		return pushAction
	default:
		return unknownAction
	}
}
