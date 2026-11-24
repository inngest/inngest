package driver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const inngestReleaseBaseURLTemplate = "https://github.com/inngest/inngest/releases/download/v%s/"

func workerBinaryName() string {
	return workerBinaryNameFor(runtime.GOOS)
}

func workerBinaryNameFor(goos string) string {
	if goos == "windows" {
		return "inngest-duckdb-worker.exe"
	}
	return "inngest-duckdb-worker"
}

func normalizeWorkerVersion(ver string) string {
	return strings.TrimPrefix(strings.TrimSpace(ver), "v")
}

// WorkerBinaryPath returns the versioned cache path where EnsureWorkerBinary
// installs (or finds already installed) the inngest-duckdb-worker binary under
// stateDir.
func WorkerBinaryPath(stateDir, ver string) string {
	return filepath.Join(stateDir, "bin", "inngest-duckdb-worker", normalizeWorkerVersion(ver), workerBinaryName())
}

// EnsureWorkerBinary ensures the matching version of inngest-duckdb-worker is
// available locally and returns its executable path.
//
// For development builds (ver == "" or "dev"), it checks for a sibling binary
// next to os.Executable() first, then falls back to PATH via exec.LookPath.
//
// For release builds, it serves from WorkerBinaryPath if cached, or otherwise
// fetches checksums.txt and the platform archive from GitHub releases, verifies
// the SHA256 digest (failing closed on mismatch or missing entry), extracts the
// binary, and installs it atomically.
func EnsureWorkerBinary(ctx context.Context, stateDir, ver string) (string, error) {
	norm := normalizeWorkerVersion(ver)
	if norm == "" || norm == "dev" {
		return resolveDevWorkerBinary()
	}
	baseURL := fmt.Sprintf(inngestReleaseBaseURLTemplate, norm)
	return ensureWorkerBinary(ctx, stateDir, norm, baseURL)
}

func resolveDevWorkerBinary() (string, error) {
	name := workerBinaryName()
	if exe, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(exe), name)
		if info, statErr := os.Stat(sibling); statErr == nil && !info.IsDir() {
			return sibling, nil
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("duckdb: %s not found next to executable or on PATH (dev build)", name)
}

func ensureWorkerBinary(ctx context.Context, stateDir, ver, baseURL string) (string, error) {
	norm := normalizeWorkerVersion(ver)
	if norm == "" || norm == "dev" {
		return resolveDevWorkerBinary()
	}

	path := WorkerBinaryPath(stateDir, norm)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("duckdb: checking for cached worker binary at %q: %w", path, err)
	}

	assetName, err := workerAssetNameFor(norm, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}

	checksumsData, err := fetchBytes(ctx, baseURL+"checksums.txt")
	if err != nil {
		return "", fmt.Errorf("duckdb: downloading checksums.txt: %w", err)
	}

	wantSHA, err := parseChecksumsTxt(checksumsData, assetName)
	if err != nil {
		return "", err
	}

	archive, err := fetchBytes(ctx, baseURL+assetName)
	if err != nil {
		return "", fmt.Errorf("duckdb: downloading %s: %w", assetName, err)
	}

	if err := verifyChecksum(archive, wantSHA); err != nil {
		return "", err
	}

	binBytes, err := extractLibrary(archive, workerBinaryName())
	if err != nil {
		return "", fmt.Errorf("duckdb: extracting %s from %s: %w", workerBinaryName(), assetName, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("duckdb: creating worker bin directory: %w", err)
	}
	if err := writeFileAtomically(path, binBytes, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func workerAssetNameFor(ver, goos, goarch string) (string, error) {
	norm := normalizeWorkerVersion(ver)
	switch goos {
	case "darwin", "linux", "windows":
	default:
		return "", fmt.Errorf("duckdb: unsupported OS %q for inngest-duckdb-worker", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("duckdb: unsupported arch %q for inngest-duckdb-worker", goarch)
	}
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("inngest-duckdb-worker_%s_%s_%s.%s", norm, goos, goarch, ext), nil
}

// parseChecksumsTxt parses a GoReleaser checksums.txt file and returns the
// lowercase hex SHA256 digest for targetAsset. Fails closed if missing or malformed.
func parseChecksumsTxt(data []byte, targetAsset string) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		digest := strings.ToLower(fields[0])
		name := strings.TrimPrefix(fields[1], "*")
		if name != targetAsset {
			continue
		}
		if len(digest) != 64 {
			return "", fmt.Errorf("duckdb: malformed sha256 digest for %q in checksums.txt", targetAsset)
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return "", fmt.Errorf("duckdb: invalid hex sha256 digest for %q in checksums.txt: %w", targetAsset, err)
		}
		return digest, nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("duckdb: scanning checksums.txt: %w", err)
	}
	return "", fmt.Errorf("duckdb: no checksum entry for %q in checksums.txt", targetAsset)
}
