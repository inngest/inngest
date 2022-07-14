# ocsql

[![Go Report Card](https://goreportcard.com/badge/contrib.go.opencensus.io/integrations/ocsql)](https://goreportcard.com/report/contrib.go.opencensus.io/integrations/ocsql)
[![GoDoc](https://godoc.org/contrib.go.opencensus.io/integrations/ocsql?status.svg)](https://godoc.org/contrib.go.opencensus.io/integrations/ocsql)
[![Sourcegraph](https://sourcegraph.com/github.com/opencensus-integrations/ocsql/-/badge.svg)](https://sourcegraph.com/github.com/opencensus-integrations/ocsql?badge)

OpenCensus SQL database driver wrapper.

Add an ocsql wrapper to your existing database code to instrument the
interactions with the database.

## installation

go get -u contrib.go.opencensus.io/integrations/ocsql

## initialize

To use ocsql with your application, register an ocsql wrapper of a database
driver as shown below.

Example:
```go
import (
    _ "github.com/mattn/go-sqlite3"
    "contrib.go.opencensus.io/integrations/ocsql"
)

var (
    driverName string
    err        error
    db         *sql.DB
)

// Register our ocsql wrapper for the provided SQLite3 driver.
driverName, err = ocsql.Register("sqlite3", ocsql.WithAllTraceOptions(), ocsql.WithInstanceName("resources"))
if err != nil {
    log.Fatalf("unable to register our ocsql driver: %v\n", err)
}

// Connect to a SQLite3 database using the ocsql driver wrapper.
db, err = sql.Open(driverName, "resource.db")
```

A more explicit and alternative way to bootstrap the ocsql wrapper exists as
shown below. This will only work if the actual database driver has its driver
implementation exported.

Example:
```go
import (
    sqlite3 "github.com/mattn/go-sqlite3"
    "contrib.go.opencensus.io/integrations/ocsql"
)

var (
    driver driver.Driver
    err    error
    db     *sql.DB
)

// Explicitly wrap the SQLite3 driver with ocsql.
driver = ocsql.Wrap(&sqlite3.SQLiteDriver{})

// Register our ocsql wrapper as a database driver.
sql.Register("ocsql-sqlite3", driver)

// Connect to a SQLite3 database using the ocsql driver wrapper.
db, err = sql.Open("ocsql-sqlite3", "resource.db")
```

Projects providing their own abstractions on top of database/sql/driver can also
wrap an existing driver.Conn interface directly with ocsql.

Example:
```go
import "contrib.go.opencensus.io/integrations/ocsql"

func GetConn(...) driver.Conn {
    // Create custom driver.Conn.
    conn := initializeConn(...)

    // Wrap with ocsql.
    return ocsql.WrapConn(conn, ocsql.WithAllTraceOptions())    
}
```

Finally database drivers that support the new (Go 1.10+) driver.Connector
interface can be wrapped directly by ocsql without the need for ocsql to
register a driver.Driver.

Example:
```go
import(
    "contrib.go.opencensus.io/integrations/ocsql"
    "github.com/lib/pq"
)

var (
    connector driver.Connector
    err       error
    db        *sql.DB
)

// Get a database driver.Connector for a fixed configuration.
connector, err = pq.NewConnector("postgres://user:passt@host:5432/db")
if err != nil {
    log.Fatalf("unable to create our postgres connector: %v\n", err)
}

// Wrap the driver.Connector with ocsql.
connector = ocsql.WrapConnector(connector, ocsql.WithAllTraceOptions())

// Use the wrapped driver.Connector.
db = sql.OpenDB(connector)
```

## metrics

Next to tracing, ocsql also supports OpenCensus stats. To record call stats,
register the available views or create your own using the provided Measures.

```go
// Register default views.
ocsql.RegisterAllViews()

```

From Go 1.11 and up, ocsql also has the ability to record database connection
pool details. Use the `RecordStats` function and provide a `*sql.DB` to record
details on, as well as the required record interval.

```go
// Register default views.
ocsql.RegisterAllViews()

// Connect to a SQLite3 database using the ocsql driver wrapper.
db, err = sql.Open("ocsql-sqlite3", "resource.db")

// Record DB stats every 5 seconds until we exit.
defer ocsql.RecordStats(db, 5 * time.Second)()
```

## Recorded metrics

| Metric                 | Search suffix          | Additional tags            |
|------------------------|------------------------|----------------------------|
| Number of Calls        | "go.sql/client/calls"  |"method", "error", "status" |
| Latency in milliseconds| "go.sql/client/latency"|"method", "error", "status" |

If using RecordStats:

| Metric                                                   | Search suffix                                |
|----------------------------------------------------------|----------------------------------------------|
| Number of open connections                               | "go.sql/db/connections/open"                 |
| Number of idle connections                               | "go.sql/db/connections/idle"                 |
| Number of active connections                             | "go.sql/db/connections/active"               |
| Total number of connections waited for                   | "go.sql/db/connections/wait_count"           |
| Total time blocked waiting for new connections           | "go.sql/db/connections/wait_duration"        |
| Total number of closed connections by SetMaxIdleConns    | "go.sql/db/connections/idle_close_count"     |
| Total number of closed connections by SetConnMaxLifetime | "go.sql/db/connections/lifetime_close_count" |

## jmoiron/sqlx

If using the `sqlx` library with named queries you will need to use the
`sqlx.NewDb` function to wrap an existing `*sql.DB` connection. Do not use the
`sqlx.Open` and `sqlx.Connect` methods.
`sqlx` uses the driver name to figure out which database is being used. It uses
this knowledge to convert named queries to the correct bind type (dollar sign,
question mark) if named queries are not supported natively by the
database. Since ocsql creates a new driver name it will not be recognized by
sqlx and named queries will fail.

Use one of the above methods to first create a `*sql.DB` connection and then
create a `*sqlx.DB` connection by wrapping the `*sql.DB` like this:

```go
    // Register our ocsql wrapper for the provided Postgres driver.
    driverName, err := ocsql.Register("postgres", ocsql.WithAllTraceOptions())
    if err != nil { ... }

    // Connect to a Postgres database using the ocsql driver wrapper.
    db, err := sql.Open(driverName, "postgres://localhost:5432/my_database")
    if err != nil { ... }

    // Wrap our *sql.DB with sqlx. use the original db driver name!!!
    dbx := sqlx.NewDB(db, "postgres")
```

## context

To really take advantage of ocsql, all database calls should be made using the
*Context methods. Failing to do so will result in many orphaned ocsql traces
if the `AllowRoot` TraceOption is set to true. By default AllowRoot is disabled
and will result in ocsql not tracing the database calls if context or parent
spans are missing.

| Old            | New                   |
|----------------|-----------------------|
| *DB.Begin      | *DB.BeginTx           |
| *DB.Exec       | *DB.ExecContext       |
| *DB.Ping       | *DB.PingContext       |
| *DB.Prepare    | *DB.PrepareContext    |
| *DB.Query      | *DB.QueryContext      |
| *DB.QueryRow   | *DB.QueryRowContext   |
|                |                       |
| *Stmt.Exec     | *Stmt.ExecContext     |
| *Stmt.Query    | *Stmt.QueryContext    |
| *Stmt.QueryRow | *Stmt.QueryRowContext |
|                |                       |
| *Tx.Exec       | *Tx.ExecContext       |
| *Tx.Prepare    | *Tx.PrepareContext    |
| *Tx.Query      | *Tx.QueryContext      |
| *Tx.QueryRow   | *Tx.QueryRowContext   |

Example:
```go

func (s *svc) GetDevice(ctx context.Context, id int) (*Device, error) {
    // Assume we have instrumented our service transports and ctx holds a span.
    var device Device
    if err := s.db.QueryRowContext(
        ctx, "SELECT * FROM device WHERE id = ?", id,
    ).Scan(&device); err != nil {
        return nil, err
    }
    return device
}
```
