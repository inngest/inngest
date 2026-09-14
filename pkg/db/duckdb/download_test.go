package duckdb

import (
	"archive/tar"
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
	"testing"
)

func TestAssetNameForPlatform(t *testing.T) {
	cases := []struct {
		goos, goarch string
		want         string
		wantErr      bool
	}{
		{goos: "darwin", goarch: "arm64", want: "duckdb-cli-osx-universal.tar.gz"},
		{goos: "darwin", goarch: "amd64", want: "duckdb-cli-osx-universal.tar.gz"},
		{goos: "linux", goarch: "amd64", want: "duckdb-cli-linux-amd64.tar.gz"},
		{goos: "linux", goarch: "arm64", want: "duckdb-cli-linux-arm64.tar.gz"},
		{goos: "windows", goarch: "amd64", want: "duckdb-cli-windows-amd64.tar.gz"},
		{goos: "linux", goarch: "386", wantErr: true},
		{goos: "freebsd", goarch: "amd64", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.goos+"/"+tc.goarch, func(t *testing.T) {
			got, err := assetNameFor(tc.goos, tc.goarch)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("assetNameFor(%q, %q) = %q, want an error", tc.goos, tc.goarch, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("assetNameFor(%q, %q) returned unexpected error: %v", tc.goos, tc.goarch, err)
			}
			if got != tc.want {
				t.Fatalf("assetNameFor(%q, %q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
			}
		})
	}
}

func TestVerifyChecksumMatches(t *testing.T) {
	data := []byte("fake duckdb cli archive contents")
	sum := sha256.Sum256(data)

	if err := verifyChecksum(data, hex.EncodeToString(sum[:])); err != nil {
		t.Fatalf("expected checksum to verify, got error: %v", err)
	}
}

func TestVerifyChecksumMismatchFails(t *testing.T) {
	data := []byte("fake duckdb cli archive contents")

	if err := verifyChecksum(data, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("expected a checksum mismatch to return an error")
	}
}

// buildFakeArchive builds a .tar.gz archive containing one entry, mirroring
// the shape of a real duckdb-cli-<dist>.tar.gz release from the nightly
// channel (a single top-level file, no directory prefix).
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

func TestExtractBinaryFindsEntry(t *testing.T) {
	want := []byte("#!/bin/sh\necho fake duckdb\n")
	archive := buildFakeArchive(t, binaryName(), want)

	got, err := extractBinary(archive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("extractBinary returned %q, want %q", got, want)
	}
}

func TestExtractBinaryMissingEntryFails(t *testing.T) {
	archive := buildFakeArchive(t, "README.md", []byte("not a binary"))

	if _, err := extractBinary(archive); err == nil {
		t.Fatal("expected an error when the archive has no duckdb entry")
	}
}

func TestEnsureBinaryCachesWithoutNetwork(t *testing.T) {
	stateDir := t.TempDir()
	path := BinaryPath(stateDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	want := []byte("already cached")
	if err := os.WriteFile(path, want, 0o755); err != nil {
		t.Fatalf("writing cached binary: %v", err)
	}

	// Nothing listens on this address; a network call here would fail the
	// test with a connection error, proving the cache short-circuits before
	// any HTTP request is made.
	got, err := ensureBinary(context.Background(), stateDir, "http://127.0.0.1:1/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Fatalf("got %q, want %q", got, path)
	}
}

func TestEnsureBinaryPublicWrapperUsesCache(t *testing.T) {
	stateDir := t.TempDir()
	path := BinaryPath(stateDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("cached"), 0o755); err != nil {
		t.Fatalf("writing cached binary: %v", err)
	}

	got, err := EnsureBinary(context.Background(), stateDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Fatalf("got %q, want %q", got, path)
	}
}

// testInstallServer stands in for the nightly distribution server, serving
// archive at <server>/<assetName>, so these tests never hit the real
// network.
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

func TestEnsureBinaryDownloadsVerifiesAndInstalls(t *testing.T) {
	assetName, err := assetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform for this test: %v", err)
	}

	want := []byte("#!/bin/sh\necho fake duckdb\n")
	archive := buildFakeArchive(t, binaryName(), want)
	sum := sha256.Sum256(archive)

	baseURL := testInstallServer(t, assetName, archive)
	checksums := map[string]string{assetName: hex.EncodeToString(sum[:])}

	stateDir := t.TempDir()
	path, err := ensureBinary(context.Background(), stateDir, baseURL, checksums)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != BinaryPath(stateDir) {
		t.Fatalf("got path %q, want %q", path, BinaryPath(stateDir))
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading installed binary: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("installed binary contents = %q, want %q", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat installed binary: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("installed binary is not executable: mode %v", info.Mode())
	}

	// A second call must be served entirely from cache; point baseURL at an
	// unreachable address to prove no further network call happens.
	path2, err := ensureBinary(context.Background(), stateDir, "http://127.0.0.1:1/", checksums)
	if err != nil {
		t.Fatalf("unexpected error on cached call: %v", err)
	}
	if path2 != path {
		t.Fatalf("cached call returned %q, want %q", path2, path)
	}
}


func TestEnsureBinaryChecksumMismatchFailsClosed(t *testing.T) {
	assetName, err := assetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform for this test: %v", err)
	}

	archive := buildFakeArchive(t, binaryName(), []byte("#!/bin/sh\necho fake duckdb\n"))
	baseURL := testInstallServer(t, assetName, archive)
	// Deliberately wrong checksum.
	checksums := map[string]string{assetName: "0000000000000000000000000000000000000000000000000000000000000000"}

	stateDir := t.TempDir()
	if _, err := ensureBinary(context.Background(), stateDir, baseURL, checksums); err == nil {
		t.Fatal("expected a checksum mismatch to fail closed")
	}

	if _, err := os.Stat(BinaryPath(stateDir)); !os.IsNotExist(err) {
		t.Fatalf("expected no binary to be installed after a checksum mismatch, stat error: %v", err)
	}
}

func TestEnsureBinaryMissingChecksumEntryFailsClosed(t *testing.T) {
	assetName, err := assetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("unsupported platform for this test: %v", err)
	}

	archive := buildFakeArchive(t, binaryName(), []byte("#!/bin/sh\necho fake duckdb\n"))
	baseURL := testInstallServer(t, assetName, archive)

	stateDir := t.TempDir()
	if _, err := ensureBinary(context.Background(), stateDir, baseURL, map[string]string{}); err == nil {
		t.Fatal("expected a missing checksum entry to fail closed")
	}
}

// TestPinnedChecksumsCoverEveryKnownAsset guards against PinnedVersion or the
// asset table drifting out of sync with the hardcoded pinnedChecksums map —
// every asset assetNameFor can produce must have a checksum pinned, or
// EnsureBinary would fail closed on that platform even though the correct
// archive is being served.
func TestPinnedChecksumsCoverEveryKnownAsset(t *testing.T) {
	platforms := []struct{ goos, goarch string }{
		{"darwin", "arm64"},
		{"darwin", "amd64"},
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"windows", "amd64"},
	}
	for _, p := range platforms {
		name, err := assetNameFor(p.goos, p.goarch)
		if err != nil {
			t.Fatalf("assetNameFor(%q, %q): %v", p.goos, p.goarch, err)
		}
		if _, ok := pinnedChecksums[name]; !ok {
			t.Errorf("no pinned checksum for asset %q (GOOS=%s GOARCH=%s)", name, p.goos, p.goarch)
		}
	}
}
