// Command tygo-collect scans Go packages for types annotated with //tygo:generate
// and collects them into a single barrel file for TypeScript generation via tygo.
//
// Usage:
//
//	tygo-collect -o output.go pkg1 pkg2 ...
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const annotation = "//tygo:generate"

func main() {
	output := flag.String("o", "", "output file path (required)")
	pkgName := flag.String("pkg", "types", "package name for generated file")
	flag.Parse()

	if *output == "" {
		fmt.Fprintln(os.Stderr, "error: -o output file is required")
		os.Exit(1)
	}

	packages := flag.Args()
	if len(packages) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one package path is required")
		os.Exit(1)
	}

	collected, err := collectAnnotatedTypes(packages)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := writeBarrelFile(*output, *pkgName, collected); err != nil {
		fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d type(s)\n", *output, len(collected))
}

type collectedType struct {
	Name       string // Used for duplicate detection
	Source     string
	SourceFile string
}

func collectAnnotatedTypes(packages []string) ([]collectedType, error) {
	var collected []collectedType

	for _, pkgPath := range packages {
		types, err := scanPackage(pkgPath)
		if err != nil {
			return nil, fmt.Errorf("scanning %s: %w", pkgPath, err)
		}
		collected = append(collected, types...)
	}

	// Check for duplicate type names
	if err := checkDuplicates(collected); err != nil {
		return nil, err
	}

	return collected, nil
}

func checkDuplicates(collected []collectedType) error {
	seen := make(map[string]string) // name -> source file
	for _, t := range collected {
		if t.Name == "" {
			continue // Skip unnamed entries (shouldn't happen, but be safe)
		}
		if prev, ok := seen[t.Name]; ok {
			return fmt.Errorf("duplicate type %q: found in both %s and %s", t.Name, prev, t.SourceFile)
		}
		seen[t.Name] = t.SourceFile
	}
	return nil
}

func scanPackage(pkgPath string) ([]collectedType, error) {
	fset := token.NewFileSet()

	// Parse all Go files in the package directory
	pkgs, err := parser.ParseDir(fset, pkgPath, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing package: %w", err)
	}

	var collected []collectedType

	// Sort package names for deterministic output order
	pkgNames := make([]string, 0, len(pkgs))
	for name := range pkgs {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	for _, pkgName := range pkgNames {
		pkg := pkgs[pkgName]

		// Sort filenames for deterministic output order
		filenames := make([]string, 0, len(pkg.Files))
		for filename := range pkg.Files {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)

		for _, filename := range filenames {
			file := pkg.Files[filename]
			types, err := scanFile(fset, filename, file)
			if err != nil {
				return nil, fmt.Errorf("scanning %s: %w", filename, err)
			}
			collected = append(collected, types...)
		}
	}

	return collected, nil
}

func scanFile(fset *token.FileSet, filename string, file *ast.File) ([]collectedType, error) {
	var collected []collectedType

	// Build a map of comment positions to their text
	commentMap := make(map[int]string)
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			line := fset.Position(c.Pos()).Line
			commentMap[line] = c.Text
		}
	}

	// Track which annotation lines have been used to avoid a later declaration
	// accidentally matching an earlier annotation.
	usedAnnotations := make(map[int]bool)

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// Check if the declaration has our annotation
			declLine := fset.Position(d.Pos()).Line
			annotationLine := findAnnotationLine(commentMap, usedAnnotations, declLine)
			if annotationLine == 0 {
				continue
			}
			usedAnnotations[annotationLine] = true

			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					source, err := nodeToString(fset, d, false)
					if err != nil {
						return nil, err
					}
					collected = append(collected, collectedType{
						Name:       s.Name.Name,
						Source:     source,
						SourceFile: filepath.Base(filename),
					})

				case *ast.ValueSpec:
					if d.Tok == token.CONST {
						source, err := nodeToString(fset, d, true)
						if err != nil {
							return nil, err
						}
						// Use first name for identification
						name := ""
						if len(s.Names) > 0 {
							name = s.Names[0].Name
						}
						collected = append(collected, collectedType{
							Name:       name,
							Source:     source,
							SourceFile: filepath.Base(filename),
						})
					}
				}
			}
		}
	}

	return collected, nil
}

// findAnnotationLine looks for a tygo:generate annotation before the declaration.
// It returns the line number of the annotation, or 0 if not found.
// usedAnnotations tracks annotations that have already been consumed by prior declarations.
func findAnnotationLine(commentMap map[int]string, usedAnnotations map[int]bool, declLine int) int {
	// Check up to 3 lines before the declaration to handle blank lines
	// between the annotation and the declaration.
	for offset := 1; offset <= 3; offset++ {
		line := declLine - offset
		if comment, ok := commentMap[line]; ok {
			if strings.HasPrefix(strings.TrimSpace(comment), annotation) {
				if usedAnnotations[line] {
					// This annotation was already used by a previous declaration
					return 0
				}
				return line
			}
			// Found a different comment, stop looking
			return 0
		}
		// Line was empty (not in commentMap), continue checking
	}
	return 0
}

// hasAnnotation checks if a declaration has a tygo:generate annotation.
// This is a convenience wrapper used by tests.
func hasAnnotation(commentMap map[int]string, declLine int) bool {
	return findAnnotationLine(commentMap, nil, declLine) != 0
}

func nodeToString(fset *token.FileSet, node ast.Node, isConst bool) (string, error) {
	// For const declarations, remove type annotations so the barrel file compiles
	// (the original types like metadata.Kind aren't imported)
	// We clone the node to avoid mutating the original AST.
	if isConst {
		if genDecl, ok := node.(*ast.GenDecl); ok {
			genDeclCopy := *genDecl
			genDeclCopy.Specs = make([]ast.Spec, len(genDecl.Specs))
			for i, spec := range genDecl.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					vsCopy := *vs
					vsCopy.Type = nil // Remove type annotation
					genDeclCopy.Specs[i] = &vsCopy
				} else {
					genDeclCopy.Specs[i] = spec
				}
			}
			node = &genDeclCopy
		}
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return "", err
	}

	// Strip the //tygo:generate annotation from output
	source := buf.String()
	source = strings.ReplaceAll(source, annotation+"\n", "")
	source = strings.ReplaceAll(source, annotation, "")

	return source, nil
}

func writeBarrelFile(output string, pkgName string, collected []collectedType) error {
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}

	var buf bytes.Buffer

	buf.WriteString("// Code generated by tygo-collect. DO NOT EDIT.\n")
	buf.WriteString("// This file collects types from multiple packages for TypeScript generation.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

	for _, t := range collected {
		buf.WriteString(fmt.Sprintf("// From %s\n", t.SourceFile))
		buf.WriteString(t.Source)
		buf.WriteString("\n\n")
	}

	return os.WriteFile(output, buf.Bytes(), 0644)
}
