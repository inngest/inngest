import { inngest } from "@/inngest/client";

export const testCancel = inngest.createFunction(
  {
    name: "Cancel test",
    cancel: [
      {
        event: "cancel/please",
        timeout: "1h",
        if: "async.data.request_id == event.data.request_id",
      },
    ]
  },
  { event: "tests/cancel.test" },
  async ({ event, step }) => {
    await step.sleep("10s");
    return { name: event.name, body: "ok" };
  }
);
