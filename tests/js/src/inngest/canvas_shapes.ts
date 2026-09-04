// Fixture functions for the Run Canvas PoC. Additive: existing tests are untouched.
import { inngest } from "@/inngest/client";

// A child function to invoke, so we get a nested run linked via InvokeStepInfo.runID.
export const canvasChild = inngest.createFunction(
  { id: "canvas-child" },
  { event: "tests/canvas.child" },
  async ({ step }) => {
    await step.run("child work", async () => "child done");
    return { child: true };
  }
);

// step.invoke -> nested run.
export const canvasInvoke = inngest.createFunction(
  { id: "canvas-invoke" },
  { event: "tests/canvas.invoke" },
  async ({ step }) => {
    await step.run("before", async () => "before");
    const res = await step.invoke("call child", {
      function: canvasChild,
      data: {},
    });
    await step.run("after", async () => "after");
    return { res };
  }
);

// A run that fails outright, mid-run, after a successful step.
export const canvasFailure = inngest.createFunction(
  { id: "canvas-failure", retries: 0 },
  { event: "tests/canvas.failure" },
  async ({ step }) => {
    await step.run("ok step", async () => "fine");
    await step.run("doomed step", async () => {
      throw new Error("this step always fails");
    });
    return "never reached";
  }
);

// waitForEvent that is never satisfied, so it times out.
export const canvasWaitTimeout = inngest.createFunction(
  { id: "canvas-wait-timeout" },
  { event: "tests/canvas.wait-timeout" },
  async ({ step }) => {
    const got = await step.waitForEvent("never arrives", {
      event: "tests/canvas.never",
      timeout: "3s",
    });
    return { timedOut: got === null };
  }
);

// An agent-shaped while-loop: N sequential iterations of the same sub-shape.
// This is the readability problem for a left-to-right layout.
export const canvasLoop = inngest.createFunction(
  { id: "canvas-loop" },
  { event: "tests/canvas.loop" },
  async ({ event, step }) => {
    const iterations = Number(event.data?.iterations ?? 8);
    let acc = 0;
    for (let i = 0; i < iterations; i++) {
      const thought = await step.run(`think-${i}`, async () => i * 2);
      acc += await step.run(`act-${i}`, async () => thought + 1);
    }
    return { acc, iterations };
  }
);

// A wide fan-out, to see how the layout copes.
export const canvasWideFanout = inngest.createFunction(
  { id: "canvas-wide-fanout" },
  { event: "tests/canvas.wide" },
  async ({ step }) => {
    const n = 12;
    const results = await Promise.all(
      Array.from({ length: n }, (_, i) => step.run(`w${i}`, async () => i))
    );
    await step.run("collect", async () => results.length);
    return { n };
  }
);

// Parallel branches that are chains, not single steps.
export const canvasParallelChains = inngest.createFunction(
  { id: "canvas-parallel-chains" },
  { event: "tests/canvas.chains" },
  async ({ step }) => {
    const chain = async (name: string) => {
      const one = await step.run(`${name}-1`, async () => 1);
      const two = await step.run(`${name}-2`, async () => one + 1);
      return two;
    };
    const [x, y] = await Promise.all([chain("left"), chain("right")]);
    await step.run("join", async () => x + y);
    return { x, y };
  }
);

// Slow parallel steps, so an in-flight snapshot catches steps mid-execution.
// They must be planned before they run, which is what gives them step spans.
export const canvasSlow = inngest.createFunction(
  { id: "canvas-slow" },
  { event: "tests/canvas.slow" },
  async ({ step }) => {
    await step.run("quick", async () => "quick");
    const sleep = (n: number) => new Promise((r) => setTimeout(r, n));
    await Promise.all([
      step.run("slow-a", async () => { await sleep(9000); return "a"; }),
      step.run("slow-b", async () => { await sleep(3000); return "b"; }),
    ]);
    await step.run("after", async () => "after");
    return "done";
  }
);
