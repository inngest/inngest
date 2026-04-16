package authn

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBasicAuthMiddleware(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	t.Run("no-op when credentials are not configured", func(t *testing.T) {
		mw := BasicAuthMiddleware("", "", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		mw(okHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("no-op when only username is set", func(t *testing.T) {
		mw := BasicAuthMiddleware("admin", "", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		mw(okHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 when password empty, got %d", rec.Code)
		}
	})

	t.Run("challenges request without credentials", func(t *testing.T) {
		mw := BasicAuthMiddleware("admin", "secret", "Inngest Dashboard")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		mw(okHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		auth := rec.Header().Get("WWW-Authenticate")
		if !strings.Contains(auth, `realm="Inngest Dashboard"`) {
			t.Fatalf("expected WWW-Authenticate to include realm, got %q", auth)
		}
	})

	t.Run("rejects invalid credentials", func(t *testing.T) {
		mw := BasicAuthMiddleware("admin", "secret", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetBasicAuth("admin", "wrong")
		mw(okHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for invalid password, got %d", rec.Code)
		}
	})

	t.Run("rejects wrong username", func(t *testing.T) {
		mw := BasicAuthMiddleware("admin", "secret", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetBasicAuth("wrong", "secret")
		mw(okHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for invalid user, got %d", rec.Code)
		}
	})

	t.Run("passes valid credentials", func(t *testing.T) {
		mw := BasicAuthMiddleware("admin", "secret", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetBasicAuth("admin", "secret")
		mw(okHandler).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("defaults realm when not provided", func(t *testing.T) {
		mw := BasicAuthMiddleware("admin", "secret", "")
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		mw(okHandler).ServeHTTP(rec, req)

		auth := rec.Header().Get("WWW-Authenticate")
		if !strings.Contains(auth, `realm="Inngest"`) {
			t.Fatalf("expected default realm, got %q", auth)
		}
	})
}
