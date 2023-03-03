import { inngest } from "@/inngest/client";

export const testSdkSteps = inngest.createFunction(
  { name: "SDK Step Test" },
  { event: "tests/step.test" },
  async ({ event, step }) => {

    const data = await step.run("first step", async () => {
      return "first step";
    });

    await step.sleep("2s");

    await step.run("second step", async () => {
      return { first: data, second: true };
    });

    return { name: event.name, body: "ok" };
  }
);
