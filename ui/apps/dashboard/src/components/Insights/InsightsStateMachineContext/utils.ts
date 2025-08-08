import type { InfiniteData } from '@tanstack/react-query';

import type { InsightsFetchResult, InsightsStatus } from './types';

export function getInsightsStatus(
  isError: boolean,
  isLoading: boolean,
  isFetching: boolean,
  data: InsightsFetchResult | undefined,
  fetchMoreError: string | undefined
): InsightsStatus {
  if (isError) return 'error';
  if (isLoading) return 'loading';
  if (isFetching && data !== undefined) return 'fetchingMore';
  if (fetchMoreError) return 'fetchMoreError';
  if (data !== undefined) return 'success';
  return 'initial';
}

export function selectInsightsData(
  infiniteData: InfiniteData<InsightsFetchResult, unknown>
): undefined | InsightsFetchResult {
  if (!infiniteData?.pages?.length) return undefined;

  const firstPage = infiniteData.pages[0];
  if (firstPage === undefined) return undefined;

  const lastPage = infiniteData.pages[infiniteData.pages.length - 1];
  if (lastPage === undefined) return undefined;

  return {
    columns: firstPage.columns,
    entries: infiniteData.pages.flatMap((page) => page.entries),
    pageInfo: lastPage.pageInfo,
    totalCount: firstPage.totalCount,
  };
}
