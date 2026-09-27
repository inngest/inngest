"""HTTP fixture for directly probing inngest-python protocol behavior."""

from __future__ import annotations

import datetime
import os
from typing import Final

import inngest
import inngest.fast_api
from fastapi import FastAPI, Response

SDK_VERSION: Final = "0.5.19"
SDK_COMMIT: Final = "c69d9a756a61d8596e1b3284f880d19fdf75dd77"
SERVE_PATH: Final = "/api/inngest"
CLOUD_SERVE_PATH: Final = "/api/inngest/cloud"

signing_key = os.getenv(
    "INNGEST_SIGNING_KEY", "7468697320697320612074657374206b6579"
)
signing_key_fallback = os.getenv(
    "INNGEST_SIGNING_KEY_FALLBACK", "66616c6c6261636b20636f6e666f726d616e6365"
)

client = inngest.Inngest(
    app_id="sdk-conformance",
    event_key=os.getenv("INNGEST_EVENT_KEY", "test"),
    signing_key=signing_key,
    is_production=False,
)

cloud_client = inngest.Inngest(
    app_id="sdk-conformance",
    event_key=os.getenv("INNGEST_EVENT_KEY", "test"),
    signing_key=signing_key,
    is_production=True,
)


@client.create_function(
    fn_id="basic-completion",
    trigger=inngest.TriggerEvent(event="conformance/basic-completion"),
)
async def basic_completion(ctx: inngest.Context) -> object:
    return {"kind": "basic-completion", "value": ctx.event.data.get("value")}


@client.create_function(
    fn_id="step-run",
    trigger=inngest.TriggerEvent(event="conformance/step-run"),
)
async def step_run(ctx: inngest.Context) -> object:
    async def operation() -> dict[str, str]:
        return {"value": "step-ran"}

    result = await ctx.step.run("deterministic-step", operation)
    return {"kind": "step-run", **result}


@client.create_function(
    fn_id="step-sleep",
    trigger=inngest.TriggerEvent(event="conformance/step-sleep"),
)
async def step_sleep(ctx: inngest.Context) -> object:
    await ctx.step.sleep("deterministic-sleep", datetime.timedelta(seconds=1))
    return {"kind": "step-sleep", "slept": True}


@client.create_function(
    fn_id="wait-for-event",
    trigger=inngest.TriggerEvent(event="conformance/wait-for-event"),
)
async def wait_for_event(ctx: inngest.Context) -> object:
    received = await ctx.step.wait_for_event(
        "deterministic-wait",
        event="conformance/wait-for-event.resume",
        timeout=datetime.timedelta(hours=1),
    )
    return {
        "kind": "wait-for-event",
        "received": None if received is None else received.data,
    }


@client.create_function(
    fn_id="function-error",
    retries=0,
    trigger=inngest.TriggerEvent(event="conformance/function-error"),
)
async def function_error(ctx: inngest.Context) -> object:
    del ctx
    raise RuntimeError("conformance function error")


@client.create_function(
    fn_id="step-error",
    retries=0,
    trigger=inngest.TriggerEvent(event="conformance/step-error"),
)
async def step_error(ctx: inngest.Context) -> object:
    async def operation() -> object:
        raise RuntimeError("conformance step error")

    return await ctx.step.run("deterministic-error", operation)


FUNCTIONS = [
    basic_completion,
    step_run,
    step_sleep,
    wait_for_event,
    function_error,
    step_error,
]

FIXTURES = [
    {
        "id": "basic-completion",
        "behavior": "basic_completion",
        "function_id": "sdk-conformance-basic-completion",
        "event": "conformance/basic-completion",
    },
    {
        "id": "step-run",
        "behavior": "step_run",
        "function_id": "sdk-conformance-step-run",
        "event": "conformance/step-run",
    },
    {
        "id": "step-sleep",
        "behavior": "step_sleep",
        "function_id": "sdk-conformance-step-sleep",
        "event": "conformance/step-sleep",
    },
    {
        "id": "wait-for-event",
        "behavior": "wait_for_event",
        "function_id": "sdk-conformance-wait-for-event",
        "event": "conformance/wait-for-event",
    },
    {
        "id": "function-error",
        "behavior": "function_error",
        "function_id": "sdk-conformance-function-error",
        "event": "conformance/function-error",
    },
    {
        "id": "step-error",
        "behavior": "step_error",
        "function_id": "sdk-conformance-step-error",
        "event": "conformance/step-error",
    },
]

app = FastAPI(title="Inngest Python SDK conformance fixture")


@app.get("/__conformance")
async def conformance_manifest() -> object:
    return {
        "schema_version": "0.1",
        "sdk": {
            "language": "python",
            "version": SDK_VERSION,
            "commit": SDK_COMMIT,
        },
        "profiles": ["serve.http.v1", "serve.execution.v1"],
        "capabilities": [
            "serve.introspection",
            "fixture.reset",
            "function.basic_completion",
            "step.run",
            "step.sleep",
            "step.wait_for_event",
            "error.function",
            "error.step",
        ],
        "serve_path": SERVE_PATH,
        "cloud_serve_path": CLOUD_SERVE_PATH,
        "fixtures": FIXTURES,
    }


@app.post("/__conformance/reset")
async def conformance_reset() -> Response:
    # All behavior is intentionally stateless; this endpoint remains part of
    # the fixture lifecycle contract and is safe to call between every case.
    return Response(status_code=204)


inngest.fast_api.serve(
    app,
    client,
    FUNCTIONS,
    enable_unauthed_sync=True,
    serve_path=SERVE_PATH,
)

inngest.fast_api.serve(
    app,
    cloud_client,
    FUNCTIONS,
    enable_unauthed_sync=False,
    serve_path=CLOUD_SERVE_PATH,
)
