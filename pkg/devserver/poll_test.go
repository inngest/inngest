package devserver

import "testing"

func TestShouldPollSDK(t *testing.T) {
	tests := []struct {
		name      string
		poll      bool
		appError  string
		firstSync bool
		want      bool
	}{
		{name: "no-poll healthy app still syncs once for -u", poll: false, firstSync: true, want: true},
		{name: "no-poll healthy app skipped after that sync", poll: false, want: false},
		{name: "no-poll app with an error still syncs", poll: false, appError: "unreachable", want: true},
		{name: "poll syncs a healthy app", poll: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldPollSDK(tt.poll, tt.appError, tt.firstSync)
			if got != tt.want {
				t.Fatalf("shouldPollSDK(%v, %q, %v) = %v, want %v", tt.poll, tt.appError, tt.firstSync, got, tt.want)
			}
		})
	}
}
