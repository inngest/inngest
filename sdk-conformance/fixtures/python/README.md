# Python SDK conformance fixture

This is a standalone HTTP target for the independent SDK conformance harness.
It does not use the old conformance runner. Dependencies are pinned in
`uv.lock`, including `inngest==0.5.19`.

## Run

Install [`uv`](https://docs.astral.sh/uv/), then from this directory:

```bash
uv sync --frozen --python 3.12
export PORT=3000
INNGEST_EVENT_KEY=test-event-key \
INNGEST_SIGNING_KEY='signkey-test-00000000000000000000000000000000' \
uv run uvicorn app:app --host 127.0.0.1 --port "$PORT"
```

The fixture exposes:

- `GET /__conformance` for its v0.1 manifest;
- `POST /__conformance/reset` for the stateless reset lifecycle hook;
- `GET`, `PUT`, and `POST /api/inngest` through the SDK's FastAPI adapter.

The six event-triggered functions use stable IDs and cover basic completion,
`step.run`, `step.sleep`, `wait_for_event`, a function-level error, and a
step-level error. Their exact event names and fully qualified function IDs are
listed in the manifest.

## Checks

```bash
uv run python -m compileall -q app.py
uv run mypy app.py
curl --fail http://127.0.0.1:${PORT:-3000}/__conformance
curl --fail -X POST http://127.0.0.1:${PORT:-3000}/__conformance/reset
curl --fail http://127.0.0.1:${PORT:-3000}/api/inngest
```

In development mode, Python SDK 0.5.19's unauthenticated `PUT` path performs
out-of-band registration rather than returning an in-band registration body.
Direct inspection and invocation probing should account for that SDK-specific
behavior.
