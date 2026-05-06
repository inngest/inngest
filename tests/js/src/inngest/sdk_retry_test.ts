import { inngest } from "@/inngest/client";

// In v4 with immediate execution, step.run executes inline.
// If the step succeeds but the function body throws, the step result
// is NOT persisted — on retry the step re-executes. We use separate
// flags to avoid infinite retry loops.
let stepFirstCall = true;
let funcAttempt = 0;

export const testRetry = inngest.createFunction(
  { id: "retry-test", triggers: [{ event: "tests/retry.test" }] },
  async ({ event, step }) => {

    const data = await step.run("first step", async () => {
      if (stepFirstCall) {
        stepFirstCall = false;
        throw new Error("broken");
      }
      return "yes";
    });

    funcAttempt += 1;
    if (funcAttempt === 1) {
      throw new Error("broken func");
    }
    funcAttempt = 0;
    stepFirstCall = true;

    return { name: event.name, body: "ok" };
  }
);
