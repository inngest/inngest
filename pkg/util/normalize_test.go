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
			name:        "host should not have port when using https scheme",
			inputURL:    "https://api.example.com:80/api/inngest?fnId=hello&step=step",
			expectedURL: "https://api.example.com/api/inngest?fnId=hello&step=step",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NormalizeAppURL(test.inputURL)
			require.Equal(t, test.expectedURL, result)
		})
	}
}
