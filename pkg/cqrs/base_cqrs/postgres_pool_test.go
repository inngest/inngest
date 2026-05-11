package base_cqrs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type postgresPoolRecorder struct {
	maxIdleConns    int
	maxOpenConns    int
	connMaxIdleTime time.Duration
	connMaxLifetime time.Duration
}

func (r *postgresPoolRecorder) SetMaxIdleConns(n int) {
	r.maxIdleConns = n
}

func (r *postgresPoolRecorder) SetMaxOpenConns(n int) {
	r.maxOpenConns = n
}

func (r *postgresPoolRecorder) SetConnMaxIdleTime(d time.Duration) {
	r.connMaxIdleTime = d
}

func (r *postgresPoolRecorder) SetConnMaxLifetime(d time.Duration) {
	r.connMaxLifetime = d
}

func TestApplyPostgresPoolOptions(t *testing.T) {
	recorder := &postgresPoolRecorder{}

	applyPostgresPoolOptions(recorder, PostgresPoolOptions{
		MaxIdleConns:    17,
		MaxOpenConns:    171,
		ConnMaxIdleTime: 11,
		ConnMaxLifetime: 61,
	})

	assert.Equal(t, 17, recorder.maxIdleConns)
	assert.Equal(t, 171, recorder.maxOpenConns)
	assert.Equal(t, 11*time.Minute, recorder.connMaxIdleTime)
	assert.Equal(t, 61*time.Minute, recorder.connMaxLifetime)
}

func TestNewSQLiteIgnoresPostgresPoolOptions(t *testing.T) {
	db, err := New(t.Context(), BaseCQRSOptions{
		Persist: false,
		ForTest: true,
		PostgresPool: &PostgresPoolOptions{
			MaxIdleConns:    17,
			MaxOpenConns:    171,
			ConnMaxIdleTime: 11,
			ConnMaxLifetime: 61,
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	assert.Equal(t, 0, db.Stats().MaxOpenConnections)
}
