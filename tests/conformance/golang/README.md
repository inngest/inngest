# Go Conformance Fixture

This directory contains a small standalone Go SDK app that matches the current
`inngest conformance` serve runner.

It is meant to be run locally while testing the current branch of the CLI.

## Start the fixture

```bash
go run ./tests/conformance/golang
```

The app serves:

- `POST /api/inngest`
- `GET /api/introspect`
- `GET /health`

By default it listens on `127.0.0.1:3000`.

## Start the dev server

```bash
INNGEST_EVENT_KEY=test \
INNGEST_SIGNING_KEY=7468697320697320612074657374206b6579 \
./inngest-bin dev --no-discovery
```

## Run doctor

```bash
./inngest-bin conformance doctor \
  --config ./tests/conformance/golang/inngest.conformance.yaml
```

## Run the current serve showcase

```bash
./inngest-bin conformance run \
  --config ./tests/conformance/golang/inngest.conformance.yaml \
  --report-out /tmp/golang-conformance-report.json
```

## Supported cases in this fixture

- `serve-introspection`
- `basic-invoke`
- `steps-serial`
- `retry-basic`
- `cancel-basic`
- `wait-for-event-basic`
