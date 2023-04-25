package inngest

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalRuntimeWrapper(t *testing.T) {
	inputs := []struct {
		name     string
		r        RuntimeWrapper
		expected []byte
	}{
		{
			name: "docker",
			r: RuntimeWrapper{
				Runtime: RuntimeDocker{
					Entrypoint: []string{"main", "--json"},
				},
			},
			expected: []byte(`{"entrypoint":["main","--json"],"type":"docker"}`),
		},
		{
			name: "http",
			r: RuntimeWrapper{
				Runtime: RuntimeHTTP{
					URL: "http://www.this-is-a-really-good-domain-name-buy-now.com",
				},
			},
			expected: []byte(`{"type":"http","url":"http://www.this-is-a-really-good-domain-name-buy-now.com"}`),
		},
	}

	for _, test := range inputs {
		t.Run(test.name, func(t *testing.T) {
			output, err := json.Marshal(test.r)
			require.NoError(t, err)
			require.Equal(t, test.expected, output)
		})
	}
}
