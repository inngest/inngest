package event

import (
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestDeferEventID(t *testing.T) {
	fixedParent := ulid.MustParse("01HKQJZ5R7XR4MNTQGZ8Z3KPAB")
	id, err := DeferEventID(fixedParent, "fixed-hashed-id")
	require.NoError(t, err)
	require.Equal(t,
		"01HKQJZ5R7GC5N0NPVNFBX80JK",
		id.String(),
		"pinned to detect any change to the defer-event seed template")
}

func TestDeferredScheduleMetadataValidate(t *testing.T) {
	m := DeferredScheduleMetadata{}
	err := m.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "fn_slug")
	require.Contains(t, err.Error(), "parent_fn_slug")
	require.Contains(t, err.Error(), "parent_run_id")
	require.Contains(t, err.Error(), "parent_defer_span")
}

func TestEventDeferredScheduleMetadata(t *testing.T) {
	wantMeta := DeferredScheduleMetadata{
		FnSlug:       "child-fn",
		ParentFnSlug: "parent-fn",
		ParentRunID:  ulid.MustParse("01HKQJZ5R7XR4MNTQGZ8Z3KPAB"),
		ParentDeferSpan: &meta.SpanReference{
			TraceParent:            "00-deadbeefcafebabe00000000000000-0011223344556677-00",
			DynamicSpanID:          "0011223344556677",
			DynamicSpanTraceParent: "00-deadbeefcafebabe00000000000000-aabbccddeeff0011-00",
		},
	}

	t.Run("missing _inngest prefix returns an error", func(t *testing.T) {
		r := require.New(t)
		e := Event{Data: map[string]any{"x": 1}}
		got, err := e.DeferredScheduleMetadata()
		r.Error(err)
		r.Contains(err.Error(), consts.InngestEventDataPrefix)
		r.Nil(got)
	})

	t.Run("decodes a map[string]any payload", func(t *testing.T) {
		r := require.New(t)
		e := Event{Data: map[string]any{
			consts.InngestEventDataPrefix: map[string]any{
				"fn_slug":        wantMeta.FnSlug,
				"parent_fn_slug": wantMeta.ParentFnSlug,
				"parent_run_id":  wantMeta.ParentRunID.String(),
				"parent_defer_span": map[string]any{
					"tp":   wantMeta.ParentDeferSpan.TraceParent,
					"dsid": wantMeta.ParentDeferSpan.DynamicSpanID,
					"dstp": wantMeta.ParentDeferSpan.DynamicSpanTraceParent,
				},
			},
		}}
		got, err := e.DeferredScheduleMetadata()
		r.NoError(err)
		r.NotNil(got)
		r.Equal(wantMeta, *got)
	})

	t.Run("bad payload returns an error", func(t *testing.T) {
		r := require.New(t)
		e := Event{Data: map[string]any{consts.InngestEventDataPrefix: "not-an-object"}}
		got, err := e.DeferredScheduleMetadata()
		r.Error(err)
		r.Nil(got)
	})
}
