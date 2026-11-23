package worker

import (
	"slices"
	"testing"
)

func TestMinimalChildEnv(t *testing.T) {
	t.Setenv("LOCALAPPDATA", `C:\Users\test\AppData\Local`)
	t.Setenv("LocalAppData", `C:\Users\test\AppData\LocalMixed`)
	t.Setenv("SUPER_SECRET_TOKEN", "do-not-leak")

	env := minimalChildEnv([]string{"EXTRA_VAR=1"})

	if !slices.Contains(env, `LOCALAPPDATA=C:\Users\test\AppData\Local`) &&
		!slices.Contains(env, `LocalAppData=C:\Users\test\AppData\LocalMixed`) {
		t.Fatalf("expected LOCALAPPDATA/LocalAppData to be preserved, got %v", env)
	}
	if slices.Contains(env, "SUPER_SECRET_TOKEN=do-not-leak") {
		t.Fatalf("expected secret env var to be dropped, got %v", env)
	}
	if !slices.Contains(env, "EXTRA_VAR=1") {
		t.Fatalf("expected extra env var to be appended, got %v", env)
	}
}
