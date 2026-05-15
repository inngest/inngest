package event

import (
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
)

func TestEvent_SetInvokeSpanRef(t *testing.T) {
	ref := &meta.SpanReference{TraceParent: "00-00112233445566778899aabbccddeeff-0011223344556677-01"}
	evt := Event{
		Name: InvokeFnName,
		Data: map[string]any{
			consts.InngestEventDataPrefix: InngestMetadata{InvokeFnID: "app-fn"},
		},
	}

	require.True(t, evt.SetInvokeSpanRef(ref))

	md := evt.Data[consts.InngestEventDataPrefix].(InngestMetadata)
	require.Equal(t, ref, md.InvokeSpanRef)
	require.Equal(t, "app-fn", md.InvokeFnID, "unrelated fields must be preserved")
}

func TestEvent_SetInvokeSpanRef_ReturnsFalseWhenNoMetadata(t *testing.T) {
	evt := Event{Name: "user/thing.happened", Data: map[string]any{"foo": "bar"}}
	require.False(t, evt.SetInvokeSpanRef(&meta.SpanReference{}))

	nilData := Event{Name: "user/thing.happened"}
	require.False(t, nilData.SetInvokeSpanRef(&meta.SpanReference{}))
}
