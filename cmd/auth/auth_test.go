package auth

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest/client"
	"github.com/inngest/inngest/pkg/inngest/clistate"
	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/require"
)

// TestLogoutKeepsIdentity pins the invariant that logout clears the session but
// not the identity telemetry reports.
func TestLogoutKeepsIdentity(t *testing.T) {
	ctx := context.Background()
	accountID, userID := uuid.New(), uuid.New()

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
	byt, err := json.Marshal(clistate.State{
		ClientID: uuid.New(),
		// No credentials, so logout skips the server-side revoke.
		Account: client.Account{ID: accountID, Name: "Test Account"},
		UserID:  userID,
		Env:     "Staging",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "state"), byt, 0600))

	require.NoError(t, logout(ctx, nil))

	state, err := clistate.GetState(ctx)
	require.NoError(t, err)
	require.Equal(t, accountID, state.Account.ID)
	require.Equal(t, userID, state.UserID)
	require.Empty(t, state.Credentials)
	require.Empty(t, state.Account.Name)
	require.Empty(t, state.Env)
}
