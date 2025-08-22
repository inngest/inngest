package base_cqrs

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/cqrs"
	sqlc_psql "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCQRSWrapper(t *testing.T) {
	cm, cleanup := initSQLiteCQRS(t)
	defer cleanup()

	t.Run("GetFunctionByInternalUUID", func(t *testing.T) {
		ctx := context.Background()

		// Generate test IDs
		accountID := uuid.New()
		envID := uuid.New()
		appID := uuid.New()
		fnID := uuid.New()

		// Upsert the app first
		_, err := cm.UpsertApp(ctx, cqrs.UpsertAppParams{
			ID:   appID,
			Name: "test-app",
		})
		require.NoError(t, err)

		// Create function config
		fnConfig := map[string]any{
			"triggers": []map[string]any{
				{"event": "test.event"},
			},
		}
		configJSON, err := json.Marshal(fnConfig)
		require.NoError(t, err)

		// Insert the function
		_, err = cm.InsertFunction(ctx, cqrs.InsertFunctionParams{
			ID:        fnID,
			AccountID: accountID,
			EnvID:     envID,
			AppID:     appID,
			Name:      "Test Function",
			Slug:      "test-function",
			Config:    string(configJSON),
			CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		// Test GetFunctionByInternalUUID
		function, err := cm.GetFunctionByInternalUUID(ctx, fnID)
		require.NoError(t, err)
		require.NotNil(t, function)

		// Verify function properties
		assert.Equal(t, fnID, function.ID)
		// assert.Equal(t, envID, function.EnvID)
		assert.Equal(t, appID, function.AppID)
		assert.Equal(t, "Test Function", function.Name)
		assert.Equal(t, "test-function", function.Slug)
		assert.NotEmpty(t, function.Config)
		assert.False(t, function.CreatedAt.IsZero())
		assert.Nil(t, function.ArchivedAt)

		// Verify function config can be unmarshaled
		var config map[string]any
		err = json.Unmarshal(function.Config, &config)
		require.NoError(t, err)
		assert.NotEmpty(t, config["triggers"])

		// Test non-existent function
		nonExistentID := uuid.New()
		_, err = cm.GetFunctionByInternalUUID(ctx, nonExistentID)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func initSQLiteCQRS(t *testing.T) (cqrs.Manager, func()) {
	db, err := New(BaseCQRSOptions{InMemory: true})
	require.NoError(t, err)

	cm := NewCQRS(db, "sqlite", sqlc_psql.NewNormalizedOpts{})

	cleanup := func() {
		db.Close()
	}

	return cm, cleanup
}
