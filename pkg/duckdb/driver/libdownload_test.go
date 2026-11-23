package driver

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func buildFakeArchive(t *testing.T, entryName string, contents []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	if err := tw.WriteHeader(&tar.Header{
		Name: entryName,
		Mode: 0o755,
		Size: int64(len(contents)),
	}); err != nil {
		t.Fatalf("writing tar header: %v", err)
	}
	if _, err := tw.Write(contents); err != nil {
		t.Fatalf("writing tar entry: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("closing tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("closing gzip writer: %v", err)
	}
	return buf.Bytes()
}

func testInstallServer(t *testing.T, assetName string, archive []byte) (baseURL string) {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/"+assetName, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	})

	return server.URL + "/"
}

func buildFakeZipArchive(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, contents := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("creating zip entry %q: %v", name, err)
		}
		if _, err := w.Write(contents); err != nil {
			t.Fatalf("writing zip entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestLibAssetNameForPlatform(t *testing.T) {
	cases := []struct {
		goos, goarch string
		want         string
		wantErr      bool
	}{
		{goos: "darwin", goarch: "arm64", want: "duckdb-shared-libs-osx-universal.tar.gz"},
		{goos: "darwin", goarch: "amd64", want: "duckdb-shared-libs-osx-universal.tar.gz"},
		{goos: "linux", goarch: "amd64", want: "duckdb-shared-libs-linux-amd64.tar.gz"},
		{goos: "linux", goarch: "arm64", want: "duckdb-shared-libs-linux-arm64.tar.gz"},
		{goos: "windows", goarch: "amd64", want: "duckdb-shared-libs-windows-amd64.tar.gz"},
		{goos: "linux", goarch: "386", wantErr: true},
		{goos: "freebsd", goarch: "amd64", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.goos+"/"+tc.goarch, func(t *testing.T) {
			got, err := libAssetNameFor(tc.goos, tc.goarch)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("libAssetNameFor(%q, %q) = %q, want error", tc.goos, tc.goarch, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("libAssetNameFor(%q, %q) unexpected error: %v", tc.goos, tc.goarch, err)
			}
			if got != tc.want {
				t.Fatalf("libAssetNameFor(%q, %q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
			}
		})
	}
}

func TestPinnedLibChecksumsCoverEveryKnownAsset(t *testing.T) {
	platforms := []struct{ goos, goarch string }{
		{"darwin", "arm64"},
		{"darwin", "amd64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"windows", "amd64"},
	}
	for _, p := range platforms {
		name, err := libAssetNameFor(p.goos, p.goarch)
		if err != nil {
			t.Fatalf("libAssetNameFor(%q, %q): %v", p.goos, p.goarch, err)
		}
		if _, ok := pinnedLibChecksums[name]; !ok {
			t.Errorf("no pinned checksum for library asset %q (GOOS=%s GOARCH=%s)", name, p.goos, p.goarch)
		}
	}
}

func TestExtractLibFromZipAndTarGz(t *testing.T) {
	want := []byte("fake-libduckdb-shared-library-bytes")

	t.Run("zip", func(t *testing.T) {
		archive := buildFakeZipArchive(t, map[string][]byte{
			"duckdb.h":    []byte("// header"),
			libraryName(): want,
		})
		got, err := extractLibrary(archive, libraryName())
		if err != nil {
			t.Fatalf("extractLibrary(zip): %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("zip missing entry", func(t *testing.T) {
		archive := buildFakeZipArchive(t, map[string][]byte{
			"duckdb.h": []byte("// header"),
		})
		if _, err := extractLibrary(archive, libraryName()); err == nil {
			t.Fatal("expected error when library entry is missing from zip")
		}
	})

	t.Run("targz", func(t *testing.T) {
		archive := buildFakeArchive(t, libraryName(), want)
		got, err := extractLibrary(archive, libraryName())
		if err != nil {
			t.Fatalf("extractLibrary(tar.gz): %v", err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("targz missing entry", func(t *testing.T) {
		archive := buildFakeArchive(t, "duckdb.h", []byte("// header"))
		if _, err := extractLibrary(archive, libraryName()); err == nil {
			t.Fatal("expected error when library entry is missing from tar.gz")
		}
	})

	t.Run("unknown format", func(t *testing.T) {
		if _, err := extractLibrary([]byte("not an archive"), libraryName()); err == nil {
			t.Fatal("expected error for unrecognized archive format")
		}
	})
}

func TestEnsureLibCachesWithoutNetwork(t *testing.T) {
	stateDir := t.TempDir()
	path := LibraryPath(stateDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("cached-lib"), 0o755); err != nil {
		t.Fatalf("write cached lib: %v", err)
	}

	got, err := ensureLibrary(context.Background(), stateDir, "http://127.0.0.1:1/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Fatalf("got %q, want %q", got, path)
	}

	gotPublic, err := EnsureLibrary(context.Background(), stateDir)
	if err != nil {
		t.Fatalf("EnsureLibrary cached call: %v", err)
	}
	if gotPublic != path {
		t.Fatalf("EnsureLibrary got %q, want %q", gotPublic, path)
	}
}

func TestEnsureLibDownloadsVerifiesAndInstalls(t *testing.T) {
	assetName, err := libAssetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}

	for _, tc := range []struct {
		name    string
		archive func(want []byte) []byte
	}{
		{
			name: "zip",
			archive: func(want []byte) []byte {
				return buildFakeZipArchive(t, map[string][]byte{
					libraryName(): want,
					"duckdb.h":    []byte("// header"),
				})
			},
		},
		{
			name: "targz",
			archive: func(want []byte) []byte {
				return buildFakeArchive(t, libraryName(), want)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := []byte("fake-libduckdb-payload-" + tc.name)
			archive := tc.archive(want)
			sum := sha256.Sum256(archive)

			baseURL := testInstallServer(t, assetName, archive)
			checksums := map[string]string{assetName: hex.EncodeToString(sum[:])}

			stateDir := t.TempDir()
			path, err := ensureLibrary(context.Background(), stateDir, baseURL, checksums)
			if err != nil {
				t.Fatalf("ensureLibrary: %v", err)
			}
			if path != LibraryPath(stateDir) {
				t.Fatalf("got path %q, want %q", path, LibraryPath(stateDir))
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read installed lib: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("installed contents = %q, want %q", got, want)
			}

			// Second call must hit cache without network.
			path2, err := ensureLibrary(context.Background(), stateDir, "http://127.0.0.1:1/", checksums)
			if err != nil {
				t.Fatalf("cached ensureLibrary: %v", err)
			}
			if path2 != path {
				t.Fatalf("cached path = %q, want %q", path2, path)
			}
		})
	}
}

func TestEnsureLibChecksumMismatchFailsClosed(t *testing.T) {
	assetName, err := libAssetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}

	archive := buildFakeZipArchive(t, map[string][]byte{libraryName(): []byte("fake")})
	baseURL := testInstallServer(t, assetName, archive)
	checksums := map[string]string{assetName: "0000000000000000000000000000000000000000000000000000000000000000"}

	stateDir := t.TempDir()
	if _, err := ensureLibrary(context.Background(), stateDir, baseURL, checksums); err == nil {
		t.Fatal("expected checksum mismatch to fail closed")
	}
	if _, err := os.Stat(LibraryPath(stateDir)); !os.IsNotExist(err) {
		t.Fatalf("expected no library installed on checksum mismatch, stat err: %v", err)
	}
}

func TestEnsureLibMissingChecksumFailsClosed(t *testing.T) {
	assetName, err := libAssetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}

	archive := buildFakeZipArchive(t, map[string][]byte{libraryName(): []byte("fake")})
	baseURL := testInstallServer(t, assetName, archive)

	stateDir := t.TempDir()
	if _, err := ensureLibrary(context.Background(), stateDir, baseURL, map[string]string{}); err == nil {
		t.Fatal("expected missing checksum entry to fail closed")
	}
}

func TestEnsureLibConcurrentDownloadSafety(t *testing.T) {
	assetName, err := libAssetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform: %v", err)
	}

	want := bytes.Repeat([]byte("duckdb-shared-lib-"), 1024)
	archive := buildFakeZipArchive(t, map[string][]byte{libraryName(): want})
	sum := sha256.Sum256(archive)

	baseURL := testInstallServer(t, assetName, archive)
	checksums := map[string]string{assetName: hex.EncodeToString(sum[:])}

	stateDir := t.TempDir()
	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	paths := make(chan string, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := ensureLibrary(context.Background(), stateDir, baseURL, checksums)
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
		t.Fatalf("concurrent ensureLibrary failed: %v", err)
	}
	for p := range paths {
		if p != LibraryPath(stateDir) {
			t.Fatalf("got path %q, want %q", p, LibraryPath(stateDir))
		}
	}

	got, err := os.ReadFile(LibraryPath(stateDir))
	if err != nil {
		t.Fatalf("reading installed lib: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("installed library contents corrupted under concurrent download")
	}

	// Verify no leftover temp files remain in the lib directory.
	entries, err := os.ReadDir(filepath.Dir(LibraryPath(stateDir)))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != libraryName() {
		t.Fatalf("unexpected files in lib dir: %v", entries)
	}
}
