package executor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/stretchr/testify/require"
)

func stepMetadataUpdate(kind metadata.Kind, values metadata.Values) metadata.ScopedUpdate {
	return metadata.ScopedUpdate{
		Scope: enums.MetadataScopeStep,
		Update: metadata.Update{
			RawUpdate: metadata.RawUpdate{
				Kind:   kind,
				Op:     enums.MetadataOpcodeMerge,
				Values: values,
			},
		},
	}
}

// writtenMetadataKinds returns the kind of every metadata span the recording
// tracer saw, in creation order.
func writtenMetadataKinds(t *testing.T, tp *recordingTracerProvider) []metadata.Kind {
	t.Helper()

	var kinds []metadata.Kind
	for _, call := range tp.createCalls {
		if call.name != meta.SpanNameMetadata {
			continue
		}
		kind, ok := meta.GetAttr(call.opts.Attributes, meta.Attrs.MetadataKind)
		require.True(t, ok, "metadata span missing kind attribute")
		kinds = append(kinds, *kind)
	}
	return kinds
}

func TestHandleGeneratorMetadata_ValidatesSDKMetadata(t *testing.T) {
	tests := []struct {
		name    string
		update  metadata.ScopedUpdate
		written bool
	}{
		{
			name:    "allowed inngest kind is written",
			update:  stepMetadataUpdate("inngest.ai", metadata.Values{"model": json.RawMessage(`"gpt-4o"`)}),
			written: true,
		},
		{
			name:    "userland kind is written",
			update:  stepMetadataUpdate("userland.custom", metadata.Values{"k": json.RawMessage(`"v"`)}),
			written: true,
		},
		{
			name:    "valid score is written",
			update:  stepMetadataUpdate(metadata.KindInngestScore, metadata.Values{"accuracy": json.RawMessage(`{"value":0.9}`)}),
			written: true,
		},
		{
			name:   "disallowed inngest kind is dropped",
			update: stepMetadataUpdate("inngest.usage", metadata.Values{"k": json.RawMessage(`"v"`)}),
		},
		{
			name:   "invalid score value is dropped",
			update: stepMetadataUpdate(metadata.KindInngestScore, metadata.Values{"accuracy": json.RawMessage(`{"value":"high"}`)}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tp := newRecordingTracerProvider()
			e := &executor{
				log:            logger.From(context.Background()),
				tracerProvider: tp,
			}
			gen := &state.GeneratorOpcode{
				ID:       "step-1",
				Op:       enums.OpcodeStepRun,
				Metadata: []metadata.ScopedUpdate{tc.update},
			}

			e.handleGeneratorMetadata(context.Background(), newTestRunContext(), gen)

			if tc.written {
				require.Equal(t, []metadata.Kind{tc.update.Kind()}, writtenMetadataKinds(t, tp))
			} else {
				require.Empty(t, writtenMetadataKinds(t, tp))
			}
		})
	}
}

func TestHandleGeneratorMetadata_DropsOnlyInvalidEntries(t *testing.T) {
	tp := newRecordingTracerProvider()
	e := &executor{
		log:            logger.From(context.Background()),
		tracerProvider: tp,
	}
	gen := &state.GeneratorOpcode{
		ID: "step-1",
		Op: enums.OpcodeStepRun,
		Metadata: []metadata.ScopedUpdate{
			stepMetadataUpdate("inngest.ai.summary", metadata.Values{"k": json.RawMessage(`"v"`)}),
			stepMetadataUpdate("userland.custom", metadata.Values{"k": json.RawMessage(`"v"`)}),
		},
	}

	// Server-generated extra metadata uses kinds SDKs cannot set and must
	// bypass the allowlist.
	extra := &mockStructured{
		kind:   "inngest.timing",
		values: metadata.Values{"k": json.RawMessage(`"v"`)},
	}

	e.handleGeneratorMetadata(context.Background(), newTestRunContext(), gen, extra)

	require.Equal(t, []metadata.Kind{"userland.custom", "inngest.timing"}, writtenMetadataKinds(t, tp))
}
