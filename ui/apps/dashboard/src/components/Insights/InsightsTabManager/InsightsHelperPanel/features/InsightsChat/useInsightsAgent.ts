import type { RunRef } from './types';

export type ClientState = {
  sqlQuery: string;
  eventTypes: string[];
  schemas: { name: string; schema: string }[];
  currentQuery: string;
  tabTitle: string;
  mode: 'insights_sql_playground';
  timestamp: number;
};

/**
 * Send a chat message to the insights agent backend. The run reference it
 * returns is how the answer gets read back — see RunWatcher.
 */
export async function sendChatMessage(params: {
  content: string;
  messageId: string;
  threadId: string;
  userId: string;
  channelKey?: string;
  state?: Record<string, unknown>;
  history?: Array<Record<string, unknown>>;
}): Promise<{ success: boolean; threadId?: string } & Partial<RunRef>> {
  const res = await fetch('/api/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      userMessage: {
        id: params.messageId,
        content: params.content,
        role: 'user',
        state: params.state,
      },
      threadId: params.threadId,
      userId: params.userId,
      channelKey: params.channelKey,
      history: params.history,
    }),
  });

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || 'Failed to send message');
  }

  return res.json();
}

export type RunResult = {
  answer: Record<string, unknown> | null;
  failed: boolean;
};

/**
 * Read an agent run back from the event that triggered it. Resolves to null
 * when the lookup itself fails, which the caller treats as "still running".
 *
 * While a run is in flight its output holds whichever step finished last, so an
 * answer has to be recognised rather than assumed: only the function's own
 * return value carries the thread id it was asked about.
 */
export async function fetchRunResult(
  run: RunRef,
  threadId: string,
): Promise<RunResult | null> {
  const res = await fetch('/api/chat-result', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(run),
  });
  if (!res.ok) return null;

  const { runs, failed } = (await res.json()) as {
    runs?: { output: Record<string, unknown> | null }[];
    failed?: boolean;
  };

  const answer =
    (runs ?? []).find(
      (r) => (r.output as { threadId?: unknown } | null)?.threadId === threadId,
    )?.output ?? null;

  return { answer, failed: Boolean(failed) };
}
