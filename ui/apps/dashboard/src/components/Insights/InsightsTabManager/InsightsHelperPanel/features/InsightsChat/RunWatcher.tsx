import { useQuery } from '@tanstack/react-query';
import { useEffect } from 'react';

import { fetchRunResult } from './useInsightsAgent';
import type { RunRef } from './types';

// The lookup is uncached, and every Insights user shares one per-account bucket
// of 300 requests / 5s. A waiting chat costs 0.5 of that per second.
const POLL_INTERVAL_MS = 2_000;

type Props = {
  run: RunRef;
  threadId: string;
  turnId: string;
  // Identify the turn rather than closing over it: unstable callbacks would
  // re-run the effect below on every parent render, outside of a fresh poll.
  onResult: (
    threadId: string,
    turnId: string,
    output: Record<string, unknown>,
  ) => void;
  onFail: (threadId: string, turnId: string, message: string) => void;
};

/**
 * Watches one in-flight agent run and reports its result.
 *
 * The run's output *is* the answer: there is no second delivery path, so a
 * hidden tab, a dropped connection or a reload costs latency and nothing else.
 * Mounting starts the watch and unmounting stops it, so the provider only has to
 * settle the turn.
 *
 * An answer arriving is the success signal, and a run reported as over is the
 * only thing that ends the wait without one. See runLookup for why those two
 * facts come from different places, and fetchRunResult for how an answer is
 * told apart from the step output of a run still working.
 */
export function RunWatcher({ run, threadId, turnId, onResult, onFail }: Props) {
  // Structural sharing keeps `data` identical across unchanged polls, so the
  // timestamp is what tells us a fresh poll landed.
  const { data, dataUpdatedAt } = useQuery({
    queryKey: ['insights-chat-run', run.eventId],
    queryFn: () => fetchRunResult(run, threadId),
    refetchInterval: POLL_INTERVAL_MS,
    // A hidden tab is the case that started all this: React Query would
    // otherwise hold the interval until the window is focused again, so a run
    // that finishes while the user is elsewhere would sit there unanswered.
    refetchIntervalInBackground: true,
    // And the two moments a waiting answer is worth chasing straight away.
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
    gcTime: 0,
  });

  useEffect(() => {
    if (!data) return;

    if (data.answer) {
      onResult(threadId, turnId, data.answer);
      return;
    }

    if (data.failed) {
      onFail(
        threadId,
        turnId,
        'The Insights agent could not complete this request. Please try again.',
      );
    }

    // Otherwise the answer is still coming, and there's no attempt cap: a run
    // this thread is waiting on either answers or reports that it can't.
  }, [data, dataUpdatedAt, threadId, turnId, onResult, onFail]);

  return null;
}
