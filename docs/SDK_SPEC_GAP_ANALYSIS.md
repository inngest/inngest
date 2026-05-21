# SDK Spec Gap Analysis & Update Proposal

This document identifies gaps between the current SDK spec (`SDK_SPEC.md`) and the actual implementations in the Go, TypeScript, and Python SDKs, then proposes spec updates.

---

## Part 1: Gap Analysis

### 1. Environment Variables

#### 1.1 `INNGEST_SERVE_HOST` vs `INNGEST_SERVE_ORIGIN`

| SDK    | Variable Used                                                            | Spec Variable          |
|--------|--------------------------------------------------------------------------|------------------------|
| Go     | `INNGEST_SERVE_HOST`                                                     | `INNGEST_SERVE_ORIGIN` |
| TS     | Both (`INNGEST_SERVE_HOST` deprecated, `INNGEST_SERVE_ORIGIN` preferred) | `INNGEST_SERVE_ORIGIN` |
| Python | `INNGEST_SERVE_ORIGIN`                                                   | `INNGEST_SERVE_ORIGIN` |

**Gap**: Go SDK uses `INNGEST_SERVE_HOST` instead of `INNGEST_SERVE_ORIGIN`.

#### 1.2 Extra env vars not in spec

All three SDKs support these additional variables:

| Variable                                 | Go  | TS  | Python | In Spec?                |
|------------------------------------------|-----|-----|--------|-------------------------|
| `INNGEST_BASE_URL`                       | Yes | Yes | Yes    | **Yes** (section 3.2)   |
| `INNGEST_STREAMING`                      | Yes | Yes | Yes    | **Yes** (section 3.2)   |
| `INNGEST_ALLOW_IN_BAND_SYNC`             | No  | Yes | Yes    | **Yes** (section 4.3.5) |
| `INNGEST_CONNECT_MAX_WORKER_CONCURRENCY` | Yes | Yes | Yes    | **Yes** (section 8.1)   |
| `INNGEST_CONNECT_ISOLATE_EXECUTION`      | No  | Yes | No     | **Yes** (section 8.1)   |
| `INNGEST_CONNECT_GATEWAY_URL`            | No  | Yes | No     | **Yes** (section 8.1)   |
| `INNGEST_LOG_LEVEL`                      | No  | No  | Yes    | Recommended             |
| `INNGEST_THREAD_POOL_MAX_WORKERS`        | No  | No  | Yes    | No                      |

Platform detection env vars (Vercel, Railway, Render, etc.) are also read by all SDKs but not in the spec. **Note**: The spec now mentions platform detection in section 3.2.

---

### 2. HTTP Headers

#### 2.1 `X-Inngest-Req-Version` value

~~**Spec says**: MUST be `1`.~~
~~**All three SDKs send**: `2`.~~

**Status**: DONE. Spec now says MUST be `2` (section 4.1.2).

#### 2.2 Extra headers not in spec

| Header                               | Go  | TS  | Python | In Spec?                |
|--------------------------------------|-----|-----|--------|-------------------------|
| `X-Inngest-Sync-Kind`                | Yes | Yes | Yes    | **Yes** (section 4.1.1) |
| `X-Inngest-Event-Id-Seed`            | Yes | No  | Yes    | No                      |
| `Server-Timing`                      | No  | Yes | Yes    | **Yes** (section 4.1.1) |
| `X-Inngest-Signature` (on responses) | Yes | No  | Yes    | No                      |

#### 2.3 Response signing

The Go SDK signs responses back to Inngest using `X-Inngest-Signature`. This is not mentioned in the spec and is not implemented in the other SDKs.

---

### 3. Sync Payload / Function Configuration

The spec documents function config fields in section 4.3.2. Several fields present in all SDKs are missing from the spec:

#### 3.1 `throttle` (all SDKs)

Distinct from `rateLimit`. Used for rate-based flow control with different semantics.

```typescript
throttle?: {
  key?: string;
  limit: number;
  period: TimeStr;
  burst?: number;
}
```

**Status**: DONE. `throttle` with `key`, `limit`, `period`, and `burst` is now in the spec (section 4.3.2).

#### 3.2 `singleton` (all SDKs)

Ensures only one run per key executes at a time.

```typescript
singleton?: {
  key?: string;
  mode: "skip" | "cancel";
}
```

**Status**: DONE. `singleton` with `key` and `mode` is now in the spec (section 4.3.2).

#### 3.3 `timeouts` (all SDKs)

Controls how long a function has to start and finish.

```typescript
timeouts?: {
  start?: TimeStr;
  finish?: TimeStr;
}
```

**Status**: DONE. `timeouts` with `start` and `finish` is now in the spec (section 4.3.2).

#### 3.4 `batchEvents` extra fields

~~The spec defines `batchEvents` with only `maxSize` and `timeout`.~~

```typescript
batchEvents?: {
  maxSize: number;
  timeout: string;
  key?: string;    // Now in spec
  if?: string;     // Not in spec (TS only, possibly)
}
```

**Status**: `key` is DONE (section 4.3.2). `if` is still not in the spec.

#### 3.5 `checkpointing` (Go and TS SDKs)

Checkpointing allows step results to be persisted to the Inngest Server during execution via API calls, providing durability for long-running sync executions. Configurable with `bufferedSteps`, `maxInterval`, and `maxRuntime`. Present in Go and TypeScript SDKs; not yet in Python.

**Status**: DONE. Checkpointing is now fully documented in section 10, covering configuration, sync/async opcode classification, checkpoint API endpoints, execution flows (async and sync modes), graceful fallback, and implementation differences between Go and TS SDKs.

#### 3.6 Step runtime `type`

~~The spec only documents `type: "http"`.~~

**Status**: DONE. The spec now documents both `type: "http"` (section 4.3.2) and `type: "ws"` (section 8.2).

---

### 4. Step Types

The spec documents four step types in section 5.3: Run, Sleep, WaitForEvent, Invoke. All three SDKs implement additional step types:

| Step Type         | Opcode                      | Go  | TS  | Python | In Spec?        |
|-------------------|-----------------------------|-----|-----|--------|-----------------|
| Run               | `StepRun`                   | Yes | Yes | Yes    | Yes             |
| Sleep             | `Sleep`                     | Yes | Yes | Yes    | Yes             |
| WaitForEvent      | `WaitForEvent`              | Yes | Yes | Yes    | Yes             |
| Invoke            | `InvokeFunction`            | Yes | Yes | Yes    | Yes             |
| ~~**SendEvent**~~ | (uses `StepRun` internally) | Yes | Yes | Yes    | **Yes** (5.3.5) |
| **WaitForSignal** | `WaitForSignal`             | Yes | Yes | No     | **No**          |
| ~~**AI Infer**~~  | `AiGateway`                 | Yes | Yes | Yes    | **Yes** (5.3.6) |
| ~~**Fetch**~~     | `Gateway`                   | Yes | Yes | No     | **Yes** (5.3.7) |

#### 4.1 `step.sendEvent()` / `step.send()`

Sends events as a step. Implemented as a wrapper around `step.run()` in Go, but a distinct method in TS/Python.

#### 4.2 `step.waitForSignal()` (opcode: `WaitForSignal`)

Signal-based synchronization between runs. Supports conflict modes: `"fail"` or `"replace"`. Present in Go and TS.

#### 4.3 `step.ai.infer()` (opcode: `AiGateway`)

AI model inference via Inngest's gateway. Supports multiple providers (OpenAI, Anthropic, Gemini, Bedrock). Present in all three SDKs (experimental in Python).

#### 4.4 `step.fetch()` (opcode: `Gateway`)

HTTP gateway calls through Inngest. Present in Go and TS.

#### 4.5 Additional internal opcodes

These exist in the codebase but are not developer-facing step types:

| Opcode | Purpose |
|--------|--------|
| `StepFailed` | Final step failure marker |
| `RunComplete` | Function completion marker |
| `SyncRunComplete` | Sync API completion |
| `DiscoveryRequest` | Continued discovery |

---

### 5. `StepNotFound` Implementation

| SDK    | Implements `StepNotFound`? |
|--------|----------------------------|
| Go     | Yes                        |
| TS     | Yes                        |
| Python | **No**                     |

**Gap**: Python SDK does not implement the `StepNotFound` opcode, which the spec requires.

---

### 6. Introspection Response

#### 6.1 `capabilities` field

All three SDKs include a `capabilities` field in the authenticated introspection response:

```typescript
capabilities?: {
  in_band_sync?: string;  // e.g. "v1"
  trust_probe?: string;   // e.g. "v1"
  connect?: string;       // e.g. "v1"
}
```

**Status**: DONE. `capabilities` is now in both the authenticated and unauthenticated introspection response schemas (section 4.5). The spec explicitly allows `capabilities` as an exception to the "no unspecified top-level keys" rule.

#### 6.2 `extra` field usage

The TS SDK puts `is_streaming` and `native_crypto` in `extra`. This is spec-compliant.

---

### 7. Middleware Lifecycle

The spec defines 7 lifecycle methods for function runs (section 6.3.1). Implementation varies:

| Lifecycle Method   | Spec     | Go     | TS                             | Python |
|--------------------|----------|--------|--------------------------------|--------|
| Transform input    | Required | Yes    | Yes (`transformFunctionInput`) | Yes    |
| Before memoization | Required | **No** | **No**                         | **No** |
| After memoization  | Required | **No** | Yes (`onMemoizationEnd`)       | **No** |
| Before execution   | Required | Yes    | Yes (`onRunStart`)             | Yes    |
| After execution    | Required | Yes    | Yes (`onRunComplete`)          | Yes    |
| Transform output   | Required | Yes    | Yes                            | Yes    |
| Before response    | Required | **No** | Yes (`wrapRequest`)            | Yes    |

**Gap**: No SDK implements `before_memoization`. Go SDK is missing `after_memoization` and `before_response`.

**Status**: DONE. The spec has been simplified: `before_memoization` was removed. `after_memoization` and `before_response` are now RECOMMENDED (not REQUIRED). The spec also acknowledges additional step-level and event send hooks as optional extensions (section 6.3.1).

#### 7.1 Extra middleware hooks (not in spec)

| Hook                         | Go  | TS  | Python |
|------------------------------|-----|-----|--------|
| `onPanic` / error recovery   | Yes | No  | No     |
| `onStepStart/Complete/Error` | No  | Yes | No     |
| `wrapStep/wrapStepHandler`   | No  | Yes | No     |
| `onRegister` (static)        | No  | Yes | No     |
| `before/after_send_events`   | No  | Yes | Yes    |
| `transformStepInput`         | No  | Yes | No     |
| `transformSendEvent`         | No  | Yes | No     |

The TS SDK has the richest middleware system with step-level hooks and wrapping patterns.

---

### 8. Connect Protocol

All three SDKs implement a WebSocket-based Connect protocol for persistent connections between the SDK and Inngest.

- **Go**: `inngestgo/connect/` (handler.go, connection.go, handshake.go, invoke.go, buffer.go, workerapi.go, worker_pool.go)
- **TS**: `inngest-js/packages/inngest/src/components/connect/` (index.ts, strategies/core/connection.ts, buffer.ts, messages.ts, util.ts)
- **Python**: `inngest-py/pkg/inngest/inngest/connect/`

**Status**: Section 8 of the spec has been expanded with full client-side protocol documentation covering:
- Wire format (Protobuf over binary WebSocket frames)
- All 15 message types with direction and purpose
- Full connection lifecycle (HTTP start API → WebSocket handshake → steady-state)
- Function execution flow (ack → lease extension → reply → reply ack)
- Reconnection logic with exponential backoff
- Message buffering and reliable delivery (at-least-once semantics)
- Graceful shutdown protocol
- Gateway draining and transparent reconnection
- Worker pool configuration
- Connection state machine
- Implementation differences between Go and TS SDKs

---

### 9. Streaming

All SDKs support HTTP streaming responses via `INNGEST_STREAMING`.

**Status**: DONE. Streaming is documented in the spec (section 9) and `INNGEST_STREAMING` is in section 3.2.

---

### 10. In-Band Sync

SDKs support in-band synchronization (sync during function execution) controlled by `INNGEST_ALLOW_IN_BAND_SYNC` and signaled via the `X-Inngest-Sync-Kind` header.

**Status**: DONE. In-band sync is documented in section 4.3.5, `X-Inngest-Sync-Kind` header in section 4.1.1.

---

### 11. `on_failure` Handler

The Python SDK auto-creates internal failure handler functions triggered by `inngest/function.failed`. This pattern exists in all SDKs.

**Status**: DONE. Failure handlers are documented in section 10.

---

## Part 2: Spec Update Proposal

Based on the gap analysis, here are the proposed changes to `SDK_SPEC.md`, ordered by priority:

### Priority 1: Fix Incorrect/Outdated Spec Content

#### P1.1: ~~Update `X-Inngest-Req-Version` to `2`~~ DONE

Spec section 4.1.2 now says MUST be `2`.

#### P1.2: ~~Add `throttle` to function configuration~~ DONE

`throttle` with `key`, `limit`, `period`, and `burst` is now in section 4.3.2.

#### P1.3: ~~Add `singleton` to function configuration~~ DONE

`singleton` with `key` and `mode` is now in section 4.3.2.

#### P1.4: ~~Add `timeouts` to function configuration~~ DONE

`timeouts` with `start` and `finish` is now in section 4.3.2.

#### P1.5: ~~Extend `batchEvents` with `key`~~ DONE

`batchEvents.key` is now in section 4.3.2.

---

### Priority 2: Document New Step Types

#### P2.1: ~~Add `step.sendEvent()` (section 5.3.5)~~ DONE

`step.sendEvent()` is now documented in section 5.3.5.

#### P2.2: Add `step.waitForSignal()` — NOT DONE

`step.waitForSignal()` (opcode: `WaitForSignal`) is still not documented in the spec. Present in Go and TS SDKs.

Proposed spec content:

```
### 5.3.X. Wait for Signal

A Wait for Signal Step pauses the Run until a signal is received from
another Run or external source, or until a timeout is reached.

{
  id: string;
  op: "WaitForSignal";
  opts: {
    signal: string;
    timeout: TimeStr;
    if?: string;
    on_conflict?: "fail" | "replace";
  };
  displayName?: string;
}

The memoized result will be the signal payload if received, or `null`
if the timeout elapsed.
```

#### P2.3: ~~Add `step.ai.infer()` (section 5.3.7)~~ DONE

`step.ai.infer()` (opcode: `AiGateway`) is now documented in section 5.3.6.

#### P2.4: ~~Add `step.fetch()` (section 5.3.8)~~ DONE

`step.fetch()` (opcode: `Gateway`) is now documented in section 5.3.7.

---

### Priority 3: Document New Features

#### P3.1: ~~Add Connect protocol section (section 8)~~ DONE

Section 8 of `SDK_SPEC.md` has been expanded with comprehensive client-side protocol documentation derived from the Go SDK (`inngestgo/connect/`) and TypeScript SDK (`inngest-js/packages/inngest/src/components/connect/`) implementations. The updated section covers the full connection lifecycle (HTTP start API, 3-step WebSocket handshake, steady-state message loop), all 15 message types, function execution flow, reconnection with exponential backoff, message buffering for reliable delivery, graceful shutdown, gateway draining, worker pool management, and implementation differences between the two SDKs.

#### P3.2: ~~Add streaming section (section 9)~~ DONE

Section 9 now documents streaming, including the `INNGEST_STREAMING` env var (also in section 3.2).

#### P3.3: ~~Add in-band sync documentation~~ DONE

In-band sync is now documented in section 4.3.5. The `X-Inngest-Sync-Kind` header is defined in section 4.1.1.

#### P3.4: ~~Add `capabilities` to introspection response~~ DONE

`capabilities` is now in both the authenticated and unauthenticated introspection response schemas (section 4.5). The spec explicitly allows `capabilities` as an exception to the "no unspecified top-level keys" rule.

---

### Priority 4: Normalize SDK Differences

#### P4.1: Standardize `INNGEST_SERVE_ORIGIN` naming

The Go SDK should migrate from `INNGEST_SERVE_HOST` to `INNGEST_SERVE_ORIGIN` (TS already did this, marking `INNGEST_SERVE_HOST` as deprecated).

#### P4.2: Python SDK should implement `StepNotFound`

The spec requires `StepNotFound` to be returned when a targeted step cannot be found. Python does not implement this.

#### P4.3: ~~Align middleware lifecycle~~ DONE

The spec has been simplified in section 6.3.1: `before_memoization` was removed entirely. `after_memoization` and `before_response` are now RECOMMENDED (not REQUIRED). The spec also acknowledges additional step-level and event send hooks as optional extensions.

#### P4.4: ~~Add `on_failure` handler to spec~~ DONE

Failure handlers are now documented in section 10.

---

### Priority 5: Housekeeping

#### P5.1: ~~Document `INNGEST_BASE_URL`~~ DONE

`INNGEST_BASE_URL` is now documented in section 3.2.

#### P5.2: ~~Document platform detection~~ DONE

Platform detection is now mentioned in section 3.2.

#### P5.3: ~~Add `Server-Timing` header~~ DONE

`Server-Timing` header is now documented as MAY in section 4.1.1.

#### P5.4: Document `X-Inngest-Event-Id-Seed` header

Used by Go and Python SDKs for deterministic event ID generation.

---

## Summary Table

| Gap                                     | Priority | Effort      | Status                            |
|-----------------------------------------|----------|-------------|-----------------------------------|
| ~~`X-Inngest-Req-Version` = 2~~         | ~~P1~~   | ~~Trivial~~ | **DONE**                          |
| ~~Add `throttle`~~                      | ~~P1~~   | ~~Small~~   | **DONE**                          |
| ~~Add `singleton`~~                     | ~~P1~~   | ~~Small~~   | **DONE**                          |
| ~~Add `timeouts`~~                      | ~~P1~~   | ~~Small~~   | **DONE**                          |
| ~~Extend `batchEvents` with `key`~~     | ~~P1~~   | ~~Trivial~~ | **DONE**                          |
| ~~Add `step.sendEvent`~~                | ~~P2~~   | ~~Small~~   | **DONE**                          |
| Add `step.waitForSignal`                | P2       | Medium      | **NOT DONE**                      |
| ~~Add `step.ai.infer`~~                 | ~~P2~~   | ~~Medium~~  | **DONE**                          |
| ~~Add `step.fetch`~~                    | ~~P2~~   | ~~Medium~~  | **DONE**                          |
| ~~Document Connect~~                    | ~~P3~~   | ~~Large~~   | **DONE**                          |
| ~~Document streaming~~                  | ~~P3~~   | ~~Small~~   | **DONE**                          |
| ~~Document in-band sync~~               | ~~P3~~   | ~~Small~~   | **DONE**                          |
| ~~Add `capabilities` to introspection~~ | ~~P3~~   | ~~Trivial~~ | **DONE**                          |
| Standardize `INNGEST_SERVE_ORIGIN`      | P4       | Small       | NOT DONE (Go SDK code change)     |
| Python `StepNotFound`                   | P4       | Small       | NOT DONE (Python SDK code change) |
| ~~Simplify middleware lifecycle~~       | ~~P4~~   | ~~Medium~~  | **DONE**                          |
| ~~Add `on_failure` handler~~            | ~~P4~~   | ~~Small~~   | **DONE**                          |
| ~~Document `INNGEST_BASE_URL`~~         | ~~P5~~   | ~~Trivial~~ | **DONE**                          |
| ~~Document platform detection~~         | ~~P5~~   | ~~Small~~   | **DONE**                          |
| ~~Add `Server-Timing` header~~          | ~~P5~~   | ~~Trivial~~ | **DONE**                          |
| Document `X-Inngest-Event-Id-Seed`      | P5       | Trivial     | NOT DONE                          |
| Document response signing (Go only)     | P5       | Small       | NOT DONE                          |
|                                         |          |             |                                   |
