import { inngest } from "@/inngest/client";

export const testSdkFunctions = inngest.createFunction(
  {
    name: "SDK Function Test" },
  { event: "tests/function.test" },
  async ({ event }) => {
    return { name: event.name, body: "ok" };
  }
);
