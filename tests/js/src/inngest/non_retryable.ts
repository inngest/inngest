import { inngest } from "@/inngest/client";
import { NonRetriableError } from "inngest";

export const testNonRetriableError = inngest.createFunction(
  { name: "SDK No Retry" },
  { event: "tests/no-retry.test" },
  async ({ step }) => {
    await step.run("first step", async () => {
      throw new NonRetriableError("no retry plz")
    });
    return "ok"; 
  }
);

