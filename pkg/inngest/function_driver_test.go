package inngest

import (
	"net/url"
	"testing"

	"github.com/inngest/inngest/pkg/execution/driver/schemes"
	"github.com/stretchr/testify/require"
)

func init() {
	schemes.Register("test", func(u *url.URL) string {
		return "test-" + u.Host
	})
}

func TestDriver(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		metadata map[string]any
		expected string
	}{
		{
			name:     "http",
			uri:      "http://example.com/api/inngest?fnId=x&step=step",
			expected: "http",
		},
		{
			name:     "https",
			uri:      "https://example.com/api/inngest?fnId=x&step=step",
			expected: "http",
		},
		{
			name: "https sync",
			uri:  "https://example.com/api/inngest?fnId=x&step=step",
			metadata: map[string]any{
				"type": "sync",
			},
			expected: "httpv2",
		},
		{
			name:     "ws",
			uri:      "ws://connect.inngest.com?fnId=x&step=step",
			expected: "connect",
		},
		{
			name:     "wss",
			uri:      "wss://connect.inngest.com?fnId=x&step=step",
			expected: "connect",
		},
		{
			name:     "registered scheme",
			uri:      "test://a?fnId=x&step=step",
			expected: "test-a",
		},
		{
			name:     "unregistered scheme",
			uri:      "other://a?fnId=x&step=step",
			expected: "http",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fn := Function{
				Driver: FunctionDriver{
					URI:      test.uri,
					Metadata: test.metadata,
				},
			}
			require.Equal(t, test.expected, Driver(fn))
		})
	}
}
