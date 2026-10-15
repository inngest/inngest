package main

import "testing"

func TestResolveLogHandler(t *testing.T) {
	tests := []struct {
		name      string
		jsonSet   bool
		jsonValue bool
		isTTY     bool
		current   string
		want      string
	}{
		{
			name:      "explicit --json=false in a non-TTY forces human-readable output",
			jsonSet:   true,
			jsonValue: false,
			isTTY:     false,
			current:   "",
			want:      "",
		},
		{
			name:      "explicit --json=false overrides a json LOG_HANDLER env var",
			jsonSet:   true,
			jsonValue: false,
			isTTY:     false,
			current:   "json",
			want:      "dev",
		},
		{
			name:      "explicit --json=false preserves an explicit text LOG_HANDLER env var",
			jsonSet:   true,
			jsonValue: false,
			isTTY:     false,
			current:   "text",
			want:      "",
		},
		{
			name:      "explicit --json=false in a TTY stays human-readable",
			jsonSet:   true,
			jsonValue: false,
			isTTY:     true,
			current:   "",
			want:      "",
		},
		{
			name:      "explicit --json in a TTY forces JSON",
			jsonSet:   true,
			jsonValue: true,
			isTTY:     true,
			current:   "",
			want:      "json",
		},
		{
			name:      "explicit --json in a non-TTY stays JSON",
			jsonSet:   true,
			jsonValue: true,
			isTTY:     false,
			current:   "",
			want:      "json",
		},
		{
			name:    "no --json in a non-TTY defaults to JSON",
			jsonSet: false,
			isTTY:   false,
			current: "",
			want:    "json",
		},
		{
			name:    "no --json in a TTY leaves the handler untouched",
			jsonSet: false,
			isTTY:   true,
			current: "",
			want:    "",
		},
		{
			name:    "no --json in a TTY preserves an existing env handler",
			jsonSet: false,
			isTTY:   true,
			current: "text",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveLogHandler(tt.jsonSet, tt.jsonValue, tt.isTTY, tt.current); got != tt.want {
				t.Errorf("resolveLogHandler(jsonSet=%v, jsonValue=%v, isTTY=%v, current=%q) = %q, want %q",
					tt.jsonSet, tt.jsonValue, tt.isTTY, tt.current, got, tt.want)
			}
		})
	}
}

func TestIsJSONHandler(t *testing.T) {
	tests := []struct {
		handler string
		want    bool
	}{
		{handler: "json", want: true},
		{handler: "JSON", want: true},
		{handler: "  json  ", want: true},
		{handler: "", want: false},
		{handler: "dev", want: false},
		{handler: "text", want: false},
		{handler: "txt", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.handler, func(t *testing.T) {
			if got := isJSONHandler(tt.handler); got != tt.want {
				t.Errorf("isJSONHandler(%q) = %v, want %v", tt.handler, got, tt.want)
			}
		})
	}
}
