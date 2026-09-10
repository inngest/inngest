package insights

import "strings"

// PathSegment is one step of a path into a JSON value: either a literal
// object key (Wildcard false, Key the key's own exact text — which may
// itself legitimately contain a "." if that's the object's real key, as
// with an OTel span attribute name) or "every element of the array here"
// (Wildcard true, Key ignored).
//
// A path is authored/matched as a []PathSegment, root-to-leaf, rather
// than a single dotted string, specifically so a literal key containing
// its own "." can never be confused with multiple levels of real JSON
// nesting. DuckDB itself draws that exact line: a bare string operand to
// ->>'/json_extract_string is one literal top-level key, dots and all,
// while a "$"-prefixed one is a JSONPath expression whose "."s are
// structural (confirmed empirically — see queryPath's own doc comment).
// Splitting a path into explicit segments reflects that same rule
// instead of letting both cases serialize to indistinguishable dotted
// text, which is exactly the ambiguity a single map[string]ColumnHint
// used to have.
type PathSegment struct {
	Key      string
	Wildcard bool
}

// seg builds a literal-key PathSegment. Each call is exactly one JSON
// object field access — a compound-looking key like "_inngest.app.name"
// passed to a single seg() call is one flat key (matching how it's
// actually stored, e.g. an OTel attribute name), never split into
// several; genuine nested access is instead written as multiple seg()
// calls (or wc, for an array step) in sequence.
func seg(key string) PathSegment { return PathSegment{Key: key} }

// wc is the array-wildcard PathSegment: "every element of the array at
// this point in the path."
var wc = PathSegment{Wildcard: true}

// PathHint pairs one path (root-to-leaf; nil/empty for the column's own
// whole value) with the hint the value found there carries.
type PathHint struct {
	Path []PathSegment
	Hint ColumnHint
}

// hint builds one PathHint from a hint and its path, written as an
// explicit segment sequence (see PathSegment/seg's own doc comments for
// why that's the point).
func hint(h ColumnHint, path ...PathSegment) PathHint {
	return PathHint{Path: path, Hint: h}
}

// rootHint builds a single-entry []PathHint for a column whose only hint
// is its own whole value (an empty Path), or nil if h is HintNone — for
// a column with no other JSON sub-path entries. A column that also needs
// sub-path entries (event_ids' wc, say) instead writes out its pathHints
// slice literal directly.
func rootHint(h ColumnHint) []PathHint {
	if h == HintNone {
		return nil
	}
	return []PathHint{{Hint: h}}
}

// withRootHint returns pathHints with an empty-Path (whole-value) entry
// present for hint -- unchanged if hint is HintNone or pathHints already
// has one. buildColumnPathHints uses this to fold resolveItemHint's (or,
// for a UNION operand, unionColumnHints') whole-value resolution directly
// into a column's own PathHints, so a computed expression or JSON
// sub-path access that resolveColumnPathHints itself can't trace back to
// a known column's full pathHints (an "arr[3]" or "attributes ->>
// 'run.id'", say) still gets its one resolvable hint represented, without
// a separate ColumnHints-shaped result living alongside PathHints.
func withRootHint(pathHints []PathHint, hint ColumnHint) []PathHint {
	if hint == HintNone {
		return pathHints
	}
	for _, ph := range pathHints {
		if len(ph.Path) == 0 {
			return pathHints
		}
	}
	return append([]PathHint{{Hint: hint}}, pathHints...)
}

// lookupPath returns the hint declared for exactly this path among c's
// pathHints, HintNone if none matches. A linear scan is fine — no
// knownColumn declares more than a handful of entries.
func (c knownColumn) lookupPath(path []PathSegment) ColumnHint {
	for _, ph := range c.pathHints {
		if pathsEqual(ph.Path, path) {
			return ph.Hint
		}
	}
	return HintNone
}

// hint returns c's whole-column hint (its empty-Path pathHints entry),
// HintNone if it has none.
func (c knownColumn) hint() ColumnHint {
	return c.lookupPath(nil)
}

func pathsEqual(a, b []PathSegment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// queryPath converts one jsonPathAccess literal (the raw text between the
// quotes of a ->>'/->'/json_extract(_string) call) into its canonical
// []PathSegment, mirroring DuckDB's own rule for what a JSON path operand
// means: a literal starting with "$" is a JSONPath expression, whose "."
// and "[*]" are structural; anything else is one literal, atomic
// top-level key — even one containing a literal "." — confirmed against
// DuckDB directly:
//
//	'{"a.b":1,"a":{"b":2}}'::JSON ->> 'a.b'   -- 1 (literal key "a.b")
//	'{"a.b":1,"a":{"b":2}}'::JSON ->> '$.a.b' -- 2 (nested a -> b)
//
// This is the load-bearing rule that resolves the "does this dot mean
// nesting or a literal key" ambiguity a flat dotted string can't
// otherwise answer: DuckDB itself already draws this line based on the
// "$" prefix, so this function just mirrors that rule instead of
// guessing from the string's own shape.
func queryPath(path string) []PathSegment {
	if !strings.HasPrefix(path, "$") {
		return []PathSegment{seg(path)}
	}
	return parseJSONPath(strings.TrimPrefix(path, "$"))
}

// parseJSONPath tokenizes p (a JSONPath expression with its leading "$"
// already stripped) into segments. Only the two forms this package's own
// pathHints/rewrite.go ever produce or consult are recognized: ".key"
// (a named field, possibly the first token with no leading ".") and
// "[*]" (a wildcard, which DuckDB allows glued directly onto the
// preceding token with no "." — e.g. "$.x[*]" — or right after "$" — e.g.
// "$[*].a.b", both confirmed empirically). A general numeric index
// ("[3]") never appears in a pathHints key (single-index access is
// resolved via resolveArrayElementHint's own "[*]" convention instead,
// regardless of the actual index), so it isn't handled here.
func parseJSONPath(p string) []PathSegment {
	var segs []PathSegment
	for len(p) > 0 {
		switch {
		case strings.HasPrefix(p, "[*]"):
			segs = append(segs, wc)
			p = p[len("[*]"):]
		case strings.HasPrefix(p, "."):
			p = p[1:]
		default:
			end := strings.IndexAny(p, ".[")
			if end < 0 {
				end = len(p)
			}
			segs = append(segs, seg(p[:end]))
			p = p[end:]
		}
	}
	return segs
}
