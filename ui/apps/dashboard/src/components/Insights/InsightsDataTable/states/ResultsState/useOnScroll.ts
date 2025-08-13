'use client';

import { useCallback, type UIEventHandler } from 'react';

import type {
  InsightsFetchResult,
  InsightsStatus,
} from '@/components/Insights/InsightsStateMachineContext/types';

// TODO: Handle case where a single page of data doesn't fill the container.
// When this happens, the user will not be able to scroll, and will not be able to load more data.
// For now, we'll just fetch 30 entries per page, mitigating the issue on any reasonable screen size.

export function useOnScroll(
  data: InsightsFetchResult | undefined,
  status: InsightsStatus,
  fetchMore: () => void
): { onScroll: UIEventHandler<HTMLDivElement> } {
  const onScroll: UIEventHandler<HTMLDivElement> = useCallback(
    (event) => {
      handleScroll(event, {
        fetchMore,
        hasEntries: Boolean(data?.entries.length),
        hasNextPage: Boolean(data?.pageInfo.hasNextPage),
        status,
      });
    },
    [fetchMore, data?.pageInfo.hasNextPage, data?.entries.length, status]
  );

  return { onScroll };
}

const SCROLL_THRESHOLD = 200;

export function handleScroll(
  event: React.UIEvent<HTMLDivElement>,
  options: {
    hasEntries: boolean;
    hasNextPage: boolean;
    status: string;
    fetchMore: () => void;
  }
) {
  const { fetchMore, hasEntries, hasNextPage, status } = options;

  if (hasEntries && hasNextPage && status === 'success') {
    const { scrollHeight, scrollTop, clientHeight } = event.target as HTMLDivElement;

    const reachedBottom = scrollHeight - scrollTop - clientHeight < SCROLL_THRESHOLD;
    if (reachedBottom) fetchMore();
  }
}
