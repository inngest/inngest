package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage: convert-openapi <input-dir> <output-dir>")
	}

	inputDir := os.Args[1]
	outputDir := os.Args[2]

	if err := convertOpenAPIFiles(inputDir, outputDir); err != nil {
		log.Fatalf("Error converting OpenAPI files: %v", err)
	}

	fmt.Printf("Successfully converted OpenAPI v2 files from %s to OpenAPI v3 in %s\n", inputDir, outputDir)
}

func convertOpenAPIFiles(inputDir, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Walk through input directory and find all .json files
	return filepath.WalkDir(inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".json") {
			return nil
		}

		// Read the OpenAPI v2 file
		v2Data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Parse as OpenAPI v2
		var v2Doc openapi2.T
		if err := json.Unmarshal(v2Data, &v2Doc); err != nil {
			fmt.Printf("Warning: skipping file %s (not a valid OpenAPI v2 file): %v\n", path, err)
			return nil
		}

		// Convert to OpenAPI v3
		v3Doc, err := openapi2conv.ToV3(&v2Doc)
		if err != nil {
			return fmt.Errorf("failed to convert %s to OpenAPI v3: %w", path, err)
		}

		// Generate output filename
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		outputPath := filepath.Join(outputDir, relPath)

		// Create output subdirectories if needed
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output subdirectory: %w", err)
		}

		// Marshal OpenAPI v3 to JSON
		v3Data, err := json.MarshalIndent(v3Doc, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal OpenAPI v3 for %s: %w", path, err)
		}

		// Write OpenAPI v3 file
		if err := os.WriteFile(outputPath, v3Data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", outputPath, err)
		}

		fmt.Printf("Converted %s -> %s\n", path, outputPath)
		return nil
	})
}