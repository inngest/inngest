package update

import "testing"

func TestClassifyPath(t *testing.T) {
	cases := []struct {
		path string
		want Method
	}{
		{"/Users/alice/proj/node_modules/inngest-cli/bin/inngest", MethodNPM},
		{"/Users/alice/.npm/_npx/abc/node_modules/.bin/inngest", MethodNPM},
		{"/opt/homebrew/bin/inngest", MethodHomebrew},
		{"/opt/homebrew/Cellar/inngest/0.35.0/bin/inngest", MethodHomebrew},
		{"/usr/local/Cellar/inngest/0.35.0/bin/inngest", MethodHomebrew},
		{"/home/linuxbrew/.linuxbrew/bin/inngest", MethodHomebrew},
		{"/usr/local/bin/inngest", MethodBash},
		{"/usr/bin/inngest", MethodBash},
		{"/Users/alice/Downloads/inngest", MethodBinary},
		{"/tmp/inngest", MethodBinary},
	}
	for _, tc := range cases {
		if got := classifyPath(tc.path); got != tc.want {
			t.Errorf("classifyPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestNormalizeMethod(t *testing.T) {
	cases := []struct {
		in   string
		want Method
	}{
		{"npm", MethodNPM},
		{"NPX", MethodNPM},
		{" Homebrew ", MethodHomebrew},
		{"brew", MethodHomebrew},
		{"bash", MethodBash},
		{"install.sh", MethodBash},
		{"binary", MethodBinary},
		{"", Method("")},
		{"unknown", Method("")},
	}
	for _, tc := range cases {
		if got := normalize(tc.in); got != tc.want {
			t.Errorf("normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestUpgradeCommand(t *testing.T) {
	cases := []struct {
		m    Method
		want string
	}{
		{MethodNPM, "npm install -g inngest-cli@latest"},
		{MethodHomebrew, "brew upgrade inngest"},
		{MethodBash, "curl -fsSL https://cli.inngest.com/install.sh | sh"},
		{MethodBinary, "curl -fsSL https://cli.inngest.com/install.sh | sh"},
	}
	for _, tc := range cases {
		if got := UpgradeCommand(tc.m); got != tc.want {
			t.Errorf("UpgradeCommand(%q) = %q, want %q", tc.m, got, tc.want)
		}
	}
}

func TestDetectEnvOverride(t *testing.T) {
	t.Setenv(EnvInstallMethod, "homebrew")
	if got := Detect(); got != MethodHomebrew {
		t.Errorf("Detect() with INNGEST_INSTALL_METHOD=homebrew = %q, want %q", got, MethodHomebrew)
	}
}
