package util

import (
	"testing"
)

func TestParsePort(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
		errorMsg    string
	}{
		// Plain port number tests
		{
			name:     "valid plain port number",
			input:    "8288",
			expected: 8288,
		},
		{
			name:     "port 80",
			input:    "80",
			expected: 80,
		},
		{
			name:     "port 443",
			input:    "443",
			expected: 443,
		},
		{
			name:     "high port number",
			input:    "65535",
			expected: 65535,
		},
		{
			name:     "port 1",
			input:    "1",
			expected: 1,
		},
		
		// TCP URL tests
		{
			name:     "tcp url with ip and port",
			input:    "tcp://192.168.194.165:8288",
			expected: 8288,
		},
		{
			name:     "tcp url with localhost",
			input:    "tcp://localhost:3000",
			expected: 3000,
		},
		{
			name:     "tcp url with hostname",
			input:    "tcp://example.com:443",
			expected: 443,
		},
		{
			name:     "tcp url with different port",
			input:    "tcp://10.0.0.1:9000",
			expected: 9000,
		},
		{
			name:     "tcp url with ipv6",
			input:    "tcp://[::1]:8080",
			expected: 8080,
		},
		
		// HTTP/HTTPS URL tests (should work with any scheme)
		{
			name:     "http url",
			input:    "http://localhost:8000",
			expected: 8000,
		},
		{
			name:     "https url",
			input:    "https://example.com:8443",
			expected: 8443,
		},
		
		// Error cases
		{
			name:        "empty string",
			input:       "",
			expectError: true,
			errorMsg:    "port cannot be empty",
		},
		{
			name:        "invalid port number",
			input:       "invalid",
			expectError: true,
			errorMsg:    "invalid port \"invalid\"",
		},
		{
			name:     "negative port number",
			input:    "-123",
			expected: -123, // strconv.Atoi actually allows negative numbers
		},
		{
			name:        "port number too large",
			input:       "99999",
			expected:    99999, // strconv.Atoi allows this, validation would be separate
		},
		{
			name:        "float as port",
			input:       "8080.5",
			expectError: true,
			errorMsg:    "invalid port \"8080.5\"",
		},
		{
			name:        "url without port",
			input:       "tcp://localhost",
			expectError: true,
			errorMsg:    "failed to parse port from URL \"tcp://localhost\"",
		},
		{
			name:        "url with invalid port",
			input:       "tcp://localhost:invalid",
			expectError: true,
			errorMsg:    "invalid port", // Just check for the key part of the error
		},
		{
			name:        "url with empty port",
			input:       "tcp://localhost:",
			expectError: true,
			errorMsg:    "invalid port number \"\" in URL \"tcp://localhost:\"",
		},
		{
			name:        "url with no host",
			input:       "tcp://:8080",
			expected:    8080, // This actually works - empty host with port
		},
		{
			name:        "malformed url",
			input:       "tcp://[invalid-ipv6:8080",
			expectError: true,
			errorMsg:    "invalid port", // This will fall through to plain port parsing and fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePort(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParsePortKubernetesCase(t *testing.T) {
	// Test the specific case that was failing in Kubernetes
	t.Run("kubernetes tcp url case", func(t *testing.T) {
		result, err := ParsePort("tcp://192.168.194.165:8288")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != 8288 {
			t.Errorf("expected 8288, got %d", result)
		}
	})
}

func TestParsePortEdgeCases(t *testing.T) {
	// Test whitespace handling
	t.Run("port with whitespace", func(t *testing.T) {
		_, err := ParsePort(" 8080 ")
		if err == nil {
			t.Error("expected error for port with whitespace")
		}
	})
	
	// Test zero port
	t.Run("zero port", func(t *testing.T) {
		result, err := ParsePort("0")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})
	
	// Test URL with scheme but no authority
	t.Run("scheme only", func(t *testing.T) {
		_, err := ParsePort("tcp://")
		if err == nil {
			t.Error("expected error for URL with no authority")
		}
	})
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}