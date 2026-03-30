# SDK Spec Gap Analysis & Update Proposal

This document identifies gaps between the current SDK spec (`SDK_SPEC.md`) and the actual implementations in the Go, TypeScript, and Python SDKs, then proposes spec updates.

---

## Part 1: Gap Analysis

### 1. Environment Variables

#### 1.1 `INNGEST_SERVE_HOST` vs `INNGEST_SERVE_ORIGIN`

| SDK | Variable Used | Spec Variable |
|-----|--------------|---------------|
| Go | `INNGEST_SERVE_HOST` | `INNGEST_SERVE_ORIGIN` |
| TS | Both (`INNGEST_SERVE_HOST` deprecated, `INNGEST_SERVE_ORIGIN` preferred) | `INNGEST_SERVE_ORIGIN` |
| Python | `INNGEST_SERVE_ORIGIN` | `INNGEST_SERVE_ORIGIN` |

**Gap**: Go SDK uses `INNGEST_SERVE_HOST` instead of `INNGEST_SERVE_ORIGIN`.

#### 1.2 Extra env vars not in spec

All three SDKs support these additional variables:

| Variable | Go | TS | Python | In Spec? |
|----------|----|----|--------|----------|
| `INNGEST_BASE_URL` | Yes | Yes | Yes | No |
| `INNGEST_STREAMING` | Yes | Yes | Yes | No |
| `INNGEST_ALLOW_IN_BAND_SYNC` | No | Yes | Yes | No |
| `INNGEST_CONNECT_MAX_WORKER_CONCURRENCY` | Yes | Yes | Yes | No |
| `INNGEST_CONNECT_ISOLATE_EXECUTION` | No | Yes | No | No |
| `INNGEST_CONNECT_GATEWAY_URL` | No | Yes | No | No |
| `INNGEST_LOG_LEVEL` | No | No | Yes | Recommended |
| `INNGEST_THREAD_POOL_MAX_WORKERS` | No | No | Yes | No |

Platform detection env vars (Vercel, Railway, Render, etc.) are also read by all SDKs but not in the spec.

---

### 2. HTTP Headers

#### 2.1 `X-Inngest-Req-Version` value

**Spec says**: MUST be `1`.
**All three SDKs send**: `2`.

This is the most significant divergence -- the spec is outdated. Execution version 2 is the current standard across all SDKs.

#### 2.2 Extra headers not in spec

| Header | Go | TS | Python | In Spec? |
|--------|----|----|--------|----------|
| `X-Inngest-Sync-Kind` | Yes | Yes | Yes | No |
| `X-Inngest-Event-Id-Seed` | Yes | No | Yes | No |
| `Server-Timing` | No | Yes | Yes | No |
| `X-Inngest-Signature` (on responses) | Yes | No | No | No |

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
  burst?: number;  // Not in spec, present in all SDKs
}
```

**Status**: `throttle` is entirely absent from the spec. `burst` sub-field is in all three SDKs.

#### 3.2 `singleton` (all SDKs)

Ensures only one run per key executes at a time.

```typescript
singleton?: {
  key?: string;
  mode: "skip" | "cancel";
}
```

**Status**: Not in the spec at all.

#### 3.3 `timeouts` (all SDKs)

Controls how long a function has to start and finish.

```typescript
timeouts?: {
  start?: TimeStr;
  finish?: TimeStr;
}
```

**Status**: Not in the spec at all.

#### 3.4 `batchEvents` extra fields

The spec defines `batchEvents` with only `maxSize` and `timeout`. All SDKs also support:

```typescript
batchEvents?: {
  maxSize: number;
  timeout: string;
  key?: string;    // Not in spec
  if?: string;     // Not in spec (TS only, possibly)
}
```

#### 3.5 `checkpointing` (Go and TS SDKs)

Checkpointing allows step results to be persisted to the Inngest Server during execution via API calls, providing durability for long-running sync executions. Configurable with `bufferedSteps`, `maxInterval`, and `maxRuntime`. Present in Go and TypeScript SDKs; not yet in Python.

#### 3.6 Step runtime `type`

The spec only documents `type: "http"`. The TS SDK also supports `type: "ws"` for WebSocket-based Connect.

---

### 4. Step Types

The spec documents four step types in section 5.3: Run, Sleep, WaitForEvent, Invoke. All three SDKs implement additional step types:

| Step Type | Opcode | Go | TS | Python | In Spec? |
|-----------|--------|----|----|--------|----------|
| Run | `StepRun` | Yes | Yes | Yes | Yes |
| Sleep | `Sleep` | Yes | Yes | Yes | Yes |
| WaitForEvent | `WaitForEvent` | Yes | Yes | Yes | Yes |
| Invoke | `InvokeFunction` | Yes | Yes | Yes | Yes |
| **SendEvent** | (uses `StepRun` internally) | Yes | Yes | Yes | **No** |
| **WaitForSignal** | `WaitForSignal` | Yes | Yes | No | **No** |
| **AI Infer** | `AiGateway` | Yes | Yes | Yes | **No** |
| **Fetch** | `Gateway` | Yes | Yes | No | **No** |

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

| SDK | Implements `StepNotFound`? |
|-----|---------------------------|
| Go | Yes |
| TS | Yes |
| Python | **No** |

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

**Status**: Not in the spec. The spec says "An SDK MUST NOT set any top-level keys not specified in the aforementioned schemas" but allows `extra` for arbitrary data. The `capabilities` field is at the top level, violating this rule.

#### 6.2 `extra` field usage

The TS SDK puts `is_streaming` and `native_crypto` in `extra`. This is spec-compliant.

---

### 7. Middleware Lifecycle

The spec defines 7 lifecycle methods for function runs (section 6.3.1). Implementation varies:

| Lifecycle Method | Spec | Go | TS | Python |
|-----------------|------|----|----|--------|
| Transform input | Required | Yes | Yes (`transformFunctionInput`) | Yes |
| Before memoization | Required | **No** | **No** | **No** |
| After memoization | Required | **No** | Yes (`onMemoizationEnd`) | **No** |
| Before execution | Required | Yes | Yes (`onRunStart`) | Yes |
| After execution | Required | Yes | Yes (`onRunComplete`) | Yes |
| Transform output | Required | Yes | Yes | Yes |
| Before response | Required | **No** | Yes (`wrapRequest`) | Yes |

**Gap**: No SDK implements `before_memoization`. Go SDK is missing `after_memoization` and `before_response`.

#### 7.1 Extra middleware hooks (not in spec)

| Hook | Go | TS | Python |
|------|----|----|--------|
| `onPanic` / error recovery | Yes | No | No |
| `onStepStart/Complete/Error` | No | Yes | No |
| `wrapStep/wrapStepHandler` | No | Yes | No |
| `onRegister` (static) | No | Yes | No |
| `before/after_send_events` | No | Yes | Yes |
| `transformStepInput` | No | Yes | No |
| `transformSendEvent` | No | Yes | No |

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

All SDKs support HTTP streaming responses via `INNGEST_STREAMING`. Not mentioned in the spec.

---

### 10. In-Band Sync

SDKs support in-band synchronization (sync during function execution) controlled by `INNGEST_ALLOW_IN_BAND_SYNC` and signaled via the `X-Inngest-Sync-Kind` header. Not in the spec.

---

### 11. `on_failure` Handler

The Python SDK auto-creates internal failure handler functions triggered by `inngest/function.failed`. This pattern exists in all SDKs but is not documented in the spec.

---

## Part 2: Spec Update Proposal

Based on the gap analysis, here are the proposed changes to `SDK_SPEC.md`, ordered by priority:

### Priority 1: Fix Incorrect/Outdated Spec Content

#### P1.1: Update `X-Inngest-Req-Version` to `2`

**Section 4.1.2**: Change "MUST be `1`" to "MUST be `2`".

All SDKs use version 2. Version 1 is effectively dead.

#### P1.2: Add `throttle` to function configuration

**Section 4.3.2**: Add alongside `rateLimit`:

```typescript
/**
 * Throttle function execution to a given number of runs per period.
 * Unlike rateLimit, throttle queues excess runs rather than dropping them.
 */
throttle?: {
  key?: string;
  limit: number;
  period: TimeStr;
  burst?: number;
};
```

#### P1.3: Add `singleton` to function configuration

**Section 4.3.2**: Add new field:

```typescript
/**
 * Ensures only one run per key executes at a time.
 * New runs are either skipped or cancel the existing run.
 */
singleton?: {
  key?: string;
  mode: "skip" | "cancel";
};
```

#### P1.4: Add `timeouts` to function configuration

**Section 4.3.2**: Add new field:

```typescript
/**
 * Controls how long a function has to start and finish execution.
 */
timeouts?: {
  /** Maximum time from trigger to first execution attempt. */
  start?: TimeStr;
  /** Maximum total time for the function run to complete. */
  finish?: TimeStr;
};
```

#### P1.5: Extend `batchEvents` with `key`

**Section 4.3.2**: Add optional `key` field to `batchEvents`:

```typescript
batchEvents?: {
  maxSize: number;
  timeout: string;
  /** Optional expression to partition batches by key. */
  key?: string;
};
```

---

### Priority 2: Document New Step Types

#### P2.1: Add `step.sendEvent()` (section 5.3.5)

```
### 5.3.5. Send Event

A Send Event Step sends one or more events to the Inngest Server as a
retriable step. An SDK MAY implement this as a wrapper around a Run Step
or as a distinct operation.

The step returns an array of event IDs for the sent events.
```

#### P2.2: Add `step.waitForSignal()` (section 5.3.6)

```
### 5.3.6. Wait for Signal

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

#### P2.3: Add `step.ai.infer()` (section 5.3.7)

```
### 5.3.7. AI Gateway

An AI Gateway Step sends an inference request through Inngest's AI
gateway, which handles provider routing, retries, and cost tracking.

{
  id: string;
  op: "AiGateway";
  opts: {
    // Provider-specific configuration
    url: string;
    headers?: Record<string, string>;
    body: any;
    format?: string;
  };
  displayName?: string;
}

The memoized result will be the inference response from the provider.
```

#### P2.4: Add `step.fetch()` (section 5.3.8)

```
### 5.3.8. Gateway (HTTP Fetch)

A Gateway Step makes an HTTP request through Inngest's gateway,
providing retries and observability for external API calls.

{
  id: string;
  op: "Gateway";
  opts: {
    url: string;
    method?: string;
    headers?: Record<string, string>;
    body?: any;
  };
  displayName?: string;
}

The memoized result will be the HTTP response.
```

---

### Priority 3: Document New Features

#### P3.1: ~~Add Connect protocol section (section 8)~~ DONE

Section 8 of `SDK_SPEC.md` has been expanded with comprehensive client-side protocol documentation derived from the Go SDK (`inngestgo/connect/`) and TypeScript SDK (`inngest-js/packages/inngest/src/components/connect/`) implementations. The updated section covers the full connection lifecycle (HTTP start API, 3-step WebSocket handshake, steady-state message loop), all 15 message types, function execution flow, reconnection with exponential backoff, message buffering for reliable delivery, graceful shutdown, gateway draining, worker pool management, and implementation differences between the two SDKs.

#### P3.2: Add streaming section (section 9)

```
# 9. Streaming

An SDK MAY support HTTP streaming responses using the
`INNGEST_STREAMING` environment variable. When enabled, the SDK sends
keep-alive data on the HTTP response while executing functions,
preventing platform timeouts on serverless environments.
```

#### P3.3: Add in-band sync documentation

**Section 4.3**: Add subsection for in-band sync:

```
### 4.3.5. In-Band Sync

An SDK MAY support in-band sync, where registration data is sent
alongside normal execution responses rather than as a separate POST
request. This is signaled via the `X-Inngest-Sync-Kind` header with
a value of `"in-band"`.

Controlled by the `INNGEST_ALLOW_IN_BAND_SYNC` environment variable.
```

#### P3.4: Add `capabilities` to introspection response

**Section 4.5**: Add `capabilities` to both authenticated and unauthenticated schemas, or move it into `extra`:

```typescript
// Authenticated response
{
  // ... existing fields ...
  capabilities?: {
    in_band_sync?: string;
    trust_probe?: string;
    connect?: string;
  };
}
```

Note: Currently all SDKs put `capabilities` at the top level, violating the spec rule about not setting unspecified top-level keys. The spec should either formalize `capabilities` or SDKs should move it to `extra`.

---

### Priority 4: Normalize SDK Differences

#### P4.1: Standardize `INNGEST_SERVE_ORIGIN` naming

The Go SDK should migrate from `INNGEST_SERVE_HOST` to `INNGEST_SERVE_ORIGIN` (TS already did this, marking `INNGEST_SERVE_HOST` as deprecated).

#### P4.2: Python SDK should implement `StepNotFound`

The spec requires `StepNotFound` to be returned when a targeted step cannot be found. Python does not implement this.

#### P4.3: Align middleware lifecycle

The spec's 7-hook lifecycle is not fully implemented by any SDK. Options:
1. **Simplify the spec** to match reality (remove `before_memoization` since no SDK implements it)
2. **Require implementation** of missing hooks

Recommendation: Simplify. The TS SDK's approach of `onMemoizationEnd` + wrapping patterns is more flexible than the spec's rigid 7-phase model. Consider rewriting section 6.3.1 to reflect a minimal required set:

Required:
- Transform input
- Before execution / After execution
- Transform output

Recommended:
- After memoization
- Before response
- Step-level hooks

#### P4.4: Add `on_failure` handler to spec

All SDKs support auto-creating failure handler functions. This should be documented.

---

### Priority 5: Housekeeping

#### P5.1: Document `INNGEST_BASE_URL`

All SDKs support this as a shorthand for setting both API and event base URLs.

#### P5.2: Document platform detection

All SDKs detect hosting platforms (Vercel, Railway, Render, etc.) via environment variables. This should be mentioned in section 3.

#### P5.3: Add `Server-Timing` header

TS and Python SDKs send `Server-Timing` headers for observability. This should be documented as optional.

#### P5.4: Document `X-Inngest-Event-Id-Seed` header

Used by Go and Python SDKs for deterministic event ID generation.

---

## Summary Table

| Gap | Priority | Effort | Impact |
|-----|----------|--------|--------|
| `X-Inngest-Req-Version` = 2 | P1 | Trivial | Spec is wrong |
| Add `throttle` | P1 | Small | Core feature undocumented |
| Add `singleton` | P1 | Small | Core feature undocumented |
| Add `timeouts` | P1 | Small | Core feature undocumented |
| Extend `batchEvents` with `key` | P1 | Trivial | Field exists in all SDKs |
| Add `step.sendEvent` | P2 | Small | Used in all SDKs |
| Add `step.waitForSignal` | P2 | Medium | New primitive |
| Add `step.ai.infer` | P2 | Medium | Growing feature |
| Add `step.fetch` | P2 | Medium | Growing feature |
| ~~Document Connect~~ | ~~P3~~ | ~~Large~~ | **DONE** -- Section 8 expanded |
| Document streaming | P3 | Small | All SDKs support |
| Document in-band sync | P3 | Small | All SDKs support |
| Add `capabilities` to introspection | P3 | Trivial | Spec violation |
| Standardize `INNGEST_SERVE_ORIGIN` | P4 | Small | Go SDK change |
| Python `StepNotFound` | P4 | Small | Spec compliance |
| Simplify middleware lifecycle | P4 | Medium | Spec/reality mismatch |
| Add `on_failure` handler | P4 | Small | All SDKs support |
| Document `INNGEST_BASE_URL` | P5 | Trivial | Convenience |
| Document platform detection | P5 | Small | All SDKs do it |
