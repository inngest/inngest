# Durable Endpoints: Streaming Output Endpoint

## Problem

The existing `GET /checkpoint/{runID}/output` endpoint polls for a complete `APIResult` and writes it all at once. It cannot handle the case where an app's HTTP handler streams a response back through the Dev Server to a waiting client.

Additionally, the client may connect **after** the app has already started streaming, so we need to buffer early chunks.

## Existing Infrastructure

The realtime package already provides everything we need:

- **Broadcaster** (`pkg/execution/realtime/broadcaster.go`) — in-process pub/sub with topic-based routing
- **SSE subscriptions** (`sub_sse.go`) — `http.Flusher`-aware, writes chunks immediately
- **`PostPublishTee`** (`api.go:408`) — streams raw bytes from an HTTP request body directly to subscribers via `io.Copy` + `channelWriter`
- **`publishStream`** (`api.go:451`) — structured chunk-based streaming with `DataStreamStart/Chunk/End` message kinds
- **`Write()` on Broadcaster** — raw byte forwarding to a channel, no JSON wrapping

## Design

### Two-part flow

```
  App (SDK)                    Dev Server                     Client
  ─────────                    ──────────                     ──────
  HTTP handler streams    ──►  POST /checkpoint/{runID}/stream
  response body chunks         │
                               ├─ Buffers chunks in a ring buffer
                               ├─ Publishes each chunk to broadcaster
                               │     topic: {envID}:{runID}:$stream
                               │
                               │                         GET /checkpoint/{runID}/stream?token=...
                               │                              │
                               │                              ├─ Replays buffered chunks
                               │                              └─ Subscribes to broadcaster
                               │                                   for new chunks
                               ├─ On EOF from app:
                               │     publish sentinel "end" chunk
                               └─ Close
```

### Part 1: Ingest endpoint (app → Dev Server)

**`POST /checkpoint/{runID}/stream`**

- Authenticated via signing key (same as other checkpoint endpoints)
- Reads the request body as a stream (not buffered to completion)
- For each chunk read:
  1. Appends to a per-run in-memory ring buffer (capped, e.g. 4MB)
  2. Publishes via the broadcaster to topic `{envID}:{hash(runID)}:$stream`
- On EOF: publishes a sentinel end-of-stream message, marks the buffer as complete
- On error/timeout: publishes an error sentinel

### Part 2: Client subscribe endpoint (Dev Server → client)

**`GET /checkpoint/{runID}/stream?token=<jwt>`**

- Authenticated via the run JWT (same as existing `Output` endpoint)
- Sets response headers for streaming (`Transfer-Encoding: chunked`, `Cache-Control: no-cache`)
- Looks up the run's stream buffer:
  - **If buffer exists and has data**: replays all buffered chunks first (catches up a late-joining client)
  - **If buffer is marked complete**: writes all buffered data, then returns (no need to subscribe)
  - **If still streaming**: after replay, subscribes to the broadcaster topic and forwards new chunks as they arrive, flushing after each write
- On end-of-stream sentinel or client disconnect: clean up subscription, return
- Timeout: 5 minutes max (matching existing `CheckpointOutputWaitMax`)

### Stream buffer

A new lightweight struct to hold per-run stream state:

```go
// streamBuffer holds chunks for a single run's stream, supporting
// late-joining clients that need to catch up.
type streamBuffer struct {
    mu       sync.RWMutex
    chunks   [][]byte     // ordered list of chunks received
    size     int          // total bytes buffered
    done     bool         // true when the app finished streaming
    err      error        // non-nil if the stream errored
    notify   chan struct{} // closed when done or errored, wakes waiting subscribers
}
```

Stored in a `sync.Map` or `ccache` on the `checkpointAPI` struct, keyed by run ID. Entries expire after a TTL (e.g. 10 minutes) to prevent leaks.

### Headers forwarding

The app may set response headers (e.g. `Content-Type: text/event-stream`). The first chunk from the SDK should be a small JSON header frame:

```json
{"headers": {"content-type": "text/event-stream"}, "status_code": 200}
```

The client-facing endpoint reads this first frame, sets the response headers accordingly, then streams the remaining chunks as raw bytes.

## Changes

### New/modified files

1. **`pkg/api/apiv1/checkpoint.go`**
   - Add `StreamIngest(w, r)` handler — reads app's stream, buffers + publishes
   - Add `StreamOutput(w, r)` handler — replays buffer + subscribes for client

2. **`pkg/api/apiv1/checkpoint_stream.go`** (new)
   - `streamBuffer` struct and its `streamRegistry` (map + TTL eviction)
   - Logic for buffering, replay, and cleanup

3. **`pkg/api/apiv1/checkpoint.go` router** (in `NewCheckpointAPI`)
   - Add route: `api.Post("/{runID}/stream", api.StreamIngest)`
   - Add route: `api.Get("/{runID}/stream", api.StreamOutput)`

### What we reuse

- **JWT auth** from `apiv1auth` (same `CreateRunJWT`/`VerifyRunJWT` for the client endpoint)
- **Signing key auth** for the ingest endpoint (same as `CheckpointSteps`)
- **Broadcaster** for fan-out (already initialized in dev server and injected into the API)
- **SSE-style flushing** pattern from `sub_sse.go`

### What we do NOT change

- Existing `Output` endpoint — it continues to work for non-streaming responses
- Existing realtime API routes — the new endpoint lives under `/checkpoint/`, not `/realtime/`
- Broadcaster internals — we use it as-is

## Late-joiner handling

The key challenge: the client may connect after some chunks have already been published.

Solution: the **ingest handler** owns a buffer. The **output handler** reads from that buffer first, then subscribes. Because both handlers share the same `streamRegistry` (in-memory on the same process, which is fine for the Dev Server), there's no race:

1. Ingest writes chunk N to buffer, then publishes to broadcaster
2. Client connects, locks buffer, reads chunks 0..N
3. Client subscribes to broadcaster (will receive chunk N+1 onward)
4. No gap, no duplication — the buffer is the source of truth for catch-up

For ordering correctness, the output handler tracks how many chunks it replayed from the buffer, and skips that many from the broadcaster (or uses a sequence number on each chunk).

## Out of scope (for POC)

- Multi-process / distributed buffering (Dev Server is single-process)
- Persistent stream storage
- Backpressure from slow clients
- Compression
