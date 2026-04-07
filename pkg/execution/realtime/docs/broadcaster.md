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

## Topics and subscriptions

A **topic** is a keyed Redis pub/sub channel that the broadcaster manages. Each unique topic key (`envID:xxhash(channel):name`) maps to one `runTopic` goroutine and a ordered set of local subscriptions. Topics are reference-counted: the first `Subscribe` call for a topic starts the goroutine, and the last `Unsubscribe` stops it.

A **subscription** is a connected client. The `Subscription` interface wraps a single client connection and exposes `WriteMessage`, `WriteChunk`, `Write` (raw bytes), `SendKeepalive`, and `Close`. There are three implementations:

- **SSE** (`sub_sse.go`): Writes to an `http.ResponseWriter`. All writes are mutex-protected. Keepalives are SSE comments (`:\n\n`). Messages are formatted as `data: {json}\n\n`.
- **WebSocket** (`sub_websocket.go`): Writes text frames. Keepalives are WebSocket pings. Also implements `ReadWriteSubscription` — the `Poll` method reads incoming frames to handle subscribe/unsubscribe requests from the client.
- **In-memory** (`sub_memory.go`): Backed by a callback function. Used in tests.

## Local delivery

When `runTopic` receives a message from Redis, it delivers to local subscriptions:

1. The message is classified by prefix (none → structured, `RAW:` → raw, `CHUNK:` → chunk).
2. The broadcaster acquires a read lock and looks up the topic's ordered set.
3. `eachSubscription` iterates the ordered set (up to 5,000 subscribers) and calls the write method on each subscription.
4. For structured messages and chunks, writes use `doPublish`, which attempts an immediate write and, on failure, spawns a background goroutine that retries up to 3 times at 3-second intervals.
5. For raw bytes, writes are attempted once — failures are logged but not retried.

Retries are async (they don't block fan-out to other subscriptions). If all retries are exhausted, the failure is logged and a metric is recorded, but the subscription is not closed.

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
- **5,000 subscriber cap per topic.** The ordered set iteration is hard-capped at 5,000 subscribers.
- **Publish is fire-and-forget.** `Publish`/`Write`/`PublishChunk` do not return errors. Failures are logged and retried (3 attempts), but the caller has no signal.
