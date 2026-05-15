export interface InsightsFetchResult {
  columns: Array<{
    name: string;
    type: 'date' | 'string' | 'number';
  }>;
  rows: Array<{
    id: string;
    values: Record<string, null | Date | string | number>;
  }>;
  diagnostics: Array<InsightsDiagnostic>;
}

export interface InsightsDiagnostic {
  position?: {
    start: number;
    end: number;
    context: string;
  };
  severity: InsightsDiagnosticSeverity;
  code: InsightsDiagnosticCode;
  message: string;
}

export type InsightsDiagnosticSeverity = 'none' | 'info' | 'warning' | 'error';

export type InsightsDiagnosticCode = string; // TODO: Define specific diagnostic codes

export type InsightsStatus = 'error' | 'initial' | 'loading' | 'success';
