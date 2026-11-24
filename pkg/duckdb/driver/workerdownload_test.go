package driver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func testWorkerReleaseServer(t *testing.T, files map[string][]byte) string {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	for name, data := range files {
		name := name
		data := data
		mux.HandleFunc("/"+name, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(data)
		})
	}
	return server.URL + "/"
}

func TestWorkerBinaryAssetNameForPlatform(t *testing.T) {
	cases := []struct {
		ver, goos, goarch string
		want              string
		wantErr           bool
	}{
		{"v1.2.3", "darwin", "arm64", "inngest-duckdb-worker_1.2.3_darwin_arm64.tar.gz", false},
		{"1.2.3", "linux", "amd64", "inngest-duckdb-worker_1.2.3_linux_amd64.tar.gz", false},
		{"1.2.3", "linux", "arm64", "inngest-duckdb-worker_1.2.3_linux_arm64.tar.gz", false},
		{"1.2.3", "windows", "amd64", "inngest-duckdb-worker_1.2.3_windows_amd64.zip", false},
		{"1.2.3", "windows", "arm64", "inngest-duckdb-worker_1.2.3_windows_arm64.zip", false},
		{"1.2.3", "freebsd", "amd64", "", true},
		{"1.2.3", "linux", "386", "", true},
	}
	for _, tc := range cases {
		got, err := workerAssetNameFor(tc.ver, tc.goos, tc.goarch)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("workerAssetNameFor(%q,%q,%q) = %q, want error", tc.ver, tc.goos, tc.goarch, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("workerAssetNameFor(%q,%q,%q) unexpected error: %v", tc.ver, tc.goos, tc.goarch, err)
		}
		if got != tc.want {
			t.Fatalf("workerAssetNameFor(%q,%q,%q) = %q, want %q", tc.ver, tc.goos, tc.goarch, got, tc.want)
		}
	}
}

func TestEnsureWorkerBinaryCachesWithoutNetwork(t *testing.T) {
	stateDir := t.TempDir()
	path := WorkerBinaryPath(stateDir, "v1.5.0")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("cached-worker"), 0o755); err != nil {
		t.Fatalf("write cached worker: %v", err)
	}

	// Unreachable URL proves cache short-circuits before any network request.
	got, err := ensureWorkerBinary(context.Background(), stateDir, "v1.5.0", "http://127.0.0.1:1/")
	if err != nil {
		t.Fatalf("ensureWorkerBinary cached: %v", err)
	}
	if got != path {
		t.Fatalf("got %q, want %q", got, path)
	}

	gotPublic, err := EnsureWorkerBinary(context.Background(), stateDir, "1.5.0")
	if err != nil {
		t.Fatalf("EnsureWorkerBinary cached: %v", err)
	}
	if gotPublic != path {
		t.Fatalf("EnsureWorkerBinary got %q, want %q", gotPublic, path)
	}
}

func TestEnsureWorkerBinaryDownloadsVerifiesAndExtracts(t *testing.T) {
	const ver = "1.9.0"
	assetName, err := workerAssetNameFor(ver, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}

	for _, tc := range []struct {
		name    string
		archive func(want []byte) []byte
	}{
		{
			name: "targz",
			archive: func(want []byte) []byte {
				return buildFakeArchive(t, workerBinaryName(), want)
			},
		},
		{
			name: "zip",
			archive: func(want []byte) []byte {
				return buildFakeZipArchive(t, map[string][]byte{
					workerBinaryName(): want,
					"LICENSE.md":       []byte("MIT"),
				})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := []byte("#!/bin/sh\necho fake worker " + tc.name + "\n")
			archive := tc.archive(want)
			sum := sha256.Sum256(archive)
			checksumsTxt := fmt.Sprintf(
				"0000000000000000000000000000000000000000000000000000000000000000  other_file.tar.gz\n%s  %s\n",
				hex.EncodeToString(sum[:]),
				assetName,
			)

			baseURL := testWorkerReleaseServer(t, map[string][]byte{
				"checksums.txt": []byte(checksumsTxt),
				assetName:       archive,
			})

			stateDir := t.TempDir()
			path, err := ensureWorkerBinary(context.Background(), stateDir, "v"+ver, baseURL)
			if err != nil {
				t.Fatalf("ensureWorkerBinary: %v", err)
			}
			if path != WorkerBinaryPath(stateDir, ver) {
				t.Fatalf("got path %q, want %q", path, WorkerBinaryPath(stateDir, ver))
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read installed worker: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("installed contents = %q, want %q", got, want)
			}

			// Second call must hit cache without network.
			path2, err := ensureWorkerBinary(context.Background(), stateDir, ver, "http://127.0.0.1:1/")
			if err != nil {
				t.Fatalf("cached call: %v", err)
			}
			if path2 != path {
				t.Fatalf("cached path = %q, want %q", path2, path)
			}
		})
	}
}

func TestEnsureWorkerBinaryChecksumMismatchFailsClosed(t *testing.T) {
	const ver = "1.9.1"
	assetName, err := workerAssetNameFor(ver, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}

	archive := buildFakeArchive(t, workerBinaryName(), []byte("fake"))
	badChecksums := fmt.Sprintf("0000000000000000000000000000000000000000000000000000000000000000  %s\n", assetName)

	baseURL := testWorkerReleaseServer(t, map[string][]byte{
		"checksums.txt": []byte(badChecksums),
		assetName:       archive,
	})

	stateDir := t.TempDir()
	if _, err := ensureWorkerBinary(context.Background(), stateDir, ver, baseURL); err == nil {
		t.Fatal("expected checksum mismatch to fail closed")
	}
	if _, err := os.Stat(WorkerBinaryPath(stateDir, ver)); !os.IsNotExist(err) {
		t.Fatalf("expected no binary installed after mismatch, stat err: %v", err)
	}
}

func TestEnsureWorkerBinaryMissingChecksumEntryFailsClosed(t *testing.T) {
	const ver = "1.9.2"
	assetName, err := workerAssetNameFor(ver, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}

	archive := buildFakeArchive(t, workerBinaryName(), []byte("fake"))
	baseURL := testWorkerReleaseServer(t, map[string][]byte{
		"checksums.txt": []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  unrelated.tar.gz\n"),
		assetName:       archive,
	})

	stateDir := t.TempDir()
	if _, err := ensureWorkerBinary(context.Background(), stateDir, ver, baseURL); err == nil {
		t.Fatal("expected missing checksum entry to fail closed")
	}
}

func TestEnsureWorkerBinaryDevModePathLookup(t *testing.T) {
	fakeBinDir := t.TempDir()
	fakeWorker := filepath.Join(fakeBinDir, workerBinaryName())
	if err := os.WriteFile(fakeWorker, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake worker: %v", err)
	}

	t.Setenv("PATH", fakeBinDir)

	for _, ver := range []string{"", "dev", "vdev"} {
		got, err := EnsureWorkerBinary(context.Background(), t.TempDir(), ver)
		if err != nil {
			t.Fatalf("EnsureWorkerBinary(ver=%q): %v", ver, err)
		}
		if got != fakeWorker {
			t.Fatalf("EnsureWorkerBinary(ver=%q) = %q, want %q", ver, got, fakeWorker)
		}
	}

	// Empty PATH should return a clear error.
	t.Setenv("PATH", t.TempDir())
	if _, err := EnsureWorkerBinary(context.Background(), t.TempDir(), "dev"); err == nil {
		t.Fatal("expected error when worker binary is not on PATH in dev mode")
	}
}

func TestEnsureWorkerBinaryConcurrentDownloadSafety(t *testing.T) {
	const ver = "1.9.3"
	assetName, err := workerAssetNameFor(ver, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}

	want := bytes.Repeat([]byte("worker-binary-bytes-"), 1024)
	archive := buildFakeArchive(t, workerBinaryName(), want)
	sum := sha256.Sum256(archive)
	checksumsTxt := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), assetName)

	baseURL := testWorkerReleaseServer(t, map[string][]byte{
		"checksums.txt": []byte(checksumsTxt),
		assetName:       archive,
	})

	stateDir := t.TempDir()
	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	paths := make(chan string, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := ensureWorkerBinary(context.Background(), stateDir, ver, baseURL)
			if err != nil {
				errs <- err
				return
			}
			paths <- p
		}()
	}
	wg.Wait()
	close(errs)
	close(paths)

	for err := range errs {
		t.Fatalf("concurrent ensureWorkerBinary failed: %v", err)
	}
	wantPath := WorkerBinaryPath(stateDir, ver)
	for p := range paths {
		if p != wantPath {
			t.Fatalf("got path %q, want %q", p, wantPath)
		}
	}

	got, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read installed worker: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("installed worker binary corrupted under concurrent download")
	}

	entries, err := os.ReadDir(filepath.Dir(wantPath))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != workerBinaryName() {
		t.Fatalf("unexpected files left in worker bin dir: %v", entries)
	}
}
