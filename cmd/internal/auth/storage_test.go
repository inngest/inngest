package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

type memoryKeyring struct {
	values map[string]string
}

func newMemoryKeyring() *memoryKeyring {
	return &memoryKeyring{values: map[string]string{}}
}

func (m *memoryKeyring) Get(service, user string) (string, error) {
	value, ok := m.values[service+"\x00"+user]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return value, nil
}

func (m *memoryKeyring) Set(service, user, password string) error {
	m.values[service+"\x00"+user] = password
	return nil
}

func (m *memoryKeyring) Delete(service, user string) error {
	key := service + "\x00" + user
	if _, ok := m.values[key]; !ok {
		return keyring.ErrNotFound
	}
	delete(m.values, key)
	return nil
}

func TestStoreKeyringKeepsSecretsOutOfMetadata(t *testing.T) {
	store := newStore(t.TempDir(), newMemoryKeyring())
	metadata, credential := testSession("session-1")

	require.NoError(t, store.Save(metadata, credential, false))

	storedMetadata, storedCredential, err := store.Load()
	require.NoError(t, err)
	require.Equal(t, metadata.SessionID, storedMetadata.SessionID)
	require.Equal(t, credential, *storedCredential)

	data, err := os.ReadFile(filepath.Join(store.dir, metadataFile))
	require.NoError(t, err)
	require.NotContains(t, string(data), credential.AccessToken)
	require.NotContains(t, string(data), credential.RefreshToken)
}

func TestStoreReplacesCredentialAcrossBackends(t *testing.T) {
	keyringStore := newMemoryKeyring()
	store := newStore(t.TempDir(), keyringStore)
	firstMetadata, firstCredential := testSession("session-1")
	secondMetadata, secondCredential := testSession("session-2")

	require.NoError(t, store.Save(firstMetadata, firstCredential, false))
	firstUser := keyringUser(&firstMetadata)
	require.Contains(t, keyringStore.values, keyringService+"\x00"+firstUser)

	require.NoError(t, store.Save(secondMetadata, secondCredential, true))
	require.NotContains(t, keyringStore.values, keyringService+"\x00"+firstUser)

	storedMetadata, storedCredential, err := store.Load()
	require.NoError(t, err)
	require.Equal(t, storageFile, storedMetadata.Storage)
	require.Equal(t, secondCredential, *storedCredential)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(store.credentialPath(storedMetadata))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestStoreRejectsReadableCredentialFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
	store := newStore(t.TempDir(), newMemoryKeyring())
	metadata, credential := testSession("session-1")
	require.NoError(t, store.Save(metadata, credential, true))
	require.NoError(t, os.Chmod(store.credentialPath(&metadata), 0o644))

	_, _, err := store.Load()
	require.ErrorContains(t, err, "permissions must be 0600")
}

func TestStoreDeleteWorksWhenCredentialIsMissing(t *testing.T) {
	store := newStore(t.TempDir(), newMemoryKeyring())
	metadata, credential := testSession("session-1")
	require.NoError(t, store.Save(metadata, credential, true))
	require.NoError(t, os.Remove(store.credentialPath(&metadata)))

	require.NoError(t, store.Delete(nil))
	_, err := store.Metadata()
	require.ErrorIs(t, err, ErrNotLoggedIn)
}

func TestStoreRequiresCompleteSession(t *testing.T) {
	store := newStore(t.TempDir(), newMemoryKeyring())
	metadata, credential := testSession("")

	err := store.Save(metadata, credential, false)
	require.ErrorContains(t, err, "incomplete")
}

func testSession(sessionID string) (Metadata, Credential) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	return Metadata{
			Issuer:           "https://api.inngest.com",
			Resource:         "https://api.inngest.com/v2",
			ClientID:         ClientID,
			SessionID:        sessionID,
			SessionExpiresAt: now.Add(30 * 24 * time.Hour),
			AccountID:        "account-id",
		}, Credential{
			AccessToken:  "inngest_at_secret",
			RefreshToken: "inngest_rt_secret",
			TokenType:    "Bearer",
			Expiry:       now.Add(time.Hour),
		}
}
