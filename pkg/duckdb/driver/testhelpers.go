package driver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

var (
	testWorkerOnce sync.Once
	testWorkerPath string
	testWorkerSkip string
	testWorkerErr  error
)

// RequireDuckDBBinary ensures libduckdb and an inngest-duckdb-worker binary are
// available for tests, skipping the calling test if libduckdb cannot be found
// locally or if no worker binary or buildable source tree is available.
// Resolution order: DUCKDB_BINARY env var -> inngest-duckdb-worker on PATH ->
// go build ./cmd/inngest-duckdb-worker from this repo or the current module
// root (CGO_ENABLED=0), cached once per test process in os.TempDir().
func RequireDuckDBBinary(t testing.TB) string {
	t.Helper()

	if os.Getenv("DUCKDB_LIB_PATH") == "" {
		const fallback = "/tmp/duckdb-purego-test/libduckdb.dylib"
		if _, err := os.Stat(fallback); err == nil {
			_ = os.Setenv("DUCKDB_LIB_PATH", fallback)
		} else {
			t.Skip("libduckdb not found (set DUCKDB_LIB_PATH to run DuckDB worker tests)")
		}
	}

	if bin := os.Getenv("DUCKDB_BINARY"); bin != "" {
		return bin
	}
	if bin, err := exec.LookPath("inngest-duckdb-worker"); err == nil {
		return bin
	}

	testWorkerOnce.Do(func() {
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			thisFile = ""
		}
		cwd, _ := os.Getwd()
		buildDir := findWorkerBuildDir(thisFile, cwd)
		if buildDir == "" {
			testWorkerSkip = "inngest-duckdb-worker binary not found (set DUCKDB_BINARY, put inngest-duckdb-worker on PATH, or run from a repo with cmd/inngest-duckdb-worker)"
			return
		}
		out := filepath.Join(os.TempDir(), fmt.Sprintf("inngest-duckdb-worker-test-%d", os.Getpid()))
		if runtime.GOOS == "windows" {
			out += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", out, "./cmd/inngest-duckdb-worker")
		cmd.Dir = buildDir
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if output, err := cmd.CombinedOutput(); err != nil {
			testWorkerErr = fmt.Errorf("building inngest-duckdb-worker: %w\n%s", err, string(output))
			return
		}
		testWorkerPath = out
	})

	if testWorkerSkip != "" {
		t.Skip(testWorkerSkip)
	}
	if testWorkerErr != nil {
		t.Fatalf("RequireDuckDBBinary: %v", testWorkerErr)
	}
	return testWorkerPath
}

func findWorkerBuildDir(sourceFile, cwd string) string {
	const workerPkgRel = "cmd/inngest-duckdb-worker"
	if sourceFile != "" {
		repoRoot := filepath.Join(filepath.Dir(sourceFile), "..", "..", "..")
		if isDir(filepath.Join(repoRoot, filepath.FromSlash(workerPkgRel))) {
			return repoRoot
		}
	}
	if cwd != "" {
		for dir := cwd; ; dir = filepath.Dir(dir) {
			if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
				if isDir(filepath.Join(dir, filepath.FromSlash(workerPkgRel))) {
					return dir
				}
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
	return ""
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
