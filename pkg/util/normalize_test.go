package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAppURL(t *testing.T) {
	tests := []struct {
		name        string
		inputURL    string
		expectedURL string
		forceHTTPS  bool
	}{
		{
			name:        "valid URI should return without modification",
			inputURL:    "https://api.example.com/api/inngest?fnId=hello&step=step",
			expectedURL: "https://api.example.com/api/inngest?fnId=hello&step=step",
		},
		{
			name:        "localhost related identifier should translate to 'localhost'",
			inputURL:    "http://127.0.0.1:3000/api/inngest?fnId=hello&step=step",
			expectedURL: "http://localhost:3000/api/inngest?fnId=hello&step=step",
		},
		{
			name:        "force https",
			inputURL:    "http://api.example.com/api/inngest",
			expectedURL: "https://api.example.com/api/inngest",
			forceHTTPS:  true,
		},
		{
			name:        "strip deployId query param",
			inputURL:    "https://api.example.com/api/inngest?deployId=1234",
			expectedURL: "https://api.example.com/api/inngest",
		},
		{
			name:        "insecure WebSocket URL should be normalized to wss",
			inputURL:    "ws://api.example.com/api/inngest",
			expectedURL: "wss://api.example.com/api/inngest",
			forceHTTPS:  true,
		},
		{
			name:        "secure WebSocket URL should stay the same with force",
			inputURL:    "wss://api.example.com/api/inngest",
			expectedURL: "wss://api.example.com/api/inngest",
			forceHTTPS:  true,
		},
		{
			name:        "insecure WebSocket URL should stay the same without force",
			inputURL:    "ws://api.example.com/api/inngest",
			expectedURL: "ws://api.example.com/api/inngest",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NormalizeAppURL(test.inputURL, test.forceHTTPS)
			require.Equal(t, test.expectedURL, result)
		})
	}
}
