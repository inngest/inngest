# Inngest Connect SDK Specification

This document is the common SDK contract for Inngest Connect. It describes the
worker-side protocol used by SDKs to maintain a persistent WebSocket connection
to an Inngest Connect Gateway, receive executor requests, renew request leases,
return SDK responses, and drain cleanly during gateway or client shutdown.

The wire contract is defined by `proto/connect/v1/connect.proto`. The gateway
implementation lives in `pkg/connect/gateway.go`, and the worker API endpoints
live in `pkg/connect/rest/v0/workerapi.go`.

## Protocol Overview

Connect has two transport surfaces:

- HTTP worker API, rooted at `{apiBaseUrl}/v0/connect`.
- Gateway WebSocket API, rooted at `{gatewayEndpoint}/v0/connect`.

SDKs use the HTTP worker API to select a gateway and obtain short-lived session
credentials. SDKs then use the WebSocket API for the long-lived execution
channel.

All WebSocket messages are binary protobuf frames. The top-level frame is always
`ConnectMessage`:

```proto
message ConnectMessage {
  GatewayMessageType kind = 1;
  bytes payload = 2;
}
```

`kind` identifies the payload type. Empty payloads MUST be encoded as empty
bytes or omitted. Receivers MUST ignore unknown fields in protobuf payloads.

The WebSocket subprotocol MUST be:

```text
v0.connect.inngest.com
```

## HTTP Worker API

### Start

Before opening a WebSocket, the SDK MUST call:

```text
POST {apiBaseUrl}/v0/connect/start
Authorization: Bearer {hashedSigningKey}
X-Inngest-Env: {envName}  # optional
Content-Type: application/octet-stream
```

Request body: protobuf `StartRequest`.

```proto
message StartRequest {
  repeated string exclude_gateways = 1;
}
```

`exclude_gateways` is a best-effort list of gateway groups the SDK prefers not
to reconnect to. The API may still return an excluded group if no better gateway
is available.

Success response body: protobuf `StartResponse`.

```proto
message StartResponse {
  string connection_id = 1;
  string gateway_endpoint = 2;
  string gateway_group = 3;
  string session_token = 4;
  string sync_token = 5;
}
```

The SDK MUST use:

- `connection_id` in the subsequent `WORKER_CONNECT` message.
- `gateway_endpoint` as the WebSocket base URL.
- `gateway_group` for reconnect routing and drain avoidance.
- `session_token` and `sync_token` in `AuthData`.

Start errors:

| HTTP status | Meaning | SDK behavior |
| --- | --- | --- |
| 400 | Invalid or unreadable protobuf request body. | Treat as configuration or SDK bug; retry only after changing the request. |
| 401 | Missing or invalid signing key/environment. | If a fallback key is configured, switch keys and retry. Otherwise fail terminally. |
| 429 | Account/environment connection limit reached. | Retry with backoff unless the SDK has a user-facing terminal policy. |
| 5xx | API or gateway selection failure. | Retry with backoff. |

### Flush Buffered Response

If a request completes while no WebSocket connection can accept the reply, the
SDK SHOULD flush the response over HTTP:

```text
POST {apiBaseUrl}/v0/connect/flush
Authorization: Bearer {hashedSigningKey}
X-Inngest-Env: {envName}  # optional
Content-Type: application/octet-stream
```

Request body: protobuf `SDKResponse`.

Success response body:

```proto
message FlushResponse {
  string request_id = 1;
}
```

The server stores the response in the request buffer and best-effort notifies
the executor. `FlushResponse.request_id` MUST match the flushed SDK response.

Flush errors:

| HTTP status | Meaning | SDK behavior |
| --- | --- | --- |
| 400 | Missing body, invalid protobuf, or invalid trace context. | Treat as SDK bug for that response. |
| 401 | Auth failure. | Switch fallback key if available; otherwise fail terminally. |
| 5xx | Buffering failure. | Retry with backoff while the response is still relevant. |

## WebSocket Connection Lifecycle

### Handshake

The handshake is ordered and strict:

1. SDK opens `{gatewayEndpoint}/v0/connect` with subprotocol
   `v0.connect.inngest.com`.
2. Gateway sends `GATEWAY_HELLO`.
3. SDK sends `WORKER_CONNECT` within 5 seconds of gateway hello.
4. Gateway authenticates, syncs apps, subscribes the connection for routing, and
   sends `GATEWAY_CONNECTION_READY`.
5. If `worker_manual_readiness_ack` is false, the gateway marks the connection
   `READY` immediately after sending `GATEWAY_CONNECTION_READY`.
6. If `worker_manual_readiness_ack` is true, the SDK MUST send `WORKER_READY`
   after it is ready to receive executor requests.

SDKs SHOULD apply their own handshake timeout, usually greater than the gateway
5 second read timeout.

### Worker Connect Payload

`WORKER_CONNECT` payload is `WorkerConnectRequestData`.

```proto
message WorkerConnectRequestData {
  string connection_id = 1;
  string instance_id = 2;
  AuthData auth_data = 3;
  bytes capabilities = 4;
  repeated AppConfiguration apps = 5;
  bool worker_manual_readiness_ack = 6;
  SystemAttributes system_attributes = 7;
  optional string environment = 8;
  string framework = 9;
  optional string platform = 10;
  string sdk_version = 11;
  string sdk_language = 12;
  google.protobuf.Timestamp started_at = 13;
  optional int64 max_worker_concurrency = 14;
}
```

Nested payloads:

```proto
message AppConfiguration {
  string app_name = 1;
  optional string app_version = 2;
  bytes functions = 4;
}

message AuthData {
  string session_token = 1;
  string sync_token = 2;
}

message SystemAttributes {
  int32 cpu_cores = 1;
  int64 mem_bytes = 2;
  string os = 3;
}
```

Required fields:

- `connection_id`: ULID returned by `/start`.
- `instance_id`: stable identifier for this SDK process. It MUST remain the same
  across reconnects for the same running process so request leases and capacity
  accounting continue to work.
- `auth_data.session_token`: session token returned by `/start`.
- `auth_data.sync_token`: sync token returned by `/start`.
- `apps`: one or more app configurations, unless the SDK is intentionally
  connecting with no apps for a specialized mode.
- `framework`: framework identifier. Connect SDKs commonly use `connect`.
- `sdk_version` and `sdk_language`.
- `started_at`: timestamp when this SDK process started.

`max_worker_concurrency` controls per-instance request capacity. `0` or unset
means unlimited capacity from the gateway's perspective.

`AppConfiguration.functions` is SDK-defined bytes consumed by app sync. SDKs
MUST encode it in the format expected by the app sync implementation for their
language.

Handshake failures:

| Case | Gateway close/error | SDK behavior |
| --- | --- | --- |
| SDK does not send `WORKER_CONNECT` within 5 seconds. | Close with `connect_worker_hello_timeout`. | Reconnect with backoff. |
| First SDK message is not `WORKER_CONNECT`. | Close with `connect_worker_hello_invalid_msg`. | Treat as SDK bug; reconnect only after fixing behavior. |
| `WORKER_CONNECT` payload is invalid protobuf. | Close with `connect_worker_hello_invalid_payload`. | Treat as SDK bug. |
| `connection_id` is not a ULID. | Close with `connect_worker_hello_invalid_payload`. | Restart from `/start`. |
| `instance_id` is empty. | Close with `connect_worker_hello_invalid_payload`. | Treat as SDK bug. |
| Auth returns nil. | Close with `connect_authentication_failed`. | Switch fallback key if available; otherwise fail terminally. |
| App count exceeds entitlement. | Close with `connect_too_many_apps_per_connection`. | Reduce apps per connection; do not hot-loop. |
| App sync fails with a user-facing connect socket error. | Gateway sends `SYNC_FAILED`, then closes with that syscode. | Surface the sync error and reconnect only after config changes or normal backoff policy. |
| Gateway is draining. | Close with status 1001 and `connect_gateway_closing`. | Reconnect using `/start` and add the old `gateway_group` to `exclude_gateways`. |

## WebSocket Message Reference

All message kinds are values of `GatewayMessageType`.

| Kind | Value | Direction | Payload | Purpose |
| --- | ---: | --- | --- | --- |
| `GATEWAY_HELLO` | 0 | Gateway -> SDK | Empty | Starts the handshake. |
| `WORKER_CONNECT` | 1 | SDK -> Gateway | `WorkerConnectRequestData` | Authenticates and registers the worker connection. |
| `GATEWAY_CONNECTION_READY` | 2 | Gateway -> SDK | `GatewayConnectionReadyData` | Communicates heartbeat, lease, and status intervals; connection can become ready. |
| `GATEWAY_EXECUTOR_REQUEST` | 3 | Gateway -> SDK | `GatewayExecutorRequestData` | Delivers a function or step execution request. |
| `WORKER_READY` | 4 | SDK -> Gateway | Empty | Marks a manual-readiness connection ready for traffic. |
| `WORKER_REQUEST_ACK` | 5 | SDK -> Gateway | `WorkerRequestAckData` | Confirms the SDK received and started handling a request. |
| `WORKER_REPLY` | 6 | SDK -> Gateway | `SDKResponse` | Sends the execution response. |
| `WORKER_REPLY_ACK` | 7 | Gateway -> SDK | `WorkerReplyAckData` | Confirms the gateway buffered/accepted the response. |
| `WORKER_PAUSE` | 8 | SDK -> Gateway | Empty | Marks the connection draining so it receives no new requests. |
| `WORKER_HEARTBEAT` | 9 | SDK -> Gateway | Empty | Keeps the connection live. |
| `GATEWAY_HEARTBEAT` | 10 | Gateway -> SDK | Empty | Acknowledges a worker heartbeat. |
| `GATEWAY_CLOSING` | 11 | Gateway -> SDK | Empty | Gateway is draining; SDK should establish a replacement connection. |
| `WORKER_REQUEST_EXTEND_LEASE` | 12 | SDK -> Gateway | `WorkerRequestExtendLeaseData` | Extends a long-running request lease. |
| `WORKER_REQUEST_EXTEND_LEASE_ACK` | 13 | Gateway -> SDK | `WorkerRequestExtendLeaseAckData` | Returns a renewed lease ID or a lease-extension NACK. |
| `SYNC_FAILED` | 14 | Gateway -> SDK | `SystemError` | Reports app sync failure before closing. |
| `WORKER_STATUS` | 15 | SDK -> Gateway | `WorkerStatusData` | Periodic observability for in-flight work and shutdown state. |

### GATEWAY_HELLO

Empty payload. The gateway sends this immediately after accepting a non-draining
WebSocket. The SDK MUST respond with `WORKER_CONNECT`.

### WORKER_CONNECT

Payload: `WorkerConnectRequestData`. See "Worker Connect Payload".

### GATEWAY_CONNECTION_READY

Payload:

```proto
message GatewayConnectionReadyData {
  string heartbeat_interval = 1;
  string extend_lease_interval = 2;
  string status_interval = 3;
}
```

Durations are Go duration strings, for example `10s`, `30s`, or `0s`.

SDK behavior:

- Parse `heartbeat_interval`; if empty or invalid, use the SDK default of 10s.
- Parse `extend_lease_interval`; if empty or invalid, use the SDK default of 30s.
- Parse `status_interval`; `0s` or empty disables worker status reporting.
- Start heartbeat scheduling after this message.
- Start status reporting only when `status_interval` is greater than zero.
- If manual readiness is enabled, send `WORKER_READY` only after local request
  handlers are ready.

Current gateway defaults:

- `heartbeat_interval`: 10s.
- `extend_lease_interval`: 30s.
- `status_interval`: 0s, disabled.

### GATEWAY_EXECUTOR_REQUEST

Payload:

```proto
message GatewayExecutorRequestData {
  string request_id = 1;
  string account_id = 2;
  string env_id = 3;
  string app_id = 4;
  string app_name = 5;
  string function_id = 6;
  string function_slug = 7;
  optional string step_id = 8;
  bytes request_payload = 9;
  bytes system_trace_ctx = 10;
  bytes user_trace_ctx = 11;
  string run_id = 12;
  string lease_id = 13;
  string job_id = 14;
}
```

SDK behavior:

1. Validate the connection is active and accepting work.
2. Validate the request targets a registered app/function.
3. Immediately send `WORKER_REQUEST_ACK` before executing user code.
4. Track the request as in-flight using `request_id`.
5. Track `lease_id` and start periodic `WORKER_REQUEST_EXTEND_LEASE`.
6. Execute the request.
7. Send `WORKER_REPLY` when execution completes.

If a request cannot be accepted because the SDK is already paused or shutting
down, the SDK SHOULD avoid ACKing it. The gateway/executor will retry routing
after the existing lease expires or the forward path fails.

### WORKER_READY

Empty payload. Only required when `worker_manual_readiness_ack` is true in
`WORKER_CONNECT`.

If the gateway service is draining, `WORKER_READY` is rejected with
`connect_gateway_closing`. If this specific connection is already draining,
`WORKER_READY` is ignored and MUST NOT reset the connection back to ready.

### WORKER_REQUEST_ACK

Payload:

```proto
message WorkerRequestAckData {
  string request_id = 1;
  string account_id = 2;
  string env_id = 3;
  string app_id = 4;
  string function_slug = 5;
  optional string step_id = 6;
  bytes system_trace_ctx = 7;
  bytes user_trace_ctx = 8;
  string run_id = 9;
}
```

The SDK MUST send this immediately after accepting
`GATEWAY_EXECUTOR_REQUEST`. The gateway forwards the ACK to the executor via
gRPC. The gateway does not send a reverse ACK for `WORKER_REQUEST_ACK`.

Invalid protobuf payload closes the connection with
`connect_worker_request_ack_invalid_payload`.

### WORKER_REQUEST_EXTEND_LEASE

Payload:

```proto
message WorkerRequestExtendLeaseData {
  string request_id = 1;
  string account_id = 2;
  string env_id = 3;
  string app_id = 4;
  string function_slug = 5;
  optional string step_id = 6;
  bytes system_trace_ctx = 7;
  bytes user_trace_ctx = 8;
  string run_id = 9;
  string lease_id = 10;
}
```

SDK behavior:

- Send every `extend_lease_interval` while the request is in-flight.
- Use the latest known `lease_id`.
- If a new active connection exists, send extensions over it. During gateway
  draining, in-flight requests may have been delivered on the old connection but
  can renew through the new connection because leases are keyed by
  `instance_id`, `request_id`, and `lease_id`, not by the old WebSocket.
- If no WebSocket is open, skip that interval tick and try again on the next
  interval after reconnect.

Current gateway lease constants:

- Initial request lease duration: 2 minutes.
- Grace period used by forwarding/ACK waits: 5 seconds.
- Default lease extension interval: 30 seconds.

Invalid protobuf payload or invalid `lease_id` ULID closes the connection with
`connect_worker_request_extend_lease_invalid_payload`.

### WORKER_REQUEST_EXTEND_LEASE_ACK

Payload:

```proto
message WorkerRequestExtendLeaseAckData {
  string request_id = 1;
  string account_id = 2;
  string env_id = 3;
  string app_id = 4;
  string function_slug = 5;
  optional string new_lease_id = 6;
}
```

SDK behavior:

- If `new_lease_id` is present, replace the tracked lease ID and continue
  extensions.
- If `new_lease_id` is absent, treat the extension as a NACK. The request lease
  was expired, claimed elsewhere, deleted, or the worker capacity record no
  longer exists. Stop extending that request. The SDK may continue local
  execution, but the final reply might be ignored or race with another attempt.

### WORKER_REPLY

Payload:

```proto
enum SDKResponseStatus {
  NOT_COMPLETED = 0;
  DONE = 1;
  ERROR = 2;
}

message SDKResponse {
  string request_id = 1;
  string account_id = 2;
  string env_id = 3;
  string app_id = 4;
  SDKResponseStatus status = 5;
  bytes body = 6;
  bool no_retry = 7;
  optional string retry_after = 8;
  string sdk_version = 9;
  uint32 request_version = 10;
  bytes system_trace_ctx = 11;
  bytes user_trace_ctx = 12;
  string run_id = 13;
}
```

The gateway stores the response in a reliable buffer, then best-effort notifies
the executor. The SDK MUST keep the response pending until it receives
`WORKER_REPLY_ACK` or successfully flushes via HTTP.

If `WORKER_REPLY` cannot be sent because no WebSocket is open, the SDK SHOULD
buffer the response locally and call `/v0/connect/flush`. If a WebSocket
reconnects before HTTP flush succeeds, the SDK MAY send `WORKER_REPLY` over the
new connection.

Invalid response protobuf or buffering failure closes the connection with
`connect_internal_error`.

### WORKER_REPLY_ACK

Payload:

```proto
message WorkerReplyAckData {
  string request_id = 1;
}
```

The gateway sends this after it has accepted and attempted to buffer the
response. The SDK MAY delete any local response buffer for `request_id` after
receiving this ACK.

### WORKER_PAUSE

Empty payload. The SDK sends this when it wants to stop receiving new work, most
commonly during client shutdown.

Gateway behavior:

- Marks the connection `DRAINING` in state.
- Removes the connection from the in-memory routing map.
- Stops forwarding new executor requests to this WebSocket.
- Accepts future `WORKER_HEARTBEAT`, `WORKER_REQUEST_ACK`,
  `WORKER_REQUEST_EXTEND_LEASE`, and `WORKER_REPLY` messages so in-flight
  requests can finish.
- Does not let later heartbeats reset the connection to `READY`.

The gateway accepts `WORKER_PAUSE` even while the gateway itself is draining.

### WORKER_HEARTBEAT

Empty payload. The SDK sends this every `heartbeat_interval`.

Gateway behavior:

- Updates connection status to `READY`, unless the gateway or connection is
  draining, in which case it updates/keeps `DRAINING`.
- Refreshes worker capacity TTL for the connection `instance_id`.
- Sends `GATEWAY_HEARTBEAT`.
- Records the last heartbeat time for missed-heartbeat detection.

If `instance_id` is missing, the gateway closes with `connect_internal_error`.
That should be impossible after a valid handshake.

### GATEWAY_HEARTBEAT

Empty payload. The gateway sends this in response to `WORKER_HEARTBEAT`.

SDK behavior:

- Reset any pending-heartbeat/missed-heartbeat counter.
- Treat missing gateway heartbeat responses as a connection health failure.
  A common policy is to reconnect after 2 consecutive missed responses.

### GATEWAY_CLOSING

Empty payload. The gateway sends this when it is draining for shutdown or
deployment.

SDK behavior:

1. Mark the current active WebSocket as draining, not dead.
2. Keep the draining WebSocket open so in-flight requests can continue sending
   heartbeats, lease extensions, ACKs, and replies.
3. Start a new `/v0/connect/start` request immediately.
4. Include the old `gateway_group` in `exclude_gateways`.
5. Establish a replacement WebSocket and complete the handshake.
6. Once the replacement connection is active or ready, close the old draining
   WebSocket with normal closure.

The old draining connection MUST NOT accept newly delivered requests from the
SDK's local scheduler, but it may still process messages already received.

Gateway timing:

- Gateway write timeout for `GATEWAY_CLOSING`: 5 seconds.
- Gateway drain ACK timeout: 25 seconds by default.
- After the timeout or after the worker closes, the gateway marks the old
  connection `DRAINING`, stops forwarding to it, and force-closes it with status
  1001 and reason `connect_gateway_closing`.

### SYNC_FAILED

Payload:

```proto
message SystemError {
  string code = 1;
  optional bytes data = 2;
  string message = 3;
}
```

The gateway may send this when app sync fails after `WORKER_CONNECT`. The
gateway then closes the WebSocket with the same underlying connect socket error.
SDKs SHOULD surface the error to the user with `code` and `message`.

### WORKER_STATUS

Payload:

```proto
message WorkerStatusData {
  repeated string in_flight_request_ids = 1;
  bool shutdown_requested = 2;
}
```

SDK behavior:

- Send only when `GATEWAY_CONNECTION_READY.status_interval` is greater than
  zero.
- Include all request IDs currently executing locally.
- Set `shutdown_requested` after graceful shutdown begins.
- Stop status reporting when the connection closes or status interval is zero.

Gateway behavior:

- Rate limits status processing to at most once every 2 seconds.
- Invalid protobuf payload is logged and ignored.
- Status currently provides observability only; it does not mutate routing
  state.

## Normal Execution Flow

```text
SDK -> API:      POST /v0/connect/start
API -> SDK:      StartResponse(connection_id, gateway_endpoint, gateway_group, tokens)
Gateway -> SDK:  GATEWAY_HELLO
SDK -> Gateway:  WORKER_CONNECT
Gateway -> SDK:  GATEWAY_CONNECTION_READY
SDK -> Gateway:  WORKER_READY                 # only when manual readiness is enabled

Gateway -> SDK:  GATEWAY_EXECUTOR_REQUEST
SDK -> Gateway:  WORKER_REQUEST_ACK
SDK -> Gateway:  WORKER_REQUEST_EXTEND_LEASE  # repeated while in-flight
Gateway -> SDK:  WORKER_REQUEST_EXTEND_LEASE_ACK(new_lease_id)
SDK -> Gateway:  WORKER_REPLY
Gateway -> SDK:  WORKER_REPLY_ACK
```

The SDK MUST ACK before running user code so the gateway/executor knows the
request reached the worker. The SDK MUST keep extending the lease until the
request completes or the gateway NACKs the extension.

## Heartbeat Flow

```text
SDK -> Gateway:  WORKER_HEARTBEAT
Gateway -> SDK:  GATEWAY_HEARTBEAT
```

SDKs SHOULD send the first heartbeat after receiving
`GATEWAY_CONNECTION_READY`, then continue at `heartbeat_interval`.

Gateway missed-heartbeat behavior:

- The gateway checks once per `heartbeat_interval`.
- If `time_since_last_heartbeat > 5 * heartbeat_interval`, the gateway cancels
  the read loop and disconnects the worker with reason
  `CONSECUTIVE_HEARTBEATS_MISSED`.
- With the default 10 second heartbeat interval, that is about 50 seconds.

SDK missed-heartbeat behavior:

- Track heartbeats sent but not answered by `GATEWAY_HEARTBEAT`.
- If the SDK misses its configured threshold, it SHOULD treat the connection as
  unexpectedly terminated and reconnect.

Heartbeats during drain:

- Heartbeats MUST continue while either gateway drain or client shutdown drain
  is in progress.
- The gateway MUST respond with `GATEWAY_HEARTBEAT`.
- The connection status MUST remain `DRAINING`, not return to `READY`.

## Unexpected Connection Termination

Unexpected termination includes WebSocket close without
`WORKER_SHUTDOWN`, network errors, read/write errors, missed heartbeat
thresholds, process-level gateway failure, and a gateway force close.

SDK behavior:

1. Mark the WebSocket dead and remove it from the active slot.
2. Keep local in-flight request records, lease IDs, timers, and response buffers.
3. Start reconnect with exponential backoff and jitter.
4. Call `/v0/connect/start` for each new attempt. Do not reuse an old
   `connection_id`.
5. Keep the same `instance_id` for the running SDK process.
6. Resume lease extensions for in-flight requests when a new WebSocket is ready.
7. Send final replies over the new WebSocket or via `/v0/connect/flush`.
8. Do not send new local work to a dead connection.

Recommended reconnect triggers:

- WebSocket close or error event.
- WebSocket read or write failure.
- Missing `GATEWAY_HEARTBEAT` responses beyond the SDK threshold.
- `GATEWAY_CLOSING`.
- Non-terminal `/start` failure.

Recommended backoff schedule:

```text
1s, 2s, 5s, 10s, 20s, 30s, 60s, 120s, 300s
```

SDKs SHOULD add jitter and SHOULD reset the attempt counter after a stable
connection is established.

## Gateway Draining Flow

Gateway draining is initiated by the gateway during shutdown, deployment, or
maintenance.

Gateway behavior:

1. Gateway marks itself draining and rejects new WebSocket connections with
   status 1001 and reason `connect_gateway_closing`.
2. For existing connections, gateway sends `GATEWAY_CLOSING`.
3. Gateway marks the connection as draining in memory immediately so heartbeats
   cannot return it to ready.
4. Gateway waits for the SDK to close the old WebSocket or for the drain ACK
   timeout.
5. Gateway marks the old connection `DRAINING` in state and stops forwarding new
   requests to it.
6. Gateway force-closes the old WebSocket if needed.

SDK behavior:

1. Keep old connection open and mark it draining.
2. Establish a new connection before closing the old one.
3. Include the old `gateway_group` in `StartRequest.exclude_gateways`.
4. Continue in-flight request ACKs, lease extensions, heartbeats, status, and
   replies while the old connection is open.
5. Prefer the new active connection for future lease extensions and replies once
   available.
6. Close the old connection after the new connection is active/ready.

This is an overlap reconnect. The SDK MUST NOT tear down the old connection
first unless the gateway already closed it, because the old connection may still
be needed to drain in-flight requests while the replacement is handshaking.

## Client Shutdown Flow

Client shutdown is initiated by the SDK process, usually from `close()`,
SIGINT, SIGTERM, or hosting-platform shutdown.

SDK behavior:

1. Set a local `shutdown_requested` flag.
2. Stop accepting new local work.
3. If an active WebSocket is open, send `WORKER_PAUSE`.
4. Keep heartbeat and status reporting running while in-flight requests drain.
5. Continue lease extensions for in-flight requests.
6. Allow in-flight requests to complete and send `WORKER_REPLY`.
7. Wait for `WORKER_REPLY_ACK` where possible; otherwise flush responses via
   `/v0/connect/flush`.
8. When no in-flight requests remain and all pending responses are acknowledged
   or flushed, close WebSockets with:

```text
WebSocket status: 1000
Reason: WORKER_SHUTDOWN
```

Gateway behavior after `WORKER_PAUSE`:

- Marks the connection `DRAINING`.
- Removes it from routing.
- Continues to accept heartbeats, ACKs, lease extensions, and replies.
- Cleans up connection state when the WebSocket closes.
- Records normal disconnect reason when the close reason is `WORKER_SHUTDOWN`.

Shutdown during reconnect backoff:

- If shutdown is requested while the SDK is waiting to reconnect and there are
  no in-flight requests or pending responses, cancel the backoff and exit.
- If in-flight requests exist, continue attempting to reconnect or flush so
  leases and replies can complete.

## Error and Close Handling

Connect socket errors are represented by WebSocket close status plus a string
syscode reason. Important syscodes:

| Syscode | Meaning | SDK behavior |
| --- | --- | --- |
| `connect_worker_hello_timeout` | SDK did not send `WORKER_CONNECT` fast enough. | Reconnect with backoff; investigate blocked event loop/startup. |
| `connect_worker_hello_invalid_msg` | First SDK message was not `WORKER_CONNECT`. | SDK bug; fix protocol ordering. |
| `connect_worker_hello_invalid_payload` | `WORKER_CONNECT` payload invalid. | SDK bug or stale `/start`; restart from `/start`. |
| `connect_authentication_failed` | Session token/sync token invalid. | Switch fallback signing key if available; otherwise terminal auth failure. |
| `connect_internal_error` | Gateway internal failure. | Reconnect with backoff unless repeated with same config. |
| `connect_gateway_closing` | Gateway is draining. | Overlap reconnect to another gateway group. |
| `connect_invalid_function_config` | App sync found invalid function config. | Surface to user; retry after config change. |
| `connect_worker_request_ack_invalid_payload` | `WORKER_REQUEST_ACK` payload invalid. | SDK bug; connection closes. |
| `connect_too_many_apps_per_connection` | App count exceeded entitlement. | Split apps across connections or reduce app count. |
| `connect_worker_request_extend_lease_invalid_payload` | Lease extension payload or lease ID invalid. | SDK bug for that request/connection. |

Worker disconnect reasons recorded by the gateway:

| Reason | When used |
| --- | --- |
| `WORKER_SHUTDOWN` | SDK closed normally with status 1000 and reason `WORKER_SHUTDOWN`. |
| `UNEXPECTED` | Default for unexpected closes without a more specific reason. |
| `GATEWAY_DRAINING` | Gateway initiated drain or force-closed for drain. |
| `CONSECUTIVE_HEARTBEATS_MISSED` | Gateway missed too many worker heartbeats. |
| `MESSAGE_TOO_LARGE` | Worker sent a message larger than gateway read limit. |

Message size:

- The gateway sets the WebSocket read limit to `MaxSDKResponseBodySize`.
- If the SDK sends a frame over that limit, the gateway records
  `MESSAGE_TOO_LARGE` and disconnects.

Unknown message kinds:

- The gateway logs and ignores unexpected message kinds after handshake.
- SDKs SHOULD ignore unknown gateway message kinds unless they are required for
  a negotiated capability.

Malformed payloads:

- Malformed critical payloads (`WORKER_CONNECT`, `WORKER_REQUEST_ACK`,
  `WORKER_REQUEST_EXTEND_LEASE`, `WORKER_REPLY`) can close the connection.
- Malformed `WORKER_STATUS` is ignored.

## SDK State Machine

SDKs SHOULD model at least these states:

```text
IDLE
CONNECTING
ACTIVE
DRAINING_OLD_CONNECTION
RECONNECTING
PAUSING
DRAINING_IN_FLIGHT
CLOSED
```

Recommended transitions:

```text
IDLE -> CONNECTING
CONNECTING -> ACTIVE
CONNECTING -> RECONNECTING
ACTIVE -> DRAINING_OLD_CONNECTION      # GATEWAY_CLOSING
ACTIVE -> RECONNECTING                 # unexpected termination
ACTIVE -> PAUSING                      # client shutdown
DRAINING_OLD_CONNECTION -> ACTIVE      # replacement connected
PAUSING -> DRAINING_IN_FLIGHT          # WORKER_PAUSE sent
DRAINING_IN_FLIGHT -> CLOSED           # no in-flight requests or pending replies
RECONNECTING -> ACTIVE
RECONNECTING -> CLOSED                 # shutdown and no in-flight work
```

The SDK should maintain separate references for:

- Active connection: receives new work.
- Draining connection: old gateway connection kept open during gateway drain.
- In-flight requests: keyed by `request_id`.
- Pending replies: completed responses waiting for `WORKER_REPLY_ACK` or HTTP
  flush success.
- Excluded gateway groups: best-effort reconnect avoidance list.

## Conformance Checklist

An SDK implementation is Connect-compatible when it:

- Calls `/v0/connect/start` and uses the returned connection/session/sync data.
- Opens the gateway WebSocket with subprotocol `v0.connect.inngest.com`.
- Sends and receives binary protobuf `ConnectMessage` frames.
- Implements all `GatewayMessageType` values listed in this document.
- Sends `WORKER_CONNECT` within the gateway handshake timeout.
- Supports automatic readiness and manual `WORKER_READY`.
- ACKs accepted executor requests before executing user code.
- Tracks and extends leases until completion or lease-extension NACK.
- Buffers completed replies until `WORKER_REPLY_ACK` or successful HTTP flush.
- Sends regular heartbeats and reconnects after missed gateway heartbeat ACKs.
- Keeps old gateway-draining connections open while establishing replacements.
- Sends `WORKER_PAUSE` for client shutdown and drains in-flight work before
  closing with `WORKER_SHUTDOWN`.
- Handles auth fallback, connection limits, sync errors, malformed messages, and
  gateway close syscodes without hot-looping.
