# Durable Endpoint streaming; broadcaster unification

The primary goal of this PR is to add support for Durable Endpoint streaming. Along the way, we noticed issues with the broadcaster implementation and fixed them.

## Durable Endpoint streaming

Add support for Durable Endpoints to stream data to clients (e.g. browsers) through the Inngest backend via Redis pub/sub. When a run goes async, the checkpoint response includes a short-lived realtime JWT so the client can subscribe to the stream via SSE.

- `POST /v1/realtime/publish/tee` accepts signing key auth (in addition to JWT), with request body size limits
- Checkpoint new-run response includes a `realtime_token` for client-side subscription. This JWT has a 15 minute expiry, to match the max duration of a Durable Endpoint streaming.

## Broadcaster unification

Collapse the two-layer broadcaster design (in-memory + Redis wrapper) into a single Redis-backed `broadcaster`. All message delivery (structured, raw bytes, and streaming chunks) flows through Redis pub/sub.

- `Write` publishes to topic key `envID:xxhash(channel):$stream` (where `channel` is the run ID for Durable Endpoints).
- `startTopic` returns a ready channel so `Subscribe` confirms the Redis subscription is active before returning.
- Dev Server uses miniredis when no Redis URI is provided.

## Bug fixes

- Double delivery in `Write` (local + Redis).
- `Write` silently dropped data when no local subscribers existed (broke multi-instance).
- SSE response headers not flushed on connect (clients blocked until first data write or keepalive).
- Stream-start `Data` field was raw bytes instead of a JSON string, causing chunk routing to fail.
