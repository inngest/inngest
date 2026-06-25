package extractors_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"

	"github.com/inngest/inngest/pkg/tracing/metadata/extractors"
)

// goldenAIMetadata mirrors extractors.AIMetadata without omitempty so the
// golden files keep empty fields visible — what an instrumentation does NOT
// emit is part of what these tests lock in. LatencyMs is excluded as
// span-timing dependent and EstimatedCost as covered by EstimateCost's own
// tests (and coupled to the pricing table).
type goldenAIMetadata struct {
	Model         string   `json:"model"`
	System        string   `json:"system"`
	OperationName string   `json:"operation_name"`
	ResponseModel string   `json:"response_model"`
	ResponseID    string   `json:"response_id"`
	FinishReasons []string `json:"finish_reasons"`
	InputTokens   int64    `json:"input_tokens"`
	OutputTokens  int64    `json:"output_tokens"`
	TotalTokens   *int64   `json:"total_tokens"`
}

// TestAIMetadataExtractor_CapturedFixtures asserts AIMetadata extraction
// against every captured OTLP fixture under testdata/, with a golden file
// alongside each fixture (<fixture>.out). Every span is rendered in document
// order. Regenerate with `go test -update`; see testdata/README.md for how
// fixtures were captured and per-instrumentation quirks.
func TestAIMetadataExtractor_CapturedFixtures(t *testing.T) {
	// goldenAIMetadata must mirror every asserted field of AIMetadata (9
	// rendered + the 2 deliberately-blanked LatencyMs/EstimatedCost). If this
	// fails, a field was added: update goldenAIMetadata and regenerate.
	require.Equal(t, 11, reflect.TypeFor[extractors.AIMetadata]().NumField(),
		"AIMetadata changed shape; update goldenAIMetadata and the goldens")

	paths, err := fs.Glob(fixtures, "testdata/*/*.otlp.json")
	require.NoError(t, err)
	require.NotEmpty(t, paths, "no OTLP fixtures discovered; check the glob")

	for _, path := range paths {
		name := strings.TrimPrefix(path, "testdata/")
		t.Run(name, func(t *testing.T) {
			spans := loadOTLPSpans(t, name)

			var buf bytes.Buffer
			for i, span := range spans {
				if i > 0 {
					buf.WriteRune('\n')
				}
				buf.WriteString("SPAN ")
				buf.WriteString(span.Name)
				buf.WriteRune('\n')

				structured, err := extractors.NewAIMetadataExtractor().
					ExtractSpanMetadata(context.Background(), span)
				require.NoError(t, err)

				if len(structured) == 0 {
					buf.WriteString("no AI metadata extracted\n")
					continue
				}
				require.Len(t, structured, 1)
				md, ok := structured[0].(extractors.AIMetadata)
				require.True(t, ok, "expected AIMetadata, got %T", structured[0])

				enc := json.NewEncoder(&buf)
				enc.SetIndent("", "  ")
				require.NoError(t, enc.Encode(goldenAIMetadata{
					Model:         md.Model,
					System:        md.Provider,
					OperationName: md.OperationName,
					ResponseModel: md.ResponseModel,
					ResponseID:    md.ResponseID,
					FinishReasons: md.FinishReasons,
					InputTokens:   md.InputTokens,
					OutputTokens:  md.OutputTokens,
					TotalTokens:   md.TotalTokens,
				}))
			}

			g := goldie.New(t,
				goldie.WithFixtureDir("testdata"),
				goldie.WithNameSuffix(".out"),
				goldie.WithDiffEngine(goldie.ColoredDiff),
			)
			g.Assert(t, name, buf.Bytes())
		})
	}
}
