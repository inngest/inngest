import { useQuery } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';

import { fetchRunResult } from './useInsightsAgent';

// The runs endpoint is cached and shares one rate-limit bucket across every
// Insights user, so there's nothing to gain from polling faster than this.
const POLL_INTERVAL_MS = 20_000;

// Run status and run output are read from different stores, so a run can
// report Completed a moment before its output is readable. Give that a few
// polls before calling the result lost.
const EMPTY_TERMINAL_POLL_LIMIT = 3;

type Props = {
  eventId: string;
  threadId: string;
  onResult: (threadId: string, output: Record<string, unknown>) => void;
  onFail: (threadId: string, message: string) => void;
};

/**
 * Watches one in-flight agent run and reports its result.
 *
 * Realtime has no replay: a run that finishes while the browser is hidden or
 * disconnected publishes a `run.completed` that nobody receives, and the thread
 * spins forever. The run's own output is the same payload that publish carries,
 * so we read it back from the run instead.
 *
 * Mounting starts the watch and unmounting stops it, so the provider only has
 * to add and remove the thread from its pending map.
 */
export function RunResultWatcher({
  eventId,
  threadId,
  onResult,
  onFail,
}: Props) {
  // Structural sharing keeps `data` identical across unchanged polls, so the
  // timestamp is what tells us a fresh poll landed.
  const { data, dataUpdatedAt } = useQuery({
    queryKey: ['insights-chat-result', eventId],
    queryFn: () => fetchRunResult(eventId, threadId),
    refetchInterval: POLL_INTERVAL_MS,
    // The two moments a missed message is worth chasing straight away.
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
    gcTime: 0,
  });

  const emptyTerminalPolls = useRef(0);

  useEffect(() => {
    if (!data) return;

    if (data.status === 'Completed' && data.output) {
      onResult(threadId, data.output);
      return;
    }

    if (data.status === 'Failed' || data.status === 'Cancelled') {
      onFail(
        threadId,
        'The Insights agent could not complete this request. Please try again.',
      );
      return;
    }

    if (data.status === 'Completed') {
      emptyTerminalPolls.current += 1;
      if (emptyTerminalPolls.current >= EMPTY_TERMINAL_POLL_LIMIT) {
        onFail(
          threadId,
          "This request finished but its result couldn't be read back. Please try sending it again.",
        );
      }
    }

    // Still Running: nothing to report, and no attempt cap — the answer is
    // still coming.
  }, [data, dataUpdatedAt, threadId, onResult, onFail]);

  return null;
}
