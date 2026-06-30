import { inngest } from "@/inngest/client";

// Helper function invoked by testParallelFanIn.  Sleeps a random 1-5s so
// concurrent invokes complete at different times, widening the race window.
export const sleepRandom = inngest.createFunction(
  { id: "sleep-random" },
  { event: "tests/sleep-random.test" },
  async ({ step }) => {
    const ms = 1000 + Math.floor(Math.random() * 4000);
    await step.sleep("zzz", `${ms}ms`);
    return { slept: ms };
  }
);

// Regression test for parallel fan-in runs wedging.  Drives 100 concurrent
// ops (50 step.run, 50 step.invoke) through one Promise.all and asserts the
// parent run completes with the expected aggregate.  step.run results sum to
// a known constant; the invoke count confirms every child finished — so the
// Go driver catches both "run wedged" and "step ran the wrong number of
// times" regressions.
export const testParallelFanIn = inngest.createFunction(
  { id: "parallel-fan-in" },
  { event: "tests/parallel-fan-in.test" },
  async ({ step }) => {
    const tasks: Promise<unknown>[] = [];

    for (let i = 0; i < 50; i++) {
      tasks.push(
        step.invoke(`invoke-${i}`, {
          function: sleepRandom,
          data: { name: "tests/sleep-random.test", data: {} },
        })
      );
    }

    for (let i = 0; i < 50; i++) {
      tasks.push(step.run(`run-${i}`, () => i * i));
    }

    const results = await Promise.all(tasks);

    // First 50 results are invokes (random sleep duration); last 50 are
    // step.run squares.  Sum the squares — 0² + 1² + … + 49² = 40425 — and
    // count completed invokes by checking the response shape.
    const invokeResults = results.slice(0, 50);
    const stepResults = results.slice(50) as number[];

    const stepSquares = stepResults.reduce((a, b) => a + b, 0);
    const invokeCount = invokeResults.filter(
      (r: any) => r && typeof r.slept === "number"
    ).length;

    return { stepSquares, invokeCount };
  }
);
