package gotesplit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type goListPkg struct {
	ImportPath   string
	Dir          string
	TestGoFiles  []string
	XTestGoFiles []string
	Incomplete   bool
	Error        *goListError
}

type goListError struct {
	Err string
}

func runGoList(pkgs []string, tags string, withRace bool) ([]goListPkg, error) {
	args := []string{"list", "-json", "-e"}
	if tags != "" {
		args = append(args, tags)
	}
	if withRace {
		args = append(args, "-race")
	}
	args = append(args, pkgs...)
	cmd := exec.Command("go", args...)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list failed: %w: %s", err, stderr.String())
	}
	dec := json.NewDecoder(stdout)
	var result []goListPkg
	for {
		var p goListPkg
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode go list output: %w", err)
		}
		result = append(result, p)
	}
	return result, nil
}

// starSuffix returns the trailing type name "X" from *X or *foo.X.
func starSuffix(ptr *ast.StarExpr) (string, bool) {
	switch x := ptr.X.(type) {
	case *ast.Ident:
		return x.Name, true
	case *ast.SelectorExpr:
		return x.Sel.Name, true
	}
	return "", false
}

// isTestFunc reports whether fn should be picked up as a shard-eligible test.
// The logic mirrors cmd/go isTestFunc:
// https://github.com/golang/go/blob/master/src/cmd/go/internal/load/test.go#L551
func isTestFunc(fn *ast.FuncDecl) bool {
	name := fn.Name.Name
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	suffix := name[len("Test"):]
	if suffix == "" {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(suffix); unicode.IsLower(r) {
		return false
	}
	// Signature check: func(t *<X>.T) — no return value, exactly one parameter (one name), of pointer type *<X>.T.
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return false
	}
	params := fn.Type.Params
	if params == nil || len(params.List) != 1 || len(params.List[0].Names) > 1 {
		return false
	}
	ptr, ok := params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	typeName, ok := starSuffix(ptr)
	return ok && typeName == "T"
}

func extractTestNames(dir string, testGoFiles, xTestGoFiles []string) ([]string, error) {
	fset := token.NewFileSet()
	var files []*ast.File
	all := append([]string{}, testGoFiles...)
	all = append(all, xTestGoFiles...)
	for _, name := range all {
		path := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		files = append(files, f)
	}

	var names []string

	// TestXXX
	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Recv != nil {
				continue // skip methods
			}
			if isTestFunc(fn) {
				names = append(names, fn.Name.Name)
			}
		}
	}

	// ExampleXXX
	// https://github.com/golang/go/blob/master/src/cmd/go/internal/load/test.go#L755-L764
	for _, ex := range doc.Examples(files...) {
		if ex.Output == "" && !ex.EmptyOutput {
			continue
		}
		names = append(names, "Example"+ex.Name)
	}
	return names, nil
}

func getTestListsFromPkgsAST(pkgs []string, tags string, withRace bool) ([]testList, error) {
	// Delegate package path resolution and build tag evaluation to the Go toolchain by invoking `go list -json` internally.
	infos, err := runGoList(pkgs, tags, withRace)
	if err != nil {
		return nil, err
	}
	var lists []testList
	for _, p := range infos {
		if p.Error != nil {
			return nil, fmt.Errorf("go list %s: %s", p.ImportPath, p.Error.Err)
		}
		if p.Incomplete {
			return nil, fmt.Errorf("go list %s: incomplete package info", p.ImportPath)
		}
		if len(p.TestGoFiles) == 0 && len(p.XTestGoFiles) == 0 {
			continue
		}
		names, err := extractTestNames(p.Dir, p.TestGoFiles, p.XTestGoFiles)
		if err != nil {
			return nil, err
		}
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)
		lists = append(lists, testList{pkg: p.ImportPath, list: names})
	}
	sort.Slice(lists, func(i, j int) bool {
		cmp := len(lists[i].list) - len(lists[j].list)
		if cmp != 0 {
			return cmp < 0
		}
		return strings.Compare(lists[i].pkg, lists[j].pkg) < 0
	})
	return lists, nil
}
