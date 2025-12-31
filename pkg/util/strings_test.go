package util

import (
	"strconv"
	"testing"
)

func TestSanitizeLogField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes carriage returns",
			input:    "hello\rworld",
			expected: "helloworld",
		},
		{
			name:     "removes newlines",
			input:    "hello\nworld",
			expected: "helloworld",
		},
		{
			name:     "removes both carriage returns and newlines",
			input:    "hello\r\nworld",
			expected: "helloworld",
		},
		{
			name:     "preserves other special characters",
			input:    "user@domain.com/path-name_file[0]%20",
			expected: "user@domain.com/path-name_file[0]%20",
		},
		{
			name:     "string without special characters",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeLogField(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeLogField(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStringToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		hasError bool
	}{
		// Integer strings
		{"simple integer", "5", 5, false},
		{"negative integer", "-10", -10, false},
		{"zero", "0", 0, false},
		{"large integer", "1234567890", 1234567890, false},

		// Float strings (should truncate)
		{"simple float", "5.0", 5, false},
		{"negative float", "-10.0", -10, false},
		{"float with decimals", "5.7", 5, false},
		{"negative float with decimals", "-10.3", -10, false},
		{"zero float", "0.0", 0, false},
		{"longer float", "123.456789", 123, false},
		{"very long float", "123.456789012345", 123, false},

		// Redis/Garnet specific cases
		{"redis integer enum", "5", 5, false},
		{"garnet float enum", "5.0", 5, false},
		{"large enum value", "100", 100, false},
		{"large enum float", "100.0", 100, false},

		// Edge cases
		{"large float", "999999.123", 999999, false},
		{"scientific notation", "1e3", 1000, false},

		// Error cases
		{"empty string", "", 0, true},
		{"non-numeric", "abc", 0, true},
		{"mixed alphanumeric", "5a", 0, true},
		{"multiple dots", "5.0.1", 0, true},
		{"infinity", "inf", 0, true},
		{"NaN", "NaN", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToInt[int](tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("StringToInt(%q) expected error, got result %d", tt.input, result)
				}
				return
			}

			if err != nil {
				t.Errorf("StringToInt(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("StringToInt(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

// BenchmarkStringToInt benchmarks our utility function
func BenchmarkStringToInt(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"integer", "5"},
		{"float", "5.0"},
		{"longer_float", "123.456789"},
		{"very_long_float", "123.456789012345"},
		{"large_integer", "1234567890"},
		{"large_float", "1234567890.0"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_, _ = StringToInt[int](tc.input)
			}
		})
	}
}

// BenchmarkStringToIntVsStrconvAtoi compares our function with direct strconv.Atoi
func BenchmarkStringToIntVsStrconvAtoi(b *testing.B) {
	testInputs := []string{"5", "5.0", "123.456789"}

	b.Run("StringToInt", func(b *testing.B) {
		for b.Loop() {
			for _, input := range testInputs {
				_, _ = StringToInt[int](input)
			}
		}
	})

	b.Run("strconv.Atoi", func(b *testing.B) {
		for b.Loop() {
			for _, input := range testInputs {
				_, _ = strconv.Atoi(input)
			}
		}
	})
}

// BenchmarkParsingStrategies compares different parsing approaches
func BenchmarkParsingStrategies(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"integer", "5"},
		{"float", "5.0"},
		{"longer_float", "123.456789"},
		{"very_long_float", "123.456789012345"},
	}

	for _, tc := range testCases {
		b.Run(tc.name+"_StringToInt", func(b *testing.B) {
			for b.Loop() {
				_, _ = StringToInt[int](tc.input)
			}
		})

		b.Run(tc.name+"_Atoi", func(b *testing.B) {
			for b.Loop() {
				_, _ = strconv.Atoi(tc.input)
			}
		})

		b.Run(tc.name+"_ParseInt", func(b *testing.B) {
			for b.Loop() {
				_, _ = strconv.ParseInt(tc.input, 10, 64)
			}
		})

		b.Run(tc.name+"_ParseFloat", func(b *testing.B) {
			for b.Loop() {
				if f, err := strconv.ParseFloat(tc.input, 64); err == nil {
					_ = int(f)
				}
			}
		})
	}
}

// BenchmarkRedisGarnetScenario benchmarks the specific Redis/Garnet use case
func BenchmarkRedisGarnetScenario(b *testing.B) {
	// Simulate the Redis vs Garnet cjson behavior difference
	redisValues := []string{"5", "10", "100", "1000"}          // Redis stores as integers
	garnetValues := []string{"5.0", "10.0", "100.0", "1000.0"} // Garnet stores as floats

	b.Run("Redis_Integer_Values", func(b *testing.B) {
		for b.Loop() {
			for _, val := range redisValues {
				_, _ = StringToInt[int](val)
			}
		}
	})

	b.Run("Garnet_Float_Values", func(b *testing.B) {
		for b.Loop() {
			for _, val := range garnetValues {
				_, _ = StringToInt[int](val)
			}
		}
	})

	b.Run("Mixed_Redis_Garnet", func(b *testing.B) {
		allValues := append(redisValues, garnetValues...)
		for b.Loop() {
			for _, val := range allValues {
				_, _ = StringToInt[int](val)
			}
		}
	})
}
