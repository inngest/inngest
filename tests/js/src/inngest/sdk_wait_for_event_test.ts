import { inngest } from "@/inngest/client";

export const testWaitForEvent = inngest.createFunction(
  {
    id: "wait-for-event",
  },
  { event: "tests/wait.test" },
  async ({ event, step }) => {
    // Wait for 10 seconds.

    // hacky types, please ignore me.
    const result = await step.waitForEvent(
      "test/resume",
      {
        event: "test/resume",
        if: "async.data.resume == true && async.data.id == event.data.id",
        timeout: "10s",
      },
    );

    return { result: result?.data };
  }
);
