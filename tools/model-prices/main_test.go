package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRun_SlimsAndFilters(t *testing.T) {
	t.Parallel()

	input := `{
		"sample_spec": {
			"input_cost_per_token": 0.0,
			"output_cost_per_token": 0.0,
			"deprecation_date": "a description, not a real date"
		},
		"whisper-1": {
			"litellm_provider": "openai",
			"mode": "audio_transcription"
		},
		"some-local-model": {
			"input_cost_per_token": 0.0,
			"output_cost_per_token": 0.0,
			"litellm_provider": "ollama",
			"mode": "chat"
		},
		"input-only-free-model": {
			"input_cost_per_token": 0.0,
			"output_cost_per_token": 1e-05,
			"litellm_provider": "openai",
			"mode": "chat"
		},
		"gpt-4o": {
			"max_tokens": 16384,
			"max_input_tokens": 128000,
			"input_cost_per_token": 2.5e-06,
			"output_cost_per_token": 1e-05,
			"litellm_provider": "openai",
			"mode": "chat"
		}
	}`

	dir := t.TempDir()
	inPath := filepath.Join(dir, "in.json")
	outPath := filepath.Join(dir, "out.json")
	if err := os.WriteFile(inPath, []byte(input), 0644); err != nil {
		t.Fatalf("writing input fixture: %v", err)
	}

	if err := run(inPath, outPath); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	var got map[string]slimEntry
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output isn't valid JSON: %v\n%s", err, out)
	}

	if _, ok := got["sample_spec"]; ok {
		t.Fatalf("sample_spec placeholder should have been excluded, got: %+v", got)
	}
	if _, ok := got["whisper-1"]; ok {
		t.Fatalf("entry without token costs should have been excluded, got: %+v", got)
	}
	if _, ok := got["some-local-model"]; ok {
		t.Fatalf("entry priced at zero for both input and output should have been excluded, got: %+v", got)
	}

	wantHalfFree := slimEntry{InputCostPerToken: 0, OutputCostPerToken: 1e-05}
	if got["input-only-free-model"] != wantHalfFree {
		t.Fatalf("input-only-free-model entry = %+v, want %+v (zero on only one side must be kept)", got["input-only-free-model"], wantHalfFree)
	}

	want := slimEntry{InputCostPerToken: 2.5e-06, OutputCostPerToken: 1e-05}
	if got["gpt-4o"] != want {
		t.Fatalf("gpt-4o entry = %+v, want %+v", got["gpt-4o"], want)
	}
	if len(got) != 2 {
		t.Fatalf("expected exactly 2 surviving entries, got %d: %+v", len(got), got)
	}

	// The slimmed entry must not carry over unrelated fields (max_tokens, mode, etc).
	var rawOut map[string]map[string]json.RawMessage
	if err := json.Unmarshal(out, &rawOut); err != nil {
		t.Fatalf("re-parsing output as raw: %v", err)
	}
	if fields := rawOut["gpt-4o"]; len(fields) != 2 {
		t.Fatalf("gpt-4o should only have 2 fields, got %d: %+v", len(fields), fields)
	}
}

func TestRun_DefaultsOutToIn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "prices.json")
	input := `{"gpt-4o":{"input_cost_per_token":2.5e-06,"output_cost_per_token":1e-05}}`
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if err := run(path, path); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading in-place output: %v", err)
	}

	var got map[string]slimEntry
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output isn't valid JSON: %v\n%s", err, out)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d: %+v", len(got), got)
	}
}
