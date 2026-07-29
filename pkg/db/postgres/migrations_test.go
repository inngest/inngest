package postgres

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigurePool(t *testing.T) {
	// sql.Open is lazy, so no live server is needed to assert pool settings.
	open := func(t *testing.T) *sql.DB {
		conn, err := sql.Open("pgx", "postgres://user:pass@localhost:5432/inngest")
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, conn.Close()) })
		return conn
	}

	t.Run("zero options leave driver defaults", func(t *testing.T) {
		conn := open(t)
		configurePool(conn, Options{})
		// database/sql reports 0 for an unbounded pool.
		require.Equal(t, 0, conn.Stats().MaxOpenConnections)
	})

	t.Run("applies explicit options", func(t *testing.T) {
		conn := open(t)
		configurePool(conn, Options{
			MaxIdleConns:    3,
			MaxOpenConns:    7,
			ConnMaxIdleTime: time.Minute,
			ConnMaxLifetime: time.Hour,
		})
		require.Equal(t, 7, conn.Stats().MaxOpenConnections)
	})
}
