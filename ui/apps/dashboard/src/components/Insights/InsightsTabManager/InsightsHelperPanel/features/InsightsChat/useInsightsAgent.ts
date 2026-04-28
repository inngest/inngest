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
}): Promise<{ success: boolean; threadId?: string }> {
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
