package grammar

import (
	"embed"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser/peg"
)

//go:embed vendor
var vendorFS embed.FS

type KeywordLists struct {
	Reserved, Unreserved, ColumnName, FuncName, TypeName []string
}

func readLines(path string) ([]string, error) {
	data, err := vendorFS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// Load parses the vendored .gram files into a single peg.Grammar and reads
// the keyword lists as plain string slices — kept free of any dependency on
// the root package's KeywordSets type (built from these by the caller) so
// this package and the root package never import each other.
func Load() (*peg.Grammar, *KeywordLists, error) {
	var sources []string
	for _, name := range []string{"select", "expression", "common", "pivot", "describe", "base"} {
		data, err := vendorFS.ReadFile("vendor/statements/" + name + ".gram")
		if err != nil {
			return nil, nil, err
		}
		sources = append(sources, string(data))
	}
	g, err := peg.ParseGrammar(sources...)
	if err != nil {
		return nil, nil, err
	}

	kl := &KeywordLists{}
	for _, kw := range []struct {
		file string
		dst  *[]string
	}{
		{"reserved_keyword", &kl.Reserved},
		{"unreserved_keyword", &kl.Unreserved},
		{"column_name_keyword", &kl.ColumnName},
		{"func_name_keyword", &kl.FuncName},
		{"type_name_keyword", &kl.TypeName},
	} {
		lines, err := readLines("vendor/keywords/" + kw.file + ".list")
		if err != nil {
			return nil, nil, err
		}
		*kw.dst = lines
	}
	return g, kl, nil
}
