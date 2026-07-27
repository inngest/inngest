package clistate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/require"
)

// writeState points ~ at a temp dir so tests never read the real state file.
func writeState(t *testing.T, state State) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	// go-homedir caches the resolved home dir, so $HOME alone isn't enough.
	homedir.DisableCache = true
	homedir.Reset()
	t.Cleanup(func() {
		homedir.DisableCache = false
		homedir.Reset()
	})

	dir := filepath.Join(home, ".config", "inngest")
	require.NoError(t, os.MkdirAll(dir, 0755))
	byt, err := json.Marshal(state)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "state"), byt, 0600))
}

func TestStoredCredentialsIgnoresEnvOverride(t *testing.T) {
	ctx := context.Background()
	writeState(t, State{Credentials: []byte("stored-key")})
	t.Setenv(EnvApiKey, "env-key")

	state, err := GetState(ctx)
	require.NoError(t, err)
	require.Equal(t, "env-key", string(state.Credentials))

	// Diverging from GetState here is what keeps logout from revoking a key
	// the CLI didn't mint.
	creds, err := StoredCredentials(ctx)
	require.NoError(t, err)
	require.Equal(t, "stored-key", string(creds))
}

func TestStoredCredentialsEmptyWithoutLogin(t *testing.T) {
	ctx := context.Background()
	writeState(t, State{})
	t.Setenv(EnvApiKey, "env-key")

	creds, err := StoredCredentials(ctx)
	require.NoError(t, err)
	require.Empty(t, creds)
}
