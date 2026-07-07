import { inngest } from "@/v3/client";

export const testSdkFunctions = inngest.createFunction(
  { id: "simple-fn" },
  { event: "tests/function.test" },
  async ({ event }) => {
    return { name: event.name, body: "ok" };
  }
);
