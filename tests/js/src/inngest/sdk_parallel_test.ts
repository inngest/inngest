import { inngest } from "@/inngest/client";

export const testParallelism = inngest.createFunction(
  { id: "step-parallelism" },
  { event: "tests/parallel.test" },
  async ({ step }) => {

    const [a, b, c] = await Promise.all([
      step.run("a", () => "a"),
      step.run("b", () => "b"),
      step.run("c", () => "c"),
    ]);

    const d = await step.run("d", () => "d");

    return { a, b, c, d };
  }
);
