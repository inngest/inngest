export interface InsightsFetchResult {
  columns: Array<{
    name: string;
    type: "date" | "string" | "number";
  }>;
  rows: Array<{
    id: string;
    values: Record<string, null | Date | string | number>;
  }>;
}

export type InsightsStatus = "error" | "initial" | "loading" | "success";
