import { useRealtime } from 'inngest/react';
import { useCallback, useMemo } from 'react';

import { insightsChannel } from '@/lib/inngest/realtime';

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
 * Thin hook wrapping useRealtime for the insights agent channel.
 * Handles token fetching and channel setup.
 */
export function useInsightsRealtime({
  channelKey,
  enabled = true,
}: {
  channelKey?: string;
  enabled?: boolean;
}) {
  const channel = useMemo(
    () => (channelKey ? insightsChannel(channelKey) : undefined),
    [channelKey],
  );

  const tokenFactory = useCallback(async () => {
    if (!channelKey) throw new Error('No channel key');
    const res = await fetch('/api/realtime/token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ channelKey }),
    });
    if (!res.ok) throw new Error('Failed to get subscription token');
    return res.json();
  }, [channelKey]);

  return useRealtime({
    channel,
    topics: ['agent_stream'] as const,
    token: channelKey ? tokenFactory : undefined,
    enabled: enabled && !!channelKey,
    autoCloseOnTerminal: false,
    reconnect: true,
    historyLimit: 200,
    // Hiding the tab would otherwise tear the subscription down entirely, and
    // realtime has no replay: anything the run publishes while we're away is
    // lost. Agent runs routinely outlast a tab switch.
    pauseOnHidden: false,
  });
}

/**
 * Send a chat message to the insights agent backend.
 */
export async function sendChatMessage(params: {
  content: string;
  messageId: string;
  threadId: string;
  userId: string;
  channelKey?: string;
  state?: Record<string, unknown>;
  history?: Array<Record<string, unknown>>;
}): Promise<{ success: boolean; threadId?: string; eventId?: string }> {
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
  status: string;
  output: Record<string, unknown> | null;
};

/**
 * Read a chat run back from its triggering event, for when the realtime
 * run.completed never arrived. Resolves to null when the run can't be
 * determined, which the caller treats the same as a failure.
 */
export async function fetchRunResult(
  eventId: string,
  threadId: string,
): Promise<RunResult | null> {
  const res = await fetch('/api/chat-result', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ eventId }),
  });
  if (!res.ok) return null;

  const { runs } = (await res.json()) as { runs?: RunResult[] };
  if (!runs?.length) return null;

  // One event triggers one agent run today, but match on the thread so a
  // second run on the same event could never answer for this one.
  return (
    runs.find(
      (run) =>
        run.output &&
        typeof run.output === 'object' &&
        (run.output as { threadId?: unknown }).threadId === threadId,
    ) ??
    runs[0] ??
    null
  );
}
