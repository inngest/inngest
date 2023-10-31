package apiv1

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	rcache "github.com/eko/gocache/store/rueidis/v4"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/redis/rueidis"
)

func NewRedisCacheMiddleware(r rueidis.Client) CachingMiddleware {
	store := rcache.NewRueidis(r)
	cache := cache.New[[]byte](store)
	return NewCacheMiddleware(cache)
}

func NewCacheMiddleware(cache *cache.Cache[[]byte]) CachingMiddleware {
	return cacheMiddleware{cache: cache}
}

type CachingMiddleware interface {
	Middleware(next http.Handler) http.Handler
}

type cacheMiddleware struct {
	cache *cache.Cache[[]byte]
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

		// Check the cache.
		resp, err := c.cache.Get(ctx, *key)
		if err == nil && len(resp) > 0 {
			// This response is cached.
			// TODO: Metrics on cache hit.
			_, _ = w.Write(resp)
			return
		}

		// Record the response from the handler itself.
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)

		// Write headers to the actual response, inspecting the max-age HTTP
		// header which determines cacheability in the result itself.
		maxAge := 0
		for key, result := range rec.Result().Header {
			if key == "Max-Age" && len(result) == 1 {
				maxAge, _ = strconv.Atoi(result[0])
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

func writeResponse(rec *httptest.ResponseRecorder, w http.ResponseWriter) ([]byte, error) {
	byt := rec.Body.Bytes()

	_, err := w.Write(byt)
	return byt, err
}
