package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/go-homedir"
	"golang.org/x/mod/semver"
)

const (
	releasesURL  = "https://api.github.com/repos/inngest/inngest/releases/latest"
	cacheTTL     = 24 * time.Hour
	checkTimeout = 3 * time.Second
)

type cacheFile struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
	LatestURL     string    `json:"latest_url"`
}

var (
	cacheMu      sync.Mutex
	httpClient   = &http.Client{Timeout: checkTimeout}
	cachePathFn  = defaultCachePath
	releasesAddr = releasesURL
)

// Check refreshes the cached "latest version" record if the cache is stale.
// Network and I/O errors are swallowed silently — this is a best-effort
// background helper. Safe to call from a goroutine.
//
// Honors the same opt-out gates as Notify (env vars, TTY, dev builds) so
// disabling the notifier also suppresses the background network request.
func Check(ctx context.Context, currentVersion string) {
	if disabled(currentVersion) {
		return
	}
	if !shouldFetch() {
		return
	}
	latest, url, err := fetchLatest(ctx)
	if err != nil || latest == "" {
		return
	}
	_ = writeCache(cacheFile{
		CheckedAt:     time.Now().UTC(),
		LatestVersion: latest,
		LatestURL:     url,
	})
}

// Latest returns the most recently cached "latest version" tuple, or empty
// strings if no fresh-enough cache exists.
func Latest() (version, url string) {
	c, err := readCache()
	if err != nil {
		return "", ""
	}
	return c.LatestVersion, c.LatestURL
}

func shouldFetch() bool {
	c, err := readCache()
	if err != nil {
		return true
	}
	return time.Since(c.CheckedAt) > cacheTTL
}

func fetchLatest(ctx context.Context) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesAddr, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "inngest-cli")
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("releases: status %d", resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return "", "", err
	}
	return strings.TrimPrefix(body.TagName, "v"), body.HTMLURL, nil
}

func defaultCachePath() (string, error) {
	dir, err := homedir.Expand("~/.config/inngest")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "update-check.json"), nil
}

func readCache() (cacheFile, error) {
	var c cacheFile
	p, err := cachePathFn()
	if err != nil {
		return c, err
	}
	byt, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return c, err
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(byt, &c); err != nil {
		return c, err
	}
	return c, nil
}

func writeCache(c cacheFile) error {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	p, err := cachePathFn()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	byt, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, byt, 0o600)
}

// IsNewer reports whether `latest` is strictly newer than `current`.
// Accepts versions with or without a leading "v". Empty strings, "dev", or
// unparseable inputs return false.
func IsNewer(current, latest string) bool {
	cv := normalizeSemver(current)
	lv := normalizeSemver(latest)
	if cv == "" || lv == "" {
		return false
	}
	return semver.Compare(lv, cv) > 0
}

func normalizeSemver(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "dev" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if semver.IsValid(v) {
		return semver.Canonical(v)
	}
	return ""
}
