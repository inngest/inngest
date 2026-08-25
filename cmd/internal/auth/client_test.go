package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultScopes(t *testing.T) {
	require.Equal(t, []string{
		"accounts:read:*",
		"apps:read:*",
		"apps:write:*",
		"environments:read:*",
		"environments:write:*",
		"events:write:*",
		"experiments:read:*",
		"functions:read:*",
		"functions:write:*",
		"insights:read:*",
		"runs:read:*",
		"runs:write:*",
		"sandboxes:read:*",
		"sandboxes:write:*",
		"sessions:read:*",
		"webhooks:read:*",
		"webhooks:write:*",
	}, DefaultScopes())
}

func TestIssuer(t *testing.T) {
	t.Setenv("INNGEST_API_HOST", "http://127.0.0.1:8090/api/v2")
	issuer, err := Issuer()
	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:8090", issuer)
}

func TestAccessTokenChecksResourceBeforeCredentialStore(t *testing.T) {
	dir := t.TempDir()
	store := newStore(dir, newMemoryKeyring())
	metadata, _ := testSession("session-1")
	metadata.Storage = storageKeyring
	require.NoError(t, writeJSONFile(filepath.Join(dir, metadataFile), metadata, 0o600))
	manager := &Manager{store: store, now: time.Now}

	_, _, err := manager.AccessToken(context.Background(), "https://example.com/v2")
	require.ErrorIs(t, err, ErrNotLoggedIn)
}

func TestValidateAcceptsAuthorizedAndPermissionDeniedResponses(t *testing.T) {
	tests := map[string]struct {
		status  int
		wantErr bool
	}{
		"success":           {status: http.StatusOK},
		"permission denied": {status: http.StatusForbidden},
		"invalid token":     {status: http.StatusUnauthorized, wantErr: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
			}))
			defer server.Close()
			manager := &Manager{httpClient: server.Client()}

			err := manager.Validate(context.Background(), &Metadata{Resource: server.URL}, "token")
			require.Equal(t, test.wantErr, err != nil)
		})
	}
}

func TestCanonicalResourceSupportsCloudAndLocalV2Aliases(t *testing.T) {
	tests := map[string]struct {
		left  string
		right string
		equal bool
	}{
		"cloud": {
			left:  "https://api.inngest.com/v2",
			right: "https://api.inngest.com/v2/",
			equal: true,
		},
		"local alias": {
			left:  "http://localhost:8090/api/v2",
			right: "http://localhost:8090/v2",
			equal: true,
		},
		"different host": {
			left:  "https://api.inngest.com/v2",
			right: "https://example.com/v2",
			equal: false,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.equal, canonicalResource(test.left) == canonicalResource(test.right))
		})
	}
}
