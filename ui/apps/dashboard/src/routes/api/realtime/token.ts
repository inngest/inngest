import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';
import { getClientSubscriptionToken } from 'inngest/react';
import { inngest } from '@/lib/inngest/client';
import {
  insightsChannel,
  insightsUserChannelKey,
} from '@/lib/inngest/realtime';

export const Route = createFileRoute('/api/realtime/token')({
  server: {
    handlers: {
      POST: async () => {
        //
        // Authenticate the user using Clerk
        const { userId } = await auth();
        if (!userId) {
          return new Response(
            JSON.stringify({ error: 'Please sign in to create a token' }),
            {
              status: 401,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }

        try {
          //
          // Create a subscription token for the authenticated user's channel
          const token = await getClientSubscriptionToken(inngest, {
            channel: insightsChannel(insightsUserChannelKey(userId)),
            topics: ['agent_stream'],
          });

          return new Response(JSON.stringify(token), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        } catch (error) {
          return new Response(
            JSON.stringify({
              error:
                error instanceof Error
                  ? error.message
                  : 'Failed to create subscription token',
            }),
            {
              status: 500,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }
      },
    },
  },
});
