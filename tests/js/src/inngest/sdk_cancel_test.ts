import { inngest } from "@/inngest/client";

export const testCancel = inngest.createFunction(
  {
    name: "Cancel test",
    cancelOn: [
      {
        event: "cancel/please",
        timeout: "1h",
        if: "async.data.request_id == event.data.request_id",
      },
    ]
  },
  { event: "tests/cancel.test" },
  async ({ event, step }) => {
    // Wait for 10 seconds.
    await step.sleep("10s");

    // Run a step, if not cancelled.
    await step.run("After the sleep", () => {
      return "This should be cancelled if a matching cancel event is received";
    })

    return { name: event.name, body: "ok" };
  }
);
