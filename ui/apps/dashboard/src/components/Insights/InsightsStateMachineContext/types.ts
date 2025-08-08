export type InsightsAction =
  | { type: 'START_QUERY'; payload: string }
  | { type: 'QUERY_SUCCESS'; payload: InsightsFetchResult }
  | { type: 'QUERY_ERROR'; payload: string }
  | { type: 'FETCH_MORE' }
  | { type: 'FETCH_MORE_SUCCESS'; payload: InsightsFetchResult }
  | { type: 'FETCH_MORE_ERROR'; payload: string };

export interface InsightsFetchResult {
  columns: Array<{
    name: string;
    type: 'date' | 'string' | 'number';
  }>;
  entries: Array<{
    id: string;
    isLoadingRow: boolean | undefined;
    values: Record<string, null | Date | string | number>;
  }>;
  pageInfo: {
    endCursor: string | null;
    hasNextPage: boolean;
    hasPreviousPage: boolean;
    startCursor: string | null;
  };
  totalCount: number;
}

export interface InsightsState {
  data: InsightsFetchResult | undefined;
  error: string | undefined;
  fetchMoreError: string | undefined;
  /**
   * The query that was last sent to the server. The user may have edited the
   * query in the editor, but fetching more data should still use this query.
   */
  lastSentQuery: string;
  status: InsightsStatus;
}

export type InsightsStatus =
  | 'error'
  | 'fetchingMore'
  | 'fetchMoreError'
  | 'initial'
  | 'loading'
  | 'success';
