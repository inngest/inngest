# Durable Endpoint Streaming

Streams HTTP response data from an app's SDK handler through the Inngest server to a waiting client.

## Flow

Initially, the app streams directly to the client. But once the DE goes async, the Inngest Server becomes an intermediary between the app and the client.

Once in async mode:
- Client receives chunks via `GET /realtime/sse?token={jwt}`.
- App pushes chunks via `POST /realtime/publish/tee?channel={runID}`.

Under the hood, the chunks are published via Redis pub/sub. Read more in the [Broadcaster](../../pkg/execution/realtime/docs/broadcaster.md) docs.
