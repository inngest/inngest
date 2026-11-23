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
	testWorkerErr  error
)

// RequireDuckDBBinary ensures libduckdb and a freshly built
// inngest-duckdb-worker binary are available for tests, skipping the calling
// test if libduckdb cannot be found locally. The worker binary is built once
// per test process (CGO_ENABLED=0) and cached in os.TempDir().
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

	testWorkerOnce.Do(func() {
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			testWorkerErr = fmt.Errorf("cannot determine driver source path")
			return
		}
		repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
		out := filepath.Join(os.TempDir(), fmt.Sprintf("inngest-duckdb-worker-test-%d", os.Getpid()))
		if runtime.GOOS == "windows" {
			out += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", out, "./cmd/inngest-duckdb-worker")
		cmd.Dir = repoRoot
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
		if output, err := cmd.CombinedOutput(); err != nil {
			testWorkerErr = fmt.Errorf("building inngest-duckdb-worker: %w\n%s", err, string(output))
			return
		}
		testWorkerPath = out
	})

	if testWorkerErr != nil {
		t.Fatalf("RequireDuckDBBinary: %v", testWorkerErr)
	}
	return testWorkerPath
}
