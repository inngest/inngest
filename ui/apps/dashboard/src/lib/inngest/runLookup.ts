//
// Server-only: reads INNGEST_SIGNING_KEY. Never import this from client code.
//
// Reads an agent chat run back from its triggering event, for when the realtime
// run.completed never reached the browser.
//
// Every tenant's Insights chats run through one shared Inngest app, so the
// signing key scopes a lookup to that app and nothing finer. Ownership has to
// be enforced here: /api/chat stamps userId into the event from the Clerk
// session (never from the request body), so an event is only readable by the
// user who triggered it.

// A local dev server serves the same endpoints unauthenticated, so the signing
// key is only required against the real API.
const IS_DEV = Boolean(process.env.INNGEST_DEV);

const INNGEST_API_BASE_URL =
  process.env.INNGEST_API_BASE_URL ??
  (IS_DEV ? 'http://localhost:8288' : 'https://api.inngest.com');

export type EventRun = {
  status: string;
  output: Record<string, unknown> | null;
};

type FetchLike = typeof fetch;

async function getJson(
  path: string,
  signingKey: string | undefined,
  fetchImpl: FetchLike,
): Promise<unknown | null> {
  const res = await fetchImpl(`${INNGEST_API_BASE_URL}${path}`, {
    headers: signingKey ? { Authorization: `Bearer ${signingKey}` } : {},
  });
  if (!res.ok) return null;
  return res.json();
}

/**
 * Runs for an event, but only if the authenticated user is the one who
 * triggered it. Returns [] for "no such event", "not yours", and "lookup
 * failed" alike, so the caller can't use this to probe for other users' events.
 */
export async function fetchRunsForEvent({
  eventId,
  userId,
  signingKey = process.env.INNGEST_SIGNING_KEY,
  fetchImpl = fetch,
}: {
  eventId: string;
  userId: string;
  signingKey?: string;
  fetchImpl?: FetchLike;
}): Promise<EventRun[]> {
  if ((!signingKey && !IS_DEV) || !userId) return [];

  const encodedId = encodeURIComponent(eventId);

  try {
    const event = (await getJson(
      `/v1/events/${encodedId}`,
      signingKey,
      fetchImpl,
    )) as { data?: { data?: { userId?: unknown } } } | null;

    if (event?.data?.data?.userId !== userId) return [];

    const runs = (await getJson(
      `/v1/events/${encodedId}/runs`,
      signingKey,
      fetchImpl,
    )) as { data?: { status?: string; output?: unknown }[] } | null;

    return (runs?.data ?? []).map((run) => ({
      status: run.status ?? 'Unknown',
      output: (run.output ?? null) as Record<string, unknown> | null,
    }));
  } catch {
    return [];
  }
}
