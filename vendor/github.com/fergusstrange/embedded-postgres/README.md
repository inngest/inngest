<p align="center">
    <img src="https://raw.githubusercontent.com/fergusstrange/embedded-postgres/master/gopher.png" width="150">
</p>

<p align="center">
<a href="https://godoc.org/github.com/fergusstrange/embedded-postgres"><img src="https://godoc.org/github.com/fergusstrange/embedded-postgres?status.svg" alt="Godoc" /></a>
<a href='https://coveralls.io/github/fergusstrange/embedded-postgres?branch=master'><img src='https://coveralls.io/repos/github/fergusstrange/embedded-postgres/badge.svg?branch=master' alt='Coverage Status' /></a>
<a href="https://github.com/fergusstrange/embedded-postgres/actions"><img src="https://github.com/fergusstrange/embedded-postgres/workflows/Embedded%20Postgres/badge.svg" alt="Build Status" /></a>
<a href="https://app.circleci.com/pipelines/github/fergusstrange/embedded-postgres"><img src="https://circleci.com/gh/fergusstrange/embedded-postgres.svg?style=shield" alt="Build Status" /></a>
<a href="https://goreportcard.com/report/github.com/fergusstrange/embedded-postgres"><img src="https://goreportcard.com/badge/github.com/fergusstrange/embedded-postgres" alt="Go Report Card" /></a>
</p>

# embedded-postgres

Run a real Postgres database locally on Linux, OSX or Windows as part of another Go application or test.

When testing this provides a higher level of confidence than using any in memory alternative. It also requires no other
external dependencies outside of the Go build ecosystem.

Heavily inspired by Java projects [zonkyio/embedded-postgres](https://github.com/zonkyio/embedded-postgres)
and [opentable/otj-pg-embedded](https://github.com/opentable/otj-pg-embedded) and reliant on the great work being done
by [zonkyio/embedded-postgres-binaries](https://github.com/zonkyio/embedded-postgres-binaries) in order to fetch
precompiled binaries
from [Maven](https://mvnrepository.com/artifact/io.zonky.test.postgres/embedded-postgres-binaries-bom).

## Installation

embedded-postgres uses Go modules and as such can be referenced by release version for use as a library. Use the
following to add the latest release to your project.

```bash
go get -u github.com/fergusstrange/embedded-postgres
``` 

## How to use

This library aims to require as little configuration as possible, favouring overridable defaults

| Configuration       | Default Value                                   |
|---------------------|-------------------------------------------------|
| Username            | postgres                                        |
| Password            | postgres                                        |
| Database            | postgres                                        |
| Version             | 12.1.0                                          |
| RuntimePath         | $USER_HOME/.embedded-postgres-go/extracted      |
| DataPath            | $USER_HOME/.embedded-postgres-go/extracted/data |
| BinariesPath        | $USER_HOME/.embedded-postgres-go/extracted      |
| BinaryRepositoryURL | https://repo1.maven.org/maven2                  |
| Port                | 5432                                            |
| StartTimeout        | 15 Seconds                                      |

The *RuntimePath* directory is erased and recreated at each `Start()` and therefore not suitable for persistent data.

If a persistent data location is required, set *DataPath* to a directory outside *RuntimePath*.

If the *RuntimePath* directory is empty or already initialized but with an incompatible postgres version, it will be
removed and Postgres reinitialized.

Postgres binaries will be downloaded and placed in *BinaryPath* if `BinaryPath/bin` doesn't exist.
*BinaryRepositoryURL* parameter allow overriding maven repository url for Postgres binaries.
If the directory does exist, whatever binary version is placed there will be used (no version check
is done).  
If your test need to run multiple different versions of Postgres for different tests, make sure
*BinaryPath* is a subdirectory of *RuntimePath*.

A single Postgres instance can be created, started and stopped as follows

```go
postgres := embeddedpostgres.NewDatabase()
err := postgres.Start()

// Do test logic

err := postgres.Stop()
```

or created with custom configuration

```go
logger := &bytes.Buffer{}
postgres := NewDatabase(DefaultConfig().
Username("beer").
Password("wine").
Database("gin").
Version(V12).
RuntimePath("/tmp").
BinaryRepositoryURL("https://repo.local/central.proxy").	
Port(9876).
StartTimeout(45 * time.Second).
Logger(logger))
err := postgres.Start()

// Do test logic

err := postgres.Stop()
```

It should be noted that if `postgres.Stop()` is not called then the child Postgres process will not be released and the
caller will block.

## Examples

There are a number of realistic representations of how to use this library
in [examples](https://github.com/fergusstrange/embedded-postgres/tree/master/examples).

## Credits

- [Gopherize Me](https://gopherize.me) Thanks for the awesome logo template.
- [zonkyio/embedded-postgres-binaries](https://github.com/zonkyio/embedded-postgres-binaries) Without which the
  precompiled Postgres binaries would not exist for this to work.

## Contributing

View the [contributing guide](CONTRIBUTING.md).

