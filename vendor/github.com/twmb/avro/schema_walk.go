package avro

// Shared child enumeration for the raw-JSON schema walkers: the SchemaCache
// self-containment pair (collectTreeDefs, inlineTreeDefs) and the Root()
// metadata walker. All three need one rule — which keys hold child schemas,
// what a flat-form ("linkedin/goavro") field lifts to, which namespace scope
// each child resolves in — and differ only in what they do at each position.
// walkNodeChildren is that rule, built on the parser's own predicates.

// nodeChildScope returns the namespace scope a node's container children
// resolve in: a named kind opens its own (dotted-name prefix, else the
// "namespace" attribute including the explicit-empty form, else the enclosing
// scope); every other node passes the enclosing scope through.
//
// Keyed on KIND alone. Gating on name presence would be wrong — the build
// registers a fullname even with no "name" key at all (a lax-accepted empty
// short name, fullname "ns."), and the children build under it.
func nodeChildScope(v map[string]any, ns string) string {
	if typ, _ := v["type"].(string); isNamedKind(typ) {
		return nodeNamespace(v, ns)
	}
	return ns
}

// nodeChildVisitor receives one node's child-schema positions from
// walkNodeChildren. A nil callback skips its position. Every key handed to a
// callback is the exact lowercase spelling, the only one that binds, so
// mutating walkers write back to the key actually in the map.
type nodeChildVisitor struct {
	// typeValue is the node's own "type" value, resolving in the ENCLOSING
	// scope rather than the child scope: a wrapped reference {"type":"X"}
	// binds X where the node sits. It is a child position because that
	// reference can name a cache-inherited type.
	typeValue func(key, scope string)
	// fields fires once with the raw array, before any field/flatField, for
	// sizing.
	fields func(arr []any)
	// field is a normal-form record field; the child schema is fo[typeKey].
	// Every fields element that is an object with a "type" key fires exactly
	// one of field/flatField.
	field func(i int, fo map[string]any, typeKey, scope string)
	// flatField is a flat-form ("linkedin/goavro") field: fo's own keys carry
	// the lifted definition of kind, and flatLiftTypeMap(fo, kind) is the
	// lifted view. The decision is the parser's flatFieldNeedsLift, so every
	// walker lifts exactly what the parser lifts.
	flatField func(i int, fo map[string]any, kind, scope string)
	// items / values are the array-items / map-values child, in the child scope.
	items  func(key, scope string)
	values func(key, scope string)

	// fieldNoType fires for a field element with no "type" key, reachable only
	// inside a STRAY "fields" — a bound record build rejects a nil field type.
	fieldNoType func(i int, fo map[string]any)

	// strayKeys also fires the container callbacks on kinds that do not BIND
	// the key, for bodies parsing as the key's schema shape. Default is
	// bound-only, and must stay so for any walker that REGISTERS names: a
	// definition-shaped value inside a stray key would take its fullname in a
	// first-wins store and shadow the real definition everywhere that store
	// feeds.
	//
	// The metadata walker alone sets it, because surfacing stray keys
	// as-written is read-only. Pinned by
	// TestMatrix_MetadataStrayKeySurfacedAsWritten.
	strayKeys bool

	// strayShapeMemo memoizes the stray-body shape checks by subtree pointer,
	// making a nested-stray schema linear instead of O(depth^2). The metadata
	// walker sets one per Root() call; nil takes the plain check. Consulted
	// only when strayKeys is set.
	strayShapeMemo strayShapeMemo
}

// walkNodeChildren enumerates node v's child-schema positions: its own "type"
// value, each record field's type (flat-form fields lifted), array items, map
// values. Keys are read by exact name. ns is the scope v sits in, handed to
// typeValue; childNS is the scope v's containers resolve in.
//
// Order is fixed: type, fields in declaration order, items, values. Each
// container key fires only on the kind that binds it — fields on record/error,
// items on array, values on map — unless vis.strayKeys opts in. Under the
// default a node fires at most one of the three; a strayKeys walk of a
// primitive can fire several.
func walkNodeChildren(v map[string]any, ns, childNS string, vis nodeChildVisitor) {
	typ, _ := v["type"].(string)
	if vis.typeValue != nil {
		if _, ok := v["type"]; ok {
			vis.typeValue("type", ns)
		}
	}
	if vis.fields != nil || vis.field != nil || vis.flatField != nil {
		if fv, ok := v["fields"]; ok &&
			(isRecordKind(typ) || (vis.strayKeys && strayBodyShapeOKMemo(vis.strayShapeMemo, "fields", fv))) {
			if arr, ok := fv.([]any); ok {
				if vis.fields != nil {
					vis.fields(arr)
				}
				for i, f := range arr {
					fo, ok := f.(map[string]any)
					if !ok {
						continue
					}
					tv, ok := fo["type"]
					if !ok {
						// Parseable only inside a stray "fields", where the
						// record build never runs. fieldNoType fills the
						// pre-sized slot with the element's own attributes
						// rather than a fabricated zero field.
						if vis.fieldNoType != nil {
							vis.fieldNoType(i, fo)
						}
						continue
					}
					if ts, isStr := tv.(string); isStr && flatFieldNeedsLift(fo, ts) {
						if vis.flatField != nil {
							vis.flatField(i, fo, ts, childNS)
						}
						continue
					}
					if vis.field != nil {
						vis.field(i, fo, "type", childNS)
					}
				}
			}
		}
	}
	if vis.items != nil {
		if iv, ok := v["items"]; ok &&
			(typ == "array" || (vis.strayKeys && strayBodyShapeOKMemo(vis.strayShapeMemo, "items", iv))) {
			vis.items("items", childNS)
		}
	}
	if vis.values != nil {
		if vv, ok := v["values"]; ok &&
			(typ == "map" || (vis.strayKeys && strayBodyShapeOKMemo(vis.strayShapeMemo, "values", vv))) {
			vis.values("values", childNS)
		}
	}
}
