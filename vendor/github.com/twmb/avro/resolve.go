package avro

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"sync"
)

// Resolve returns a schema that decodes data written with the writer schema
// and produces values matching the reader schema's layout. The writer schema
// is what the data was encoded with (typically from an OCF file header or
// a schema registry); the reader schema is what your application expects now.
//
// Decoding with the returned schema handles field addition (defaults), field
// removal (skip), renaming (aliases), reordering, and type promotion.
// Encoding with it uses the reader's format.
//
// [CheckCompatibility] runs first, and any incompatibility comes back as a
// [*CompatibilityError]; if it passes and the canonical forms are identical,
// reader is returned as-is. The check must precede that fast path: the parsing
// canonical form strips logicalType, precision and scale, so two schemas with
// equal canonical forms can still be logically incompatible — a decimal
// precision/scale mismatch most of all — and would otherwise pass the fast path
// and silently rescale the decoded value.
//
// Note: the argument order is (writer, reader), matching source-then-destination
// convention and Java's GenericDatumReader. This differs from the Avro spec
// text and hamba/avro, which put reader first.
func Resolve(writer, reader *Schema) (*Schema, error) {
	if err := CheckCompatibility(writer, reader); err != nil {
		return nil, err
	}
	if bytes.Equal(reader.Canonical(), writer.Canonical()) {
		return reader, nil
	}
	ctx := &resolveCtx{
		seen:     make(map[nodePair]*schemaNode),
		custom:   reader.custom,
		minBytes: newMinBytesWalk(),
	}
	resolved, err := resolveNode(reader.node, writer.node, "", ctx)
	if err != nil {
		return nil, err
	}
	s := &Schema{
		ser:         reader.ser,
		deser:       resolved.deser,
		c:           reader.c,
		node:        reader.node,
		full:        reader.full,
		custom:      reader.custom,
		customBaked: reader.customBaked,
	}
	s.resolveWriter = writer
	// decodeJSONResolved transforms writer-shaped JSON into RAW writer binary
	// before the resolving decode; a writer carrying its own CustomType decoders
	// would run them during that transform and then fail to re-encode the
	// resulting Go-domain value (a Decode-only custom has no Encode to invert it).
	// Re-parse the writer custom-free for that round-trip; the reader's custom
	// types still apply in the final s.Decode. Names are accepted wholesale
	// (internalReparseNames) so any writer the user's validator already accepted
	// re-parses — names do not affect wire bytes. With no custom effects
	// anywhere in the writer's tree there is nothing to suppress, so reuse it
	// directly. customBaked, not len(writer.custom): a cache-parsed writer
	// whose customs match only SchemaCache-inherited subtrees has an empty
	// overlay while the inherited ser/deser still carry the baked conversions.
	s.resolveWriterRaw = writer
	if writer.customBaked {
		raw, err := Parse(writer.full, internalReparseNames)
		if err != nil {
			return nil, fmt.Errorf("avro: building custom-free writer view for resolved JSON decode: %w", err)
		}
		s.resolveWriterRaw = raw
	}
	// s.soe needs no assignment: the header is a pure function of the
	// canonical form, and s.c is the reader's, so s's own lazy hash produces
	// the reader's header byte for byte — without forcing the reader to hash
	// at all when nothing here ever touches single-object bytes. The WRITER's
	// header is reached through s.resolveWriter above, which DecodeSingleObject
	// asks so writer-produced wire decodes here (acceptsWriterSOE).
	//
	// The resolved Schema also accepts the READER's fingerprint, but the
	// payload after it must still be WRITER-shaped, since s.deser consumes
	// writer bytes. Feeding back reader.AppendSingleObject output is NOT a
	// supported round-trip: it errors on dropped writer fields or silently
	// default-fills added ones, exactly as reader-shaped JSON does to a
	// resolved DecodeJSON. Use the reader schema directly for reader-shaped
	// data.
	return s, nil
}

// resolvedRecord holds the compiled resolution between a reader and writer record.
type resolvedRecord struct {
	readerNames    []string
	readerNameVals []reflect.Value // pre-computed reflect.ValueOf(name); avoids alloc per SetMapIndex
	wireOps        []wireOp
	defaults       []defaultOp
	cache          sync.Map
}

// wireOp describes how to handle a single writer field during deserialization.
type wireOp struct {
	readerIdx int     // index in the reader's field list; -1 means skip
	read      deserfn // non-nil when readerIdx >= 0
	skip      skipfn  // non-nil when readerIdx == -1
}

// defaultOp fills in a reader field that is absent from the writer.
type defaultOp struct {
	readerIdx      int
	encodedDefault []byte
	deser          deserfn
}

// resolveCtx carries per-resolution state through the recursive resolve calls.
type resolveCtx struct {
	seen   map[nodePair]*schemaNode
	custom map[*schemaNode]*customWiring
	// minBytes is shared across every container this resolution derives a
	// per-element bound for. resolveArray/resolveMap and the dropped-field
	// skip compiler all consult it, so a writer pointing many containers at one
	// subtree pays for that subtree once, not once per container. See
	// newMinBytesWalk.
	minBytes *minBytesWalk
}

// customDecodersFor returns the decoder chain registered against r, or
// nil if none. Sibling of [resolveCtx.customSNFor].
func (ctx *resolveCtx) customDecodersFor(r *schemaNode) []func(any, *SchemaNode) (any, error) {
	if w := ctx.custom[r]; w != nil {
		return w.decoders
	}
	return nil
}

// maybeWrapResolvedNode re-applies custom decoders from the reader
// schema to a resolved node that uses the reader node directly.
func maybeWrapResolvedNode(r *schemaNode, ctx *resolveCtx) *schemaNode {
	if len(ctx.customDecodersFor(r)) == 0 {
		return r
	}
	nd := &schemaNode{
		kind:  r.kind,
		name:  r.name,
		ser:   r.ser,
		deser: r.deser,
	}
	ctx.applyCustomToNode(nd, r)
	return nd
}

// applyCustomToNode wraps nd.deser + nd.decodeJSON with the custom
// decoders registered against r. No-op when no CT is registered.
// Shared by the resolveArray/resolveMap/resolveEnum/promote sites in
// doResolve so all four agree on the wrap pair.
func (ctx *resolveCtx) applyCustomToNode(nd, r *schemaNode) {
	w := ctx.custom[r]
	if w == nil || len(w.decoders) == 0 {
		return
	}
	nd.deser = wrapDeserWithCustomDecoders(nd.deser, w.decoders, w.sn)
	nd.decodeJSON = wrapDecodeJSONWithCustomDecoders(w.decoders, w.sn, w.suppressLogical)
}

// resolveNode resolves a (reader, writer) schema pair, handling cycles
// from self-referencing records (e.g. a linked list node). The three
// states in the seen map are:
//   - absent: not yet visited — proceed with resolution
//   - nil:    in-progress — a recursive call hit this pair, creating a cycle
//   - *node:  resolved — reuse the result
//
// On cycle detection, we create a placeholder node whose deser is a
// trampoline closure (calls n.deser through the pointer). After the
// real resolution completes, we copy the resolved node's contents into
// the placeholder so all holders of the placeholder pointer get the
// real implementation.
func resolveNode(r, w *schemaNode, path string, ctx *resolveCtx) (*schemaNode, error) {
	pair := nodePair{r, w}
	if n, ok := ctx.seen[pair]; ok {
		if n == nil {
			n = &schemaNode{}
			n.deser = func(src []byte, v reflect.Value, sl *slab) ([]byte, error) {
				return n.deser(src, v, sl)
			}
			ctx.seen[pair] = n
		}
		return n, nil
	}
	ctx.seen[pair] = nil // mark as in-progress

	resolved, err := doResolve(r, w, path, ctx)
	if err != nil {
		return nil, err
	}

	if placeholder := ctx.seen[pair]; placeholder != nil && placeholder != resolved {
		*placeholder = *resolved
		resolved = placeholder
	}
	ctx.seen[pair] = resolved
	return resolved, nil
}

func doResolve(r, w *schemaNode, path string, ctx *resolveCtx) (*schemaNode, error) {
	if w.kind == "union" && r.kind != "union" {
		return resolveWriterUnion(r, w, path, ctx)
	}
	if r.kind == "union" && w.kind != "union" {
		return resolveReaderUnion(r, w, path, ctx)
	}
	if r.kind == "union" && w.kind == "union" {
		return resolveUnionUnion(r, w, path, ctx)
	}

	if r.kind == w.kind {
		switch r.kind {
		case "record":
			return resolveRecord(r, w, path, ctx)
		case "enum":
			return resolveEnum(r, w, ctx)
		case "array":
			return resolveArray(r, w, path, ctx)
		case "map":
			return resolveMap(r, w, path, ctx)
		default:
			// Everything else — the primitives, and fixed — resolves to the
			// reader node itself. fixed needs no arm of its own despite being
			// a named type: CheckCompatibility ran first and already required
			// the names and the sizes to match (checkSameKind), so there is
			// nothing left to reconcile.
			return maybeWrapResolvedNode(r, ctx), nil
		}
	}

	pd := promotionDeser(w.kind, r.kind)
	if pd != nil {
		deser := deserfn(pd)
		// The bare promotion deser drops the reader's logical type, so writer
		// "int" → reader {"long","logicalType":"timestamp-millis"} would give
		// int64 instead of time.Time. Java applies the conversion after the
		// widening the same way (Resolver.java:154-165).
		//
		// A CustomType that suppresses the reader's built-in logical decoder
		// must keep suppressing through promotion, so the custom decoder — or,
		// with no Decode callback, the user — sees the raw value on a promoted
		// wire exactly as on a direct one. Without the gate it gets the
		// enriched type on one and the raw type on the other.
		if pdLogical := promotionDeserForLogical(w.kind, r); pdLogical != nil {
			if cw := ctx.custom[r]; cw == nil || !cw.suppressLogical {
				deser = pdLogical
			}
		}
		nd := &schemaNode{
			kind:  r.kind,
			ser:   r.ser,
			deser: deser,
		}
		ctx.applyCustomToNode(nd, r)
		return nd, nil
	}

	return nil, &CompatibilityError{
		Path:       pathOrRoot(path),
		ReaderType: r.kind,
		WriterType: w.kind,
		Detail:     "incompatible types",
	}
}

func resolveRecord(r, w *schemaNode, path string, ctx *resolveCtx) (*schemaNode, error) {
	// One lookup for the whole record, asked once per writer field below.
	readerByName := newReaderFieldLookup(r)

	rr := &resolvedRecord{
		readerNames:    make([]string, len(r.fields)),
		readerNameVals: make([]reflect.Value, len(r.fields)),
	}
	for i, rf := range r.fields {
		rr.readerNames[i] = rf.name
		rr.readerNameVals[i] = reflect.ValueOf(rf.name)
	}

	// Track which reader fields are matched. Also tracks WHICH writer
	// field name claimed each reader slot, so a second writer field
	// resolving to the same reader index (via the alias-rename collision
	// described below) produces a useful error rather than a silent
	// last-writer-wins overwrite.
	readerMatched := make([]bool, len(r.fields))
	matchedByWriterName := make([]string, len(r.fields))

	for _, wf := range w.fields {
		ri := readerByName.index(wf.name)
		if ri < 0 {
			rr.wireOps = append(rr.wireOps, wireOp{
				readerIdx: -1,
				skip:      buildSkip(wf.node, ctx.minBytes),
			})
			continue
		}
		// Alias-rename collision: a previous writer field already
		// resolved to this reader-field index (either by exact name
		// match for one and alias match for the other, or both via
		// aliases). Java applyAliases renames the writer field and
		// then Schema.setFields rejects the resulting duplicate
		// (Schema.java:978-981). fastavro deletes the reader-field
		// from its lookup dict on first claim so the second falls
		// through to skip_data (_read_py.py:553). twmb aligns with
		// Java's fail-fast posture — matches the rest of the package
		// (writer-union incompatibility, eager schema-resolution fail)
		// and surfaces the configuration error at Resolve time rather
		// than producing silent data loss on every decode.
		if readerMatched[ri] {
			return nil, &CompatibilityError{
				Path:       pathOrRoot(path),
				ReaderType: "record",
				WriterType: "record",
				Detail: fmt.Sprintf("writer fields %q and %q both resolve to reader field %q (via name + alias collision); rename the writer or drop the alias to disambiguate",
					truncForError(matchedByWriterName[ri]), truncForError(wf.name), truncForError(r.fields[ri].name)),
			}
		}
		readerMatched[ri] = true
		matchedByWriterName[ri] = wf.name
		rf := &r.fields[ri]
		resolved, err := resolveNode(rf.node, wf.node, fieldPath(path, rf.name), ctx)
		if err != nil {
			return nil, err
		}
		rr.wireOps = append(rr.wireOps, wireOp{
			readerIdx: ri,
			read:      resolved.deser,
		})
	}

	for i, rf := range r.fields {
		if readerMatched[i] {
			continue
		}
		encoded, err := encodeDefault(nil, rf.defaultVal, rf.node)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", truncForError(fieldPath(path, rf.name)), err)
		}
		deser := rf.node.deser
		if w := ctx.custom[rf.node]; w != nil && len(w.decoders) > 0 {
			deser = wrapDeserWithCustomDecoders(deser, w.decoders, w.sn)
		}
		rr.defaults = append(rr.defaults, defaultOp{
			readerIdx:      i,
			encodedDefault: encoded,
			deser:          deser,
		})
	}

	nd := &schemaNode{
		kind:        "record",
		name:        r.name,
		aliases:     r.aliases,
		bareAliases: r.bareAliases,
		fields:      r.fields,
		ser:         r.ser,
		deser:       rr.buildDeser(),
		serRecord:   r.serRecord,
		deserRecord: r.deserRecord,
	}
	// Re-apply a record-level CustomType (AvroType:"record") to the resolved
	// node, exactly as resolveEnum / resolveArray / resolveMap / fixed /
	// promotion do via applyCustomToNode. The direct (non-resolved) build
	// wires record nodes in applyCustomTypes, so a CustomType.Decode for a
	// record node fires on a plain Decode; without this, any real evolution
	// (which bypasses the canonical-equality fast path) silently returns the
	// raw map[string]any instead of the callback's converted value — a
	// direct-vs-resolved divergence. The resolved DecodeJSON funnels through
	// this same deser, so both resolved wire formats gain the custom together.
	ctx.applyCustomToNode(nd, r)
	return nd, nil
}

// readerFieldLookup answers "which reader field does this writer field name?"
// in constant time per question, for one reader record.
//
// The two maps are SEPARATE and consulted in that order, because the rule is
// that EVERY field name outranks EVERY field alias — not that a name outranks
// an alias on the same field. A single merged map cannot express that: a
// writer name that is one reader field's alias and a LATER reader field's name
// would resolve to whichever entry was written last, silently reversing the
// routing. That routing is the contract the parse-time rejection of field
// name/alias collisions is justified by. Within each map the FIRST field wins,
// and a parse rejects the collisions that would make it observable.
//
// Built once per reader record, asked once per writer field: building it per
// question is the scan-inside-a-loop it exists to avoid, over a field count the
// schema text picks.
type readerFieldLookup struct {
	byName  map[string]int
	byAlias map[string]int
}

func newReaderFieldLookup(r *schemaNode) readerFieldLookup {
	lk := readerFieldLookup{byName: make(map[string]int, len(r.fields))}
	for i := range r.fields {
		if _, taken := lk.byName[r.fields[i].name]; !taken {
			lk.byName[r.fields[i].name] = i
		}
	}
	for i := range r.fields {
		for _, alias := range r.fields[i].aliases {
			if lk.byAlias == nil {
				lk.byAlias = make(map[string]int, len(r.fields))
			}
			if _, taken := lk.byAlias[alias]; !taken {
				lk.byAlias[alias] = i
			}
		}
	}
	return lk
}

// index reports the reader-field index writerFieldName resolves to, or -1.
func (lk readerFieldLookup) index(writerFieldName string) int {
	if i, ok := lk.byName[writerFieldName]; ok {
		return i
	}
	if i, ok := lk.byAlias[writerFieldName]; ok {
		return i
	}
	return -1
}

func (rr *resolvedRecord) buildDeser() deserfn {
	return func(src []byte, v reflect.Value, sl *slab) ([]byte, error) {
		if sl.depth >= maxDepth {
			return nil, errTooDeep
		}
		sl.depth++
		defer func() { sl.depth-- }()
		v = indirectAlloc(v)
		k := v.Kind()

		if k == reflect.Interface {
			return rr.deserInterface(src, v, sl)
		}
		t := v.Type()
		if k == reflect.Map && t.Key().Kind() == reflect.String {
			return rr.deserMap(src, v, t, sl)
		}
		if k == reflect.Struct {
			return rr.deserStruct(src, v, t, sl)
		}
		return nil, &SemanticError{GoType: t, AvroType: "record"}
	}
}

func (rr *resolvedRecord) deserInterface(src []byte, v reflect.Value, sl *slab) ([]byte, error) {
	// Mirror the natural record decoder (deserRecord.deser): when the
	// interface target already wraps a map[string]any, decode into the
	// existing map so keys outside the schema are retained — the
	// documented streaming-decode reuse contract.
	m := reuseOrMakeStringAnyMap(v, len(rr.readerNames))
	var err error
	elem := reflect.New(anyType).Elem()

	for _, op := range rr.wireOps {
		if op.readerIdx < 0 {
			if src, err = op.skip(src, sl); err != nil {
				return nil, err
			}
			continue
		}
		if src, err = op.read(src, elem, sl); err != nil {
			return nil, recordFieldError(nil, rr.readerNames[op.readerIdx], err)
		}
		m[rr.readerNames[op.readerIdx]] = elem.Interface()
		elem.SetZero()
	}

	// Apply defaults. The deserfn does not write to its src, so pass the
	// encoded default bytes directly without copying.
	for _, d := range rr.defaults {
		if _, err = d.deser(d.encodedDefault, elem, sl); err != nil {
			return nil, recordFieldError(nil, rr.readerNames[d.readerIdx], err)
		}
		m[rr.readerNames[d.readerIdx]] = elem.Interface()
		elem.SetZero()
	}

	return src, setIface(v, reflect.ValueOf(m), "record")
}

func (rr *resolvedRecord) deserMap(src []byte, v reflect.Value, t reflect.Type, sl *slab) ([]byte, error) {
	if v.IsNil() {
		v.Set(reflect.MakeMapWithSize(t, len(rr.readerNames)))
	}
	var err error
	elem := reflect.New(t.Elem()).Elem()
	keyType := t.Key()

	for _, op := range rr.wireOps {
		if op.readerIdx < 0 {
			if src, err = op.skip(src, sl); err != nil {
				return nil, err
			}
			continue
		}
		name := rr.readerNames[op.readerIdx]
		if err := validateJSONNumberMapKey(name, keyType, "record"); err != nil {
			return nil, err
		}
		if src, err = op.read(src, elem, sl); err != nil {
			return nil, recordFieldError(nil, name, err)
		}
		v.SetMapIndex(mapKeyAs(t, rr.readerNameVals[op.readerIdx]), elem)
		elem.SetZero()
	}

	for _, d := range rr.defaults {
		name := rr.readerNames[d.readerIdx]
		if err := validateJSONNumberMapKey(name, keyType, "record"); err != nil {
			return nil, err
		}
		if _, err = d.deser(d.encodedDefault, elem, sl); err != nil {
			return nil, recordFieldError(nil, name, err)
		}
		v.SetMapIndex(mapKeyAs(t, rr.readerNameVals[d.readerIdx]), elem)
		elem.SetZero()
	}

	return src, nil
}

func (rr *resolvedRecord) deserStruct(src []byte, v reflect.Value, t reflect.Type, sl *slab) ([]byte, error) {
	mapping, err := typeFieldMapping(rr.readerNames, &rr.cache, t)
	if err != nil {
		return nil, err
	}

	for _, op := range rr.wireOps {
		if op.readerIdx < 0 {
			if src, err = op.skip(src, sl); err != nil {
				return nil, err
			}
			continue
		}
		fv, ferr := fieldByIndex(v, mapping.indices[op.readerIdx])
		if ferr != nil {
			return nil, recordFieldError(t, rr.readerNames[op.readerIdx], ferr)
		}
		if src, err = op.read(src, fv, sl); err != nil {
			return nil, recordFieldError(t, rr.readerNames[op.readerIdx], err)
		}
	}

	for _, d := range rr.defaults {
		fv, ferr := fieldByIndex(v, mapping.indices[d.readerIdx])
		if ferr != nil {
			return nil, recordFieldError(t, rr.readerNames[d.readerIdx], ferr)
		}
		if _, err = d.deser(append([]byte(nil), d.encodedDefault...), fv, sl); err != nil {
			return nil, recordFieldError(t, rr.readerNames[d.readerIdx], err)
		}
	}

	return src, nil
}

func resolveEnum(r, w *schemaNode, ctx *resolveCtx) (*schemaNode, error) {
	readerIdx := make(map[string]int, len(r.symbols))
	for i, s := range r.symbols {
		readerIdx[s] = i
	}

	identity := len(r.symbols) == len(w.symbols)
	mapping := make([]int, len(w.symbols))
	for i, ws := range w.symbols {
		if ri, ok := readerIdx[ws]; ok {
			mapping[i] = ri
			if ri != i {
				identity = false
			}
		} else {
			identity = false
			if !r.hasEnumDef {
				return nil, &CompatibilityError{
					Path:       r.name,
					ReaderType: r.name,
					WriterType: w.name,
					Detail:     fmt.Sprintf("writer symbol %q not in reader and no default", truncForError(ws)),
				}
			}
			defIdx, ok := readerIdx[r.enumDef]
			if !ok {
				return nil, fmt.Errorf("enum default %q not found in reader symbols", truncForError(r.enumDef))
			}
			mapping[i] = defIdx
		}
	}

	if identity {
		return maybeWrapResolvedNode(r, ctx), nil
	}

	readerSymbols := r.symbols
	deser := deserfn(func(src []byte, v reflect.Value, _ *slab) ([]byte, error) {
		idx, src, err := readVarint(src)
		if err != nil {
			return nil, err
		}
		if idx < 0 || int(idx) >= len(mapping) {
			return nil, fmt.Errorf("enum index %d out of range [0, %d)", idx, len(mapping))
		}
		ri := mapping[idx]
		return src, setEnumTarget(indirectAlloc(v), ri, readerSymbols[ri])
	})
	nd := &schemaNode{
		kind:        "enum",
		name:        r.name,
		aliases:     r.aliases,
		bareAliases: r.bareAliases,
		symbols:     r.symbols,
		// The symbol slice is the reader's, so its lookup is too. A
		// resolved node that carries the siblings without the table sends
		// every consumer back to scanning them.
		symbolIdx: r.symbolIdx,
		ser:       r.ser,
		deser:     deser,
	}
	ctx.applyCustomToNode(nd, r)
	return nd, nil
}

func resolveArray(r, w *schemaNode, path string, ctx *resolveCtx) (*schemaNode, error) {
	resolved, err := resolveNode(r.items, w.items, path+".items", ctx)
	if err != nil {
		return nil, err
	}
	if resolved == r.items {
		return maybeWrapResolvedNode(r, ctx), nil
	}
	nd := &schemaNode{
		kind:  "array",
		items: resolved,
		ser:   r.ser,
		// minItemBytes: bound against the WRITER's wire format, not the
		// reader's resolved schema. Per the spec, items are encoded
		// with the writer's type — an int-on-wire promoted to a double
		// reader is still 1 byte minimum on the wire. Mirrors the
		// resolveMap site below.
		deser: (&deserArray{deserItem: resolved.deser, minItemBytes: ctx.minBytes.minBytesOf(w.items)}).deser,
	}
	ctx.applyCustomToNode(nd, r)
	return nd, nil
}

func resolveMap(r, w *schemaNode, path string, ctx *resolveCtx) (*schemaNode, error) {
	resolved, err := resolveNode(r.values, w.values, path+".values", ctx)
	if err != nil {
		return nil, err
	}
	if resolved == r.values {
		return maybeWrapResolvedNode(r, ctx), nil
	}
	nd := &schemaNode{
		kind:   "map",
		values: resolved,
		ser:    r.ser,
		// minEntryBytes: bound against the WRITER's wire format, not the
		// reader's resolved schema (a long-on-wire promoted to a double
		// reader is still 1 byte minimum on the wire).
		deser: (&deserMap{deserItem: resolved.deser, minEntryBytes: mapEntryMinBytes(ctx.minBytes.minBytesOf(w.values))}).deser,
	}
	ctx.applyCustomToNode(nd, r)
	return nd, nil
}

// resolveWriterUnion: writer is a union, reader is not. Every writer
// branch must resolve against the reader; the first failure is eagerly
// returned.
//
// Spec ("Schema Resolution"): "if writer's is a union, but reader's is
// not: if the reader's schema matches the selected writer's schema, it
// is recursively resolved against it. If they do not match, an error is
// signalled." Java reads "selected" as license to defer per-branch failures to
// decode time via ErrorAction; this requires every branch compatible at resolve
// time, matching the package's fail-fast posture. A producer that narrowed
// during evolution but never emits the dropped branch must update its schema,
// and in exchange mismatches surface at config time rather than mid-stream.
func resolveWriterUnion(r, w *schemaNode, path string, ctx *resolveCtx) (*schemaNode, error) {
	branchDesers := make([]deserfn, len(w.branches))
	bnames := make([]string, len(w.branches))
	lnames := make([]string, len(w.branches))
	for i, wb := range w.branches {
		resolved, err := resolveNode(r, wb, path, ctx)
		if err != nil {
			return nil, err
		}
		branchDesers[i] = resolved.deser
		bnames[i], lnames[i] = unionBranchNames(wb)
	}
	// noWrap: reader is non-union, so the TaggedUnions wrap on
	// du.deser would leak the WRITER's branch name onto a target that
	// has no union to dispatch through. Sibling resolveReaderUnion
	// handles its own wrap (with the READER's branch name) when the
	// reader IS a union; resolveUnionUnion (both union) keeps wrap on.
	du := &deserUnion{fns: branchDesers, branchNames: bnames, logicalNames: lnames, noWrap: true}
	return &schemaNode{
		kind:  r.kind,
		name:  r.name,
		ser:   r.ser,
		deser: du.deser,
	}, nil
}

// resolveReaderUnion: reader is union, writer is not.
// Find first matching reader branch — two-pass to match Java's
// Resolver.firstMatchingBranch (exact match scanned first, numeric
// promotion as a fallback pass only if no exact match exists,
// Resolver.java:634/:666). Single-pass would silently produce float64
// for an int writer when the reader is ["double","int"].
func resolveReaderUnion(r, w *schemaNode, path string, ctx *resolveCtx) (*schemaNode, error) {
	rb := newReaderBranchLookup(r).match(w)
	if rb == nil {
		return nil, &CompatibilityError{
			Path:       pathOrRoot(path),
			ReaderType: "union",
			WriterType: w.kind,
			Detail:     detailWriterTypeNoReaderBranch,
		}
	}
	resolved, err := resolveNode(rb, w, path, ctx)
	if err != nil {
		return nil, err
	}
	// The wire format has no union index (writer wrote a non-union
	// value), so we can't use deserUnion.deser which reads a varint
	// index. Wrap the resolved deser with deserUnion.maybeWrap on a
	// single-branch name table — the same code the natural union path
	// runs — so the two paths share one TaggedUnions contract: targets
	// that map[string]any is not assignable to (concrete types,
	// non-empty interfaces) skip the wrap silently rather than erroring.
	// unionEmitTag, not the raw logical qualifier: the tag is resolved against
	// the READER union, whose other branches may own that spelling exactly.
	bn, _ := unionBranchNames(rb)
	wrap := &deserUnion{branchNames: []string{bn}, logicalNames: []string{unionEmitTag(r, rb, true)}}
	inner := resolved.deser
	deser := func(src []byte, v reflect.Value, sl *slab) ([]byte, error) {
		src, err := inner(src, v, sl)
		if err == nil {
			wrap.maybeWrap(v, sl, 0)
		}
		return src, err
	}
	return &schemaNode{
		kind:     "union",
		branches: r.branches,
		// Same slice, same table: the indexes in tags address r.branches,
		// which is exactly what this node carries.
		tags:  r.tags,
		ser:   r.ser,
		deser: deser,
	}, nil
}

// resolveUnionUnion: both reader and writer are unions.
// Map each writer branch to its best matching reader branch.
func resolveUnionUnion(r, w *schemaNode, path string, ctx *resolveCtx) (*schemaNode, error) {
	branchDesers := make([]deserfn, len(w.branches))
	bnames := make([]string, len(w.branches))
	lnames := make([]string, len(w.branches))
	// Built once for the whole loop: this asks per WRITER branch, and the
	// answer is a property of the READER union.
	readerBranches := newReaderBranchLookup(r)
	for i, wb := range w.branches {
		rb := readerBranches.match(wb)
		if rb == nil {
			return nil, &CompatibilityError{
				Path:       pathOrRoot(path),
				ReaderType: "union",
				WriterType: fmt.Sprintf("union[%d]:%s", i, wb.kind),
				Detail:     detailWriterBranchNoReaderBranch,
			}
		}
		resolved, err := resolveNode(rb, wb, path, ctx)
		if err != nil {
			return nil, err
		}
		branchDesers[i] = resolved.deser
		// Tag name comes from the READER branch — what the consumer's
		// schema declares — not the writer's. Otherwise a promoted
		// int→long branch decoded with TaggedUnions would emit
		// {"int": ...} against a reader that knows the field as "long".
		// The logical spelling goes through unionEmitTag so it degrades to
		// the unqualified name when another READER branch owns it exactly.
		bnames[i], _ = unionBranchNames(rb)
		lnames[i] = unionEmitTag(r, rb, true)
	}
	du := &deserUnion{fns: branchDesers, branchNames: bnames, logicalNames: lnames}
	deser := du.deser
	if len(w.branches) == 2 && w.branches[0].kind == "null" {
		deser = deserNullUnion(du)
	}
	return &schemaNode{
		kind:     "union",
		branches: r.branches,
		// Same slice, same table: the indexes in tags address r.branches,
		// which is exactly what this node carries.
		tags:  r.tags,
		ser:   r.ser,
		deser: deser,
	}, nil
}

// extractDefaultBytes converts the encodeDefault "bytes"/"fixed" arm's
// raw default value into the wire-form []byte. A literal []byte passes
// through; a string is codepoint-mapped via avroJSONBytesToBytes (the
// fwd-ref fixup path that bypasses convertDefaultBytes); other types
// yield a typed error. typeLabel is "bytes" or "fixed".
func extractDefaultBytes(val any, typeLabel string) ([]byte, error) {
	switch v := val.(type) {
	case []byte:
		return v, nil
	case string:
		return avroJSONBytesToBytes(v)
	}
	return nil, fmt.Errorf("expected []byte or string for %s default, got %T", typeLabel, val)
}

// defaultChargeSink collects producer-compliance verdicts raised while a field
// default is pre-encoded, WITHOUT failing the walk.
//
// The two must stay separate. A default that cannot be WRITTEN is still a
// schema that must PARSE — a reader dropping the field never writes it — and
// the same walk is used by the union try-each, where an error means "this
// branch does not accept the value" and a compliance verdict would silently
// select a different branch. So the verdict rides out here and is surfaced by
// the encode-side consumers of the pre-encoded bytes.
type defaultChargeSink struct{ err error }

func (s *defaultChargeSink) record(err error) {
	if s != nil && err != nil && s.err == nil {
		s.err = err
	}
}

func encodeDefault(dst []byte, val any, node *schemaNode) ([]byte, error) {
	return encodeDefaultDepth(dst, val, node, 0, nil)
}

// encodeDefaultCharged is encodeDefault for the one caller that installs the
// result as a field's pre-encoded default: it additionally reports the
// producer-compliance verdict for the payload it just built.
func encodeDefaultCharged(val any, node *schemaNode) ([]byte, error, error) {
	var sink defaultChargeSink
	b, err := encodeDefaultDepth(nil, val, node, 0, &sink)
	return b, sink.err, err
}

// encodeDefaultDepth bounds the recursion encodeDefault performs while filling
// absent nested record fields from their own defaults. Unlike validateDefault
// (which skips absent fields and so terminates vacuously), encodeDefault fills
// them — so a default that has no finite encoding because a required field
// recurses into its own type (e.g. record R{ R self = {} }, or
// R{ array<R> kids = [{}] }) would recurse forever and overflow the stack.
// The same maxDepth ceiling the wire codec enforces turns that into an
// errTooDeep parse error. A legitimately finite default nests far below the
// bound (each level resolves a concrete value), so this never false-rejects a
// real default.
func encodeDefaultDepth(dst []byte, val any, node *schemaNode, depth int, sink *defaultChargeSink) ([]byte, error) {
	if depth >= maxDepth {
		return nil, errTooDeep
	}
	switch node.kind {
	case "null":
		if val != nil {
			return nil, fmt.Errorf("expected nil for null default, got %T", val)
		}
		return dst, nil
	case "boolean":
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool for boolean default, got %T", val)
		}
		if b {
			return append(dst, 1), nil
		}
		return append(dst, 0), nil
	case "int":
		n, err := defaultAsInt32(val)
		if err != nil {
			return nil, fmt.Errorf("int default: %w", err)
		}
		return appendVarint(dst, n), nil
	case "long":
		n, err := defaultAsInt64(val)
		if err != nil {
			return nil, fmt.Errorf("long default: %w", err)
		}
		return appendVarlong(dst, n), nil
	case "float":
		f, err := defaultAsFloat(val)
		if err != nil {
			return nil, fmt.Errorf("float default: %w", err)
		}
		// Lossy-destination policy: float64 → float32 narrowing to ±Inf
		// is accepted at encode (matches appendAvroFloat32 / Java).
		return appendUint32(dst, math.Float32bits(float32(f))), nil
	case "double":
		f, err := defaultAsFloat(val)
		if err != nil {
			return nil, fmt.Errorf("double default: %w", err)
		}
		return appendUint64(dst, math.Float64bits(f)), nil
	case "string":
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for string default, got %T", val)
		}
		dst = appendVarlong(dst, int64(len(s)))
		return append(dst, s...), nil
	case "bytes":
		b, err := extractDefaultBytes(val, "bytes")
		if err != nil {
			return nil, err
		}
		sink.record(chargeDecimalLeaf(b, node.logical))
		dst = appendVarlong(dst, int64(len(b)))
		return append(dst, b...), nil
	case "enum":
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for enum default, got %T", val)
		}
		for i, sym := range node.symbols {
			if sym == s {
				return appendVarint(dst, int32(i)), nil
			}
		}
		return nil, fmt.Errorf("unknown enum symbol %q in default", truncForError(s))
	case "fixed":
		b, err := extractDefaultBytes(val, "fixed")
		if err != nil {
			return nil, err
		}
		if len(b) != node.size {
			return nil, fmt.Errorf("fixed default length %d != size %d", len(b), node.size)
		}
		sink.record(chargeDecimalLeaf(b, node.logical))
		return append(dst, b...), nil
	case "array":
		// null is not an array. Rejecting it here keeps the union try-
		// each loop honest: a [Array,null] union with default null would
		// otherwise match the Array branch (producing an empty-array
		// wire form) instead of the null branch. Mirrors
		// validateDefault's nil-reject for parse-time symmetry.
		arr, err := defaultArrayShape(val)
		if err != nil {
			return nil, err
		}
		if len(arr) == 0 {
			return appendVarlong(dst, 0), nil
		}
		dst = appendVarlong(dst, int64(len(arr)))
		bodyStart := len(dst)
		for _, item := range arr {
			dst, err = encodeDefaultDepth(dst, item, node.items, depth+1, sink)
			if err != nil {
				return nil, err
			}
		}
		// Ask the array encoders' own shared compliance helper, whose doc
		// requires exactly this: every array encoder routes through it, or the
		// paths drift. This default walk is one of them.
		sink.record(arrayZeroByteEncodeCompliance(len(dst) == bodyStart, len(arr)))
		return append(dst, 0), nil
	case "map":
		m, err := defaultObjectShape(val, "map")
		if err != nil {
			return nil, err
		}
		if len(m) == 0 {
			return appendVarlong(dst, 0), nil
		}
		dst = appendVarlong(dst, int64(len(m)))
		for k, v := range m {
			dst = appendVarlong(dst, int64(len(k)))
			dst = append(dst, k...)
			dst, err = encodeDefaultDepth(dst, v, node.values, depth+1, sink)
			if err != nil {
				return nil, err
			}
		}
		return append(dst, 0), nil
	case "record":
		m, err := defaultObjectShape(val, "record")
		if err != nil {
			return nil, err
		}
		for _, f := range node.fields {
			fval, exists := m[f.name]
			if !exists {
				if !f.hasDefault {
					return nil, fmt.Errorf("record default missing field %q with no default", truncForError(f.name))
				}
				fval = f.defaultVal
			}
			dst, err = encodeDefaultDepth(dst, fval, f.node, depth+1, sink)
			if err != nil {
				return nil, err
			}
		}
		return dst, nil
	case "union":
		// Avro 1.12+ relaxed the union-default rule (AVRO-3649): the default
		// may match any branch, not just the first. Walk in declaration
		// order and pick the first that accepts; encode its index as the
		// wire prefix.
		//
		// No type-name fast path. The runtime dispatchers use
		// unionTypeNameForValue, correct for user values because the Go type
		// names the intended branch, but a stored default's branch was already
		// chosen at parse time by firstUnionBranchAcceptingDefault, which
		// iterates declaration order with no kind filter. The wire index MUST
		// agree with that picker: an [enum, string] default "A" picks enum at
		// validate time, while the type-name shortcut picks the later string
		// branch, emitting wire bytes that name a different branch than the
		// metadata API reports.
		if len(node.branches) == 0 {
			return nil, fmt.Errorf("empty union")
		}
		base := len(dst)
		for i, branch := range node.branches {
			attempt := appendVarlong(dst[:base], int64(i))
			// Each attempt charges into its OWN sink, and only the WINNER's
			// verdict is merged. Selection must stay byte-identical: it is
			// decided by err alone, so a compliance verdict — which says the
			// payload is too large to read back, not that the branch rejects
			// the value — can never move the branch index. Handing the verdict
			// back as the attempt's err would look like a fix (no unreadable
			// wire is emitted) while silently selecting a LATER branch, and the
			// metadata API would then report a different branch than the wire
			// names. Passing nil instead keeps selection right but charges
			// nothing at all.
			var attemptSink defaultChargeSink
			if encoded, err := encodeDefaultDepth(attempt, val, branch, depth+1, &attemptSink); err == nil {
				sink.record(attemptSink.err)
				return encoded, nil
			}
		}
		return nil, fmt.Errorf("union default does not match any branch: %T(%s)", val, truncValueForError(val))
	default:
		return nil, fmt.Errorf("unsupported default encoding for type %q", node.kind)
	}
}
