# Use Tuning Engines as a governed AI endpoint

This example shows how to call a Tuning Engines OpenAI-compatible endpoint from
an Inngest function step. Inngest owns durable execution, retries, and event
flow. Tuning Engines owns model routing, policy checks, budgets, audit logs,
and runtime trace correlation.

Use this pattern when your function needs a governed model endpoint instead of
calling a model provider directly.

## Environment

```bash
export TE_INFERENCE_KEY=sk-te-your-inference-key
export TE_MODEL=auto
```

## Function

```ts
import { Inngest } from "inngest";

export const inngest = new Inngest({ id: "support-workflows" });

export const triageTicket = inngest.createFunction(
  { id: "triage-ticket-with-tuning-engines" },
  { event: "ticket/created" },
  async ({ event, step, runId }) => {
    const teRunId = `inngest_${runId}`;

    const response = await step.run("governed-model-call", async () => {
      const res = await fetch("https://api.tuningengines.com/v1/chat/completions", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${process.env.TE_INFERENCE_KEY}`,
          "Content-Type": "application/json",
          "X-TE-Run-ID": teRunId,
        },
        body: JSON.stringify({
          model: process.env.TE_MODEL || "auto",
          messages: [
            {
              role: "user",
              content: `Triage this ticket: ${event.data.ticketText}`,
            },
          ],
          metadata: {
            run_id: teRunId,
            request_id: crypto.randomUUID(),
            runtime: "inngest",
            event_type: "model.call",
          },
        }),
      });

      if (!res.ok) {
        throw new Error(`Tuning Engines request failed: ${res.status}`);
      }

      return res.json();
    });

    return { runId: teRunId, response };
  },
);
```

## Notes

- Keep the inference key in your normal secrets manager or deployment
  environment.
- Use the Inngest run id as the Tuning Engines `run_id` so model usage, policy
  decisions, approvals, and traces can be correlated.
- If a Tuning Engines policy requires approval, the endpoint returns an
  approval-required response. Approve it in Tuning Engines, then retry the same
  step with the returned approval id in the request headers.
