export type InsightsAction =
  | { type: 'START_QUERY'; payload: string }
  | { type: 'QUERY_SUCCESS'; payload: InsightsFetchResult }
  | { type: 'QUERY_ERROR'; payload: string };

export interface InsightsFetchResult {
  columns: Array<{
    name: string;
    type: 'date' | 'string' | 'number';
  }>;
  rows: Array<{
    id: string;
    values: Record<string, null | Date | string | number>;
  }>;
}

export interface InsightsState {
  /**
   * The query that was last sent to the server. The user may have edited the
   * query in the editor, but fetching more data should still use this query.
   */
  activeQuery: string;
  data: undefined | InsightsFetchResult;
  error: undefined | string;
  query: string;
  status: InsightsStatus;
}

export type InsightsStatus = 'error' | 'initial' | 'loading' | 'success';
