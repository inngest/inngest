package driver

import (
	"archive/tar"
	"archive/zip"
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

// defaultInstallBaseURL is where EnsureLibrary downloads
// duckdb-shared-libs-<dist>.tar.gz assets from.
const defaultInstallBaseURL = "https://github.com/duckdb/duckdb/releases/download/" + PinnedVersion + "/"

// maxDownloadSize bounds the release archive download.
const maxDownloadSize = 200 << 20 // 200MB

// maxLibExtractSize bounds the extracted shared library size (the macOS
// universal libduckdb.dylib is ~135MB, so 300MB gives generous headroom).
const maxLibExtractSize = 300 << 20 // 300MB

// BinaryPath is an alias for LibraryPath retained for compatibility with
// callers such as `inngest duckdb download`.
func BinaryPath(stateDir string) string {
	return LibraryPath(stateDir)
}

// EnsureBinary is an alias for EnsureLibrary retained for compatibility with
// callers such as `inngest duckdb download`.
func EnsureBinary(ctx context.Context, stateDir string) (string, error) {
	return EnsureLibrary(ctx, stateDir)
}

// libraryName returns the shared library filename for the current OS.
func libraryName() string {
	return libraryNameFor(runtime.GOOS)
}

func libraryNameFor(goos string) string {
	switch goos {
	case "darwin":
		return "libduckdb.dylib"
	case "windows":
		return "duckdb.dll"
	default:
		return "libduckdb.so"
	}
}

// LibraryPath is where EnsureLibrary installs (or finds already installed) the
// pinned DuckDB shared library under stateDir — versioned so a future
// PinnedVersion bump downloads fresh rather than silently reusing an old
// library left on disk.
func LibraryPath(stateDir string) string {
	return filepath.Join(stateDir, "lib", PinnedVersion, libraryName())
}

// EnsureLibrary makes sure the pinned DuckDB shared library (libduckdb.dylib /
// libduckdb.so / duckdb.dll) is present under stateDir (see LibraryPath) and
// returns its path, downloading and verifying it on first use. Later calls with
// the same stateDir are served entirely from the cached file, with no network
// access.
func EnsureLibrary(ctx context.Context, stateDir string) (string, error) {
	return ensureLibrary(ctx, stateDir, defaultInstallBaseURL, pinnedLibChecksums)
}

// ensureLibrary is EnsureLibrary with an injectable base URL and checksum
// table, so tests can point it at an httptest server and known-good checksums
// instead of the real network and pinnedLibChecksums.
func ensureLibrary(ctx context.Context, stateDir, baseURL string, checksums map[string]string) (string, error) {
	path := LibraryPath(stateDir)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("duckdb: checking for cached library at %q: %w", path, err)
	}

	assetName, err := libAssetNameFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}

	want, ok := checksums[assetName]
	if !ok {
		return "", fmt.Errorf("duckdb: no pinned checksum for library asset %q", assetName)
	}

	archive, err := fetchBytes(ctx, baseURL+assetName)
	if err != nil {
		return "", fmt.Errorf("duckdb: downloading %s: %w", assetName, err)
	}

	libBytes, err := verifyAndExtractLibrary(archive, want, libraryName())
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("duckdb: creating lib directory: %w", err)
	}
	if err := writeFileAtomically(path, libBytes, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

// fetchBytes GETs url with ctx and returns its body, capped at maxDownloadSize.
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

// verifyChecksum checks that sha256(data) matches want (lowercase hex).
func verifyChecksum(data []byte, want string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("duckdb: checksum mismatch: got %s, want %s", got, want)
	}
	return nil
}

func verifyAndExtractLibrary(archive []byte, wantSHA256, wantEntry string) ([]byte, error) {
	if err := verifyChecksum(archive, wantSHA256); err != nil {
		return nil, err
	}
	return extractLibrary(archive, wantEntry)
}

// libAssetNameFor maps a Go GOOS/GOARCH pair to the DuckDB shared library
// release asset name for PinnedVersion (v2.0.0).
func libAssetNameFor(goos, goarch string) (string, error) {
	switch goos {
	case "darwin":
		return "duckdb-shared-libs-osx-universal.tar.gz", nil
	case "linux":
		switch goarch {
		case "amd64":
			return "duckdb-shared-libs-linux-amd64.tar.gz", nil
		case "arm64":
			return "duckdb-shared-libs-linux-arm64.tar.gz", nil
		}
	case "windows":
		if goarch == "amd64" {
			return "duckdb-shared-libs-windows-amd64.tar.gz", nil
		}
	}
	return "", fmt.Errorf("duckdb: no known shared library asset for GOOS=%s GOARCH=%s", goos, goarch)
}

// extractLibrary extracts the named shared library entry from either a .zip
// archive (PK magic) or a .tar.gz archive (gzip magic).
func extractLibrary(archive []byte, wantName string) ([]byte, error) {
	if len(archive) >= 4 && archive[0] == 'P' && archive[1] == 'K' && archive[2] == 0x03 && archive[3] == 0x04 {
		return extractLibraryFromZip(archive, wantName)
	}
	if len(archive) >= 2 && archive[0] == 0x1f && archive[1] == 0x8b {
		return extractLibraryFromTarGz(archive, wantName)
	}
	return nil, fmt.Errorf("duckdb: unrecognized library archive format (expected .tar.gz or .zip)")
}

func extractLibraryFromZip(archive []byte, wantName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("duckdb: opening zip library archive: %w", err)
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if filepath.Base(f.Name) != wantName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("duckdb: opening %q in zip archive: %w", f.Name, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(rc, maxLibExtractSize+1))
		_ = rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("duckdb: reading %q from zip archive: %w", f.Name, readErr)
		}
		if len(data) > maxLibExtractSize {
			return nil, fmt.Errorf("duckdb: %q exceeds maximum extracted size (%d bytes)", wantName, maxLibExtractSize)
		}
		return data, nil
	}
	return nil, fmt.Errorf("duckdb: no %q entry found in zip library archive", wantName)
}

func extractLibraryFromTarGz(archive []byte, wantName string) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("duckdb: opening tar.gz library archive: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("duckdb: reading tar.gz library archive: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		if filepath.Base(hdr.Name) != wantName {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(tr, maxLibExtractSize+1))
		if err != nil {
			return nil, fmt.Errorf("duckdb: reading %q from tar.gz archive: %w", hdr.Name, err)
		}
		if len(data) > maxLibExtractSize {
			return nil, fmt.Errorf("duckdb: %q exceeds maximum extracted size (%d bytes)", wantName, maxLibExtractSize)
		}
		return data, nil
	}
	return nil, fmt.Errorf("duckdb: no %q entry found in tar.gz library archive", wantName)
}

// writeFileAtomically writes data to a unique temp file in the same directory
// via os.CreateTemp, sets permissions, and renames it into place. Using a
// unique temp file ensures concurrent downloads never race on the same temp
// path and readers never observe a partially-written library.
func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".duckdb-lib-*.tmp")
	if err != nil {
		return fmt.Errorf("duckdb: creating temporary library file: %w", err)
	}
	tmpPath := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("duckdb: writing temporary library file: %w", err)
	}
	if err := f.Chmod(perm); err != nil {
		_ = f.Close()
		return fmt.Errorf("duckdb: setting permissions on temporary library file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("duckdb: closing temporary library file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("duckdb: installing library at %q: %w", path, err)
	}
	cleanup = false
	return nil
}
