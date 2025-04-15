package httpdriver

import (
	"fmt"
	"testing"
)

func TestIsDNSLookupTimeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "real dns lookup timeout error",
			err:      fmt.Errorf(`Post "http://na-ashburn.sdkgateway.infra.inngest.lol:8080/request": lookup na-ashburn.sdkgateway.infra.inngest.lol on [2607:e3c0:a040:f100::a]:53: read udp [2607:e3c0:a040:f00e::a8bc]:49196->[2607:e3c0:a040:f100::a]:53: i/o timeout`),
			expected: true,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "different error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDNSLookupTimeout(tt.err)
			if result != tt.expected {
				t.Errorf("IsDNSLookupTimeout() = %v, want %v", result, tt.expected)
			}
		})
	}
}
