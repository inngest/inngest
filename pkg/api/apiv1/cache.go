package apiv1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	rcache "github.com/eko/gocache/store/rueidis/v4"
	"github.com/go-chi/chi/v5"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/pquerna/cachecontrol/cacheobject"
	"github.com/redis/rueidis"
)

func NewRedisCacheMiddleware(r rueidis.Client) CachingMiddleware[string] {
	store := rcache.NewRueidis(r)
	cache := cache.New[string](store)
	return NewCacheMiddleware(cache)
}

func NewCacheMiddleware[T ~string | ~[]byte](cache *cache.Cache[T]) CachingMiddleware[T] {
	return cacheMiddleware[T]{cache: cache}
}

type CachingMiddleware[T ~string | ~[]byte] interface {
	Middleware(next http.Handler) http.Handler
}

type cacheMiddleware[T ~string | ~[]byte] struct {
	cache *cache.Cache[T]
}

func (c cacheMiddleware[T]) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		key := cacheKey(r)
		if key == nil {
			// Not cacheable as we have no cache key, so pass through.
			next.ServeHTTP(w, r)
			return
		}

		route := chi.RouteContext(ctx).RoutePattern()

		// Check the cache.
		// XXX: We need to support both types because freecache does not support storing strings
		// and gocache's `Get` is buggy with []byte.
		// https://github.com/eko/gocache/issues/197#issuecomment-1679348756
		resp, err := c.cache.Get(ctx, *key)
		if err == nil {
			var data []byte
			switch v := any(resp).(type) {
			case string:
				if len(v) > 0 {
					data = []byte(v)
				}
			case []byte:
				if len(v) > 0 {
					data = v
				}
			}

			if len(data) > 0 {
				metrics.IncrAPICacheHit(ctx, metrics.CounterOpt{
					PkgName: pkgName,
					Tags:    map[string]any{"route": route},
				})
				_, _ = w.Write(data)
				return
			}
		} else if !errors.Is(err, store.NotFound{}) {
			logger.StdlibLogger(ctx).Error("error retrieving cache", "error", err, "key", key)
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

		var cacheValue T
		switch any(cacheValue).(type) {
		case string:
			cacheValue = any(rec.Body.String()).(T)
		case []byte:
			cacheValue = any(body).(T)
		}
		if err := c.cache.Set(ctx, *key, cacheValue, store.WithExpiration(time.Duration(maxAge)*time.Second)); err != nil {
			logger.StdlibLogger(ctx).Warn("error caching api response", "error", err)
		}
		metrics.IncrAPICacheMiss(ctx, metrics.CounterOpt{
			PkgName: pkgName,
			Tags:    map[string]any{"route": route},
		})
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
