package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/require"
)

func TestResolveStateDirDefaultsToInngestConfigDir(t *testing.T) {
	dir, err := ResolveStateDir("")
	require.NoError(t, err)
	require.Equal(t, consts.DefaultInngestConfigDir, dir)
}

func TestResolveStateDirKeepsAbsolutePathsAsIs(t *testing.T) {
	abs := t.TempDir()
	dir, err := ResolveStateDir(abs)
	require.NoError(t, err)
	require.Equal(t, abs, dir)
}

func TestResolveStateDirMakesRelativePathsAbsolute(t *testing.T) {
	t.Chdir(t.TempDir())

	// Compare against os.Getwd rather than the TempDir path directly: on macOS
	// t.TempDir() lives under a symlinked /var, so the two spellings of the
	// same directory would not compare equal.
	wd, err := os.Getwd()
	require.NoError(t, err)

	dir, err := ResolveStateDir("some/state")
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(dir), "expected an absolute path, got %q", dir)
	require.Equal(t, filepath.Join(wd, "some", "state"), dir)
}
