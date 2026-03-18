# Durable Endpoint streaming

This feature allows our users to publish to their subscribers via our backend. It uses Redis pub/sub for cross-instance streaming.

The primary use case is "Durable Endpoint streaming". This allows a Durable Endpoint to stream to a client (e.g. a browser) via our backend.

## Auth

Tries 2 strategies in order:

1. **Realtime JWT** — HS256, issued by `rt.inngest.com`, 1-min expiry. Extracted from `Authorization: Bearer <token>` header or `?token=` query param. Claims embed `{sub: accountID, env: workspaceID, topics: [], publish: bool}`.
2. **Standard auth fallback** — API key / signing key middleware. If no valid JWT is found, the request falls through to the standard auth middleware.

**Subscribing** (`GET /sse`): Requires a JWT with embedded topics. The topics in the JWT define what the subscriber can listen to. The JWT is only validated at connection time — once connected, the SSE stream stays open regardless of JWT expiry (1-min expiry just limits the connect window).

**Publishing** (`POST /publish/tee`): Accepts either a JWT with `publish: true` or standard signing key auth.

**Token creation** (`POST /token`): Only accepts standard auth (not JWTs). Returns a scoped subscription JWT for the requested topics.

Two token constructors exist: `NewJWT` (subscribe, topics embedded) and `NewPublishJWT` (publish, no topics).

## Publish

Stream to `POST /v1/realtime/publish/tee?channel={runID}`. The request body is forwarded directly to the broadcaster as raw bytes (no JSON wrapping or framing).

Auth uses the signing key (not a JWT). The SDK already has the signing key and doesn't need a separate publish token.

## Broadcast

Since publish may go to a different instance than subscribe, we need broadcasting. This is implemented using Redis pub/sub.

Finds relevant topic by env ID and channel (i.e. run ID). Writes bytes to the topic's subscribers. Supports fanout.

Topics keys are `envID:xxhash(runID):$stream`.

## Subscribe

Subscribe via `GET /v1/realtime/sse?token={jwt}`. The JWT embeds topic objects `{kind: "run", env_id, channel: runID, name: "$stream"}`. The broadcaster internally hashes these to `envID:xxhash(runID):$stream` for routing.

## Limitations

- **Late-joining**: We do not buffer or replay, so publishing without a subscriber leads to lost data.
- **15-minute SSE cap**: SSE connections are hard-capped at 15 minutes. No reconnect or token refresh mechanism exists, and data published during a disconnect window is lost.
