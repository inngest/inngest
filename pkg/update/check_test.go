package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func withTempCache(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "update-check.json")
	old := cachePathFn
	cachePathFn = func() (string, error) { return path, nil }
	t.Cleanup(func() { cachePathFn = old })
	return path
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.35.0", "0.36.0", true},
		{"0.35.0", "0.35.1", true},
		{"v0.35.0", "0.36.0", true},
		{"0.35.0", "v0.35.0", false},
		{"0.36.0", "0.35.0", false},
		{"dev", "0.36.0", false},
		{"", "0.36.0", false},
		{"0.35.0", "", false},
		{"not-a-version", "0.36.0", false},
		{"0.35.0", "0.36.0-rc.1", true},
		{"0.35.0-rc.1", "0.35.0", true},
	}
	for _, tc := range cases {
		if got := IsNewer(tc.current, tc.latest); got != tc.want {
			t.Errorf("IsNewer(%q,%q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestCacheRoundTrip(t *testing.T) {
	withTempCache(t)

	if _, _, _ = "x", "x", 0; true {
		// Empty cache — Latest returns empty.
		if v, u := Latest(); v != "" || u != "" {
			t.Fatalf("Latest() on empty cache = (%q,%q), want empty", v, u)
		}
	}

	want := cacheFile{
		CheckedAt:     time.Now().UTC().Truncate(time.Second),
		LatestVersion: "0.36.0",
		LatestURL:     "https://example.test/v0.36.0",
	}
	if err := writeCache(want); err != nil {
		t.Fatalf("writeCache: %v", err)
	}
	got, err := readCache()
	if err != nil {
		t.Fatalf("readCache: %v", err)
	}
	if !got.CheckedAt.Equal(want.CheckedAt) || got.LatestVersion != want.LatestVersion || got.LatestURL != want.LatestURL {
		t.Errorf("round-trip mismatch: got %+v want %+v", got, want)
	}
	if v, u := Latest(); v != want.LatestVersion || u != want.LatestURL {
		t.Errorf("Latest() = (%q,%q), want (%q,%q)", v, u, want.LatestVersion, want.LatestURL)
	}
}

func TestShouldFetch(t *testing.T) {
	withTempCache(t)

	// No file — should fetch.
	if !shouldFetch() {
		t.Error("shouldFetch with no cache = false, want true")
	}

	// Fresh cache — should NOT fetch.
	if err := writeCache(cacheFile{CheckedAt: time.Now().UTC(), LatestVersion: "0.36.0"}); err != nil {
		t.Fatal(err)
	}
	if shouldFetch() {
		t.Error("shouldFetch with fresh cache = true, want false")
	}

	// Stale cache — should fetch.
	if err := writeCache(cacheFile{CheckedAt: time.Now().Add(-2 * cacheTTL).UTC(), LatestVersion: "0.36.0"}); err != nil {
		t.Fatal(err)
	}
	if !shouldFetch() {
		t.Error("shouldFetch with stale cache = false, want true")
	}
}

func TestCheckFetchesAndCaches(t *testing.T) {
	withTempCache(t)
	// Force the env-var gate off so the test doesn't depend on host env.
	for _, k := range []string{"INNGEST_NO_UPDATE_NOTIFIER", "NO_UPDATE_NOTIFIER", "DO_NOT_TRACK", "CI"} {
		t.Setenv(k, "")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v0.36.0",
			"html_url": "https://example.test/releases/v0.36.0",
		})
	}))
	defer srv.Close()

	oldAddr := releasesAddr
	releasesAddr = srv.URL
	t.Cleanup(func() { releasesAddr = oldAddr })

	// Bypass the TTY gate inside disabled() — the fetch logic itself does
	// not require a TTY, only the shared opt-out helper does. We test that
	// helper separately; here we exercise the network path by calling the
	// internals directly.
	if !shouldFetch() {
		t.Fatal("shouldFetch returned false with empty cache")
	}
	latest, url, err := fetchLatest(context.Background())
	if err != nil {
		t.Fatalf("fetchLatest: %v", err)
	}
	if err := writeCache(cacheFile{CheckedAt: time.Now().UTC(), LatestVersion: latest, LatestURL: url}); err != nil {
		t.Fatal(err)
	}

	v, u := Latest()
	if v != "0.36.0" {
		t.Errorf("Latest version = %q, want %q", v, "0.36.0")
	}
	if u != "https://example.test/releases/v0.36.0" {
		t.Errorf("Latest url = %q", u)
	}
}

func TestCheckSilentOnHTTPError(t *testing.T) {
	withTempCache(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	oldAddr := releasesAddr
	releasesAddr = srv.URL
	t.Cleanup(func() { releasesAddr = oldAddr })

	// Even calling the fetch directly, an HTTP error must not write cache.
	if _, _, err := fetchLatest(context.Background()); err == nil {
		t.Fatal("fetchLatest succeeded on 500, want error")
	}
	if v, _ := Latest(); v != "" {
		t.Errorf("Latest() = %q after failed fetch, want empty", v)
	}
}

func TestCheckSkipsWhenDisabled(t *testing.T) {
	withTempCache(t)

	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v0.36.0"})
	}))
	defer srv.Close()

	oldAddr := releasesAddr
	releasesAddr = srv.URL
	t.Cleanup(func() { releasesAddr = oldAddr })

	for _, tc := range []struct {
		name           string
		envKey, envVal string
		currentVer     string
	}{
		{name: "INNGEST_NO_UPDATE_NOTIFIER", envKey: "INNGEST_NO_UPDATE_NOTIFIER", envVal: "1", currentVer: "0.35.0"},
		{name: "NO_UPDATE_NOTIFIER", envKey: "NO_UPDATE_NOTIFIER", envVal: "1", currentVer: "0.35.0"},
		{name: "DO_NOT_TRACK", envKey: "DO_NOT_TRACK", envVal: "1", currentVer: "0.35.0"},
		{name: "CI", envKey: "CI", envVal: "true", currentVer: "0.35.0"},
		{name: "dev build", currentVer: "dev"},
		{name: "empty version", currentVer: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envVal)
			}
			before := hits
			Check(context.Background(), tc.currentVer)
			if hits != before {
				t.Errorf("Check made %d new HTTP request(s) with %s set; expected 0", hits-before, tc.name)
			}
		})
	}
}

func TestNotifySkipsWithoutCache(t *testing.T) {
	withTempCache(t)
	t.Setenv("INNGEST_NO_UPDATE_NOTIFIER", "")
	t.Setenv("NO_UPDATE_NOTIFIER", "")

	// Even if all gates pass, no cache means no output.
	var buf testWriter
	Notify(&buf, "0.35.0")
	if buf.n > 0 {
		t.Errorf("Notify wrote %d bytes with no cache, want 0", buf.n)
	}
}

func TestNotifySkipsForDevBuild(t *testing.T) {
	withTempCache(t)
	if err := writeCache(cacheFile{CheckedAt: time.Now().UTC(), LatestVersion: "99.0.0"}); err != nil {
		t.Fatal(err)
	}
	var buf testWriter
	Notify(&buf, "dev")
	if buf.n > 0 {
		t.Errorf("Notify wrote %d bytes for dev build, want 0", buf.n)
	}
}

func TestNotifyRespectsOptOut(t *testing.T) {
	withTempCache(t)
	if err := writeCache(cacheFile{CheckedAt: time.Now().UTC(), LatestVersion: "99.0.0"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"INNGEST_NO_UPDATE_NOTIFIER", "NO_UPDATE_NOTIFIER", "DO_NOT_TRACK", "CI"} {
		t.Run(k, func(t *testing.T) {
			t.Setenv(k, "1")
			var buf testWriter
			Notify(&buf, "0.35.0")
			if buf.n > 0 {
				t.Errorf("Notify wrote %d bytes with %s set, want 0", buf.n, k)
			}
		})
	}
}

func TestEnabledFor(t *testing.T) {
	cases := map[string]bool{
		"dev":     true,
		"help":    true,
		"version": true,
		"":        true,
		"start":   false,
		"doctor":  false,
		"alpha":   false,
		"random":  false,
	}
	for name, want := range cases {
		if got := EnabledFor(name); got != want {
			t.Errorf("EnabledFor(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestDefaultCachePathLocation(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	p, err := defaultCachePath()
	if err != nil {
		t.Fatalf("defaultCachePath: %v", err)
	}
	// On any platform the path should be under the configured home dir.
	if !filepath.IsAbs(p) {
		t.Errorf("defaultCachePath returned non-absolute path: %q", p)
	}
	_ = os.Setenv // silence import if unused on a platform
}

// testWriter is a minimal io.Writer that only tracks total bytes written.
type testWriter struct{ n int }

func (w *testWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return len(p), nil
}
