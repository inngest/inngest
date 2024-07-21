package apiv1

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	rcache "github.com/eko/gocache/store/rueidis/v4"
	"github.com/inngest/inngest/pkg/headers"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/pquerna/cachecontrol/cacheobject"
	"github.com/redis/rueidis"
)

func NewRedisCacheMiddleware(
	r rueidis.Client,
	serverKind string,
) CachingMiddleware {
	store := rcache.NewRueidis(r)
	cache := cache.New[[]byte](store)
	return NewCacheMiddleware(cache, serverKind)
}

func NewCacheMiddleware(
	cache *cache.Cache[[]byte],
	serverKind string,
) CachingMiddleware {
	return cacheMiddleware{
		// Allow cache skipping for the Dev Server so that SDK tests can run
		// faster. Never allow cache skipping in Cloud
		allowCacheSkip: serverKind == headers.ServerKindDev,

		cache: cache,
	}
}

type CachingMiddleware interface {
	Middleware(next http.Handler) http.Handler
}

type cacheMiddleware struct {
	allowCacheSkip bool
	cache          *cache.Cache[[]byte]
}

func (c cacheMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		key := cacheKey(r)
		if key == nil {
			// Not cacheable as we have no cache key, so pass through.
			next.ServeHTTP(w, r)
			return
		}

		// Cache skipping must only be allowed in the Dev Server
		checkCache := true
		if c.allowCacheSkip && r.Header.Get(headers.HeaderKeySkipCache) == "true" {
			checkCache = false
		}

		if checkCache {
			// Check the cache.
			resp, err := c.cache.Get(ctx, *key)
			if err == nil && len(resp) > 0 {
				// TODO: Metrics on cache hit.
				_, _ = w.Write(resp)
				return
			}
		}

		// Record the response from the handler itself.
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)

		// Write headers to the actual response, inspecting the max-age HTTP
		// header which determines cacheability in the result itself.
		maxAge := int32(0)
		for key, result := range rec.Result().Header {
			if key == "Cache-Control" && len(result) == 1 {
				if res, err := cacheobject.ParseResponseCacheControl(result[0]); err == nil {
					maxAge = int32(res.MaxAge)
				}
			}
			for _, item := range result {
				w.Header().Add(key, item)
			}
		}

		if maxAge == 0 {
			// If there's no max-age header, we cannot cache the response.
			// Ignore.
			if _, err := io.Copy(w, rec.Body); err != nil {
				logger.StdlibLogger(ctx).Error("error writing api response", "error", err)
			}
			return
		}

		// Read the response into a byte stream;  we need to write this and cache
		// it.
		body := rec.Body.Bytes()
		if _, err := w.Write(body); err != nil {
			logger.StdlibLogger(ctx).Error("error writing api response", "error", err)
			return
		}

		if err := c.cache.Set(ctx, *key, body, store.WithExpiration(time.Duration(maxAge)*time.Second)); err != nil {
			logger.StdlibLogger(ctx).Warn("error caching api response", "error", err)
		}
		// TODO: Metrics on cache miss.
	})
}

func cacheKey(r *http.Request) *string {
	if r.Method != http.MethodGet {
		return nil
	}

	// This is a simple check for now.  We're not quite respecting RFC7234; we're
	// adding a max-age header then handling this internally.
	//
	// In the future we can implement a heavyweight caching library such as
	// https://github.com/darkweak/souin.
	//
	// Note that we still want auth'd requests to be cached, against the spec's standards.
	key := r.URL.String()
	auth := r.Header.Get("Authorization")
	sum := sha256.Sum256([]byte(key + auth))
	hash := hex.EncodeToString(sum[:])
	return &hash
}
