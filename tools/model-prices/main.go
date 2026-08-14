// Command model-prices slims down a raw LiteLLM model_prices_and_context_window.json
// snapshot to only the fields pkg/tracing/metadata/extractors actually uses for AI
// cost estimation (input/output cost per token), dropping everything else
// (context windows, capability flags, provider metadata, etc.) along with the
// upstream "sample_spec" documentation placeholder and any entries that aren't
// priced per token (image/audio/embedding models).
//
// Run this on a freshly downloaded snapshot before committing it, so the
// embedded file stays small. See scripts/update-model-prices.sh.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// placeholderKey is a documentation-only entry in the upstream file (a
// template showing every possible field, with dummy zero-value costs) - it
// isn't a real model and must be excluded.
const placeholderKey = "sample_spec"

// rawEntry mirrors only the fields we care about from the upstream file;
// unknown fields (context windows, capability flags, provider metadata, etc.)
// are ignored by json.Unmarshal.
type rawEntry struct {
	InputCostPerToken  *float64 `json:"input_cost_per_token"`
	OutputCostPerToken *float64 `json:"output_cost_per_token"`
}

// slimEntry is the trimmed-down shape written back out - the only fields
// pkg/tracing/metadata/extractors.mustLoadModelPricing reads.
type slimEntry struct {
	InputCostPerToken  float64 `json:"input_cost_per_token"`
	OutputCostPerToken float64 `json:"output_cost_per_token"`
}

func main() {
	inPath := flag.String("in", "pkg/tracing/metadata/extractors/model_prices.json", "path to the raw model_prices.json to slim down")
	outPath := flag.String("out", "", "path to write the slimmed JSON to (defaults to -in, overwriting it in place)")
	flag.Parse()

	if *outPath == "" {
		*outPath = *inPath
	}

	if err := run(*inPath, *outPath); err != nil {
		fmt.Fprintln(os.Stderr, "model-prices:", err)
		os.Exit(1)
	}
}

func run(inPath, outPath string) error {
	raw, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inPath, err)
	}

	var entries map[string]rawEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return fmt.Errorf("parsing %s: %w", inPath, err)
	}

	slimmed := make(map[string]slimEntry, len(entries))
	for model, entry := range entries {
		if model == placeholderKey {
			continue
		}
		if entry.InputCostPerToken == nil || entry.OutputCostPerToken == nil {
			continue
		}
		if *entry.InputCostPerToken == 0 && *entry.OutputCostPerToken == 0 {
			// A model priced at exactly zero for both input and output is
			// almost always an unfilled upstream placeholder, not a
			// genuinely free model - excluding it avoids a real model's
			// usage silently costing nothing.
			continue
		}
		slimmed[model] = slimEntry{
			InputCostPerToken:  *entry.InputCostPerToken,
			OutputCostPerToken: *entry.OutputCostPerToken,
		}
	}

	out, err := json.MarshalIndent(slimmed, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling slimmed pricing: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(outPath, out, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	fmt.Printf("model-prices: kept %d of %d entries (%d bytes -> %d bytes)\n", len(slimmed), len(entries), len(raw), len(out))
	return nil
}
