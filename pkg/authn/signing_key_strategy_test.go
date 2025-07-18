package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleSigningKey(t *testing.T) {
	trustedKey := "signkey-test-abc123def456"
	trustedKeyNormalized := "abc123def456"
	hashedTrusted, err := HashedSigningKey(trustedKeyNormalized)
	require.NoError(t, err)

	t.Run("should accept valid plain text key", func(t *testing.T) {
		authCtx, err := HandleSigningKey(context.Background(), trustedKey, trustedKey)
		require.NoError(t, err)
		require.NotNil(t, authCtx)
		require.True(t, authCtx.isAuthenticated)
	})

	t.Run("should accept valid normalized key", func(t *testing.T) {
		authCtx, err := HandleSigningKey(context.Background(), trustedKeyNormalized, trustedKey)
		require.NoError(t, err)
		require.NotNil(t, authCtx)
		require.True(t, authCtx.isAuthenticated)
	})

	t.Run("should accept valid hashed key", func(t *testing.T) {
		authCtx, err := HandleSigningKey(context.Background(), hashedTrusted, trustedKey)
		require.NoError(t, err)
		require.NotNil(t, authCtx)
		require.True(t, authCtx.isAuthenticated)
	})

	t.Run("should accept hashed key with prefix", func(t *testing.T) {
		hashedWithPrefix := "signkey-prod-" + hashedTrusted
		authCtx, err := HandleSigningKey(context.Background(), hashedWithPrefix, trustedKey)
		require.NoError(t, err)
		require.NotNil(t, authCtx)
		require.True(t, authCtx.isAuthenticated)
	})

	t.Run("should reject invalid key", func(t *testing.T) {
		invalidKey := "signkey-test-wrongkey123"
		authCtx, err := HandleSigningKey(context.Background(), invalidKey, trustedKey)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid signing key")
		require.Nil(t, authCtx)
	})

	t.Run("should reject empty client key", func(t *testing.T) {
		authCtx, err := HandleSigningKey(context.Background(), "", trustedKey)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid signing key")
		require.Nil(t, authCtx)
	})

	t.Run("should accept key with different prefix", func(t *testing.T) {
		differentPrefix := "signkey-prod-abc123def456"
		authCtx, err := HandleSigningKey(context.Background(), differentPrefix, trustedKey)
		require.NoError(t, err)
		require.NotNil(t, authCtx)
		require.True(t, authCtx.isAuthenticated)
	})
}

func TestSigningKeyMiddleware(t *testing.T) {
	trustedKey := "signkey-test-abc123def456"

	t.Run("should pass through when no signing key configured", func(t *testing.T) {
		middleware := SigningKeyMiddleware(nil)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("success"))
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		middleware(handler).ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "success", w.Body.String())
	})

	t.Run("should authenticate with valid signing key in header", func(t *testing.T) {
		middleware := SigningKeyMiddleware(&trustedKey)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx := r.Context().Value(authContextKey)
			require.NotNil(t, authCtx)

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("authenticated"))
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+trustedKey)
		w := httptest.NewRecorder()

		middleware(handler).ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "authenticated", w.Body.String())
	})

	t.Run("should authenticate with hashed signing key", func(t *testing.T) {
		middleware := SigningKeyMiddleware(&trustedKey)

		hashedKey, err := HashedSigningKey("abc123def456")
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("authenticated"))
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+hashedKey)
		w := httptest.NewRecorder()

		middleware(handler).ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "authenticated", w.Body.String())
	})

	t.Run("should reject request with no authorization header", func(t *testing.T) {
		middleware := SigningKeyMiddleware(&trustedKey)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		middleware(handler).ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "Authentication failed")
	})

	t.Run("should reject request with invalid signing key", func(t *testing.T) {
		middleware := SigningKeyMiddleware(&trustedKey)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer signkey-test-wrongkey123")
		w := httptest.NewRecorder()

		middleware(handler).ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "Authentication failed")
	})

	t.Run("should reject request with empty authorization header", func(t *testing.T) {
		middleware := SigningKeyMiddleware(&trustedKey)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "")
		w := httptest.NewRecorder()

		middleware(handler).ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "Authentication failed")
	})

	t.Run("should reject request with malformed authorization header", func(t *testing.T) {
		middleware := SigningKeyMiddleware(&trustedKey)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()

		middleware(handler).ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		require.Contains(t, w.Body.String(), "Authentication failed")
	})
}

func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "should strip test prefix",
			input:    "signkey-test-abc123",
			expected: "abc123",
		},
		{
			name:     "should strip prod prefix",
			input:    "signkey-prod-def456",
			expected: "def456",
		},
		{
			name:     "should strip branch prefix",
			input:    "signkey-branch-ghi789",
			expected: "ghi789",
		},
		{
			name:     "should strip generic prefix",
			input:    "signkey-custom-jkl012",
			expected: "jkl012",
		},
		{
			name:     "should handle key without prefix",
			input:    "abc123def456",
			expected: "abc123def456",
		},
		{
			name:     "should handle empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeKey(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHashedSigningKey(t *testing.T) {
	t.Run("should hash valid hex key", func(t *testing.T) {
		key := "abc123def456"
		hashed, err := HashedSigningKey(key)
		require.NoError(t, err)
		require.NotEmpty(t, hashed)
		require.NotEqual(t, key, hashed)

		// Hash should be deterministic
		hashed2, err := HashedSigningKey(key)
		require.NoError(t, err)
		require.Equal(t, hashed, hashed2)
	})

	t.Run("should handle key with prefix", func(t *testing.T) {
		keyWithPrefix := "signkey-test-abc123def456"
		keyWithoutPrefix := "abc123def456"

		hashed1, err := HashedSigningKey(keyWithPrefix)
		require.NoError(t, err)

		hashed2, err := HashedSigningKey(keyWithoutPrefix)
		require.NoError(t, err)

		require.Equal(t, hashed1, hashed2)
	})

	t.Run("should return error for invalid hex", func(t *testing.T) {
		invalidHex := "xyz123"
		_, err := HashedSigningKey(invalidHex)
		require.Error(t, err)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		hashed, err := HashedSigningKey("")
		require.NoError(t, err)
		require.NotEmpty(t, hashed)
		// Empty string should produce a consistent hash
		expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // SHA256 of empty bytes
		require.Equal(t, expected, hashed)
	})
}
