package parser

import "strings"

// SplitExcludeName splits one StarExpr.Exclude entry -- kept as its raw
// source text, quotes included (see adaptExcludeNames) -- into its
// unquoted dot-separated parts: `a` -> [a], `t.a` -> [t a],
// `"T"."a.b"` -> [T a.b].
func SplitExcludeName(raw string) []string {
	var parts []string
	var cur strings.Builder
	quoted := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case c == '"' && quoted && i+1 < len(raw) && raw[i+1] == '"':
			cur.WriteByte('"')
			i++
		case c == '"':
			quoted = !quoted
		case c == '.' && !quoted:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	return append(parts, cur.String())
}
