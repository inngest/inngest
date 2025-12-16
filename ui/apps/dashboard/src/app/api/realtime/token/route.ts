/**
 * Route: POST /api/realtime/token
 *
 * Issues a short‑lived Inngest Realtime subscription token for an authenticated
 * Clerk user. The request must include a JSON body containing a `channelKey`.
 * The created token is subscribed to the `agent_stream` topic for the resolved
 * channel returned by `createChannel(channelKey)`.
 *
 * Request body:
 *   { channelKey: string }
 *
 * Success response:
 *   200 OK with JSON token payload returned by `getSubscriptionToken`.
 *
 * Error responses:
 *   401 if the user is not authenticated.
 *   400 if `channelKey` is missing.
 *   500 if token creation fails.
 */
import { NextRequest, NextResponse } from 'next/server';
import { auth } from '@clerk/nextjs/server';
import { getSubscriptionToken } from '@inngest/realtime';

import { inngest } from '@/lib/inngest/client';
import { createChannel } from '@/lib/inngest/realtime';

export type RequestBody = {
  userId?: string;
  channelKey?: string;
};

export async function POST(req: NextRequest) {
  // Authenticate the user using Clerk
  const { userId } = auth();
  if (!userId) {
    return NextResponse.json({ error: 'Please sign in to create a token' }, { status: 401 });
  }

  try {
    // 1. Get the channel key from the request body and validate it
    const { channelKey } = (await req.json()) as RequestBody;
    if (!channelKey) {
      return NextResponse.json({ error: 'channelKey is required' }, { status: 400 });
    }

    // 2. Create a subscription token for the resolved channel
    //    Match publisher semantics: when channelKey is provided, we publish to that key directly.
    const token = await getSubscriptionToken(inngest, {
      channel: createChannel(channelKey),
      topics: ['agent_stream'],
    });

    // 3. Return the token
    return NextResponse.json(token);
  } catch (error) {
    // Return an error if the token creation fails
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to create subscription token' },
      { status: 500 }
    );
  }
}
