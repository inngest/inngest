package apiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coocood/freecache"
	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	freecachestore "github.com/eko/gocache/store/freecache/v4"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/inngest/inngest/pkg/api/apiv1"
	"github.com/stretchr/testify/require"
)

func newCacheTestHandler(t *testing.T, next http.Handler) http.Handler {
	t.Helper()
	c := cache.New[[]byte](freecachestore.NewFreecache(freecache.NewCache(1024 * 1024)))
	return apiv1.NewCacheMiddleware(c).Middleware(next)
}

func newCacheTestRequest(method, target, authorization string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Authorization", authorization)
	routeCtx := chi.NewRouteContext()
	routeCtx.RoutePatterns = append(routeCtx.RoutePatterns, "/v1/test")
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestCacheMiddlewareForwardsUncachedResponse(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{name: "OK", status: http.StatusOK},
		{name: "bad request", status: http.StatusBadRequest},
		{name: "not found", status: http.StatusNotFound},
		{name: "rate limited", status: http.StatusTooManyRequests},
		{name: "internal server error", status: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newCacheTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-Test-Header", "preserved")
				w.WriteHeader(tt.status)
				_, err := w.Write([]byte("response body"))
				require.NoError(t, err)
			}))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, newCacheTestRequest(http.MethodGet, "/v1/test", "key"))

			require.Equal(t, tt.status, rec.Code)
			require.Equal(t, "preserved", rec.Header().Get("X-Test-Header"))
			require.Equal(t, "response body", rec.Body.String())
		})
	}
}

func TestCacheMiddlewareCachesOnlyOKResponses(t *testing.T) {
	t.Run("caches an OK response", func(t *testing.T) {
		calls := 0
		handler := newCacheTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			w.Header().Set("Cache-Control", "private, max-age=60")
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprintf(w, `{"call":%d}`, calls)
			require.NoError(t, err)
		}))

		first := httptest.NewRecorder()
		handler.ServeHTTP(first, newCacheTestRequest(http.MethodGet, "/v1/test", "key"))
		second := httptest.NewRecorder()
		handler.ServeHTTP(second, newCacheTestRequest(http.MethodGet, "/v1/test", "key"))

		require.Equal(t, 1, calls)
		require.Equal(t, http.StatusOK, first.Code)
		require.Equal(t, `{"call":1}`, first.Body.String())
		require.Equal(t, "application/json", first.Header().Get("Content-Type"))
		require.Equal(t, http.StatusOK, second.Code)
		require.Equal(t, `{"call":1}`, second.Body.String())
		require.Equal(t, "application/json", second.Header().Get("Content-Type"))
		require.Empty(t, second.Header().Get("Cache-Control"))
	})

	for _, status := range []int{http.StatusCreated, http.StatusNotFound} {
		t.Run(fmt.Sprintf("does not cache status %d", status), func(t *testing.T) {
			calls := 0
			handler := newCacheTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.Header().Set("Cache-Control", "private, max-age=60")
				w.WriteHeader(status)
				_, err := fmt.Fprintf(w, "call %d", calls)
				require.NoError(t, err)
			}))

			first := httptest.NewRecorder()
			handler.ServeHTTP(first, newCacheTestRequest(http.MethodGet, "/v1/test", "key"))
			second := httptest.NewRecorder()
			handler.ServeHTTP(second, newCacheTestRequest(http.MethodGet, "/v1/test", "key"))

			require.Equal(t, 2, calls)
			require.Equal(t, status, first.Code)
			require.Equal(t, "call 1", first.Body.String())
			require.Equal(t, status, second.Code)
			require.Equal(t, "call 2", second.Body.String())
		})
	}
}

func TestCacheMiddlewareSupportsStringCacheValues(t *testing.T) {
	cacheStore := store.NewMockStoreInterface(gomock.NewController(t))
	cacheStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, store.NotFound{})
	cacheStore.EXPECT().Set(gomock.Any(), gomock.Any(), `{"cached":true}`, gomock.Any()).Return(nil)
	cacheStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return(`{"cached":true}`, nil)
	handler := apiv1.NewCacheMiddleware(cache.New[string](cacheStore)).Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=60")
		_, err := w.Write([]byte(`{"cached":true}`))
		require.NoError(t, err)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), newCacheTestRequest(http.MethodGet, "/v1/test", "key"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, newCacheTestRequest(http.MethodGet, "/v1/test", "key"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.Equal(t, `{"cached":true}`, rec.Body.String())
}

func TestCacheMiddlewareDoesNotCacheWithoutMaxAge(t *testing.T) {
	calls := 0
	handler := newCacheTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, err := fmt.Fprintf(w, "call %d", calls)
		require.NoError(t, err)
	}))

	for range 2 {
		handler.ServeHTTP(httptest.NewRecorder(), newCacheTestRequest(http.MethodGet, "/v1/test", "key"))
	}

	require.Equal(t, 2, calls)
}

func TestCacheMiddlewareBypassesNonGETRequests(t *testing.T) {
	calls := 0
	handler := newCacheTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Cache-Control", "private, max-age=60")
		w.WriteHeader(http.StatusCreated)
	}))

	for range 2 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newCacheTestRequest(http.MethodPost, "/v1/test", "key"))
		require.Equal(t, http.StatusCreated, rec.Code)
	}

	require.Equal(t, 2, calls)
}

func TestCacheMiddlewareKeyIncludesURLAndAuthorization(t *testing.T) {
	calls := 0
	handler := newCacheTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Cache-Control", "private, max-age=60")
		_, err := fmt.Fprintf(w, "call %d", calls)
		require.NoError(t, err)
	}))

	tests := []struct {
		target        string
		authorization string
		wantBody      string
	}{
		{target: "/v1/test?value=a", authorization: "key-a", wantBody: "call 1"},
		{target: "/v1/test?value=a", authorization: "key-a", wantBody: "call 1"},
		{target: "/v1/test?value=b", authorization: "key-a", wantBody: "call 2"},
		{target: "/v1/test?value=a", authorization: "key-b", wantBody: "call 3"},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, newCacheTestRequest(http.MethodGet, tt.target, tt.authorization))
		require.Equal(t, tt.wantBody, rec.Body.String())
	}
	require.Equal(t, 3, calls)
}
