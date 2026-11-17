package avro

import (
	"fmt"
	"slices"
	"strings"
)

// CheckCompatibility reports whether data written with the writer schema can
// be read by the reader schema. It returns nil on success or a
// [*CompatibilityError] describing the first incompatibility.
//
// See [Resolve] for a note on argument order.
func CheckCompatibility(writer, reader *Schema) error {
	return checkCompat(reader.node, writer.node, "", make(map[nodePair]bool))
}

type nodePair struct {
	r, w *schemaNode
}

// compatErr builds a *CompatibilityError, factoring the construction
// shape repeated at every site in this file.
func compatErr(path, readerType, writerType, detail string) error {
	return &CompatibilityError{
		Path:       pathOrRoot(path),
		ReaderType: readerType,
		WriterType: writerType,
		Detail:     detail,
	}
}

// CompatibilityError.Detail strings for the writer-union incompatibility rule.
// The rule is enforced from two entry points — the pre-check here
// (CheckCompatibility) and the resolver (resolve.go, Resolve) — so the
// user-visible detail must stay identical between them; sharing the literal
// makes that structural rather than a comment-enforced lockstep.
const (
	detailWriterTypeNoReaderBranch   = "writer type matches no reader union branch"
	detailWriterBranchNoReaderBranch = "writer union branch has no matching reader branch"
)

func checkCompat(r, w *schemaNode, path string, seen map[nodePair]bool) error {
	if r == nil || w == nil {
		return compatErr(path, nodeKind(r), nodeKind(w), "nil schema")
	}

	pair := nodePair{r, w}
	if seen[pair] {
		return nil
	}
	seen[pair] = true

	if w.kind == "union" {
		return checkWriterUnion(r, w, path, seen)
	}
	if r.kind == "union" {
		return checkReaderUnion(r, w, path, seen)
	}

	if r.kind == w.kind {
		return checkSameKind(r, w, path, seen)
	}

	if promotionDeser(w.kind, r.kind) != nil {
		return nil
	}

	return compatErr(path, r.kind, w.kind, "incompatible types")
}

// checkWriterUnion requires every writer branch to be compatible with the
// reader, union or not; the first failure is an eager CompatibilityError.
//
// Java and fastavro instead defer per-branch failures to decode time via
// ErrorAction sentinels, so a writer that narrowed during evolution but never
// emits the dropped branch stays readable there. This package fails fast, like
// the rest of its resolution. The cost is that such a producer must update its
// schema; the benefit is that schema problems surface before any data flows.
func checkWriterUnion(r, w *schemaNode, path string, seen map[nodePair]bool) error {
	// Built once for the whole loop: this asks per WRITER branch, and the
	// answer is a property of the READER union.
	var readerBranches readerBranchLookup
	if r.kind == "union" {
		readerBranches = newReaderBranchLookup(r)
	}
	for i, wb := range w.branches {
		if r.kind == "union" {
			rb := readerBranches.match(wb)
			if rb == nil {
				return compatErr(path, r.kind, fmt.Sprintf("union[%d]:%s", i, wb.kind),
					detailWriterBranchNoReaderBranch)
			}
			if err := checkCompat(rb, wb, path, seen); err != nil {
				return err
			}
			continue
		}
		if err := checkCompat(r, wb, path, seen); err != nil {
			return err
		}
	}
	return nil
}

func checkReaderUnion(r, w *schemaNode, path string, seen map[nodePair]bool) error {
	rb := newReaderBranchLookup(r).match(w)
	if rb == nil {
		return compatErr(path, "union", w.kind, detailWriterTypeNoReaderBranch)
	}
	return checkCompat(rb, w, path, seen)
}

func checkSameKind(r, w *schemaNode, path string, seen map[nodePair]bool) error {
	switch r.kind {
	case "record":
		if !namesMatch(r, w) {
			return compatErr(path, r.name, w.name, "record names do not match")
		}
		return checkRecordCompat(r, w, path, seen)
	case "enum":
		if !namesMatch(r, w) {
			return compatErr(path, r.name, w.name, "enum names do not match")
		}
		return checkEnumCompat(r, w, path)
	case "array":
		return checkCompat(r.items, w.items, path+".items", seen)
	case "map":
		return checkCompat(r.values, w.values, path+".values", seen)
	case "fixed":
		if !namesMatch(r, w) {
			return compatErr(path, r.name, w.name, "fixed names do not match")
		}
		if r.size != w.size {
			return compatErr(path,
				fmt.Sprintf("fixed(%d)", r.size),
				fmt.Sprintf("fixed(%d)", w.size),
				"fixed sizes differ")
		}
	}
	if r.logical == "decimal" && w.logical == "decimal" {
		if r.precision != w.precision || r.scale != w.scale {
			return compatErr(path,
				fmt.Sprintf("decimal(%d,%d)", r.precision, r.scale),
				fmt.Sprintf("decimal(%d,%d)", w.precision, w.scale),
				"decimal precision or scale differ")
		}
	}
	return nil
}

func checkRecordCompat(r, w *schemaNode, path string, seen map[nodePair]bool) error {
	writerFields := make(map[string]*fieldNode, len(w.fields))
	for i := range w.fields {
		writerFields[w.fields[i].name] = &w.fields[i]
	}

	for _, rf := range r.fields {
		wf := findWriterField(rf, writerFields)
		if wf == nil {
			if !rf.hasDefault {
				return &CompatibilityError{
					Path:       fieldPath(path, rf.name),
					ReaderType: rf.node.kind,
					Detail:     "reader field has no default and is missing from writer",
				}
			}
			continue
		}
		if err := checkCompat(rf.node, wf.node, fieldPath(path, rf.name), seen); err != nil {
			return err
		}
	}

	// Mirrors resolveRecord's duplicate-claim guard, or CheckCompatibility
	// returns nil on a pair Resolve then rejects.
	if err := checkRecordFieldClaimsUnique(r, w, path); err != nil {
		return err
	}
	return nil
}

// checkRecordFieldClaimsUnique reports two writer fields resolving to the same
// reader-field index — one by canonical name and one by alias, or both by
// aliases. The twin of resolveRecord's guard.
func checkRecordFieldClaimsUnique(r, w *schemaNode, path string) error {
	if len(r.fields) == 0 {
		return nil
	}
	// Presence and identity are SEPARATE variables. A field name is not a
	// usable presence sentinel: [WithLaxNames] admits the empty string, so a
	// writer field named "" would claim a slot while leaving it
	// indistinguishable from unclaimed, and a second field reaching it through
	// an alias would go undetected. claimedBy exists only to name the collision
	// in the error. resolveRecord splits them the same way.
	claimed := make([]bool, len(r.fields))
	claimedBy := make([]string, len(r.fields))
	readerByName := newReaderFieldLookup(r)
	for _, wf := range w.fields {
		ri := readerByName.index(wf.name)
		if ri < 0 {
			continue
		}
		if claimed[ri] {
			return &CompatibilityError{
				Path:       pathOrRoot(path),
				ReaderType: "record",
				WriterType: "record",
				Detail: fmt.Sprintf("writer fields %q and %q both resolve to reader field %q (via name + alias collision); rename the writer or drop the alias to disambiguate",
					truncForError(claimedBy[ri]), truncForError(wf.name), truncForError(r.fields[ri].name)),
			}
		}
		claimed[ri] = true
		claimedBy[ri] = wf.name
	}
	return nil
}

func checkEnumCompat(r, w *schemaNode, path string) error {
	readerSymbols := make(map[string]bool, len(r.symbols))
	for _, s := range r.symbols {
		readerSymbols[s] = true
	}
	for _, ws := range w.symbols {
		if !readerSymbols[ws] && !r.hasEnumDef {
			return compatErr(path, r.name, w.name,
				fmt.Sprintf("writer enum symbol %q not in reader and reader has no default", truncForError(ws)))
		}
	}
	return nil
}

// namesMatch checks if two named types match by fully-qualified name,
// unqualified name, or aliases. Per the Avro spec, named types in
// different namespaces match if their unqualified names are the same
// ("both schemas are records with the same (unqualified) name" — the same
// wording for enum and fixed).
//
// Aliases carry their qualification. A reader alias matches the writer's exact
// fullname, and an alias DECLARED without a dot also short-name-matches the
// writer in any namespace — fastavro's raw-string tier (executed), the
// permissive side of the two references. An explicitly-qualified alias never
// short-matches: the spec ("Aliases") makes "x.y" a fully qualified name
// denoting exactly x.y, and both references reject the cross-namespace match.
func namesMatch(r, w *schemaNode) bool {
	if r.name == w.name {
		return true
	}
	if unqualified(r.name) == unqualified(w.name) {
		return true
	}
	if slices.Contains(r.aliases, w.name) {
		return true
	}
	return slices.Contains(r.bareAliases, unqualified(w.name))
}

// unqualified returns the unqualified portion of a possibly dot-separated name.
func unqualified(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return name
}

// namespaceOf returns the namespace portion of a fullname (everything before
// the final dot), or "" when the name has no namespace.
func namespaceOf(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[:i]
	}
	return ""
}

// findWriterField finds the writer field matching a reader field by name or
// reader field aliases.
func findWriterField(rf fieldNode, writerFields map[string]*fieldNode) *fieldNode {
	if wf, ok := writerFields[rf.name]; ok {
		return wf
	}
	for _, alias := range rf.aliases {
		if wf, ok := writerFields[alias]; ok {
			return wf
		}
	}
	return nil
}

// readerBranchLookup answers "which reader branch does this writer node
// select?" in constant time, for one reader union. The rule is
// branchMatchTiers below, applied once ahead of the questions: both callers ask
// once per WRITER branch, and scanning the reader's branches inside that loop
// is quadratic in two author-chosen counts. A cost bound only — the verdict
// must be what the scan gave.
type readerBranchLookup struct {
	branches []*schemaNode
	// byTier[i] holds branchMatchTiers[i]'s keys; first branch wins, which is
	// the declaration-order tie-break.
	byTier []map[branchMatchKey]int
	// firstByKind is every reader branch kind's first index. The promotion
	// tier is keyed by KIND alone, so it reads this instead of a tier map.
	firstByKind map[string]int
}

// branchMatchKey identifies what a reader branch answers to under one tier.
// Kind, because every tier matches within a kind. SIZE, because the spec folds
// a fixed's size into the MATCH predicate rather than checking it after
// selection: a wrong-size same-name fixed must not match, and selection
// continues to a later branch (NOT_BUGS #44).
type branchMatchKey struct {
	kind string
	name string
	size int
}

// branchMatchTier is one rank of the match rule: the names a reader branch
// ANSWERS TO, and the name a writer node ASKS WITH. The builder registers
// readerNames and the query asks writerName from this one table, so the index
// and the verdict cannot describe different rules.
type branchMatchTier struct {
	name        string
	readerNames func(r *schemaNode) []string
	writerName  func(w *schemaNode) (string, bool)
}

// branchIsNamedKind reports whether a branch carries a name to match on. The
// builder normalizes "error" into "record" before a node exists, so the three
// spellings here are the whole set.
func branchIsNamedKind(n *schemaNode) bool {
	switch n.kind {
	case "record", "enum", "fixed":
		return true
	}
	return false
}

// branchSizeKey is the size a fixed matches on, and zero for every other kind,
// so the key carries the constraint exactly where the rule puts it.
func branchSizeKey(n *schemaNode) int {
	if n.kind == "fixed" {
		return n.size
	}
	return 0
}

// branchMatchTiers ranks union-branch selection: full name or alias, then
// unqualified short name, then promotion.
//
// The unqualified tier applies to record, enum AND fixed, matching fastavro's
// match_types. Java does the short-name match for records only; this package
// follows fastavro's more uniform rule deliberately (NOT_BUGS #44). Exact match
// must outrank it — the spec permits a union to hold several named types
// sharing an unqualified name across namespaces.
//
// An UNNAMED kind answers to the empty name at the exact tier and to nothing at
// the unqualified tier: same kind IS an exact match for it, and it has no name
// to shorten.
var branchMatchTiers = []branchMatchTier{
	{
		name: "full name or alias",
		readerNames: func(r *schemaNode) []string {
			if !branchIsNamedKind(r) {
				return []string{""}
			}
			return append([]string{r.name}, r.aliases...)
		},
		writerName: func(w *schemaNode) (string, bool) {
			if !branchIsNamedKind(w) {
				return "", true
			}
			return w.name, true
		},
	},
	{
		name: "unqualified short name",
		readerNames: func(r *schemaNode) []string {
			if !branchIsNamedKind(r) {
				return nil
			}
			// Only an alias DECLARED without a dot short-matches across
			// namespaces; an explicitly-qualified alias denotes exactly its
			// fullname and is already handled by the exact tier. Same rule as
			// namesMatch — see its alias-qualification comment.
			return append([]string{unqualified(r.name)}, r.bareAliases...)
		},
		writerName: func(w *schemaNode) (string, bool) {
			if !branchIsNamedKind(w) {
				return "", false
			}
			return unqualified(w.name), true
		},
	},
}

// promotionTargetKinds maps a writer kind to every reader kind it promotes to.
// DERIVED from the promotions table itself (promote.go) rather than listed, so
// a promotion added there is honored here without an edit — the promotion tier
// is keyed by kind alone, and this is that tier's vocabulary.
var promotionTargetKinds = func() map[string][]string {
	m := make(map[string][]string, len(promotions))
	for key := range promotions {
		writerKind, readerKind, ok := strings.Cut(key, ">")
		if !ok {
			continue
		}
		m[writerKind] = append(m[writerKind], readerKind)
	}
	return m
}()

func newReaderBranchLookup(r *schemaNode) readerBranchLookup {
	lk := readerBranchLookup{
		branches:    r.branches,
		byTier:      make([]map[branchMatchKey]int, len(branchMatchTiers)),
		firstByKind: make(map[string]int, len(r.branches)),
	}
	for ti := range branchMatchTiers {
		lk.byTier[ti] = make(map[branchMatchKey]int, len(r.branches))
	}
	for i, rb := range r.branches {
		if rb == nil {
			continue
		}
		if _, taken := lk.firstByKind[rb.kind]; !taken {
			lk.firstByKind[rb.kind] = i
		}
		for ti, t := range branchMatchTiers {
			for _, name := range t.readerNames(rb) {
				key := branchMatchKey{kind: rb.kind, name: name, size: branchSizeKey(rb)}
				if _, taken := lk.byTier[ti][key]; !taken {
					lk.byTier[ti][key] = i
				}
			}
		}
	}
	return lk
}

// match returns the best reader branch for w, or nil. Tiers are consulted in
// rank order and the first that answers wins, which is what the old best-tier
// scan computed by ranking every branch.
func (lk readerBranchLookup) match(w *schemaNode) *schemaNode {
	for ti, t := range branchMatchTiers {
		name, ok := t.writerName(w)
		if !ok {
			continue
		}
		if i, ok := lk.byTier[ti][branchMatchKey{kind: w.kind, name: name, size: branchSizeKey(w)}]; ok {
			return lk.branches[i]
		}
	}
	// Promotion is the last tier and the only one that crosses kinds. A
	// same-kind named branch that failed both tiers above does NOT reach it:
	// nothing promotes into record, enum or fixed, so promotionTargetKinds has
	// no entry that could match one.
	best := -1
	for _, readerKind := range promotionTargetKinds[w.kind] {
		if readerKind == w.kind {
			continue
		}
		if i, ok := lk.firstByKind[readerKind]; ok && (best < 0 || i < best) {
			best = i
		}
	}
	if best < 0 {
		return nil
	}
	return lk.branches[best]
}

func pathOrRoot(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}

func fieldPath(parent, field string) string {
	if parent == "" {
		return field
	}
	return parent + "." + field
}

func nodeKind(n *schemaNode) string {
	if n == nil {
		return "<nil>"
	}
	return n.kind
}
