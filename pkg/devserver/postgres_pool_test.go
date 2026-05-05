package devserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresPoolOptions(t *testing.T) {
	t.Run("nil without postgres", func(t *testing.T) {
		assert.Nil(t, postgresPoolOptions(StartOpts{}))
	})

	t.Run("nil when postgres pool is left unspecified", func(t *testing.T) {
		assert.Nil(t, postgresPoolOptions(StartOpts{PostgresURI: "postgres://user:pass@localhost/db"}))
	})

	t.Run("maps configured values", func(t *testing.T) {
		pool := postgresPoolOptions(StartOpts{
			PostgresURI:             "postgres://user:pass@localhost/db",
			PostgresMaxIdleConns:    17,
			PostgresMaxOpenConns:    171,
			PostgresConnMaxIdleTime: 11,
			PostgresConnMaxLifetime: 61,
		})
		require.NotNil(t, pool)

		assert.Equal(t, 17, pool.MaxIdleConns)
		assert.Equal(t, 171, pool.MaxOpenConns)
		assert.Equal(t, 11, pool.ConnMaxIdleTime)
		assert.Equal(t, 61, pool.ConnMaxLifetime)
	})
}
