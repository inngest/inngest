export type ColumnType = 'Date' | 'string' | 'number';

export interface InsightsColumn {
  name: string;
  type: ColumnType;
}

export interface InsightsEntry {
  id: string;
  values: Record<string, Date | string | number | null>;
}

export interface PageInfo {
  endCursor: string | null;
  hasNextPage: boolean;
  hasPreviousPage: boolean;
  startCursor: string | null;
}

export interface InsightsResult {
  columns: InsightsColumn[];
  entries: InsightsEntry[];
  pageInfo: PageInfo;
  totalCount: number;
}

export type InsightsState = 'initial' | 'loading' | 'success' | 'error' | 'fetchingMore';
