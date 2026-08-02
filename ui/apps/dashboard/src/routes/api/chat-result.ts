import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';
import { z } from 'zod/v3';

//
// Recovery path for the agent chat. Realtime has no replay: if the browser is
// disconnected when the run publishes run.completed, that message is gone for
// good and the UI would spin forever. The run's own output is the same payload
// the publish carries, so the chat UI polls here with the event id it got back
// from /api/chat and reconciles from the run instead.
const requestSchema = z.object({
  eventId: z.string().min(1).max(64),
});

const INNGEST_API_BASE_URL =
  process.env.INNGEST_API_BASE_URL ?? 'https://api.inngest.com';

type EventRun = {
  status?: string;
  output?: unknown;
};

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

export const Route = createFileRoute('/api/chat-result')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        const { userId } = await auth();
        if (!userId) return json({ error: 'Please sign in' }, 401);

        const body: unknown = await request.json().catch(() => null);
        const parsed = requestSchema.safeParse(body);
        if (!parsed.success) {
          return json(
            { error: parsed.error.errors[0]?.message ?? 'Invalid request' },
            400,
          );
        }

        const signingKey = process.env.INNGEST_SIGNING_KEY;
        if (!signingKey) return json({ runs: [] });

        try {
          // The signing key scopes this to the agent app's own environment, so
          // the lookup can only ever see this deployment's runs.
          const res = await fetch(
            `${INNGEST_API_BASE_URL}/v1/events/${encodeURIComponent(
              parsed.data.eventId,
            )}/runs`,
            { headers: { Authorization: `Bearer ${signingKey}` } },
          );
          if (!res.ok) return json({ runs: [] });

          const payload = (await res.json()) as { data?: EventRun[] };
          const runs = (payload.data ?? []).map((run) => ({
            status: run.status ?? 'Unknown',
            output: run.output ?? null,
          }));

          return json({ runs });
        } catch {
          // A failed lookup is not a UI error: the caller falls back to telling
          // the user to retry rather than surfacing a 500.
          return json({ runs: [] });
        }
      },
    },
  },
});
