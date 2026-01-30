export interface InsightsFetchResult {
  columns: Array<{
    name: string;
    type: 'date' | 'string' | 'number';
  }>;
  rows: Array<{
    id: string;
    values: Record<string, null | Date | string | number>;
  }>;
  diagnostics: Array<{
    position?: {
      start: number;
      end: number;
      context: string;
    };
    severity: 'ERROR' | 'WARNING' | 'INFO' | 'NONE';
    message: string;
  }>;
}

export type InsightsStatus = 'error' | 'initial' | 'loading' | 'success';
