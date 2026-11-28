package parser

import "strings"

// typeName prints t in a canonical spelling: keywords uppercased, single
// spaces, no space before a paren or bracket, `ARRAY` spelled `[]`, and an
// empty modifier list dropped. DuckDB parses the result to the same type
// as the original (TestWriteTypeNameMatchesDuckDB).
func (sw *sqlWriter) typeName(t *TypeName) {
	if t.Setof {
		sw.str("SETOF ")
	}
	switch {
	case t.Keyword:
		sw.str(t.Name[0])
	case len(t.Name) == 1:
		writeTypeNamePart(sw, t.Name[0])
	default:
		writeDottedName(sw, t.Name)
	}
	switch {
	case len(t.Fields) > 0:
		sw.str("(")
		for i, f := range t.Fields {
			sw.sep(i, ", ")
			sw.typeField(f)
		}
		sw.str(")")
	case len(t.Params) > 0:
		sw.str("(")
		for i, p := range t.Params {
			sw.sep(i, ", ")
			sw.typeName(p)
		}
		sw.str(")")
	case len(t.Modifiers) > 0:
		sw.str("(")
		sw.exprListTop(t.Modifiers)
		sw.str(")")
	}
	if t.TimeZone != "" {
		sw.str(" ")
		sw.str(t.TimeZone)
	}
	if t.Interval != "" {
		sw.str(" ")
		sw.str(t.Interval)
	}
	for _, b := range t.Arrays {
		sw.str("[")
		if b.Size != nil {
			sw.exprTop(b.Size)
		}
		sw.str("]")
	}
}

func (sw *sqlWriter) typeField(f *TypeField) {
	writeIdentPart(sw, f.Name)
	sw.str(" ")
	sw.typeName(f.Type)
}

// writeTypeNamePart prints an unqualified identifier type name, bare only
// if it lexes back as the same identifier: a keyword the TypeName
// primitive rejects needs quotes, and so does a word that starts one of
// the grammar's keyword-spelled types (`"struct"` bare would reparse as
// the STRUCT keyword form, a different TypeName for the same DuckDB type).
func writeTypeNamePart(sw *sqlWriter, s string) {
	if isSimpleIdent(s) && isTypeNameIdent(s) {
		sw.str(s)
		return
	}
	sw.str(`"`)
	sw.str(strings.ReplaceAll(s, `"`, `""`))
	sw.str(`"`)
}

func isTypeNameIdent(s string) bool {
	if _, err := loadPegParser(); err != nil {
		return false
	}
	up := strings.ToUpper(s)
	if _, ok := typeLeadWords[up]; ok {
		return false
	}
	return keywordSets.allowedAsIdentifier([]byte(up), CategoryTypeName)
}

// typeLeadWords are the first keywords of every TypeVariations alternative
// other than QualifiedSimpleType (common.gram). DOUBLE isn't one: DoubleType
// needs a following PRECISION, which Write never prints after a type.
var typeLeadWords = map[string]struct{}{
	"TIME": {}, "TIMESTAMP": {}, "INTERVAL": {}, "BIT": {}, "ROW": {}, "STRUCT": {},
	"VARIANT": {}, "MAP": {}, "TUPLE": {}, "GEOMETRY": {}, "UNION": {}, "SETOF": {},
	"INT": {}, "INTEGER": {}, "SMALLINT": {}, "BIGINT": {}, "REAL": {}, "BOOLEAN": {},
	"FLOAT": {}, "DECIMAL": {}, "DEC": {}, "NUMERIC": {},
	"CHARACTER": {}, "CHAR": {}, "NATIONAL": {}, "NCHAR": {}, "VARCHAR": {},
}
