package duckdb

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// PinnedVersion, pinnedStagedCommit, and pinnedChecksums live in the
// generated pinned.go, kept separate from this file so that bumping the pin
// (via scripts/update-duckdb-version.sh) produces a small, easy-to-review
// diff instead of touching download logic.

// defaultInstallBaseURL is where EnsureBinary downloads
// duckdb-cli-<dist>.tar.gz assets from — DuckDB's nightly/staged
// distribution server, addressed by commit+version, mirroring the URL
// duckdb's own official install script
// (https://github.com/duckdb/duckdb-install-scripts) constructs for
// DUCKDB_STAGED builds.
const defaultInstallBaseURL = "https://duckdb-staging.duckdb.org/" + pinnedStagedCommit + "/" + PinnedVersion + "/duckdb/duckdb/github_release/"

// maxDownloadSize bounds the release archive download, mirroring
// pkg/update/check.go's io.LimitReader precedent — the CLI archive is a few
// tens of MB, so this is generous headroom, not a real limit.
const maxDownloadSize = 200 << 20 // 200MB

// binaryName is the executable's filename inside the DuckDB CLI release
// archive, and the name EnsureBinary installs it under.
func binaryName() string {
	if runtime.GOOS == "windows" {
		return "duckdb.exe"
	}
	return "duckdb"
}

// BinaryPath is where EnsureBinary installs (or finds already installed) the
// pinned DuckDB CLI binary under stateDir — versioned so a future
// PinnedVersion bump downloads fresh rather than silently reusing an old
// binary left on disk. Exported so callers (e.g. the `inngest duckdb
// download` CLI command) can report or remove the cached path without
// duplicating this layout.
func BinaryPath(stateDir string) string {
	return filepath.Join(stateDir, "bin", PinnedVersion, binaryName())
}

// EnsureBinary makes sure the pinned DuckDB CLI binary is present under
// stateDir (see BinaryPath) and returns its path, downloading and verifying
// it on first use. Later calls with the same stateDir are served entirely
// from the cached file, with no network access.
func EnsureBinary(ctx context.Context, stateDir string) (string, error) {
	return ensureBinary(ctx, stateDir, defaultInstallBaseURL, pinnedChecksums)
}

// ensureBinary is EnsureBinary with an injectable base URL and checksum
// table, so tests can point it at an httptest server and known-good
// checksums instead of the real network and pinnedChecksums.
func ensureBinary(ctx context.Context, stateDir, baseURL string, checksums map[string]string) (string, error) {
	path := BinaryPath(stateDir)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("duckdb: checking for cached binary at %q: %w", path, err)
	}

	assetName, err := assetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}

	want, ok := checksums[assetName]
	if !ok {
		return "", fmt.Errorf("duckdb: no pinned checksum for asset %q", assetName)
	}

	archive, err := fetchBytes(ctx, baseURL+assetName)
	if err != nil {
		return "", fmt.Errorf("duckdb: downloading %s: %w", assetName, err)
	}

	if err := verifyChecksum(archive, want); err != nil {
		return "", err
	}

	bin, err := extractBinary(archive)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("duckdb: creating bin directory: %w", err)
	}
	if err := writeBinaryAtomically(path, bin); err != nil {
		return "", err
	}
	return path, nil
}

// assetNameFor maps a Go GOOS/GOARCH pair to the DuckDB CLI release asset
// name that runs on it. DuckDB ships a single universal binary for macOS
// (covering both amd64 and arm64), and separate amd64/arm64 builds for
// Linux, and an amd64 build for Windows — no other combination is
// supported.
func assetNameFor(goos, goarch string) (string, error) {
	switch goos {
	case "darwin":
		return "duckdb-cli-osx-universal.tar.gz", nil
	case "linux":
		switch goarch {
		case "amd64":
			return "duckdb-cli-linux-amd64.tar.gz", nil
		case "arm64":
			return "duckdb-cli-linux-arm64.tar.gz", nil
		}
	case "windows":
		if goarch == "amd64" {
			return "duckdb-cli-windows-amd64.tar.gz", nil
		}
	}
	return "", fmt.Errorf("duckdb: no known release asset for GOOS=%s GOARCH=%s", goos, goarch)
}

// fetchBytes GETs url with ctx and returns its body, capped at
// maxDownloadSize.
func fetchBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize))
}

// verifyChecksum checks that sha256(data) matches want (lowercase hex). It
// returns an error — never a silent pass — on mismatch, so callers fail
// closed rather than installing an unverified binary.
func verifyChecksum(data []byte, want string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("duckdb: checksum mismatch: got %s, want %s", got, want)
	}
	return nil
}

// extractBinary finds and returns the duckdb (or duckdb.exe) entry's
// uncompressed contents inside a duckdb-cli-<dist>.tar.gz release archive.
func extractBinary(archive []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("duckdb: opening release archive: %w", err)
	}
	defer gr.Close()

	want := binaryName()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("duckdb: reading release archive: %w", err)
		}
		if filepath.Base(hdr.Name) != want {
			continue
		}
		return io.ReadAll(tr)
	}
	return nil, fmt.Errorf("duckdb: no %q entry found in release archive", want)
}

// writeBinaryAtomically writes data to path with executable permissions via
// a temp file + rename, so a concurrent reader (or a crash mid-write) never
// observes a partially-written binary at the final path.
func writeBinaryAtomically(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return fmt.Errorf("duckdb: writing temporary binary: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("duckdb: installing binary at %q: %w", path, err)
	}
	return nil
}
