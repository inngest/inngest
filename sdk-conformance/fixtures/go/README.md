# Go SDK conformance fixture

This is the Go target for the independent SDK conformance harness. It is a
small HTTP application intended for direct protocol probing; it does not use
the legacy Go conformance runner.

Run it from the repository root:

```sh
PORT=3000 \
INNGEST_EVENT_KEY=test \
INNGEST_SIGNING_KEY='7468697320697320612074657374206b6579' \
go run ./sdk-conformance/fixtures/go
```

`PORT` must be a numeric port. The server binds to `127.0.0.1` and exposes:

- `GET /__conformance` — fixture manifest, SDK identity, profiles, and capabilities;
- `POST /__conformance/reset` — reset mutable fixture state (currently stateless);
- `/api/inngest` — the normal Inngest SDK serve handler.

The manifest is the source of truth for stable function IDs and trigger event
names. The fixture uses the repository-pinned `github.com/inngest/inngestgo`
dependency.
