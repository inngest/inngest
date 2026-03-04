package responsecache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
)

const (
	DefaultTTL             = 2 * time.Minute
	DefaultCleanupInterval = 30 * time.Second
)

// DiskCacheOpt configures a DiskCache.
type DiskCacheOpt struct {
	// Dir is the base directory for cache files.
	// Defaults to os.TempDir()/inngest-response-cache.
	Dir string

	// TTL controls how long cached responses are kept.  Files older than
	// this are removed by the background cleanup goroutine.
	// Defaults to 2 minutes.
	TTL time.Duration

	// CleanupInterval controls how often the cleanup goroutine runs.
	// Defaults to 30 seconds.
	CleanupInterval time.Duration

	Logger logger.Logger
}

// DiskCache implements ResponseCache by writing serialized responses to
// individual files on disk.  A background goroutine periodically removes
// files older than the configured TTL so that the disk doesn't fill up.
//
// In production this directory will typically be backed by a Kubernetes
// ephemeral volume.
type DiskCache struct {
	dir             string
	ttl             time.Duration
	cleanupInterval time.Duration
	done            chan struct{}
	log             logger.Logger
}

// NewDiskCache creates a DiskCache rooted at the given directory.  The
// directory is created if it does not exist.  A background goroutine is
// started immediately to clean up expired entries.
func NewDiskCache(opts DiskCacheOpt) (*DiskCache, error) {
	if opts.Dir == "" {
		opts.Dir = filepath.Join(os.TempDir(), "inngest-response-cache")
	}
	if opts.TTL == 0 {
		opts.TTL = DefaultTTL
	}
	if opts.CleanupInterval == 0 {
		opts.CleanupInterval = DefaultCleanupInterval
	}

	if err := os.MkdirAll(opts.Dir, 0700); err != nil {
		return nil, fmt.Errorf("creating response cache dir: %w", err)
	}

	dc := &DiskCache{
		dir:             opts.Dir,
		ttl:             opts.TTL,
		cleanupInterval: opts.CleanupInterval,
		done:            make(chan struct{}),
		log:             opts.Logger,
	}

	go dc.cleanupLoop()

	return dc, nil
}

// Get reads a cached response from disk.  Returns (nil, nil) on cache miss.
// Corrupt files are silently removed and treated as a miss.
func (dc *DiskCache) Get(_ context.Context, key string) (*state.DriverResponse, error) {
	path := dc.pathFor(key)

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading cached response: %w", err)
	}

	resp, err := deserialize(data)
	if err != nil {
		// Corrupt entry — delete and treat as miss.
		_ = os.Remove(path)
		return nil, nil
	}

	return resp, nil
}

// Set writes a response to disk atomically (write tmp + rename).
func (dc *DiskCache) Set(_ context.Context, key string, resp *state.DriverResponse) error {
	data, err := serialize(resp)
	if err != nil {
		return fmt.Errorf("serializing response: %w", err)
	}

	path := dc.pathFor(key)
	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming cache file: %w", err)
	}

	return nil
}

// Close stops the background cleanup goroutine.
func (dc *DiskCache) Close() error {
	close(dc.done)
	return nil
}

// pathFor returns the file path for a given cache key.  The key is hashed
// with SHA-256 to produce a safe, fixed-length filename.
func (dc *DiskCache) pathFor(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(dc.dir, fmt.Sprintf("%x.json", h))
}

func (dc *DiskCache) cleanupLoop() {
	ticker := time.NewTicker(dc.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dc.done:
			return
		case <-ticker.C:
			dc.cleanup()
		}
	}
}

func (dc *DiskCache) cleanup() {
	entries, err := os.ReadDir(dc.dir)
	if err != nil {
		if dc.log != nil {
			dc.log.Warn("response cache cleanup: read dir error", "error", err)
		}
		return
	}

	cutoff := time.Now().Add(-dc.ttl)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dc.dir, entry.Name()))
		}
	}
}

// --- serialization helpers ---

// cachedResponse is the on-disk JSON envelope.  It stores the marshalled
// DriverResponse together with the unexported "final" flag which cannot
// survive a plain json.Marshal round-trip.
type cachedResponse struct {
	Response json.RawMessage `json:"response"`
	Final    bool            `json:"final"`
}

func serialize(resp *state.DriverResponse) ([]byte, error) {
	// Normalise Output to json.RawMessage so that it round-trips
	// deterministically (the field is typed as `any`).
	if resp.Output != nil {
		normalised, err := outputToRaw(resp.Output)
		if err != nil {
			return nil, fmt.Errorf("normalising output: %w", err)
		}
		// Work on a shallow copy so we don't mutate the caller's
		// response.
		clone := *resp
		clone.Output = normalised
		resp = &clone
	}

	raw, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}

	env := cachedResponse{
		Response: raw,
		Final:    resp.Final(),
	}
	return json.Marshal(env)
}

func deserialize(data []byte) (*state.DriverResponse, error) {
	var env cachedResponse
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	var resp state.DriverResponse
	if err := json.Unmarshal(env.Response, &resp); err != nil {
		return nil, err
	}

	if env.Final {
		resp.SetFinal()
	}

	return &resp, nil
}

// outputToRaw converts the DriverResponse.Output (typed as `any`) into a
// json.RawMessage so that JSON marshal/unmarshal round-trips cleanly.
func outputToRaw(v any) (json.RawMessage, error) {
	switch t := v.(type) {
	case json.RawMessage:
		return t, nil
	case []byte:
		return json.RawMessage(t), nil
	case string:
		return json.RawMessage(t), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(b), nil
	}
}
