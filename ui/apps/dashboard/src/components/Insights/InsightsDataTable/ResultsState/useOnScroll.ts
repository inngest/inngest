import { useCallback, type UIEventHandler } from 'react';

import { type InsightsResult, type InsightsState } from '../types';

// TODO: Handle case where data doesn't fill viewport on tall monitors
// When content is short and doesn't trigger scroll, users can't access more data.

export function useOnScroll(
  data: InsightsResult | undefined,
  state: InsightsState,
  fetchMore: () => void
): { onScroll: UIEventHandler<HTMLDivElement> } {
  const onScroll: UIEventHandler<HTMLDivElement> = useCallback(
    (event) => {
      handleScroll(event, {
        fetchMore,
        hasEntries: Boolean(data?.entries.length),
        hasNextPage: Boolean(data?.pageInfo.hasNextPage),
        state,
      });
    },
    [fetchMore, data?.pageInfo.hasNextPage, data?.entries.length, state]
  );

  return { onScroll };
}

const SCROLL_THRESHOLD = 200;

export function handleScroll(
  event: React.UIEvent<HTMLDivElement>,
  options: {
    hasEntries: boolean;
    hasNextPage: boolean;
    state: string;
    fetchMore: () => void;
  }
) {
  const { hasEntries, hasNextPage, state, fetchMore } = options;

  if (hasEntries && hasNextPage && state === 'success') {
    const { scrollHeight, scrollTop, clientHeight } = event.target as HTMLDivElement;

    const reachedBottom = scrollHeight - scrollTop - clientHeight < SCROLL_THRESHOLD;
    if (reachedBottom) fetchMore();
  }
}
