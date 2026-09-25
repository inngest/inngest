package trace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The final case is a regression: URLPath used to return the empty string when
// neither the struct field nor the environment supplied a path.
func TestTracerOptsURLPath(t *testing.T) {
	const envURLPath = "OTEL_TRACE_COLLECTOR_URL_PATH"

	tests := []struct {
		name     string
		opts     TracerOpts
		env      string
		expected string
	}{
		{
			name:     "struct field wins over the environment",
			opts:     TracerOpts{TraceURLPath: "/dev/traces"},
			env:      "/env/traces",
			expected: "/dev/traces",
		},
		{
			name:     "environment is used when the struct field is empty",
			env:      "/env/traces",
			expected: "/env/traces",
		},
		{
			name:     "neither configured resolves to the OTLP trace path",
			expected: "/v1/traces",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(envURLPath, test.env)

			require.Equal(t, test.expected, test.opts.URLPath())
		})
	}
}
