import { inngest } from "@/inngest/client";

export const testParallelism = inngest.createFunction(
  { name: "SDK Parallel Test" },
  { event: "tests/parallel.test" },
  async ({ step }) => {

    const [a, b] = await Promise.all([
      step.run("a", () => "a"),
      step.run("b", () => "b"),
    ]);

    const c = await step.run("c", () => "c");

    return { a, b, c };
  }
);
