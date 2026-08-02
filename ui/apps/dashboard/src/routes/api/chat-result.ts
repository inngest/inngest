import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';
import { z } from 'zod/v3';

import { fetchRunsForEvent } from '@/lib/inngest/runLookup';

//
// Recovery path for the agent chat. Realtime has no replay: if the browser is
// disconnected when the run publishes run.completed, that message is gone for
// good and the UI would spin forever. The run's own output is the same payload
// the publish carries, so the chat UI polls here with the event id it got back
// from /api/chat.
//
// The lookup is pinned to the caller's Clerk session, the same way
// chat-validate pins its write, because one shared Inngest app serves every
// tenant's chats.
const requestSchema = z.object({
  eventId: z.string().min(1).max(64),
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

        // userId comes from the Clerk session, never the request body. An
        // unowned or unknown event returns an empty list either way, so this
        // can't be used to probe for other users' events.
        const runs = await fetchRunsForEvent({
          eventId: parsed.data.eventId,
          userId,
        });

        return json({ runs });
      },
    },
  },
});
