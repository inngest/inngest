import { inngest } from "@/inngest/client";
import { NonRetriableError } from "inngest";

// In v4 with immediate execution, the SDK intercepts NonRetriableError and
// reports OpcodeStepFailed. The server then retries the function. On the
// retry, the step should succeed so the function can complete.
let hasThrown = false;

export const testNonRetriableError = inngest.createFunction(
  { id: "no-retry", triggers: [{ event: "tests/no-retry.test" }] },
  async ({ step }) => {
    try {
    await step.run("first step", async () => {
      if (!hasThrown) {
        hasThrown = true;
        throw new NonRetriableError("no retry plz")
      }
      return "step done";
    });
    } catch(e) {
      // Do nothing with this error.
    }
    hasThrown = false;
    return "ok";
  }
);

