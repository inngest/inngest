# Broadcaster

The broadcaster delivers realtime messages from publishers to subscribers. It uses Redis pub/sub for cross-instance delivery and fans out locally to connected clients (SSE, WebSocket).

## How it works

One broadcaster per instance. All publish calls (`Publish`, `Write`, `PublishChunk`) go to Redis. Each instance runs a `runTopic` goroutine per active topic that receives from Redis and fans out to local subscriptions. This means a publisher and subscriber can be on different instances.

Topics are keyed as `envID:xxhash(channel):name`. For Durable Endpoint streaming, the channel is the run ID and the name is `$stream`.

## Message types

Three types flow through Redis, distinguished by prefix:

- **Structured** (no prefix): JSON-encoded `Message`. Used for step output, run results, etc.
- **Raw** (`RAW:` prefix): Opaque bytes from `Write`. Used by `/realtime/publish/tee` for Durable Endpoint streaming.
- **Chunk** (`CHUNK:` prefix): JSON-encoded `Chunk`. Used for data stream chunks.

## Publishing

- **`Publish(Message)`** — Structured JSON messages. The caller sets topics on the `Message`; the broadcaster publishes to each topic. Used by `/publish` (the Executor calls this for step output, run results, etc.) and for stream-start/stream-end control messages when the content-type is `text/stream`.
- **`PublishChunk(Message, Chunk)`** — Streaming chunks. The caller sets topics on the `Message`; the chunk is JSON-encoded and published to each. Used for the data chunks between stream-start and stream-end when the content-type is `text/stream`.
- **`Write(envID, channel, data)`** — Raw bytes. Always publishes to the `$stream` topic for the given env/channel. Used by `/publish/tee` for Durable Endpoint streaming. The caller doesn't need to know about topics.

`Publish` and `PublishChunk` are topic-aware (topics come from the `Message`). `Write` is topic-unaware: it always targets `$stream`, which is the convention for raw byte streams.

## Subscription lifecycle

1. `Subscribe` registers the subscription and starts a `runTopic` goroutine for any new topics.
2. `Subscribe` waits until Redis confirms the subscription is active (via a ready channel returned by `startTopic`), so no messages are missed between subscribe and the first publish. The broadcaster lock is released while waiting.
3. Keepalives are sent every 15s. After 3 consecutive failures, the subscription is closed.
4. `Unsubscribe` / `CloseSubscription` removes the subscription. When the last subscriber leaves a topic, the Redis subscription is cancelled.

## Shutdown

`Close` sends a `closing` message to all subscribers (so they can reconnect to a healthy instance), then waits for a grace period (default 5min) before force-closing connections and cancelling all `runTopic` goroutines.

## Footguns

- **No buffering or replay.** If no subscriber is connected when a message is published, it's lost. Late-joining subscribers miss everything before they connected.
- **Redis is required.** There is no in-process-only mode. The Dev Server uses miniredis to satisfy this.
- **Two Redis clients required.** A Redis client that has subscribed cannot also publish. `NewRedisBroadcaster` takes separate `pubc` (publish) and `subc` (subscribe) clients.
- **5,000 subscriber cap per topic.** The skiplist iteration is hard-capped at 5,000 nodes.
- **Publish is fire-and-forget.** `Publish`/`Write`/`PublishChunk` do not return errors. Failures are logged and retried (3 attempts), but the caller has no signal.
