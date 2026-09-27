import { createServer } from "node:http";
import { Inngest } from "inngest";
import { serve } from "inngest/node";

const SDK_VERSION = "4.21.0";
const SDK_COMMIT = "197812b9e20d87315998027b0ad810ca8644c4ee";

const signingKey =
  process.env.INNGEST_SIGNING_KEY ??
  "7468697320697320612074657374206b6579";
const signingKeyFallback =
  process.env.INNGEST_SIGNING_KEY_FALLBACK ??
  "66616c6c6261636b20636f6e666f726d616e6365";

const inngest = new Inngest({
  id: "sdk-conformance",
  eventKey: process.env.INNGEST_EVENT_KEY ?? "test",
  signingKey,
  signingKeyFallback,
  isDev: true,
});

const cloudInngest = new Inngest({
  id: "sdk-conformance",
  eventKey: process.env.INNGEST_EVENT_KEY ?? "test",
  signingKey,
  signingKeyFallback,
  isDev: false,
});

const basicCompletion = inngest.createFunction(
  { id: "basic-completion", triggers: [{ event: "conformance/basic-completion" }] },
  async ({ event }) => ({ kind: "basic-completion", value: event.data.value ?? null }),
);

const stepRun = inngest.createFunction(
  { id: "step-run", triggers: [{ event: "conformance/step-run" }] },
  async ({ step }) => {
    const value = await step.run("deterministic-step", () => ({ value: "step-ran" }));
    return { kind: "step-run", ...value };
  },
);

const stepSleep = inngest.createFunction(
  { id: "step-sleep", triggers: [{ event: "conformance/step-sleep" }] },
  async ({ step }) => {
    await step.sleep("deterministic-sleep", "1s");
    return { kind: "step-sleep", slept: true };
  },
);

const waitForEvent = inngest.createFunction(
  { id: "wait-for-event", triggers: [{ event: "conformance/wait-for-event" }] },
  async ({ step }) => {
    const received = await step.waitForEvent("deterministic-wait", {
      event: "conformance/wait-for-event.resume",
      timeout: "1h",
    });
    return {
      kind: "wait-for-event",
      received: received === null ? null : received.data,
    };
  },
);

const functionError = inngest.createFunction(
  { id: "function-error", retries: 0, triggers: [{ event: "conformance/function-error" }] },
  async () => {
    throw new Error("conformance function error");
  },
);

const stepError = inngest.createFunction(
  { id: "step-error", retries: 0, triggers: [{ event: "conformance/step-error" }] },
  async ({ step }) => {
    await step.run("deterministic-error", () => {
      throw new Error("conformance step error");
    });
  },
);

const functions = [
  basicCompletion,
  stepRun,
  stepSleep,
  waitForEvent,
  functionError,
  stepError,
];

const manifest = {
  schema_version: "0.1",
  sdk: {
    language: "typescript",
    version: SDK_VERSION,
    commit: SDK_COMMIT,
  },
  profiles: ["serve.http.v1", "serve.execution.v2"],
  capabilities: [
    "serve.introspection",
    "function.basic_completion",
    "step.run",
    "step.sleep",
    "step.wait_for_event",
    "error.function",
    "error.step",
  ],
  serve_path: "/api/inngest",
  cloud_serve_path: "/api/inngest/cloud",
  fixtures: [
    {
      id: "basic-completion",
      behavior: "basic_completion",
      function_id: "sdk-conformance-basic-completion",
      event: "conformance/basic-completion",
    },
    {
      id: "step-run",
      behavior: "step_run",
      function_id: "sdk-conformance-step-run",
      event: "conformance/step-run",
    },
    {
      id: "step-sleep",
      behavior: "step_sleep",
      function_id: "sdk-conformance-step-sleep",
      event: "conformance/step-sleep",
    },
    {
      id: "wait-for-event",
      behavior: "wait_for_event",
      function_id: "sdk-conformance-wait-for-event",
      event: "conformance/wait-for-event",
    },
    {
      id: "function-error",
      behavior: "function_error",
      function_id: "sdk-conformance-function-error",
      event: "conformance/function-error",
    },
    {
      id: "step-error",
      behavior: "step_error",
      function_id: "sdk-conformance-step-error",
      event: "conformance/step-error",
    },
  ],
} as const;

const sdkHandler = serve({ client: inngest, functions });
const cloudSdkHandler = serve({ client: cloudInngest, functions });
const port = Number.parseInt(process.env.PORT ?? "3000", 10);

createServer((request, response) => {
  const path = new URL(request.url ?? "/", `http://${request.headers.host ?? "localhost"}`).pathname;

  if (request.method === "GET" && path === "/__conformance") {
    response.writeHead(200, { "content-type": "application/json; charset=utf-8" });
    response.end(JSON.stringify(manifest));
    return;
  }

  if (request.method === "POST" && path === "/__conformance/reset") {
    response.writeHead(204);
    response.end();
    return;
  }

  if (path === "/api/inngest") {
    void sdkHandler(request, response);
    return;
  }

  if (path === "/api/inngest/cloud") {
    void cloudSdkHandler(request, response);
    return;
  }

  response.writeHead(404, { "content-type": "application/json; charset=utf-8" });
  response.end(JSON.stringify({ error: "not found" }));
}).listen(port, "127.0.0.1", () => {
  console.log(`TypeScript conformance fixture listening on http://127.0.0.1:${port}`);
});
