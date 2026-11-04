# Bun: SQL-first Golang ORM

[![build workflow](https://github.com/uptrace/bun/actions/workflows/build.yml/badge.svg)](https://github.com/uptrace/bun/actions)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/uptrace/bun)](https://pkg.go.dev/github.com/uptrace/bun)
[![Documentation](https://img.shields.io/badge/bun-documentation-informational)](https://bun.uptrace.dev/)
[![Chat](https://discordapp.com/api/guilds/752070105847955518/widget.png)](https://discord.gg/rWtp5Aj)
[![Gurubase](https://img.shields.io/badge/Gurubase-Ask%20Bun%20Guru-006BFF)](https://gurubase.io/g/bun)

**Lightweight, SQL-first Golang ORM for PostgreSQL, MySQL, MSSQL, SQLite, and Oracle**

Bun is a modern ORM that embraces SQL rather than hiding it. Write complex queries in Go with type
safety, powerful scanning capabilities, and database-agnostic code that works across multiple SQL
databases.

## ✨ Key Features

- **SQL-first approach** - Write elegant, readable queries that feel like SQL
- **Multi-database support** - PostgreSQL, MySQL/MariaDB, MSSQL, SQLite, and Oracle
- **Type-safe operations** - Leverage Go's static typing for compile-time safety
- **Flexible scanning** - Query results into structs, maps, scalars, or slices
- **Performance optimized** - Built on `database/sql` with minimal overhead
- **Rich relationships** - Define complex table relationships with struct tags
- **Production ready** - Migrations, fixtures, soft deletes, and OpenTelemetry support

## 🚀 Quick Start

```bash
go get github.com/uptrace/bun
```

### Basic Example

```go
package main

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/uptrace/bun"
    "github.com/uptrace/bun/dialect/sqlitedialect"
    "github.com/uptrace/bun/driver/sqliteshim"
)

func main() {
    ctx := context.Background()

    // Open database
    sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:")
    if err != nil {
        panic(err)
    }

    // Create Bun instance
    db := bun.NewDB(sqldb, sqlitedialect.New())

    // Define model
    type User struct {
        ID   int64  `bun:",pk,autoincrement"`
        Name string `bun:",notnull"`
    }

    // Create table
    db.NewCreateTable().Model((*User)(nil)).Exec(ctx)

    // Insert user
    user := &User{Name: "John Doe"}
    db.NewInsert().Model(user).Exec(ctx)

    // Query user
    err = db.NewSelect().Model(user).Where("id = ?", user.ID).Scan(ctx)
    fmt.Printf("User: %+v\n", user)
}
```

## 🎯 Why Choose Bun?

### Elegant Complex Queries

Write sophisticated queries that remain readable and maintainable:

```go
regionalSales := db.NewSelect().
    ColumnExpr("region").
    ColumnExpr("SUM(amount) AS total_sales").
    TableExpr("orders").
    GroupExpr("region")

topRegions := db.NewSelect().
    ColumnExpr("region").
    TableExpr("regional_sales").
    Where("total_sales > (SELECT SUM(total_sales) / 10 FROM regional_sales)")

var results []struct {
    Region       string `bun:"region"`
    Product      string `bun:"product"`
    ProductUnits int    `bun:"product_units"`
    ProductSales int    `bun:"product_sales"`
}

err := db.NewSelect().
    With("regional_sales", regionalSales).
    With("top_regions", topRegions).
    ColumnExpr("region, product").
    ColumnExpr("SUM(quantity) AS product_units").
    ColumnExpr("SUM(amount) AS product_sales").
    TableExpr("orders").
    Where("region IN (SELECT region FROM top_regions)").
    GroupExpr("region, product").
    Scan(ctx, &results)
```

### Flexible Result Scanning

Scan query results into various Go types:

```go
// Into structs
var users []User
db.NewSelect().Model(&users).Scan(ctx)

// Into maps
var userMaps []map[string]interface{}
db.NewSelect().Table("users").Scan(ctx, &userMaps)

// Into scalars
var count int
db.NewSelect().Table("users").ColumnExpr("COUNT(*)").Scan(ctx, &count)

// Into individual variables
var id int64
var name string
db.NewSelect().Table("users").Column("id", "name").Limit(1).Scan(ctx, &id, &name)
```

## 📊 Database Support

| Database      | Driver                                     | Dialect               |
| ------------- | ------------------------------------------ | --------------------- |
| PostgreSQL    | `github.com/uptrace/bun/driver/pgdriver`   | `pgdialect.New()`     |
| MySQL/MariaDB | `github.com/go-sql-driver/mysql`           | `mysqldialect.New()`  |
| SQLite        | `github.com/uptrace/bun/driver/sqliteshim` | `sqlitedialect.New()` |
| SQL Server    | `github.com/denisenkom/go-mssqldb`         | `mssqldialect.New()`  |
| Oracle        | `github.com/sijms/go-ora/v2`               | `oracledialect.New()` |

## 🔧 Advanced Features

### Table Relationships

Define complex relationships with struct tags:

```go
type User struct {
    ID      int64   `bun:",pk,autoincrement"`
    Name    string  `bun:",notnull"`
    Posts   []Post  `bun:"rel:has-many,join:id=user_id"`
    Profile Profile `bun:"rel:has-one,join:id=user_id"`
}

type Post struct {
    ID     int64 `bun:",pk,autoincrement"`
    Title  string
    UserID int64
    User   *User `bun:"rel:belongs-to,join:user_id=id"`
}

// Load users with their posts
var users []User
err := db.NewSelect().
    Model(&users).
    Relation("Posts").
    Scan(ctx)
```

### Bulk Operations

Efficient bulk operations for large datasets:

```go
// Bulk insert
users := []User{{Name: "John"}, {Name: "Jane"}, {Name: "Bob"}}
_, err := db.NewInsert().Model(&users).Exec(ctx)

// Bulk update with CTE
_, err = db.NewUpdate().
    Model(&users).
    Set("updated_at = NOW()").
    Where("active = ?", true).
    Exec(ctx)

// Bulk delete
_, err = db.NewDelete().
    Model((*User)(nil)).
    Where("created_at < ?", time.Now().AddDate(-1, 0, 0)).
    Exec(ctx)
```

### Migrations

Version your database schema:

```go
import "github.com/uptrace/bun/migrate"

migrations := migrate.NewMigrations()

migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
    _, err := db.NewCreateTable().Model((*User)(nil)).Exec(ctx)
    return err
}, func(ctx context.Context, db *bun.DB) error {
    _, err := db.NewDropTable().Model((*User)(nil)).Exec(ctx)
    return err
})

migrator := migrate.NewMigrator(db, migrations)
err := migrator.Init(ctx)
err = migrator.Up(ctx)
```

## 📈 Monitoring & Observability

### Debug Queries

Enable query logging for development:

```go
import "github.com/uptrace/bun/extra/bundebug"

db.AddQueryHook(bundebug.NewQueryHook(
    bundebug.WithVerbose(true),
))
```

### OpenTelemetry Integration

Production-ready observability with distributed tracing:

```go
import "github.com/uptrace/bun/extra/bunotel"

db.AddQueryHook(bunotel.NewQueryHook(
    bunotel.WithDBName("myapp"),
))
```

> **Monitoring made easy**: Bun is brought to you by ⭐
> [**uptrace/uptrace**](https://github.com/uptrace/uptrace). Uptrace is an open-source APM tool that
> supports distributed tracing, metrics, and logs. You can use it to monitor applications and set up
> automatic alerts to receive notifications via email, Slack, Telegram, and others.
>
> See [OpenTelemetry example](example/opentelemetry) which demonstrates how you can use Uptrace to
> monitor Bun.

## 📚 Documentation & Resources

- **[Getting Started Guide](https://bun.uptrace.dev/guide/golang-orm.html)** - Comprehensive
  tutorial
- **[API Reference](https://pkg.go.dev/github.com/uptrace/bun)** - Complete package documentation
- **[Examples](https://github.com/uptrace/bun/tree/master/example)** - Working code samples
- **[Starter Kit](https://github.com/go-bun/bun-starter-kit)** - Production-ready template
- **[Community Discussions](https://github.com/uptrace/bun/discussions)** - Get help and share ideas

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to
get started.

**Thanks to all our contributors:**

<a href="https://github.com/uptrace/bun/graphs/contributors">
  <img src="https://contributors-img.web.app/image?repo=uptrace/bun" alt="Contributors" />
</a>

## 🔗 Related Projects

- **[Golang HTTP router](https://github.com/uptrace/bunrouter)** - Fast and flexible HTTP router
- **[Golang msgpack](https://github.com/vmihailenco/msgpack)** - High-performance MessagePack
  serialization

---

<div align="center">
  <strong>Star ⭐ this repo if you find Bun useful!</strong><br>
  <sub>Join our community on <a href="https://discord.gg/rWtp5Aj">Discord</a> • Follow updates on <a href="https://github.com/uptrace/bun">GitHub</a></sub>
</div>
