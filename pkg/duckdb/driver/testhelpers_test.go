package driver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindWorkerBuildDir(t *testing.T) {
	t.Run("prefers source repo root when cmd/inngest-duckdb-worker exists", func(t *testing.T) {
		root := t.TempDir()
		srcFile := filepath.Join(root, "pkg", "duckdb", "driver", "testhelpers.go")
		require.NoError(t, os.MkdirAll(filepath.Dir(srcFile), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(root, "cmd", "inngest-duckdb-worker"), 0o755))

		got := findWorkerBuildDir(srcFile, t.TempDir())
		require.Equal(t, root, got)
	})

	t.Run("falls back to cwd module root when vendored without cmd/inngest-duckdb-worker", func(t *testing.T) {
		outerRepo := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(outerRepo, "go.mod"), []byte("module example.com/outer\n"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(outerRepo, "cmd", "inngest-duckdb-worker"), 0o755))

		vendoredSrc := filepath.Join(outerRepo, "vendor", "github.com", "inngest", "inngest", "pkg", "duckdb", "driver", "testhelpers.go")
		require.NoError(t, os.MkdirAll(filepath.Dir(vendoredSrc), 0o755))

		subPkg := filepath.Join(outerRepo, "pkg", "lake")
		require.NoError(t, os.MkdirAll(subPkg, 0o755))

		got := findWorkerBuildDir(vendoredSrc, subPkg)
		require.Equal(t, outerRepo, got)
	})

	t.Run("returns empty when neither source root nor cwd module has worker cmd", func(t *testing.T) {
		outerRepo := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(outerRepo, "go.mod"), []byte("module example.com/outer\n"), 0o644))

		vendoredSrc := filepath.Join(outerRepo, "vendor", "github.com", "inngest", "inngest", "pkg", "duckdb", "driver", "testhelpers.go")
		require.NoError(t, os.MkdirAll(filepath.Dir(vendoredSrc), 0o755))

		got := findWorkerBuildDir(vendoredSrc, outerRepo)
		require.Equal(t, "", got)
	})
}

func TestRequireDuckDBBinaryUsesEnvOverride(t *testing.T) {
	t.Setenv("DUCKDB_LIB_PATH", "/fake/libduckdb.so")
	t.Setenv("DUCKDB_BINARY", "/custom/inngest-duckdb-worker")

	require.Equal(t, "/custom/inngest-duckdb-worker", RequireDuckDBBinary(t))
}
