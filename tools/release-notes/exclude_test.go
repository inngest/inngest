package main

import "testing"

func TestParseCliffExcludePaths(t *testing.T) {
	config := `
exclude_paths = [
  # Cloud dashboard
  "ui/apps/dashboard/",
  "pkg/debugapi/", # inline comment
  # "pkg/constraintapi/",
  "cmd/debug/"
]
`

	got := ParseCliffExcludePaths(config)
	want := []string{"ui/apps/dashboard/", "pkg/debugapi/", "cmd/debug/"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("path[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestShouldExcludePR(t *testing.T) {
	excludes := []string{"ui/apps/dashboard/", "pkg/debugapi/"}

	tests := []struct {
		name string
		pr   PullRequest
		want bool
	}{
		{
			name: "all paths excluded",
			pr: PullRequest{
				Title: "feat: dashboard tweak",
				Files: []string{"ui/apps/dashboard/page.tsx", "pkg/debugapi/api.go"},
			},
			want: true,
		},
		{
			name: "mixed paths included",
			pr: PullRequest{
				Title: "feat: runtime tweak",
				Files: []string{"ui/apps/dashboard/page.tsx", "pkg/execution/run.go"},
			},
			want: false,
		},
		{
			name: "non release prefix",
			pr: PullRequest{
				Title: "internal: update tooling",
				Files: []string{"pkg/execution/run.go"},
			},
			want: true,
		},
		{
			name: "scoped non release prefix",
			pr: PullRequest{
				Title: "cloud(dashboard): update card",
				Files: []string{"pkg/execution/run.go"},
			},
			want: true,
		},
		{
			name: "similar prefix is not excluded",
			pr: PullRequest{
				Title: "internalize: expose helper",
				Files: []string{"pkg/execution/run.go"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldExcludePR(tt.pr, excludes); got != tt.want {
				t.Fatalf("ShouldExcludePR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathExcluded(t *testing.T) {
	excludes := []string{"pkg/debugapi/", "cmd/debug", "ui/apps/dashboard/"}

	tests := []struct {
		path string
		want bool
	}{
		{path: "pkg/debugapi/api.go", want: true},
		{path: "./pkg/debugapi/api.go", want: true},
		{path: "pkg/debugapi_extra/api.go", want: false},
		{path: "cmd/debug", want: true},
		{path: "cmd/debug/main.go", want: true},
		{path: "cmd/debugger/main.go", want: false},
		{path: "ui/apps/dashboard", want: false},
		{path: "ui/apps/dashboard/page.tsx", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := PathExcluded(tt.path, excludes); got != tt.want {
				t.Fatalf("PathExcluded(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
