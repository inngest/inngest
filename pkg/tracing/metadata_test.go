package tracing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/stretchr/testify/require"
)

// mockStructured implements metadata.Structured with configurable behavior.
type mockStructured struct {
	kind         metadata.Kind
	values       metadata.Values
	serializeErr error
}

func (m *mockStructured) Kind() metadata.Kind      { return m.kind }
func (m *mockStructured) Op() enums.MetadataOpcode { return enums.MetadataOpcodeMerge }
func (m *mockStructured) Serialize() (metadata.Values, error) {
	if m.serializeErr != nil {
		return nil, m.serializeErr
	}
	return m.values, nil
}

// makeValues creates a Values map with a single key whose total size (key + value) equals targetSize.
func makeValues(targetSize int) metadata.Values {
	if targetSize <= 0 {
		return metadata.Values{}
	}
	key := "k"
	valSize := targetSize - len(key)
	if valSize < 0 {
		return metadata.Values{key[:targetSize]: json.RawMessage{}}
	}
	return metadata.Values{
		key: json.RawMessage(strings.Repeat("x", valSize)),
	}
}

func TestCreateMetadataSpan_SpanExactlyAtLimit(t *testing.T) {
	tp := NewNoopTracerProvider()

	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(consts.MaxMetadataSpanSize),
	}

	ref, err := CreateMetadataSpan(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
}

func TestCreateMetadataSpan_SpanOverLimit(t *testing.T) {
	md := &mockStructured{
		kind:   "test.kind",
		values: makeValues(consts.MaxMetadataSpanSize + 1),
	}

	ref, err := CreateMetadataSpan(
		context.Background(), nil, nil,
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.True(t, errors.Is(err, metadata.ErrMetadataSpanTooLarge))
	require.Nil(t, ref)
}

func TestCreateMetadataSpan_EmptyValues(t *testing.T) {
	tp := NewNoopTracerProvider()

	md := &mockStructured{
		kind:   "test.kind",
		values: metadata.Values{},
	}

	ref, err := CreateMetadataSpan(
		context.Background(), tp, &meta.SpanReference{},
		"test.location", "test", nil, md, enums.MetadataScopeStep,
	)
	require.NoError(t, err)
	require.NotNil(t, ref)
}
