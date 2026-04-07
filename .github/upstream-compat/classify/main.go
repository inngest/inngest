// classify — Classify inngest/ changes as safe, additive, or breaking.
//
// Compares exported Go symbols between two versions of inngest/ and reports
// whether the changes would require monorepo/ updates.
//
// In CI mode (--ci), all monorepo file paths and implementation details are
// redacted. Only inngest/ symbols, signatures, counts, and classifications
// are shown.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"inngest.com/upstream-compat/internal/astdiff"
	"inngest.com/upstream-compat/internal/iface"
)

func main() {
	// Top-level panic recovery — in CI mode, avoid leaking stack traces
	// that may contain monorepo file paths.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "internal error: classify check failed")
			os.Exit(2)
		}
	}()

	var (
		inngestDir  string
		monorepoDir string
		ref         string
		oldDir      string
		newDir      string
		ciMode      bool
	)

	flag.StringVar(&inngestDir, "inngest", "../inngest", "Path to inngest/ repo")
	flag.StringVar(&monorepoDir, "monorepo", "../monorepo", "Path to monorepo/ repo")
	flag.StringVar(&ref, "ref", "", "Git ref to compare against (e.g., HEAD~3)")
	flag.StringVar(&oldDir, "old", "", "Direct path to old inngest/ source (alternative to --ref)")
	flag.StringVar(&newDir, "new", "", "Direct path to new inngest/ source (alternative to --inngest)")
	flag.BoolVar(&ciMode, "ci", false, "CI mode: redact monorepo paths and implementation details")
	flag.Parse()

	if newDir == "" {
		newDir = inngestDir
	}

	// Resolve paths
	newDir, _ = filepath.Abs(newDir)
	monorepoDir, _ = filepath.Abs(monorepoDir)

	if oldDir == "" && ref == "" {
		// Default: compare vendor vs current inngest
		vendorDir := filepath.Join(monorepoDir, "vendor", "github.com", "inngest", "inngest")
		if _, err := os.Stat(vendorDir); err == nil {
			oldDir = vendorDir
		} else {
			fmt.Fprintf(os.Stderr, "error: no --old or --ref specified and vendor dir not found\n")
			os.Exit(2)
		}
	}

	if oldDir == "" && ref != "" {
		// Create a temp checkout of inngest at the given ref
		tmpDir, err := checkoutRef(inngestDir, ref)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error checking out ref %s: %v\n", ref, err)
			os.Exit(2)
		}
		defer os.RemoveAll(tmpDir)
		oldDir = tmpDir
	}

	if !ciMode {
		fmt.Println("=== Change Classification ===")
		fmt.Printf("Old: %s\n", oldDir)
		fmt.Printf("New: %s\n", newDir)
		fmt.Println()
	}

	// Extract symbols from both versions
	if !ciMode {
		fmt.Print("Extracting exports from old version... ")
	}
	oldSymbols, err := astdiff.ExtractExports(oldDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error extracting old exports: %v\n", err)
		os.Exit(2)
	}
	if !ciMode {
		fmt.Printf("%d symbols\n", len(oldSymbols))
		fmt.Print("Extracting exports from new version... ")
	}

	newSymbols, err := astdiff.ExtractExports(newDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error extracting new exports: %v\n", err)
		os.Exit(2)
	}
	if !ciMode {
		fmt.Printf("%d symbols\n", len(newSymbols))
	}

	// Diff
	changes := astdiff.DiffSymbols(oldSymbols, newSymbols)
	if len(changes) == 0 {
		fmt.Println("No exported symbol changes detected.")
		fmt.Println("Classification: SAFE")
		os.Exit(0)
	}

	// Classify against interface registry
	registry := iface.DefaultRegistry()

	if !ciMode {
		// Verify implementors exist (skip in CI — would leak monorepo paths)
		warnings := iface.VerifyImplementors(registry, monorepoDir)
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "%s\n", w)
		}
	}

	classified := iface.ClassifyChanges(changes, registry)
	overall := iface.OverallClassification(classified)

	// Sort: breaking first, then additive, then safe
	sort.Slice(classified, func(i, j int) bool {
		return classOrder(classified[i].Classification) < classOrder(classified[j].Classification)
	})

	// Print report
	fmt.Printf("Classification: %s\n\n", overall)

	var breaking, additive, safe []iface.ClassifiedChange
	for _, c := range classified {
		switch c.Classification {
		case iface.Breaking:
			breaking = append(breaking, c)
		case iface.Additive:
			additive = append(additive, c)
		case iface.Safe:
			safe = append(safe, c)
		}
	}

	if len(breaking) > 0 {
		fmt.Printf("BREAKING CHANGES (%d):\n", len(breaking))
		for _, c := range breaking {
			fmt.Printf("  %s %s: %s\n", c.Change.Type, c.Symbol.Name, c.Reason)
			if c.OldSignature != "" {
				fmt.Printf("    was: %s\n", truncate(c.OldSignature, 120))
				fmt.Printf("    now: %s\n", truncate(c.NewSignature, 120))
			}
			if ciMode {
				// Redact: show only a count, not file paths
				if len(c.AffectedFiles) > 0 {
					fmt.Printf("    %d downstream implementations affected\n", len(c.AffectedFiles))
				}
			} else {
				if len(c.AffectedFiles) > 0 {
					fmt.Printf("    monorepo/ files to update:\n")
					for _, f := range c.AffectedFiles {
						fmt.Printf("      - %s\n", f)
					}
				}
			}
			fmt.Printf("    defined in: %s:%d\n", c.Symbol.File, c.Symbol.Line)
		}
		fmt.Println()
	}

	if len(additive) > 0 {
		fmt.Printf("ADDITIVE CHANGES (%d):\n", len(additive))
		for _, c := range additive {
			fmt.Printf("  %s %s: %s\n", c.Change.Type, c.Symbol.Name, c.Reason)
			fmt.Printf("    defined in: %s:%d\n", c.Symbol.File, c.Symbol.Line)
		}
		fmt.Println()
	}

	if len(safe) > 0 {
		fmt.Printf("SAFE CHANGES (%d):\n", len(safe))
		for _, c := range safe {
			fmt.Printf("  %s %s\n", c.Change.Type, c.Symbol.Name)
		}
		fmt.Println()
	}

	// Summary
	fmt.Printf("--- Summary ---\n")
	fmt.Printf("Total changes:    %d\n", len(classified))
	fmt.Printf("Breaking:         %d\n", len(breaking))
	fmt.Printf("Additive:         %d\n", len(additive))
	fmt.Printf("Safe:             %d\n", len(safe))

	// Exit code
	switch overall {
	case iface.Safe:
		os.Exit(0)
	case iface.Additive:
		os.Exit(1)
	case iface.Breaking:
		os.Exit(2)
	}
}

func checkoutRef(repoDir, ref string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "inngest-classify-*")
	if err != nil {
		return "", err
	}

	// Use git archive to extract the old version
	cmd := exec.Command("git", "-C", repoDir, "archive", ref)
	archive, err := cmd.Output()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("git archive failed for ref %s: %w", ref, err)
	}

	// Extract the archive
	tarCmd := exec.Command("tar", "-xf", "-", "-C", tmpDir)
	tarCmd.Stdin = strings.NewReader(string(archive))
	if err := tarCmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("tar extraction failed: %w", err)
	}

	return tmpDir, nil
}

func classOrder(c iface.Classification) int {
	switch c {
	case iface.Breaking:
		return 0
	case iface.Additive:
		return 1
	case iface.Safe:
		return 2
	}
	return 3
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
