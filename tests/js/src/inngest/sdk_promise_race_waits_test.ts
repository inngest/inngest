import { inngest } from "@/inngest/client";

// Regression: Promise.race over parallel step.waitForEvent calls used to
// hang — when one wait resolved, the others were still tracked as pending,
// so the function never continued past the race.
export const testPromiseRaceWaits = inngest.createFunction(
  { id: "promise-race-waits" },
  { event: "tests/promise-race-waits.test" },
  async ({ step }) => {
    const winner: any = await Promise.race([
      step.waitForEvent("answer", {
        event: "tests/promise-race-waits.answer",
        if: "async.data.id == event.data.id",
        timeout: "5m",
      }),
      step.waitForEvent("completed", {
        event: "tests/promise-race-waits.completed",
        if: "async.data.id == event.data.id",
        timeout: "5m",
      }),
      step.waitForEvent("deleted", {
        event: "tests/promise-race-waits.deleted",
        if: "async.data.id == event.data.id",
        timeout: "5m",
      }),
    ]);

    return { winner: winner?.name ?? null };
  }
);
