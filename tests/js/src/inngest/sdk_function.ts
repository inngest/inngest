import { inngest } from "@/inngest/client";

export const testSdkFunctions = inngest.createFunction(
  { id: "simple-fn", triggers: [{ event: "tests/function.test" }] },
  async ({ event }) => {
    return { name: event.name, body: "ok" };
  }
);
