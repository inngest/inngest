import { inngest } from "@/inngest/client";

export const testParallelism = inngest.createFunction(
  { name: "SDK Parallel Test" },
  { event: "tests/parallel.test" },
  async ({ step }) => {

    const [a, b, c] = await Promise.all([
      step.run("a", () => "a"),
      step.run("b", () => "b"),
      step.run("c", () => "c"),
      step.sleep("1s"),
    ]);

    const d = await step.run("d", () => "d");

    return { a, b, c, d };
  }
);
