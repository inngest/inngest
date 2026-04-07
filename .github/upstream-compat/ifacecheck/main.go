// ifacecheck — Check interface compatibility between vendored and current inngest/.
//
// Compares the method sets of known interfaces that monorepo/ implements.
// Reports which interfaces have changed and which monorepo/ files need updating.
//
// In CI mode (--ci), monorepo implementor file paths are replaced with
// a count ("N implementations affected") to avoid leaking monorepo structure.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"inngest.com/upstream-compat/internal/astdiff"
	"inngest.com/upstream-compat/internal/iface"
)

func main() {
	// Top-level panic recovery — in CI mode, avoid leaking stack traces
	// that may contain monorepo file paths.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "internal error: ifacecheck failed")
			os.Exit(2)
		}
	}()

	var (
		oldDir      string
		newDir      string
		monorepoDir string
		ciMode      bool
	)

	flag.StringVar(&oldDir, "old", "", "Path to old inngest/ source (default: monorepo vendor)")
	flag.StringVar(&newDir, "new", "../inngest", "Path to new inngest/ source")
	flag.StringVar(&monorepoDir, "monorepo", "../monorepo", "Path to monorepo/")
	flag.BoolVar(&ciMode, "ci", false, "CI mode: redact monorepo paths and implementation details")
	flag.Parse()

	monorepoDir, _ = filepath.Abs(monorepoDir)
	newDir, _ = filepath.Abs(newDir)

	if oldDir == "" {
		oldDir = filepath.Join(monorepoDir, "vendor", "github.com", "inngest", "inngest")
	}
	oldDir, _ = filepath.Abs(oldDir)

	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: old dir not found: %s\n", oldDir)
		os.Exit(2)
	}
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "error: new dir not found: %s\n", newDir)
		os.Exit(2)
	}

	registry := iface.DefaultRegistry()

	if !ciMode {
		fmt.Println("=== Interface Compatibility Check ===")
		fmt.Printf("Old (vendored): %s\n", oldDir)
		fmt.Printf("New (current):  %s\n", newDir)
		fmt.Println()
	}

	hasBreaking := false
	hasChanges := false

	for _, w := range registry {
		// Determine the subdirectory to check
		pkgSubdir := packageToSubdir(w.Package)

		oldPkgDir := filepath.Join(oldDir, pkgSubdir)
		newPkgDir := filepath.Join(newDir, pkgSubdir)

		if _, err := os.Stat(oldPkgDir); os.IsNotExist(err) {
			fmt.Printf("  SKIP %s.%s — old package dir not found\n", shortPkg(w.Package), w.Name)
			continue
		}
		if _, err := os.Stat(newPkgDir); os.IsNotExist(err) {
			fmt.Printf("  REMOVED %s.%s — package no longer exists!\n", shortPkg(w.Package), w.Name)
			hasBreaking = true
			hasChanges = true
			if w.Implemented && !ciMode {
				printImplementors(w)
			} else if w.Implemented && ciMode {
				fmt.Printf("    %d implementations affected\n", len(w.Implementors))
			}
			continue
		}

		// Extract exports from both versions of this specific package
		oldSymbols, err := astdiff.ExtractExports(oldPkgDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error parsing old %s: %v\n", pkgSubdir, err)
			continue
		}
		newSymbols, err := astdiff.ExtractExports(newPkgDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error parsing new %s: %v\n", pkgSubdir, err)
			continue
		}

		// Find the interface itself
		pkgBase := filepath.Base(pkgSubdir)
		ifaceKey := fmt.Sprintf("%s.%s", pkgBase, w.Name)

		oldIface, oldHas := oldSymbols[ifaceKey]
		newIface, newHas := newSymbols[ifaceKey]

		if !oldHas && !newHas {
			fmt.Printf("  SKIP %s.%s — interface not found in either version\n", shortPkg(w.Package), w.Name)
			continue
		}
		if !oldHas && newHas {
			fmt.Printf("  NEW  %s.%s — interface added\n", shortPkg(w.Package), w.Name)
			hasChanges = true
			continue
		}
		if oldHas && !newHas {
			fmt.Printf("  REMOVED %s.%s — interface removed!\n", shortPkg(w.Package), w.Name)
			hasBreaking = true
			hasChanges = true
			if w.Implemented && !ciMode {
				printImplementors(w)
			} else if w.Implemented && ciMode {
				fmt.Printf("    %d implementations affected\n", len(w.Implementors))
			}
			continue
		}

		// Compare interface signatures
		if oldIface.Signature == newIface.Signature {
			fmt.Printf("  OK   %s.%s — no changes\n", shortPkg(w.Package), w.Name)
			continue
		}

		hasChanges = true

		// Find specific method changes
		methodChanges := findMethodChanges(oldSymbols, newSymbols, pkgBase, w.Name)

		if len(methodChanges) == 0 {
			// Signature changed but no method-level changes detected (embedded interface change?)
			fmt.Printf("  CHANGED %s.%s — signature changed (possibly embedded interface)\n", shortPkg(w.Package), w.Name)
			if w.Implemented {
				hasBreaking = true
				if !ciMode {
					printImplementors(w)
				} else {
					fmt.Printf("    %d implementations affected\n", len(w.Implementors))
				}
			}
			continue
		}

		// Report method-level changes
		breaking := false
		for _, mc := range methodChanges {
			if mc.Type == astdiff.Removed || mc.Type == astdiff.Modified {
				breaking = true
			}
			if mc.Type == astdiff.Added && w.Implemented {
				breaking = true
			}
		}

		if breaking {
			hasBreaking = true
			fmt.Printf("  BREAKING %s.%s [%s]:\n", shortPkg(w.Package), w.Name, w.Criticality)
		} else {
			fmt.Printf("  CHANGED %s.%s [%s]:\n", shortPkg(w.Package), w.Name, w.Criticality)
		}

		for _, mc := range methodChanges {
			switch mc.Type {
			case astdiff.Added:
				fmt.Printf("    + %s\n", mc.Symbol.Name)
				if mc.Symbol.Signature != "" {
					fmt.Printf("      %s\n", truncate(mc.Symbol.Signature, 100))
				}
			case astdiff.Removed:
				fmt.Printf("    - %s\n", mc.Symbol.Name)
			case astdiff.Modified:
				fmt.Printf("    ~ %s\n", mc.Symbol.Name)
				fmt.Printf("      was: %s\n", truncate(mc.OldSignature, 100))
				fmt.Printf("      now: %s\n", truncate(mc.NewSignature, 100))
			}
		}

		if w.Implemented && breaking {
			if !ciMode {
				printImplementors(w)
			} else {
				fmt.Printf("    %d implementations affected\n", len(w.Implementors))
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println()
	fmt.Println("--- Summary ---")
	if !hasChanges {
		fmt.Println("All watched interfaces are compatible.")
		os.Exit(0)
	}
	if hasBreaking {
		fmt.Println("BREAKING interface changes detected.")
		if !ciMode {
			fmt.Println("Monorepo/ implementations need updating before vendoring.")
		} else {
			fmt.Println("Downstream implementations need updating before vendoring.")
		}
		os.Exit(2)
	}
	fmt.Println("Interface changes detected but none are breaking for downstream implementations.")
	os.Exit(0)
}

func findMethodChanges(oldSymbols, newSymbols map[string]astdiff.Symbol, pkgBase, ifaceName string) []astdiff.Change {
	prefix := fmt.Sprintf("%s.%s.", pkgBase, ifaceName)
	var changes []astdiff.Change

	// Find removed and modified methods
	for key, oldSym := range oldSymbols {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if newSym, ok := newSymbols[key]; !ok {
			changes = append(changes, astdiff.Change{
				Symbol: oldSym,
				Type:   astdiff.Removed,
			})
		} else if oldSym.Signature != newSym.Signature {
			changes = append(changes, astdiff.Change{
				Symbol:       newSym,
				Type:         astdiff.Modified,
				OldSignature: oldSym.Signature,
				NewSignature: newSym.Signature,
			})
		}
	}

	// Find added methods
	for key, newSym := range newSymbols {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if _, ok := oldSymbols[key]; !ok {
			changes = append(changes, astdiff.Change{
				Symbol: newSym,
				Type:   astdiff.Added,
			})
		}
	}

	return changes
}

func printImplementors(w iface.WatchedInterface) {
	fmt.Printf("    Implementations to update:\n")
	for _, f := range w.Implementors {
		fmt.Printf("      - monorepo/%s\n", f)
	}
}

func packageToSubdir(pkg string) string {
	const prefix = "github.com/inngest/inngest/"
	if strings.HasPrefix(pkg, prefix) {
		return pkg[len(prefix):]
	}
	return pkg
}

func shortPkg(pkg string) string {
	parts := strings.Split(pkg, "/")
	if len(parts) > 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return pkg
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
