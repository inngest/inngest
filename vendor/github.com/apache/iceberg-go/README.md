<!--
 - Licensed to the Apache Software Foundation (ASF) under one or more
 - contributor license agreements.  See the NOTICE file distributed with
 - this work for additional information regarding copyright ownership.
 - The ASF licenses this file to You under the Apache License, Version 2.0
 - (the "License"); you may not use this file except in compliance with
 - the License.  You may obtain a copy of the License at
 -
 -   http://www.apache.org/licenses/LICENSE-2.0
 -
 - Unless required by applicable law or agreed to in writing, software
 - distributed under the License is distributed on an "AS IS" BASIS,
 - WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 - See the License for the specific language governing permissions and
 - limitations under the License.
 -->

# Iceberg Golang

[![Go Reference](https://pkg.go.dev/badge/github.com/apache/iceberg-go.svg)](https://pkg.go.dev/github.com/apache/iceberg-go)

`iceberg` is a Golang implementation of the [Iceberg table spec](https://iceberg.apache.org/spec/).

## Build From Source

### Prerequisites

* Go 1.25 or later

### Build

```shell
$ git clone https://github.com/apache/iceberg-go.git
$ cd iceberg-go/cmd/iceberg && go build .
```

## Running Tests

Use the [Makefile](Makefile) so commands stay in sync with CI (e.g. golangci-lint version).

### Unit tests

```shell
make test
```

### Linting

```shell
make lint
```

Install the linter first 

```shell
make lint-install
# or: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
```

### Integration tests

**Prerequisites:** Docker, Docker Compose

1. Start the Docker containers using docker compose:

   ```shell
   make integration-setup
   ```

2. Export the required environment variables:

   ```shell
   export AWS_S3_ENDPOINT=http://$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' object-store):9000
   export AWS_REGION=us-east-1
   export SPARK_CONTAINER_ID=$(docker ps -qf 'name=spark-iceberg')
   export DOCKER_API_VERSION=$(docker version -f '{{.Server.APIVersion}}')
   ```

3. Run the integration tests:

   ```shell
   make integration-test
   ```

   Or run a single suite: `make integration-scanner`, `make integration-io`, `make integration-rest`, `make integration-rest-scan-planning`, `make integration-spark`, `make integration-hive`, `make integration-hadoop`.

#### Running against Spark 4.0

To validate against Spark 4.0 (used by v3 feature tests), substitute the setup step:

```shell
make integration-setup-spark4
```

The exported env vars and `make integration-test` invocation are identical to the Spark 3.5 flow above.

Tests requiring Spark 4 (currently the variant and unknown-type tests) automatically skip on the Spark 3.5 setup.

## Feature Support / Roadmap

### FileSystem Support

| Filesystem Type      | Supported |
| :------------------: | :-------: |
| S3                   |    X      |
| Google Cloud Storage |    X      |
| Azure Blob Storage   |    X      |
| Local Filesystem     |    X      |

### Metadata

| Operation                | Supported |
| :----------------------- | :-------: |
| Get Schema               |     X     |
| Get Snapshots            |     X     |
| Get Sort Orders          |     X     |
| Get Partition Specs      |     X     |
| Get Manifests            |     X     |
| Create New Manifests     |     X     |
| Plan Scan                | Local + Remote |
| Plan Scan for Snapshot   | Local + Remote |

### REST Scan Planning

REST catalogs can plan scans on the server, including asynchronous plans,
batched task retrieval, and plan-scoped storage credentials. Enable this per
scan with `table.WithScanPlanningMode`; existing scans continue to plan locally.

| Scan option | Behavior |
| :---------- | :------- |
| `table.ScanPlanningLocal` (default) | Read manifests locally; no planning endpoints required. |
| `table.ScanPlanningRemote` | Require the advertised plan endpoint. Async or fanout responses also require their continuation endpoints; missing capabilities return an error. |
| `table.ScanPlanningAuto` | Use remote planning when submission, polling, and task retrieval endpoints are advertised; otherwise plan locally. Errors after choosing remote are returned, without falling back to local. |

Capabilities come from `GET /v1/config`. `rest.Catalog.SupportsPlanTableScan()`
checks submission support, while `SupportsFullRemoteScanPlanning()` checks
submission, polling, and task retrieval. Cancellation is best-effort and is not
required for automatic remote planning. In `auto` mode, catalogs
without a planner retain local planning.

The scanner does not honor the REST `scan-planning-mode` configuration key
(`client`/`server`). A catalog requiring `server` mode may rely on planning to
vend plan-scoped storage credentials. Default local planning, explicit `local`,
or an `auto` fallback can instead read manifests with the table's storage
credentials, which may use a different identity or lack the required access.
For these deployments, explicitly select `table.ScanPlanningRemote` on every
scan; do not rely on the server configuration or `auto` to enforce remote
planning. Remote mode returns an error if a required capability is missing.

```go
// tbl is a table loaded from a rest.Catalog.
ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
defer cancel()

scan := tbl.Scan(
    table.WithScanPlanningMode(table.ScanPlanningRemote),
    table.WithRowFilter(iceberg.EqualTo(iceberg.Reference("tenant_id"), "acme")),
    table.WithSelectedFields("id", "tenant_id"),
)
defer scan.Close()

tasks, err := scan.PlanFiles(ctx)
if err != nil {
    return err
}
// Read using this scan so its plan-scoped credentials remain attached.
_, records, err := scan.ReadTasks(ctx, tasks)
if err != nil {
    return err
}
for record, err := range records {
    if err != nil {
        return err
    }
    // Consume the Arrow record batch here.
    record.Release()
}
```

`WithSnapshotID`, `WithSnapshotAsOf`, and tag scans use the historical schema;
branch scans use the current table schema. File ranges follow
`read.split.target-size` for both local and remote planning. Remote incremental
scans and `_last_updated_sequence_number` projection are not supported. For
row-lineage scans requiring that column, `auto` chooses local planning and
`remote` returns an error.

Plan and task POSTs retry HTTP 408, 429, 500, 502, 503, and 504 up to three times
with jittered backoff, reusing the same UUIDv7 idempotency key. Other HTTP errors,
transport failures, and malformed successful responses are returned directly.
`Retry-After` is honored up to the caller's deadline (or capped at five seconds
without a deadline). A new explicit call generates a new key unless one is
supplied by the caller. `WaitForPlanOptions` controls polling delays and retry
limits; use a context deadline to bound the complete operation. Cancellation
during polling and exhausted polling trigger best-effort server cleanup. Recognized expired
plans return `rest.ErrPlanExpired`; expired task handles return
`rest.ErrNoSuchPlanTask`. Callers decide whether to submit a new plan.

Close the scan after consuming its tasks, including when tasks are never read.
Plan-scoped credentials remain alive for active readers; closing or replacing
the plan releases them when those readers finish. They cannot be renewed from
the table-credentials endpoint. Once `ReadTasks` returns an iterator, iterate it
(even if stopping early) to release its reader lease.

OpenTelemetry uses the application's global tracer and meter providers, with
scope `github.com/apache/iceberg-go/scan-planning`; no SDK is installed by the
library. Logical endpoint calls emit `iceberg.scan.planning.<operation>` spans
and the following instruments:

| Instrument | Meaning |
| :--------- | :------ |
| `iceberg.scan.planning.requests` | Logical endpoint calls, labeled by operation and success/error outcome. |
| `iceberg.scan.planning.request.duration` | Endpoint call duration including retries, in milliseconds. |
| `iceberg.scan.planning.retries` | Additional attempts after transient HTTP failures. |
| `iceberg.scan.planning.expirations` | Recognized expired plan or task responses. |
| `iceberg.scan.planning.fallbacks` | Auto mode choosing local due to capability or row-lineage constraints. |

Operations are `plan`, `fetch-result`, `fetch-tasks`, and `cancel`. Attributes
contain only operation, outcome, or fallback reason; they omit table names,
filters, tokens, credentials, and raw error messages. These client metrics
complement the existing scan/commit metrics reporter.

The in-process local/remote parity test runs with `go test ./catalog/rest`.
For Java interoperability, run `make integration-rest-scan-planning` (Docker
required). Remote planning is opt-in, with no migration required for existing
local scans.

### Catalog Support

| Operation                   | REST | Hive |  Glue  | SQL  | Hadoop |
|:----------------------------|:----:|:----:|:------:|:----:|:------:|
| Load Table                  |  X   |  X   |   X    |  X   |   X    |
| List Tables                 |  X   |  X   |   X    |  X   |   X    |
| Create Table                |  X   |  X   |   X    |  X   |   X    |
| Register Table              |  X   |  X   |   X    |      |        |
| Update Current Snapshot     |  X   |  X   |   X    |  X   |   X    |
| Create New Snapshot         |  X   |  X   |   X    |  X   |   X    |
| Rename Table                |  X   |  X   |   X    |  X   |        |
| Drop Table                  |  X   |  X   |   X    |  X   |   X    |
| Alter Table                 |  X   |  X   |   X    |  X   |   X    |
| Check Table Exists          |  X   |  X   |   X    |  X   |   X    |
| Set Table Properties        |  X   |  X   |   X    |  X   |   X    |
| List Namespaces             |  X   |  X   |   X    |  X   |   X    |
| Create Namespace            |  X   |  X   |   X    |  X   |   X    |
| Check Namespace Exists      |  X   |  X   |   X    |  X   |   X    |
| Drop Namespace              |  X   |  X   |   X    |  X   |   X    |
| Update Namespace Properties |  X   |  X   |   X    |  X   |        |
| Create View                 |  X   |  X   |        |  X   |        |
| Load View                   |  X   |  X   |        |  X   |        |
| List View                   |  X   |  X   |        |  X   |        |
| Drop View                   |  X   |  X   |        |  X   |        |
| Rename View                 |  X   |      |        |      |        |
| Check View Exists           |  X   |  X   |        |  X   |        |

### Read/Write Data Support

* Data can currently be read as an Arrow Table or as a stream of Arrow record batches.

#### Supported Write Operations

As long as the FileSystem is supported and the Catalog supports altering
the table, the following tracks the current write support:

| Operation            | Supported |
|:---------------------|:---------:|
| Append Stream        |     X     |
| Append Data Files    |     X     |
| Rewrite Files        |     X     |
| Rewrite manifests    |     X     |
| Overwrite Files      |     X     |
| Copy-On-Write Delete |     X     |
| Write Pos Delete     |     X     |
| Write Eq Delete      |     X     |
| Row Delta            |     X     |


### CLI Usage
Run `go build ./cmd/iceberg` from the root of this repository to build the CLI executable, alternately you can run `go install github.com/apache/iceberg-go/cmd/iceberg@latest` to install it to the `bin` directory of your `GOPATH`.

The `iceberg` CLI usage is very similar to [pyiceberg CLI](https://py.iceberg.apache.org/cli/) \
You can pass the catalog URI with `--uri` argument.

Example:
You can start the Iceberg REST API docker image which runs on default in port `8181`
```
docker pull apache/iceberg-rest-fixture:latest
docker run -p 8181:8181 apache/iceberg-rest-fixture:latest
```
and run the `iceberg` CLI pointing to the REST API server.

```
 ./iceberg --uri http://0.0.0.0:8181 list
┌─────┐
| IDs |
| --- |
└─────┘
```
**Create Namespace**
```
./iceberg --uri http://0.0.0.0:8181 create namespace taxitrips
```

**List Namespace**
```
 ./iceberg --uri http://0.0.0.0:8181 list
┌───────────┐
| IDs       |
| --------- |
| taxitrips |
└───────────┘


```
# Get in Touch

- [Iceberg community](https://iceberg.apache.org/community/)
