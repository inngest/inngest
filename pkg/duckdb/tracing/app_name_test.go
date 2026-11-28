package tracing

import (
	"testing"

	sv2 "github.com/inngest/inngest/pkg/execution/state/v2"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
)

// addTenantAndDebugAttrs gives every span the app name the run state holds,
// so the dual-write's run rows have it (and none for a run scheduled before
// the app name was stored).
func TestAddTenantAndDebugAttrsAppName(t *testing.T) {
	for _, tc := range []struct {
		stored string
		want   *string
	}{
		{stored: "my-app", want: new("my-app")},
		{stored: "", want: nil},
	} {
		md := &sv2.Metadata{Config: *sv2.InitConfig(&sv2.Config{})}
		if tc.stored != "" {
			md.Config.SetAppName(tc.stored)
		}
		md.Config.SetFunctionSlug("my-app-fn")
		attrs := meta.NewAttrSet()
		addTenantAndDebugAttrs(attrs, md)

		var got *string
		for _, a := range attrs.Attrs {
			if a.Key() == meta.Attrs.AppName.Key() {
				v, ok := a.Value().(*string)
				require.True(t, ok, "%T", a.Value())
				got = v
			}
		}
		require.Equal(t, tc.want, got)
	}
}
