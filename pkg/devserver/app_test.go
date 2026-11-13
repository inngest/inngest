package devserver

import (
	"testing"

	"github.com/inngest/inngest/pkg/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// TestOptionsGraph checks that every dependency of the server has a
// provider.  fx.ValidateApp does not call the constructors, so the test needs
// no database, Redis, or open port.
func TestOptionsGraph(t *testing.T) {
	conf, err := config.Dev(t.Context())
	require.NoError(t, err)

	tests := []struct {
		name string
		exp  experiments
	}{
		{
			name: "default",
		},
		{
			name: "debug API",
			exp: experiments{
				debugAPI: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := StartOpts{
				Config: *conf,
			}
			require.NoError(t, fx.ValidateApp(options(t.Context(), opts, tt.exp)))
		})
	}
}
