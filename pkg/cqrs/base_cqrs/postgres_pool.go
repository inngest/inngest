package base_cqrs

import "time"

// PostgresPoolOptions configures the shared database/sql pool for Postgres.
type PostgresPoolOptions struct {
	// MaxIdleConns sets the maximum idle connections retained by database/sql.
	MaxIdleConns int

	// MaxOpenConns sets the maximum open connections allowed by database/sql.
	MaxOpenConns int

	// ConnMaxIdleTime sets the maximum idle time, in minutes, for retained connections.
	ConnMaxIdleTime int

	// ConnMaxLifetime sets the maximum lifetime, in minutes, for reused connections.
	ConnMaxLifetime int
}

type postgresPoolConfigurable interface {
	SetMaxIdleConns(int)
	SetMaxOpenConns(int)
	SetConnMaxIdleTime(time.Duration)
	SetConnMaxLifetime(time.Duration)
}

func applyPostgresPoolOptions(db postgresPoolConfigurable, opts PostgresPoolOptions) {
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetConnMaxIdleTime(time.Duration(opts.ConnMaxIdleTime) * time.Minute)
	db.SetConnMaxLifetime(time.Duration(opts.ConnMaxLifetime) * time.Minute)
}
