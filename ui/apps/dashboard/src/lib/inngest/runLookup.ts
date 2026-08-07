//
// Server-only: reads INNGEST_SIGNING_KEY. Never import this from client code.
//
// Reads an agent chat run back from its triggering event. This is the only way
// the chat UI learns its answer: the run's output is the result.
//
// It takes two endpoints, because each API version is good at one half of it:
//
//   - v2 event-runs carries `includeOutput` all the way to the reader and is
//     uncached, so an answer is readable within a poll of the run producing it.
//     Its `status`, though, comes from the run list and drifts: it reads RUNNING
//     for a run whose answer is already readable, flips to COMPLETED and back,
//     and never reports a failed run as FAILED.
//   - v1 event-runs re-reads the root span to correct the status, which is the
//     only trustworthy "this run is over" signal. It sits behind a 15s response
//     cache, which is why the answer doesn't come from here.
//
// So: v2 says what the answer is, v1 says when to stop waiting for one.
const IS_DEV = Boolean(process.env.INNGEST_DEV);

const INNGEST_API_BASE_URL =
  process.env.INNGEST_API_BASE_URL ??
  (IS_DEV ? 'http://localhost:8288' : 'https://api.inngest.com');

export type EventRunResult = {
  // Run outputs for the event. While a run is in flight this holds whichever
  // step finished last, so the caller has to recognise its own answer.
  runs: { output: Record<string, unknown> | null }[];
  // Every run for this event is over without answering.
  failed: boolean;
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
 * Runs for an event, with their output, and whether they've given up.
 *
 * An unreadable lookup answers "nothing yet, still going", so a blip costs a
 * poll rather than the result.
 *
 * Callers are responsible for proving the event belongs to the requesting user
 * before calling this; see chatReceipt.
 */
export async function fetchEventRunResult({
  eventId,
  signingKey = process.env.INNGEST_SIGNING_KEY,
  fetchImpl = fetch,
}: {
  eventId: string;
  signingKey?: string;
  fetchImpl?: FetchLike;
}): Promise<EventRunResult> {
  if (!signingKey && !IS_DEV) return { runs: [], failed: false };

  const id = encodeURIComponent(eventId);

  try {
    const [output, corrected] = await Promise.all([
      getJson(
        `/api/v2/events/${id}/runs?includeOutput=true`,
        signingKey,
        fetchImpl,
      ) as Promise<{
        data?: { output?: unknown }[];
      } | null>,
      getJson(`/v1/events/${id}/runs`, signingKey, fetchImpl) as Promise<{
        data?: { status?: string }[];
      } | null>,
    ]);

    const statuses = (corrected?.data ?? []).map((run) => run.status);

    return {
      runs: (output?.data ?? []).map((run) => ({
        output: (run.output ?? null) as Record<string, unknown> | null,
      })),
      failed:
        statuses.length > 0 &&
        statuses.every((s) => s === 'Failed' || s === 'Cancelled'),
    };
  } catch {
    return { runs: [], failed: false };
  }
}
