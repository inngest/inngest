'use server';

import { auth } from '@clerk/nextjs/server';

export async function generateActorToken(actorId: string, userId: string) {
  const user = auth();
  if (!user.userId) {
    return {
      ok: false,
      error: 'You do not have permission to access this page.',
    };
  }

  const INNGEST_ORG_ID = process.env.CLERK_INNGEST_ORG_ID;

  if (!INNGEST_ORG_ID) {
    return {
      ok: false,
      error: 'Missing CLERK_INNGEST_ORG_ID env variable',
    };
  }

  if (user.orgId !== INNGEST_ORG_ID) {
    return {
      ok: false,
      error: 'You do not have permission to access this page.',
    };
  }

  const params = JSON.stringify({
    user_id: userId,
    actor: {
      sub: actorId,
    },
  });

  if (!process.env.CLERK_SECRET_KEY) {
    return { ok: false, error: 'Missing CLERK_SECRET_KEY env variable' };
  }

  let res: Response;
  try {
    res = await fetch('https://api.clerk.com/v1/actor_tokens', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${process.env.CLERK_SECRET_KEY}`,
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: params,
      cache: 'no-store',
    });
  } catch (e) {
    return { ok: false, error: 'Network error while contacting Clerk' };
  }

  if (!res.ok) {
    return { ok: false, error: 'Failed to generate actor token' };
  }
  const data = await res.json();

  return { ok: true, error: null, token: data.token };
}
