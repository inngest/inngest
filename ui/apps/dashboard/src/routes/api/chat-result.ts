import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';
import { z } from 'zod/v3';

import { verifyReceipt } from '@/lib/inngest/chatReceipt';
import { fetchEventRunResult } from '@/lib/inngest/runLookup';

//
// How the Insights chat gets its answer. /api/chat returns the id of the event
// it sent plus a receipt; the UI polls here until the run reports a result.
// The run's output is the answer — there is no second delivery path — so a
// hidden tab, a dropped connection or a reload costs latency, never the result.
const requestSchema = z.object({
  eventId: z.string().min(1).max(64),
  receipt: z.string().min(1).max(128),
});

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

        // The receipt is re-derived from the Clerk session, never from the
        // request body, so it only verifies for the user who triggered the run.
        const { eventId, receipt } = parsed.data;
        if (!verifyReceipt(eventId, userId, receipt)) {
          return json({ error: 'Not found' }, 404);
        }

        return json(await fetchEventRunResult({ eventId }));
      },
    },
  },
});
