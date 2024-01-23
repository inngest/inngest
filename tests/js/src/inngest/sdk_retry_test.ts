import { inngest } from "@/inngest/client";

let attempt = 0;

export const testRetry = inngest.createFunction(
  { id: "retry-test" },
  { event: "tests/retry.test" },
  async ({ event, step }) => {

    const data = await step.run("first step", async () => {
      attempt += 1;
      switch (attempt) {
      case 1:
        throw new Error("broken");
      default:
        const res = "yes: " + attempt;
        attempt = 0; // reset
        return res;
      }
    });

    attempt += 1;
    switch (attempt) {
    case 1:
      throw new Error("broken func");
    default:
      attempt = 0; // reset
    }

    return { name: event.name, body: "ok" };
  }
);

