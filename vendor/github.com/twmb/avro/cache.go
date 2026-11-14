package avro

import (
	"crypto/sha256"
	"encoding/json"
	"maps"
	"strings"
	"sync"
)

// SchemaCache accumulates named types across multiple [SchemaCache.Parse]
// calls, allowing schemas to reference types defined in previously parsed
// schemas — the shape a Schema Registry's inter-schema references take.
//
// Schemas must be parsed in dependency order: referenced types must be
// parsed before the schemas that reference them.
//
// Parsing the same schema string more than once is allowed and returns the
// previously parsed result, so diamond dependencies (A→B→D, A→C→D) need no
// caller-side tracking. Options that change what the string compiles to —
// custom types or [WithLaxNames] — skip this deduplication and re-parse, since
// the string alone no longer identifies the result. Deduplication normalizes
// JSON whitespace and key order but not the Avro canonical form: schemas
// differing only in formatting dedupe, while differences in non-canonical
// fields like doc or aliases return a duplicate type error.
//
// Each returned [*Schema] is fully resolved and independent of the cache. That
// extends to sub-schemas: a node extracted from [Schema.Root] converts via
// [SchemaNode.Schema] with every cross-parse reference resolved, so the cache
// is never needed again once Parse returns.
//
// [WithLaxNames] is sticky: if a type is defined with it, pass it to every
// later Parse that references that type. A schema containing a lax name is not
// parseable without it, cache or no cache, so the referencing Parse's
// [Schema.String] and [Schema.Canonical] output also needs WithLaxNames to
// re-parse. [Schema.Encode] and [Schema.Decode] are unaffected either way.
//
// The zero value is ready to use. A SchemaCache is safe for concurrent use.
type SchemaCache struct {
	mu    sync.Mutex
	named map[string]*namedType
	dedup map[[32]byte]*Schema
	// skipDedupParsed holds the schema strings this cache has parsed under
	// options that skip dedup — custom types or WithLaxNames. It exists to
	// answer one question, asked in one place: may this string re-DEFINE a
	// name the cache already holds? A string the cache has compiled before
	// under any option may; a new string may not.
	//
	// Which option it was is not that question, and nothing asks it. Two
	// maps here would be two sets that no read tells apart, which is a
	// second thing to keep in step for no answer gained — and the same
	// string parsed both ways would sit in both.
	skipDedupParsed map[[32]byte]bool
	// defs holds each registered named type's self-contained JSON definition,
	// all attributes included, keyed by fullname. A later Parse referencing an
	// earlier type splices the definition back in at the first dangling
	// reference, keeping Canonical/Fingerprint/Root/String self-contained with
	// every original attribute. See inlineTreeDefs.
	defs map[string]any
}

// cacheNormalizeSchema re-marshals a schema string so two spellings of one
// schema — different whitespace, different key order — share a dedup key.
//
// Only a schema that is EXACTLY one JSON value normalizes. A decode that stops
// at the first value would silently drop whatever followed, turning a schema
// bare [Parse] rejects into one the cache truncates and accepts, so anything
// with trailing content is handed back untouched for parse() to refuse on its
// own terms. Same strict rule parseSchemaTree applies, asked of the same
// decoder rather than spelled again here.
func cacheNormalizeSchema(schema string) string {
	v, err := decodeSchemaAnyStrict(schema)
	if err != nil {
		return schema
	}
	normalized, err := json.Marshal(v)
	if err != nil {
		return schema
	}
	return string(normalized)
}

// Parse parses a schema string, registering any named types (records, enums,
// fixed) in the cache. Named types from previous Parse calls are available
// for reference resolution. On failure, the cache is not modified.
func (c *SchemaCache) Parse(schema string, opts ...SchemaOpt) (*Schema, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.named == nil {
		c.named = make(map[string]*namedType)
		c.dedup = make(map[[32]byte]*Schema)
		c.skipDedupParsed = make(map[[32]byte]bool)
		c.defs = make(map[string]any)
	}

	schema = cacheNormalizeSchema(schema)
	// Clone the cache's map so a failed parse doesn't corrupt the cache.
	cloned := maps.Clone(c.named)
	b := &builder{
		named:      cloned,
		building:   make(map[*schemaNode]struct{}),
		definedSet: make(map[*namedType]bool),
		minBytes:   newMinBytesWalk(),
	}
	applySchemaOpts(b, opts)
	hasCustomTypes := len(b.customTypes) > 0
	// WithLaxNames sets a non-default name validator (b.checkName), which
	// changes what the same schema string compiles to (a name strict Parse
	// rejects becomes accepted). The dedup key is the schema string only,
	// so lax parses must skip dedup the way custom types do — otherwise a
	// lax-then-strict call sequence returns the cached lax schema to the
	// strict caller (silently accepting an invalid name), and a
	// strict-then-lax sequence returns the strict schema ignoring the opt.
	hasLaxNames := b.checkName != nil
	skipDedup := hasCustomTypes || hasLaxNames

	h := sha256.Sum256([]byte(schema))
	if !skipDedup {
		if s, ok := c.dedup[h]; ok {
			return s, nil
		}
	}

	// cachedNames marks every name INHERITED from the cache. It alone cannot
	// tell an inherited REFERENCE from a name this parse RE-DEFINES, since a
	// re-registered name is in both: rejectCachedRefIfCustomTypeWouldMatch
	// therefore keys its "defined this parse" skip on definedSet membership of
	// the resolved *namedType, and reads cachedNames only to recognize a
	// genuine cross-parse reference (present here, but its nt is the cloned
	// cached node, absent from definedSet). The duplicate-name check pairs
	// cachedNames with allowReRegister to permit a same-string re-parse.
	// Populated for EVERY parse with inherited names, since the cross-parse
	// custom-boundary guard must fire even for a plain parse that references a
	// custom-built cached type.
	if len(cloned) > 0 {
		b.cachedNames = make(map[string]bool, len(cloned))
		for name := range cloned {
			b.cachedNames[name] = true
		}
	}
	// allowReRegister lets a parse re-DEFINE an inherited name instead of
	// erroring, granted ONLY when the cache has already seen this exact schema
	// string — under dedup if it was parsed strictly, under skipDedupParsed if
	// it was parsed with custom types or lax names. Such a re-parse re-enters
	// the builder and re-registers the names IT defined — the same string, not
	// a conflict. A NEW string re-defining an inherited name still errors,
	// custom mode included: a custom parse that merely REFERENCES a cached name
	// never needs this, and a conflicting redefinition must error regardless.
	b.allowReRegister = c.dedup[h] != nil || c.skipDedupParsed[h]

	s, err := parse(schema, b)
	if err != nil {
		return nil, err
	}

	// A parse REFERENCING a type from a prior cache Parse resolves it in the
	// node tree but leaves a bare reference in the JSON forms, making
	// Canonical/Fingerprint/Root/String non-self-contained. That breaks the
	// documented "independent of the cache" contract and cross-impl
	// fingerprint interop. Splice the inherited definitions into the ORIGINAL
	// JSON, preserving attributes the node tree never stored like doc, order
	// and field props, and rebuild the metadata forms from it cache-LESSLY so
	// the inlined definition is not itself seen as inherited. The
	// encode/decode path is untouched.
	//
	// The trigger is "inlineTreeDefs spliced something", not "s.full does not
	// re-parse cache-lessly". The latter misses a reference that re-parses to
	// the WRONG type: a bare reference to an inherited type that, cache-lessly,
	// FORWARD-binds to a same-named type defined LATER in this schema. Eager
	// binding sent the wire reference to the inherited type (NOT_BUGS.md #24),
	// so the node tree is right, but s.full re-parses cleanly with the
	// reference bound locally and its metadata then describes a DIFFERENT
	// schema than the wire codec. A schema referencing nothing inherited
	// splices nothing and keeps its String().
	selfContained := s.full
	if len(cloned) > 0 {
		if tree, terr := unmarshalAnyPreservePrecision(s.full); terr == nil {
			inlined := make(map[string]bool)
			tree = inlineTreeDefs(tree, "", c.defs, make(map[string]bool), inlined)
			if len(inlined) > 0 {
				if marshaled, merr := json.Marshal(tree); merr == nil {
					s2, rerr := Parse(string(marshaled), opts...)
					if rerr != nil {
						// The spliced form may carry a name an inherited type
						// was defined with under WithLaxNames, which a
						// re-parse under this call's opts rejects. Lax names
						// are sticky (see the SchemaCache doc). Retry with the
						// internal accept-everything validator, appended last
						// so it wins over any user lax fn — the retry only
						// broadens — leaving the metadata forms describing the
						// full self-contained schema rather than a dangling
						// reference. Accept-all is sound here because every
						// spliced name was validated by the parse that defined
						// it and names pass through verbatim, so the canonical
						// bytes match a standalone lax parse.
						laxOpts := make([]SchemaOpt, len(opts)+1)
						copy(laxOpts, opts)
						laxOpts[len(opts)] = internalReparseNames
						s2, rerr = Parse(string(marshaled), laxOpts...)
					}
					if rerr == nil {
						selfContained = s2.full
						// Adopting s2's canonical form adopts its SOE header
						// too: the header is a pure function of c, hashed on
						// first use, and nothing has reached that first use on
						// this still-unpublished schema.
						s.c = s2.c
						s.full = s2.full
					}
				}
			}
		}
	}

	// Record this parse's named-type definitions (from the self-contained form,
	// so each captured definition has its own transitive references spliced in)
	// for future cross-parse references.
	if tree, terr := unmarshalAnyPreservePrecision(selfContained); terr == nil {
		collectTreeDefs(tree, "", func(fn string, def any) {
			if _, ok := c.defs[fn]; !ok {
				// Store with the namespace made explicit so the definition
				// resolves to the same fullname wherever it is later spliced,
				// regardless of the enclosing namespace at the reference site.
				c.defs[fn] = defWithExplicitNamespace(def, fn)
			}
		})
	}

	// Named types are safe to cache unconditionally: applyCustomTypes
	// wraps b.ser/b.deser without mutating the node's ser/deser, so
	// cached named type nodes keep their unwrapped functions.
	c.named = b.named
	// The same condition that decided whether to CONSULT dedup decides which
	// side to record on, so the two cannot fall out of step: a schema whose
	// result the string alone does not identify is remembered as parsed, not
	// as a result to hand back.
	if skipDedup {
		c.skipDedupParsed[h] = true
	} else {
		c.dedup[h] = s
	}
	return s, nil
}

// --- cross-parse self-containment (SchemaCache.defs) ---
//
// The helpers below splice each cache-inherited definition back into the
// schema JSON at its first reference, so Canonical/Fingerprint/Root/String
// stay self-contained with every original attribute. They work on the generic
// JSON tree — a lossless round-trip via unmarshalAnyPreservePrecision +
// json.Marshal — tracking namespace scope so references resolve to the right
// fullname.

// avroNamedRef reports whether a type string names a type (a reference), as
// opposed to a primitive or a container/definition keyword.
func avroNamedRef(typ string) bool {
	switch typ {
	case "null", "boolean", "int", "long", "float", "double", "bytes", "string",
		"record", "enum", "array", "map", "union", "fixed", "error":
		return false
	}
	return typ != ""
}

// nodeNamespace returns the namespace in scope inside a named-type object: the
// prefix of a dotted name, else its "namespace" attribute, else inherited.
// Keys are read by exact name, mirroring the parser.
func nodeNamespace(obj map[string]any, enclosingNS string) string {
	// A dotted name carries its own namespace and suppresses the attribute
	// even when the prefix is empty (".x"), so presence of the dot — not a
	// non-empty split — is what decides. namespaceOf performs the split.
	if name, ok := obj["name"].(string); ok && strings.ContainsRune(name, '.') {
		return namespaceOf(name)
	}
	if ns, ok := obj["namespace"].(string); ok {
		return ns
	}
	return enclosingNS
}

// nodeFullnameTree resolves a named-type object's fullname.
func nodeFullnameTree(obj map[string]any, enclosingNS string) string {
	name, _ := obj["name"].(string)
	short := unqualified(name)
	if ns := nodeNamespace(obj, enclosingNS); ns != "" {
		return ns + "." + short
	}
	return short
}

// collectTreeDefs calls visit for every named-type definition in the tree, with
// its resolved fullname and subtree, tracking namespace scope. Child positions,
// key casing and the flat-form field lift come from walkNodeChildren, so every
// definition is collected exactly where the parser registers it. The node's own
// "type" value is a string on a parser-accepted tree and defines nothing.
//
// The visit fires for every named KIND, "name" key or not, with nodeChildScope:
// the parser registers a fullname even with no name key (fullname "ns.") and
// scopes children by its namespace attribute regardless. Gating on name-key
// presence misfiles nested defs under ENCLOSING-scoped fullnames — a
// cross-parse reference then finds nothing to splice, and a parse that
// references-then-locally-defines the misfiled short name splices the STALE def
// over its own, producing metadata for a schema the wire codec rejects. The ""
// fullname is collected but inert; no reference can spell it.
func collectTreeDefs(node any, ns string, visit func(fullname string, def any)) {
	switch v := node.(type) {
	case []any:
		for _, b := range v {
			collectTreeDefs(b, ns, visit)
		}
	case map[string]any:
		if typ, _ := v["type"].(string); isNamedKind(typ) {
			visit(nodeFullnameTree(v, ns), v)
		}
		walkNodeChildren(v, ns, nodeChildScope(v, ns), nodeChildVisitor{
			field: func(_ int, fo map[string]any, typeKey, scope string) {
				collectTreeDefs(fo[typeKey], scope, visit)
			},
			flatField: func(_ int, fo map[string]any, kind, scope string) {
				collectTreeDefs(flatLiftTypeMap(fo, kind), scope, visit)
			},
			items:  func(key, scope string) { collectTreeDefs(v[key], scope, visit) },
			values: func(key, scope string) { collectTreeDefs(v[key], scope, visit) },
		})
	}
}

// inlineTreeDefs replaces the FIRST occurrence of each reference to a cache-
// inherited named type (in defs, not defined locally before the reference, not
// already inlined) with its definition, recursing into the inlined copy so
// transitive references resolve too. Subsequent occurrences stay bare. The def
// is deep-copied before any mutation so the cache is not corrupted.
//
// Binding mirrors the parser EXACTLY — eager, in-scope-first, POSITIONAL. At
// each scopedRefKeys candidate a LOCAL definition registered AT THIS POINT IN
// THE WALK beats a cache-inherited one, and seen accumulates local fullnames in
// DFS pre-order as the parser registers them. So a ref AFTER a local def stays
// bare while a ref BEFORE it splices the cached type. A position-INdependent
// local set would keep that second ref bare and diverge the JSON forms from the
// wire codec.
//
// The walk also dedupes DEFINITIONS: spliced defs are self-contained, so two
// inherited refs sharing a transitive type would each carry the shared name's
// definition, a duplicate the rebuild Parse rejects. A second definition is
// rewritten to a reference to the first.
func inlineTreeDefs(node any, ns string, defs map[string]any, seen, inlined map[string]bool) any {
	switch v := node.(type) {
	case string:
		if !avroNamedRef(v) {
			return v
		}
		var keys [2]string
		for _, key := range scopedRefKeys(&keys, v, ns) {
			if seen[key] {
				return v // bound to a local def already in scope; keep it bare
			}
			if def, ok := defs[key]; ok {
				if inlined[key] {
					return v
				}
				inlined[key] = true
				return inlineTreeDefs(deepCopyTree(def), ns, defs, seen, inlined)
			}
		}
		return v
	case []any:
		for i := range v {
			v[i] = inlineTreeDefs(v[i], ns, defs, seen, inlined)
		}
		return v
	case map[string]any:
		// A wrapped name reference — {"type":"X"} with optional non-structural
		// keys — is the bare "X" plus its props. The general path below would
		// recurse INTO the "type" value and emit the invalid {"type":{X-def}}
		// when X splices, which the rebuild Parse rejects, degrading the
		// metadata to a dangling reference. A wrapper cannot carry
		// schema-shaped structural keys, so its extras are inert.
		//
		// When X splices, the definition replaces the wrapper and its PROPS
		// ride on it, definition winning collisions. Reserved usage-site
		// attributes do not survive; Java drops usage-site extras entirely, and
		// props are canonical-stripped, so schema identity is unchanged across
		// all three reference spellings.
		//
		// When X stays bare, a SOLE-key wrapper collapses to the bare
		// spelling: a later wrapped reference to an already-inlined type would
		// otherwise keep {"type":"X"} where "X" belongs and diverge String()
		// from its bare-spelled twin. A props-carrying wrapper keeps its shape.
		if ref, ok := v["type"].(string); ok && avroNamedRef(ref) {
			resolved := inlineTreeDefs(ref, ns, defs, seen, inlined)
			if def, isMap := resolved.(map[string]any); isMap {
				defTyp, _ := def["type"].(string)
				defLogical, _ := def["logicalType"].(string)
				// The wrapper's props are a flat key set, never a
				// nested-stray schema, so a nil verdict costs one fresh
				// shape check per key and nothing compounds. Definition-wins
				// is an exact-key presence check, and map keys are unique,
				// so merging one prop cannot change another's verdict —
				// the merge is order-independent.
				for k, wv := range v {
					if schemaReservedKeyForObject(k, wv, defTyp, defLogical, nil) {
						continue
					}
					if _, has := def[k]; has {
						continue
					}
					def[k] = wv
				}
				return def
			}
			if len(v) == 1 {
				return resolved
			}
			v["type"] = resolved
			return v
		}
		// Register this node's own name BEFORE walking its children, mirroring
		// the parser's early self-registration, so a later sibling or
		// descendant sees it. Fires for every named KIND, "name" key or not: a
		// spliced subtree can carry a keyless definition, and a later reference
		// must find it in scope or the walk splices a second copy and the
		// rebuild rejects the duplicate.
		//
		// A name ALREADY defined here means a SECOND definition inside another
		// spliced subtree — two inherited refs sharing a transitive type (the
		// diamond A→{B,C}→D), or a nested type referenced before its container.
		// The node tree shares ONE resolved type per name, so the JSON keeps
		// the first definition and references it thereafter, as Java's toString
		// does. Without the rewrite the rebuilt JSON defines the name twice,
		// Parse rejects it, and the metadata falls back to the dangling
		// original.
		if typ, _ := v["type"].(string); isNamedKind(typ) {
			fullname := nodeFullnameTree(v, ns)
			if ref, ok := dupDefRef(fullname, ns, seen); ok {
				return ref
			}
			seen[fullname] = true
		}
		inlineNodeChildren(v, ns, defs, seen, inlined)
		return v
	}
	return node
}

// inlineNodeChildren splices inherited refs in v's child-schema positions
// via walkNodeChildren: the node's own "type" value (a bare reference there
// resolves in the ENCLOSING scope — see nodeChildVisitor.typeValue), each
// record field's type, and array items / map values (the node's own child
// scope). A flat-form ("linkedin/goavro") field recurses IN PLACE: the field
// object carries the lifted type's structural keys, so the splice walks
// exactly the children the parser lifts and mutations land in the original
// tree. Its "type" value is a bare kind string, never a reference, so the
// typeValue splice leaves it alone.
//
// A flat NAMED field opens its own namespace scope and, when it re-defines an
// already-seen name, is rewritten to the equivalent normal-form reference
// field — inlineTreeDefs's duplicate case, except a field object cannot be
// replaced by a bare string. A flat UNNAMED field keeps the enclosing record's
// scope, since the lift drops name/namespace for unnamed kinds. A flat field
// carrying a structural key of a DIFFERENT kind never reaches here: the lift
// keeps the stray and the per-kind build fails the parse.
func inlineNodeChildren(v map[string]any, ns string, defs map[string]any, seen, inlined map[string]bool) {
	spliceAt := func(key, scope string) {
		v[key] = inlineTreeDefs(v[key], scope, defs, seen, inlined)
	}
	walkNodeChildren(v, ns, nodeChildScope(v, ns), nodeChildVisitor{
		typeValue: spliceAt,
		field: func(_ int, fo map[string]any, typeKey, scope string) {
			fo[typeKey] = inlineTreeDefs(fo[typeKey], scope, defs, seen, inlined)
		},
		flatField: func(_ int, fo map[string]any, kind, scope string) {
			if isNamedKind(kind) {
				fullname := nodeFullnameTree(fo, scope)
				if ref, ok := dupDefRef(fullname, scope, seen); ok {
					rewriteFlatFieldToRef(fo, ref)
					return
				}
				seen[fullname] = true
			}
			inlineNodeChildren(fo, scope, defs, seen, inlined)
		},
		items:  spliceAt,
		values: spliceAt,
	})
}

// dupDefRef decides how a SECOND definition of fullname, at a position whose
// enclosing namespace is ns, is replaced by a reference to the first. Returns
// the spelling and true when one is expressible; ("", false) leaves the
// duplicate in place, so the rebuild Parse fails and the metadata falls back
// to the dangling original — degraded, but never describing a different schema
// than the wire codec.
//
// A dotted fullname is an exact lookup on re-parse, so it is always safe. A
// null-namespace name has only its bare short name, which the parser binds
// enclosing-namespace-first: safe unless a same-short-name type qualified by
// the enclosing namespace is already defined at this point. Binding is eager
// and positional, so only an EARLIER definition can capture it. Avro has no
// absolute-reference syntax for the shadowed case (Java's toString shares the
// limitation), so it stays a definition.
func dupDefRef(fullname, ns string, seen map[string]bool) (string, bool) {
	// The "" fullname (a keyless definition with no namespace in scope)
	// has no reference spelling at all — avroNamedRef rejects the empty
	// string — so a second definition stays in place unconditionally.
	if fullname == "" || !seen[fullname] {
		return "", false
	}
	if strings.Contains(fullname, ".") {
		return fullname, true
	}
	if ns == "" || !seen[ns+"."+fullname] {
		return fullname, true
	}
	return "", false
}

// rewriteFlatFieldToRef converts a flat-form field that re-defines an
// already-defined named type into the normal-form field whose "type" is a name
// reference. It keeps exactly the keys liftFlatFieldType treats as field-only
// — name, default, order, aliases. Everything else belongs to the TYPE and is
// carried by its first definition.
func rewriteFlatFieldToRef(fo map[string]any, ref string) {
	for k := range fo {
		switch k {
		case "name", "default", "order", "aliases", "type":
		default:
			delete(fo, k)
		}
	}
	fo["type"] = ref
}

// defWithExplicitNamespace deep-copies a named-type definition and makes its
// namespace explicit from the resolved fullname: "name" becomes the short
// name, "namespace" the resolved namespace — "" for the null namespace, which
// forces null even inside a namespaced splice site. Without it, a definition
// relying on an inherited namespace would re-inherit whatever schema it is
// later spliced into and resolve to the WRONG fullname (a.b.Inner becoming
// c.d.Inner). Only the top level is rewritten, so nested types either resolve
// against this now-explicit namespace or keep their own — both inheritance and
// explicit override survive the splice.
func defWithExplicitNamespace(def any, fullname string) any {
	cp := deepCopyTree(def)
	if obj, ok := cp.(map[string]any); ok {
		obj["name"] = unqualified(fullname)
		obj["namespace"] = namespaceOf(fullname)
	}
	return cp
}

// deepCopyTree deep-copies a generic JSON tree.
func deepCopyTree(node any) any {
	switch v := node.(type) {
	case map[string]any:
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[k] = deepCopyTree(val)
		}
		return m
	case []any:
		s := make([]any, len(v))
		for i, e := range v {
			s[i] = deepCopyTree(e)
		}
		return s
	default:
		return v
	}
}
