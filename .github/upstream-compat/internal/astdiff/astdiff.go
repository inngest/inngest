// Package astdiff extracts and compares exported Go symbols between two
// versions of a directory tree (typically an inngest/ checkout at different
// commits). It operates at the AST level — fast but no type resolution.
package astdiff

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// SymbolKind classifies an exported symbol.
type SymbolKind string

const (
	KindFunc      SymbolKind = "func"
	KindType      SymbolKind = "type"
	KindInterface SymbolKind = "interface"
	KindStruct    SymbolKind = "struct"
	KindConst     SymbolKind = "const"
	KindVar       SymbolKind = "var"
	KindMethod    SymbolKind = "method"
)

// Symbol represents a single exported Go symbol.
type Symbol struct {
	Name      string     // e.g. "Queue" or "Queue.Enqueue"
	Kind      SymbolKind // func, type, interface, struct, method, etc.
	Signature string     // textual representation for comparison
	File      string     // relative file path
	Line      int
}

// ChangeType classifies a diff entry.
type ChangeType string

const (
	Added    ChangeType = "added"
	Removed  ChangeType = "removed"
	Modified ChangeType = "modified"
)

// Change represents a single symbol change between two versions.
type Change struct {
	Symbol       Symbol
	Type         ChangeType
	OldSignature string // only for Modified
	NewSignature string // only for Modified
}

// IsBreaking returns true if this change could break downstream consumers.
func (c Change) IsBreaking() bool {
	switch c.Type {
	case Removed:
		return true
	case Modified:
		return true
	case Added:
		// Adding a method to an interface is breaking for implementors.
		// The caller must check this against the implementation registry.
		return c.Symbol.Kind == KindMethod
	}
	return false
}

// ExtractExports parses all Go files under dir and returns exported symbols
// keyed by "pkg.Name" (e.g., "queue.Queue", "queue.Queue.Enqueue").
func ExtractExports(dir string) (map[string]Symbol, error) {
	symbols := make(map[string]Symbol)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip vendor, testdata, hidden dirs
		if info.IsDir() {
			base := filepath.Base(path)
			if base == "vendor" || base == "testdata" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		// Only .go files, skip tests
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, _ := filepath.Rel(dir, path)
		pkgDir := filepath.Dir(rel)
		// Use the last path component as the package key prefix
		pkgKey := filepath.Base(pkgDir)
		if pkgKey == "." {
			pkgKey = filepath.Base(dir)
		}

		fset := token.NewFileSet()
		f, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			// Skip unparseable files
			return nil
		}

		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name == nil || !d.Name.IsExported() {
					continue
				}
				if d.Recv != nil && len(d.Recv.List) > 0 {
					// Method
					recvName := receiverTypeName(d.Recv.List[0].Type)
					if recvName == "" || !ast.IsExported(recvName) {
						continue
					}
					key := fmt.Sprintf("%s.%s.%s", pkgKey, recvName, d.Name.Name)
					symbols[key] = Symbol{
						Name:      recvName + "." + d.Name.Name,
						Kind:      KindMethod,
						Signature: funcSignature(d),
						File:      rel,
						Line:      fset.Position(d.Pos()).Line,
					}
				} else {
					// Function
					key := fmt.Sprintf("%s.%s", pkgKey, d.Name.Name)
					symbols[key] = Symbol{
						Name:      d.Name.Name,
						Kind:      KindFunc,
						Signature: funcSignature(d),
						File:      rel,
						Line:      fset.Position(d.Pos()).Line,
					}
				}

			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if !s.Name.IsExported() {
							continue
						}
						key := fmt.Sprintf("%s.%s", pkgKey, s.Name.Name)
						kind := KindType
						switch st := s.Type.(type) {
						case *ast.InterfaceType:
							kind = KindInterface
							symbols[key] = Symbol{
								Name:      s.Name.Name,
								Kind:      kind,
								Signature: interfaceSignature(st),
								File:      rel,
								Line:      fset.Position(s.Pos()).Line,
							}
							// Also extract interface methods as symbols
							if st.Methods != nil {
								for _, m := range st.Methods.List {
									for _, name := range m.Names {
										if !name.IsExported() {
											continue
										}
										mKey := fmt.Sprintf("%s.%s.%s", pkgKey, s.Name.Name, name.Name)
										symbols[mKey] = Symbol{
											Name:      s.Name.Name + "." + name.Name,
											Kind:      KindMethod,
											Signature: fieldSignature(m),
											File:      rel,
											Line:      fset.Position(m.Pos()).Line,
										}
									}
								}
							}

						case *ast.StructType:
							kind = KindStruct
							symbols[key] = Symbol{
								Name:      s.Name.Name,
								Kind:      kind,
								Signature: structSignature(st),
								File:      rel,
								Line:      fset.Position(s.Pos()).Line,
							}
						default:
							symbols[key] = Symbol{
								Name:      s.Name.Name,
								Kind:      kind,
								Signature: typeExprString(s.Type),
								File:      rel,
								Line:      fset.Position(s.Pos()).Line,
							}
						}

					case *ast.ValueSpec:
						for _, name := range s.Names {
							if !name.IsExported() {
								continue
							}
							kind := KindVar
							if d.Tok == token.CONST {
								kind = KindConst
							}
							key := fmt.Sprintf("%s.%s", pkgKey, name.Name)
							symbols[key] = Symbol{
								Name: name.Name,
								Kind: kind,
								File: rel,
								Line: fset.Position(name.Pos()).Line,
							}
						}
					}
				}
			}
		}
		return nil
	})

	return symbols, err
}

// DiffSymbols compares old and new symbol maps and returns changes.
func DiffSymbols(old, new map[string]Symbol) []Change {
	var changes []Change

	// Find removed and modified
	for key, oldSym := range old {
		newSym, exists := new[key]
		if !exists {
			changes = append(changes, Change{
				Symbol: oldSym,
				Type:   Removed,
			})
		} else if oldSym.Signature != newSym.Signature {
			changes = append(changes, Change{
				Symbol:       newSym,
				Type:         Modified,
				OldSignature: oldSym.Signature,
				NewSignature: newSym.Signature,
			})
		}
	}

	// Find added
	for key, newSym := range new {
		if _, exists := old[key]; !exists {
			changes = append(changes, Change{
				Symbol: newSym,
				Type:   Added,
			})
		}
	}

	return changes
}

// --- helpers ---

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	}
	return ""
}

func funcSignature(d *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func(")
	if d.Type.Params != nil {
		b.WriteString(fieldListString(d.Type.Params))
	}
	b.WriteString(")")
	if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
		b.WriteString(" (")
		b.WriteString(fieldListString(d.Type.Results))
		b.WriteString(")")
	}
	return b.String()
}

func interfaceSignature(iface *ast.InterfaceType) string {
	if iface.Methods == nil {
		return "interface{}"
	}
	var parts []string
	for _, m := range iface.Methods.List {
		parts = append(parts, fieldSignature(m))
	}
	return "interface{" + strings.Join(parts, "; ") + "}"
}

func structSignature(st *ast.StructType) string {
	if st.Fields == nil {
		return "struct{}"
	}
	var parts []string
	for _, f := range st.Fields.List {
		if len(f.Names) > 0 && f.Names[0].IsExported() {
			parts = append(parts, fieldSignature(f))
		}
	}
	return "struct{" + strings.Join(parts, "; ") + "}"
}

func fieldSignature(f *ast.Field) string {
	var names []string
	for _, n := range f.Names {
		names = append(names, n.Name)
	}
	typeStr := typeExprString(f.Type)
	if len(names) > 0 {
		return strings.Join(names, ", ") + " " + typeStr
	}
	return typeStr
}

func fieldListString(fl *ast.FieldList) string {
	var parts []string
	for _, f := range fl.List {
		parts = append(parts, fieldSignature(f))
	}
	return strings.Join(parts, ", ")
}

func typeExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeExprString(t.Elt)
		}
		return "[...]" + typeExprString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprString(t.Key) + "]" + typeExprString(t.Value)
	case *ast.ChanType:
		return "chan " + typeExprString(t.Value)
	case *ast.FuncType:
		var b strings.Builder
		b.WriteString("func(")
		if t.Params != nil {
			b.WriteString(fieldListString(t.Params))
		}
		b.WriteString(")")
		if t.Results != nil && len(t.Results.List) > 0 {
			b.WriteString(" (")
			b.WriteString(fieldListString(t.Results))
			b.WriteString(")")
		}
		return b.String()
	case *ast.InterfaceType:
		return interfaceSignature(t)
	case *ast.StructType:
		return structSignature(t)
	case *ast.Ellipsis:
		return "..." + typeExprString(t.Elt)
	case *ast.IndexExpr:
		return typeExprString(t.X) + "[" + typeExprString(t.Index) + "]"
	case *ast.IndexListExpr:
		var indices []string
		for _, idx := range t.Indices {
			indices = append(indices, typeExprString(idx))
		}
		return typeExprString(t.X) + "[" + strings.Join(indices, ", ") + "]"
	case *ast.ParenExpr:
		return "(" + typeExprString(t.X) + ")"
	default:
		return fmt.Sprintf("<%T>", expr)
	}
}
