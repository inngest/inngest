import { inngest } from "@/inngest/client";
import { NonRetriableError } from "inngest";

export const testNonRetriableError = inngest.createFunction(
  { id: "no-retry" },
  { event: "tests/no-retry.test" },
  async ({ step }) => {
    try {
    await step.run("first step", async () => {
      throw new NonRetriableError("no retry plz")
    });
    } catch(e) {
      // Do nothing with this error.
    }
    return "ok"; 
  }
);

