package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/api/v2/apiv2oauth"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestDefaultScopes(t *testing.T) {
	require.Equal(t, apiv2oauth.CLIScopes(), DefaultScopes())
}

func TestIssuer(t *testing.T) {
	tests := map[string]struct {
		host    string
		want    string
		wantErr string
	}{
		"loopback HTTP": {
			host: "http://127.0.0.1:8090/api/v2",
			want: "http://127.0.0.1:8090",
		},
		"remote HTTPS": {
			host: "https://api.example.com/api/v2",
			want: "https://api.example.com",
		},
		"remote HTTP": {
			host:    "http://api.example.com/api/v2",
			wantErr: "HTTP is allowed only for loopback addresses",
		},
		"unsupported scheme": {
			host:    "file://localhost/tmp/api",
			wantErr: "must use HTTPS",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("INNGEST_API_HOST", test.host)
			issuer, err := Issuer()
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, issuer)
		})
	}
}

func TestAccessTokenChecksResourceBeforeCredentialStore(t *testing.T) {
	dir := t.TempDir()
	store := newStore(dir, newMemoryKeyring())
	metadata, _ := testSession("session-1")
	metadata.Storage = storageKeyring
	require.NoError(t, writeJSONFile(filepath.Join(dir, metadataFile), metadata, 0o600))
	manager := &Manager{store: store, now: time.Now}

	_, _, err := manager.AccessToken(context.Background(), "https://example.com/v2")
	require.ErrorContains(t, err, "different API host")
}

func TestAccessTokenRefreshesWithinLeeway(t *testing.T) {
	now := time.Now().UTC()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, ClientID, r.Form.Get("client_id"))
		require.Equal(t, "old-refresh", r.Form.Get("refresh_token"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"resource":      server.URL + "/v2",
		}))
	}))
	defer server.Close()

	store := newStore(t.TempDir(), newMemoryKeyring())
	metadata, credential := testSession("session-1")
	metadata.Issuer = server.URL
	metadata.Resource = server.URL + "/v2"
	metadata.SessionExpiresAt = now.Add(time.Hour)
	credential.AccessToken = "old-access"
	credential.RefreshToken = "old-refresh"
	credential.Expiry = now.Add(20 * time.Second)
	require.NoError(t, store.Save(metadata, credential, false))
	manager := &Manager{store: store, httpClient: server.Client(), now: func() time.Time { return now }}

	accessToken, _, err := manager.AccessToken(context.Background(), metadata.Resource)

	require.NoError(t, err)
	require.Equal(t, "new-access", accessToken)
	_, stored, err := store.Load()
	require.NoError(t, err)
	require.Equal(t, "new-refresh", stored.RefreshToken)
}

func TestMetadataFromTokenRejectsDifferentResource(t *testing.T) {
	token := (&oauth2.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
	}).WithExtra(map[string]any{
		"resource":           "https://other.example.com/v2",
		"session_id":         "session-id",
		"session_expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
		"account_id":         "account-id",
	})

	_, _, err := MetadataFromToken("https://api.example.com", "https://api.example.com/v2", token)

	require.ErrorContains(t, err, "different resource")
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
