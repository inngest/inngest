# SDK conformance (independent scaffold)

This directory is an independent scaffold for an Inngest SDK conformance suite.
It intentionally does not import, invoke, or derive its cases from
`pkg/conformance`.

## What the suite specifies

The contract is externally observable behavior at protocol boundaries:

1. the SDK serve endpoint (inspection, synchronization, and invocation);
2. requests made by the SDK to the Event API;
3. requests made by the SDK to Connect;
4. end-to-end run outcomes through an Inngest server.

It does not standardize language APIs, callback signatures, framework adapters,
or an exact internal execution transcript.

## Sources of authority

The current TypeScript SDK is the de facto reference implementation. Its
observable behavior supplies the default contract for a case.

[`docs/SDK_SPEC.md`](../docs/SDK_SPEC.md) is a useful design artifact, but may
lag the implementations. [`docs/SDK_SPEC_GAP_ANALYSIS.md`](../docs/SDK_SPEC_GAP_ANALYSIS.md)
tracks omissions and implementation differences; neither document overrides
current TypeScript behavior by itself.

Go and Python are independent corroborating implementations. When both agree
with TypeScript, confidence in the inferred contract increases. When both agree
with each other but differ from TypeScript, the case is disputed: TypeScript may
contain a bug. The harness must expose that disagreement rather than blessing
TypeScript automatically or deciding by majority vote.

A test case has one of three states:

- `reference`: captures current TypeScript SDK behavior;
- `compatibility`: preserves an explicitly documented older profile;
- `disputed`: credible cross-SDK evidence suggests the reference behavior may
  be wrong.

Reference and compatibility cases can block conformance. Disputed cases are
reported but cannot block another SDK until the underlying behavior is decided.

Every blocking case must record:

- the protocol profile it belongs to;
- the pinned TypeScript revision defining its behavior;
- observations from Go, TypeScript, and Python;
- supporting specification sections and decision records, when available.

The starting observations and unresolved disputes are recorded in
[`OBSERVATIONS.md`](./OBSERVATIONS.md).

## Architecture

```text
versioned case corpus ─────┐
                          ▼
SDK fixture ── HTTP ── vector runner ── semantic report
     │
     └──────── Inngest server ───────── outcome report
```

The first layer sends deterministic requests directly to an SDK fixture and
matches semantic response properties. It isolates SDK protocol behavior from
server scheduling and execution choices. A later scenario layer runs through a
real server and asserts durable outcomes such as completed output, retries, or
cancellation; it must not require one exact callback sequence.

## Runnable observation matrix

The scaffold includes equivalent fixtures for the pinned TypeScript, Go, and
Python SDKs under [`fixtures`](./fixtures), declarative direct-HTTP probes under
[`probes`](./probes), and a matrix runner under [`matrix`](./matrix).

Run the full matrix from the repository root inside the Nix development shell:

```bash
nix develop --command ./sdk-conformance/run-matrix.sh
```

The script starts all three fixtures, resets each fixture before every probe,
and writes the full unmodified response bodies plus a reference comparison to
`.amp/in/artifacts/sdk-conformance-report.json`. A difference is data, not a
test failure. The runner exits nonzero only when it cannot execute the matrix.

The initial probes cover:

- serve introspection;
- signed in-band synchronization;
- cloud-mode primary, fallback, wrong-key, stale, and future signatures;
- function completion and function errors;
- `step.run`, `step.sleep`, `step.waitForEvent`, and step errors.

Probe requests include asymmetric input and fixed IDs so implementations cannot
pass by returning a convenient constant or by generating mutually consistent
random values.

## Profiles and capabilities

Profiles version incompatible protocol shapes, for example `serve.execution.v1`
and `serve.execution.v2`. Capabilities identify orthogonal features such as
`step.wait_for_event` or `signature.fallback_key`. A target manifest declares
both. The runner reports an unmet capability as unsupported, not failed.

This distinction lets a generated SDK grow incrementally while ensuring that
every capability it claims is fully conformant.

## Fixture contract

Each SDK repository owns a deliberately small fixture application. The harness
starts it with:

- `PORT`: port on `127.0.0.1`;
- `INNGEST_EVENT_KEY`: deterministic test event key;
- `INNGEST_SIGNING_KEY`: deterministic test signing key.

The process must expose:

- `GET /__conformance`: the target manifest defined by
  [`target.schema.json`](./spec/v0.1/target.schema.json);
- `POST /__conformance/reset`: reset counters and mutable fixture state;
- the declared Inngest serve endpoint.

The fixture functions are specified as data in the manifest rather than by
copying one language's public API shape.

## Adoption plan

1. Capture raw request/response observations from pinned releases of all three
   SDKs without normalizing them.
2. Review differences and ratify the smallest common protocol requirements,
   with separate profiles where differences are intentional.
3. Implement fixtures in each SDK repository and run direct vectors in CI.
4. Add Event API vectors, then execution replay and additional opcodes.
5. Add end-to-end server scenarios only after direct protocol failures are easy
   to diagnose.
6. Use the versioned fixture manifest, cases, and conformance report as the
   acceptance contract for generated SDKs.

The matrix runner records observations without deciding which differences are
bugs. The smaller assertion runner in [`runner`](./runner) is the next layer:
reviewed observations become profile- and capability-scoped blocking cases.
Disputed cases run only when explicitly requested.

```bash
go run ./sdk-conformance/runner \
  --cases ./sdk-conformance/cases \
  --target http://127.0.0.1:3000 \
  --include-disputed
```
